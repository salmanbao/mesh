package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/domain"
)

var idCounter uint64

func nextID(prefix string) string {
	n := atomic.AddUint64(&idCounter, 1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UTC().UnixNano(), n)
}

func (s *Service) Analyze(ctx context.Context, actor Actor, in AnalyzeInput) (domain.Prediction, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.Prediction{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.Prediction{}, domain.ErrIdempotencyRequired
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return domain.Prediction{}, domain.ErrInvalidInput
	}
	model, err := s.resolveModel(ctx, in.ModelID, in.ModelVersion)
	if err != nil {
		return domain.Prediction{}, err
	}

	contentHash := hashText(content)
	requestHash := hashJSON(map[string]any{
		"op":            "analyze",
		"user_id":       userID,
		"content_id":    strings.TrimSpace(in.ContentID),
		"content_hash":  contentHash,
		"model_id":      model.ModelID,
		"model_version": model.Version,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Prediction{}, err
	} else if ok {
		var out domain.Prediction
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.Prediction{}, err
	}
	if cached, ok, err := s.predictions.FindByKey(ctx, contentHash, model.ModelID, model.Version); err != nil {
		return domain.Prediction{}, err
	} else if ok {
		_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, cached)
		return cached, nil
	}

	row := s.buildPrediction(userID, strings.TrimSpace(in.ContentID), contentHash, content, model)
	if err := s.predictions.Create(ctx, row); err != nil {
		return domain.Prediction{}, err
	}
	if s.feedback != nil {
		_ = s.feedback.Append(ctx, domain.FeedbackLog{
			FeedbackID:   nextID("fdbk"),
			PredictionID: row.PredictionID,
			UserID:       row.UserID,
			Feedback:     "prediction_created",
			CreatedAt:    row.CreatedAt,
		})
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:    nextID("audit"),
			EventType:  "ai.analysis.completed",
			ActorID:    actor.SubjectID,
			EntityID:   row.PredictionID,
			OccurredAt: s.nowFn(),
			Metadata: map[string]string{
				"model_id":      row.ModelID,
				"model_version": row.ModelVersion,
			},
		})
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, row)
	return row, nil
}

func (s *Service) BatchAnalyze(ctx context.Context, actor Actor, in BatchAnalyzeInput) (domain.BatchJob, error) {
	userID, err := s.resolveUser(actor, in.UserID)
	if err != nil {
		return domain.BatchJob{}, err
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.BatchJob{}, domain.ErrIdempotencyRequired
	}
	if len(in.Items) == 0 {
		return domain.BatchJob{}, domain.ErrInvalidInput
	}
	model, err := s.resolveModel(ctx, in.ModelID, in.ModelVersion)
	if err != nil {
		return domain.BatchJob{}, err
	}

	serializedItems := make([]map[string]string, 0, len(in.Items))
	for _, item := range in.Items {
		content := strings.TrimSpace(item.Content)
		if content == "" {
			return domain.BatchJob{}, domain.ErrInvalidInput
		}
		serializedItems = append(serializedItems, map[string]string{
			"content_id": strings.TrimSpace(item.ContentID),
			"content":    content,
		})
	}

	requestHash := hashJSON(map[string]any{
		"op":            "batch_analyze",
		"user_id":       userID,
		"model_id":      model.ModelID,
		"model_version": model.Version,
		"items":         serializedItems,
	})
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.BatchJob{}, err
	} else if ok {
		var out domain.BatchJob
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.BatchJob{}, err
	}

	now := s.nowFn()
	job := domain.BatchJob{
		JobID:          nextID("job"),
		UserID:         userID,
		Status:         domain.BatchStatusPending,
		ModelID:        model.ModelID,
		ModelVersion:   model.Version,
		RequestedCount: len(in.Items),
		CreatedAt:      now,
		PredictionIDs:  make([]string, 0, len(in.Items)),
		Predictions:    make([]domain.Prediction, 0, len(in.Items)),
	}
	job.StatusURL = "/api/v1/ai/batch-status/" + job.JobID

	for _, item := range in.Items {
		content := strings.TrimSpace(item.Content)
		contentHash := hashText(content)
		row, found, err := s.predictions.FindByKey(ctx, contentHash, model.ModelID, model.Version)
		if err != nil {
			return domain.BatchJob{}, err
		}
		if !found {
			row = s.buildPrediction(userID, strings.TrimSpace(item.ContentID), contentHash, content, model)
			if err := s.predictions.Create(ctx, row); err != nil {
				return domain.BatchJob{}, err
			}
		}
		job.PredictionIDs = append(job.PredictionIDs, row.PredictionID)
		job.Predictions = append(job.Predictions, row)
	}

	done := s.nowFn()
	job.Status = domain.BatchStatusCompleted
	job.CompletedCount = len(job.Predictions)
	job.CompletedAt = &done
	if err := s.batchJobs.Create(ctx, job); err != nil {
		return domain.BatchJob{}, err
	}
	if s.audit != nil {
		_ = s.audit.Append(ctx, domain.AuditLog{
			EventID:    nextID("audit"),
			EventType:  "ai.batch.completed",
			ActorID:    actor.SubjectID,
			EntityID:   job.JobID,
			OccurredAt: done,
			Metadata: map[string]string{
				"items": fmt.Sprintf("%d", job.CompletedCount),
			},
		})
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 200, job)
	return job, nil
}

