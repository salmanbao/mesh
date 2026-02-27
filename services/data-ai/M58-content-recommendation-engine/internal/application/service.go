package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/domain"
	"github.com/viralforge/mesh/services/data-ai/M58-content-recommendation-engine/internal/ports"
)

func (s *Service) GetRecommendations(ctx context.Context, actor Actor, input GetRecommendationsInput) (domain.RecommendationResponse, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.RecommendationResponse{}, domain.ErrUnauthorized
	}

	role := strings.TrimSpace(input.Role)
	if role == "" {
		role = actor.Role
	}
	role = normalizeRole(role)
	limit := normalizeLimit(input.Limit)
	segment := strings.TrimSpace(input.Segment)
	now := s.nowFn()

	if s.recommendations != nil {
		if batch, err := s.recommendations.GetLatestBatch(ctx, actor.SubjectID, role); err == nil {
			if now.Sub(batch.ComputedAt) <= s.cfg.RecommendationTTL {
				return domain.RecommendationResponse{
					Recommendations: trimRecommendations(batch.Recommendations, limit),
					Meta: domain.RecommendationMeta{
						ComputedAt:         batch.ComputedAt,
						CacheHit:           true,
						ModelVersion:       batch.ModelVersion,
						RecommendationMode: "cached",
					},
				}, nil
			}
		}
	}

	modelVersion := "v1.0.0"
	if s.models != nil {
		if model, err := s.models.GetDefault(ctx); err == nil && strings.TrimSpace(model.Version) != "" {
			modelVersion = model.Version
		}
	}
	if s.abTests != nil {
		_, _ = s.abTests.GetOrAssign(ctx, actor.SubjectID, now)
	}

	candidates := []domain.Recommendation{}
	if s.campaignDiscovery != nil {
		items, err := s.campaignDiscovery.ListCandidateCampaigns(ctx, actor.SubjectID, role, segment, maxInt(limit*3, 20))
		if err != nil {
			return domain.RecommendationResponse{}, err
		}
		candidates = rankCandidates(items, actor.SubjectID, role, modelVersion, now)
	}
	if len(candidates) == 0 {
		candidates = fallbackRecommendations(actor.SubjectID, role, modelVersion, now, limit)
	}

	overrides := []domain.RecommendationOverride{}
	if s.overrides != nil {
		active, err := s.overrides.ListActive(ctx, role, now)
		if err == nil {
			overrides = active
		}
	}
	candidates = applyOverrides(candidates, overrides)
	candidates = trimRecommendations(candidates, limit)
	for i := range candidates {
		candidates[i].Position = i + 1
	}

	batch := domain.RecommendationBatch{
		BatchID:         uuid.NewString(),
		UserID:          actor.SubjectID,
		Role:            role,
		Recommendations: candidates,
		ModelVersion:    modelVersion,
		ComputedAt:      now,
		CacheHit:        false,
	}
	if s.recommendations != nil {
		if err := s.recommendations.SaveBatch(ctx, batch); err != nil {
			return domain.RecommendationResponse{}, err
		}
	}
	if len(candidates) > 0 {
		_ = s.enqueueRecommendationGenerated(ctx, actor, batch, candidates[0])
	}

	return domain.RecommendationResponse{
		Recommendations: candidates,
		Meta: domain.RecommendationMeta{
			ComputedAt:         now,
			CacheHit:           false,
			ModelVersion:       modelVersion,
			RecommendationMode: "personalized",
		},
	}, nil
}