func (s *Service) GetBatchStatus(ctx context.Context, actor Actor, jobID string) (domain.BatchJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.BatchJob{}, domain.ErrUnauthorized
	}
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return domain.BatchJob{}, domain.ErrInvalidInput
	}
	job, err := s.batchJobs.GetByID(ctx, jobID)
	if err != nil {
		return domain.BatchJob{}, err
	}
	if !canActForUser(actor, job.UserID) {
		return domain.BatchJob{}, domain.ErrForbidden
	}
	return job, nil
}

func (s *Service) resolveUser(actor Actor, requested string) (string, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return "", domain.ErrUnauthorized
	}
	requested = strings.TrimSpace(requested)
	if requested == "" {
		requested = strings.TrimSpace(actor.SubjectID)
	}
	if !canActForUser(actor, requested) {
		return "", domain.ErrForbidden
	}
	return requested, nil
}

func canActForUser(actor Actor, userID string) bool {
	role := strings.ToLower(strings.TrimSpace(actor.Role))
	actorID := strings.TrimSpace(actor.SubjectID)
	userID = strings.TrimSpace(userID)
	return actorID != "" && userID != "" && (actorID == userID || role == "admin" || role == "support")
}

func (s *Service) resolveModel(ctx context.Context, modelID, version string) (domain.Model, error) {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		modelID = "vf-core"
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "2026.02"
	}
	if s.models == nil {
		return domain.Model{ModelID: modelID, Version: version, Active: true}, nil
	}
	return s.models.GetActive(ctx, modelID, version)
}

func (s *Service) buildPrediction(userID, contentID, contentHash, content string, model domain.Model) domain.Prediction {
	label := classify(content)
	flagged := label != "safe"
	confidence := 0.91
	if flagged {
		confidence = 0.97
	}
	return domain.Prediction{
		PredictionID: nextID("pred"),
		UserID:       userID,
		ContentID:    contentID,
		ContentHash:  contentHash,
		ModelID:      model.ModelID,
		ModelVersion: model.Version,
		Label:        label,
		Confidence:   confidence,
		Flagged:      flagged,
		CreatedAt:    s.nowFn(),
	}
}

func classify(content string) string {
	content = strings.ToLower(content)
	switch {
	case strings.Contains(content, "dmca"), strings.Contains(content, "copyright"), strings.Contains(content, "pirated"):
		return "copyright_risk"
	case strings.Contains(content, "fraud"), strings.Contains(content, "scam"):
		return "fraud_risk"
	default:
		return "safe"
	}
}

func hashText(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

func hashJSON(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *Service) getIdempotent(ctx context.Context, key, expectedHash string) ([]byte, bool, error) {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil, false, nil
	}
	rec, err := s.idempotency.Get(ctx, key, s.nowFn())
	if err != nil || rec == nil {
		return nil, false, err
	}
	if rec.RequestHash != expectedHash {
		return nil, false, domain.ErrIdempotencyConflict
	}
	if len(rec.ResponseBody) == 0 {
		return nil, false, nil
	}
	return rec.ResponseBody, true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, v any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(v)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}