func (s *Service) RecordFeedback(ctx context.Context, actor Actor, recommendationID string, input FeedbackInput) (domain.FeedbackRecord, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.FeedbackRecord{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.FeedbackRecord{}, domain.ErrIdempotencyRequired
	}
	if strings.TrimSpace(recommendationID) == "" {
		return domain.FeedbackRecord{}, domain.ErrInvalidInput
	}
	feedbackEventType := domain.NormalizeFeedbackEvent(input.EventType)
	if feedbackEventType == "" || strings.TrimSpace(input.EntityID) == "" {
		return domain.FeedbackRecord{}, domain.ErrInvalidInput
	}
	if strings.TrimSpace(input.EventID) == "" || strings.TrimSpace(input.TraceID) == "" || strings.TrimSpace(input.SourceService) == "" {
		return domain.FeedbackRecord{}, domain.ErrInvalidInput
	}
	if input.SchemaVersion == "" {
		input.SchemaVersion = "1.0"
	}
	if input.PartitionKeyPath == "" {
		input.PartitionKeyPath = "data.entity_id"
	}
	if input.PartitionKey == "" {
		input.PartitionKey = input.EntityID
	}
	if input.PartitionKeyPath != "data.entity_id" || input.PartitionKey != input.EntityID {
		return domain.FeedbackRecord{}, domain.ErrInvalidEnvelope
	}

	occurredAt, err := parseRFC3339OrNow(input.OccurredAt, s.nowFn())
	if err != nil {
		return domain.FeedbackRecord{}, domain.ErrInvalidInput
	}

	requestHash := hashPayload(map[string]string{
		"recommendation_id": strings.TrimSpace(recommendationID),
		"event_id":          strings.TrimSpace(input.EventID),
		"event_type":        feedbackEventType,
		"entity_id":         strings.TrimSpace(input.EntityID),
	})
	if cached, ok, err := s.getIdempotentFeedback(ctx, actor, requestHash); err != nil {
		return domain.FeedbackRecord{}, err
	} else if ok {
		return cached, nil
	}

	now := s.nowFn()
	record := domain.FeedbackRecord{
		FeedbackID:       uuid.NewString(),
		RecommendationID: strings.TrimSpace(recommendationID),
		UserID:           actor.SubjectID,
		EventType:        feedbackEventType,
		EntityID:         strings.TrimSpace(input.EntityID),
		OccurredAt:       occurredAt,
		CreatedAt:        now,
		SourceService:    strings.TrimSpace(input.SourceService),
		TraceID:          strings.TrimSpace(input.TraceID),
		SchemaVersion:    strings.TrimSpace(input.SchemaVersion),
		IdempotencyKey:   actor.IdempotencyKey,
	}
	if s.feedback != nil {
		if err := s.feedback.Create(ctx, record); err != nil {
			return domain.FeedbackRecord{}, err
		}
	}
	_ = s.enqueueRecommendationFeedbackRecorded(ctx, actor, record)
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, httpLikeStatusOK, record); err != nil {
		return domain.FeedbackRecord{}, err
	}
	return record, nil
}

func (s *Service) ApplyOverride(ctx context.Context, actor Actor, input OverrideInput) (domain.RecommendationOverride, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.RecommendationOverride{}, domain.ErrUnauthorized
	}
	if domain.NormalizeRole(actor.Role) != domain.RoleAdmin {
		return domain.RecommendationOverride{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.RecommendationOverride{}, domain.ErrIdempotencyRequired
	}
	overrideType := domain.NormalizeOverrideType(input.OverrideType)
	if overrideType == "" || strings.TrimSpace(input.EntityID) == "" || strings.TrimSpace(input.Scope) == "" || strings.TrimSpace(input.Reason) == "" {
		return domain.RecommendationOverride{}, domain.ErrInvalidInput
	}
	if input.Multiplier <= 0 || input.Multiplier > 2.0 {
		return domain.RecommendationOverride{}, domain.ErrInvalidInput
	}
	var endAt *time.Time
	if strings.TrimSpace(input.EndDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(input.EndDate))
		if err != nil {
			return domain.RecommendationOverride{}, domain.ErrInvalidInput
		}
		p := parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second).UTC()
		endAt = &p
	}

	requestHash := hashPayload(input)
	if cached, ok, err := s.getIdempotentOverride(ctx, actor, requestHash); err != nil {
		return domain.RecommendationOverride{}, err
	} else if ok {
		return cached, nil
	}

	now := s.nowFn()
	record := domain.RecommendationOverride{
		OverrideID:   uuid.NewString(),
		OverrideType: overrideType,
		EntityID:     strings.TrimSpace(input.EntityID),
		Scope:        strings.ToLower(strings.TrimSpace(input.Scope)),
		ScopeValue:   strings.TrimSpace(input.ScopeValue),
		Multiplier:   input.Multiplier,
		Reason:       strings.TrimSpace(input.Reason),
		StartAt:      now,
		EndAt:        endAt,
		CreatedAt:    now,
		CreatedBy:    actor.SubjectID,
		Active:       true,
	}
	if s.overrides != nil {
		if err := s.overrides.Upsert(ctx, record); err != nil {
			return domain.RecommendationOverride{}, err
		}
	}
	_ = s.enqueueRecommendationOverrideApplied(ctx, actor, record)
	if err := s.completeIdempotent(ctx, actor.IdempotencyKey, httpLikeStatusCreated, record); err != nil {
		return domain.RecommendationOverride{}, err
	}
	return record, nil
}

func (s *Service) ListOverrides(ctx context.Context, actor Actor, role string) ([]domain.RecommendationOverride, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return nil, domain.ErrUnauthorized
	}
	if domain.NormalizeRole(actor.Role) != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}
	if s.overrides == nil {
		return []domain.RecommendationOverride{}, nil
	}
	return s.overrides.ListActive(ctx, normalizeRole(role), s.nowFn())
}

func (s *Service) getIdempotentFeedback(ctx context.Context, actor Actor, requestHash string) (domain.FeedbackRecord, bool, error) {
	if s.idempotency == nil {
		return domain.FeedbackRecord{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.FeedbackRecord{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.FeedbackRecord{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.FeedbackRecord
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.FeedbackRecord{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.FeedbackRecord{}, false, err
	}
	return domain.FeedbackRecord{}, false, nil
}

func (s *Service) getIdempotentOverride(ctx context.Context, actor Actor, requestHash string) (domain.RecommendationOverride, bool, error) {
	if s.idempotency == nil {
		return domain.RecommendationOverride{}, false, nil
	}
	now := s.nowFn()
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.RecommendationOverride{}, false, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.RecommendationOverride{}, false, domain.ErrIdempotencyConflict
		}
		var cached domain.RecommendationOverride
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.RecommendationOverride{}, false, err
		}
		return cached, true, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.RecommendationOverride{}, false, err
	}
	return domain.RecommendationOverride{}, false, nil
}

func (s *Service) completeIdempotent(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.idempotency.Complete(ctx, key, code, encoded, s.nowFn())
}

func normalizeLimit(v int) int {
	if v <= 0 {
		return 10
	}
	if v < 10 {
		return 10
	}
	if v > 50 {
		return 50
	}
	return v
}

func trimRecommendations(in []domain.Recommendation, limit int) []domain.Recommendation {
	if limit <= 0 || len(in) <= limit {
		out := make([]domain.Recommendation, len(in))
		copy(out, in)
		return out
	}
	out := make([]domain.Recommendation, limit)
	copy(out, in[:limit])
	return out
}

func fallbackRecommendations(userID, role, modelVersion string, now time.Time, limit int) []domain.Recommendation {
	out := make([]domain.Recommendation, 0, limit)
	for i := 0; i < limit; i++ {
		score := 0.8 - float64(i)*0.03
		out = append(out, domain.Recommendation{
			RecommendationID:    uuid.NewString(),
			UserID:              userID,
			Role:                role,
			EntityID:            fmt.Sprintf("cmp_fallback_%02d", i+1),
			EntityType:          domain.EntityTypeCampaign,
			Score:               domain.Clamp(0, score, 1),
			Position:            i + 1,
			Reason:              "Trending campaign fallback",
			ConfidenceScore:     0.6,
			ConfidenceLevel:     domain.ConfidenceLevel(0.6),
			ContributingFactors: []domain.Factor{{Name: "Popularity", Weight: 1.0}},
			Campaign:            &domain.CampaignSnapshot{CampaignID: fmt.Sprintf("cmp_fallback_%02d", i+1), Title: "Trending Campaign", CreatorID: "creator_trending", RewardRate: 1.5, Platform: "TikTok"},
			ModelVersion:        modelVersion,
			ComputedAt:          now,
		})
	}
	return out
}

func rankCandidates(candidates []ports.CampaignCandidate, userID, role, modelVersion string, now time.Time) []domain.Recommendation {
	out := make([]domain.Recommendation, 0, len(candidates))
	for i, item := range candidates {
		base := scoreCandidate(item, role)
		factors := factorsForCandidate(item, role)
		out = append(out, domain.Recommendation{
			RecommendationID:    uuid.NewString(),
			UserID:              userID,
			Role:                role,
			EntityID:            item.CampaignID,
			EntityType:          domain.EntityTypeCampaign,
			Score:               base,
			Position:            i + 1,
			Reason:              reasonForRole(role, item),
			ConfidenceScore:     domain.Clamp(0, 0.55+base*0.45, 1),
			ConfidenceLevel:     domain.ConfidenceLevel(domain.Clamp(0, 0.55+base*0.45, 1)),
			ContributingFactors: factors,
			Campaign: &domain.CampaignSnapshot{
				CampaignID:   item.CampaignID,
				Title:        item.Title,
				CreatorID:    item.CreatorID,
				RewardRate:   item.RewardRate,
				Platform:     item.Platform,
				Category:     item.Category,
				ApprovalRate: item.ApprovalRate,
			},
			ModelVersion: modelVersion,
			ComputedAt:   now,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	for i := range out {
		out[i].Position = i + 1
	}
	return out
}

func scoreCandidate(item ports.CampaignCandidate, role string) float64 {
	reward := domain.Clamp(0, item.RewardRate/3.0, 1)
	approval := domain.Clamp(0, item.ApprovalRate, 1)
	velocity := domain.Clamp(0, item.VelocityScore, 1)
	recency := 1.0
	if item.AgeDays > 90 {
		recency = 0.5
	} else if item.AgeDays > 30 {
		recency = 0.8
	}
	switch role {
	case domain.RoleCreator:
		return domain.Clamp(0, approval*0.35+velocity*0.35+reward*0.15+recency*0.15, 1)
	case domain.RoleBuyer:
		return domain.Clamp(0, velocity*0.45+approval*0.25+recency*0.2+reward*0.1, 1)
	default:
		return domain.Clamp(0, approval*0.30+reward*0.30+velocity*0.20+recency*0.20, 1)
	}
}

func factorsForCandidate(item ports.CampaignCandidate, role string) []domain.Factor {
	switch role {
	case domain.RoleCreator:
		return []domain.Factor{{Name: "Niche Match", Weight: 0.30}, {Name: "Historical Performance", Weight: 0.30}, {Name: "Engagement Rate", Weight: 0.20}, {Name: "Reputation", Weight: 0.20}}
	case domain.RoleBuyer:
		return []domain.Factor{{Name: "Trend Velocity", Weight: 0.30}, {Name: "Engagement Rate", Weight: 0.25}, {Name: "Creator Reputation", Weight: 0.20}, {Name: "Category Preference", Weight: 0.15}, {Name: "Freshness", Weight: 0.10}}
	default:
		return []domain.Factor{{Name: "Collaborative Filtering", Weight: 0.45}, {Name: "Creator Reputation", Weight: 0.30}, {Name: "Niche Match", Weight: 0.25}}
	}
}

func reasonForRole(role string, item ports.CampaignCandidate) string {
	switch role {
	case domain.RoleCreator:
		return "Recommended creator collaboration candidate based on niche and performance"
	case domain.RoleBuyer:
		return "Trending content recommendation in your preferred category"
	default:
		if item.Platform != "" && item.Category != "" {
			return fmt.Sprintf("Matches your %s activity and %s interests", item.Platform, item.Category)
		}
		return "Recommended because you submitted to similar campaigns"
	}
}

func applyOverrides(in []domain.Recommendation, overrides []domain.RecommendationOverride) []domain.Recommendation {
	if len(in) == 0 || len(overrides) == 0 {
		return in
	}
	out := make([]domain.Recommendation, len(in))
	copy(out, in)
	for i := range out {
		for _, ov := range overrides {
			if !ov.Active {
				continue
			}
			if ov.EntityID != out[i].EntityID && (out[i].Campaign == nil || ov.EntityID != out[i].Campaign.CreatorID) {
				continue
			}
			switch ov.OverrideType {
			case domain.OverrideTypePromoteCampaign, domain.OverrideTypeSuppressCampaign:
				if ov.EntityID != out[i].EntityID {
					continue
				}
			case domain.OverrideTypePromoteCreator, domain.OverrideTypeSuppressCreator:
				if out[i].Campaign == nil || ov.EntityID != out[i].Campaign.CreatorID {
					continue
				}
			}
			out[i].Score = domain.Clamp(0, out[i].Score*ov.Multiplier, 1)
			out[i].Reason = out[i].Reason + " (admin override applied)"
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}

func parseRFC3339OrNow(raw string, fallback time.Time) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	v, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, err
	}
	return v.UTC(), nil
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const (
	httpLikeStatusOK      = 200
	httpLikeStatusCreated = 201
)
