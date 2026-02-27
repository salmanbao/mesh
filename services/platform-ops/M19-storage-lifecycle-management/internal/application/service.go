package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/platform-ops/M19-storage-lifecycle-management/internal/domain"
)

func (s *Service) CreatePolicy(ctx context.Context, actor Actor, in CreatePolicyInput) (domain.StoragePolicy, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.StoragePolicy{}, domain.ErrUnauthorized
	}
	if !isAdminLike(actor) {
		return domain.StoragePolicy{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.StoragePolicy{}, domain.ErrIdempotencyRequired
	}
	in.Scope = strings.TrimSpace(in.Scope)
	in.TierFrom = strings.ToUpper(strings.TrimSpace(in.TierFrom))
	in.TierTo = strings.ToUpper(strings.TrimSpace(in.TierTo))
	if in.Scope == "" || !domain.IsValidTier(in.TierFrom) || !domain.IsValidTier(in.TierTo) || in.AfterDays <= 0 {
		return domain.StoragePolicy{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.StoragePolicy{}, err
	} else if ok {
		var out domain.StoragePolicy
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.StoragePolicy{}, err
	}

	now := s.nowFn()
	id := strings.TrimSpace(in.PolicyID)
	if id == "" {
		id = "pol-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	}
	row := domain.StoragePolicy{
		PolicyID:        id,
		Scope:           in.Scope,
		TierFrom:        in.TierFrom,
		TierTo:          in.TierTo,
		AfterDays:       in.AfterDays,
		LegalHoldExempt: in.LegalHoldExempt,
		Status:          domain.PolicyStatusActive,
		CreatedAt:       now,
	}
	if err := s.policies.Create(ctx, row); err != nil {
		return domain.StoragePolicy{}, err
	}
	_ = s.appendAudit(ctx, domain.AuditRecord{
		AuditID:     uuid.NewString(),
		Action:      "policy_created",
		TriggeredBy: actor.SubjectID,
		Reason:      row.Scope,
		InitiatedAt: now,
		CompletedAt: now,
	})
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 201, row)
	return row, nil
}

func (s *Service) GetAnalyticsSummary(ctx context.Context, actor Actor) (domain.AnalyticsSummary, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.AnalyticsSummary{}, domain.ErrUnauthorized
	}
	if s.lifecycle == nil {
		return domain.AnalyticsSummary{ByTier: map[string]int64{}}, nil
	}
	return s.lifecycle.AnalyticsSummary(ctx)
}

func (s *Service) MoveToGlacier(ctx context.Context, actor Actor, in MoveToGlacierInput) (domain.LifecycleJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.LifecycleJob{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.LifecycleJob{}, domain.ErrIdempotencyRequired
	}
	in.FileID = strings.TrimSpace(in.FileID)
	in.SourceBucket = strings.TrimSpace(in.SourceBucket)
	in.SourceKey = strings.TrimSpace(in.SourceKey)
	in.DestinationBucket = strings.TrimSpace(in.DestinationBucket)
	in.DestinationKey = strings.TrimSpace(in.DestinationKey)
	if in.FileID == "" || in.SourceBucket == "" || in.SourceKey == "" || in.DestinationBucket == "" || in.DestinationKey == "" {
		return domain.LifecycleJob{}, domain.ErrInvalidInput
	}

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LifecycleJob{}, err
	} else if ok {
		var out domain.LifecycleJob
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.LifecycleJob{}, err
	}

	now := s.nowFn()
	row := domain.LifecycleFile{
		FileID:            in.FileID,
		CampaignID:        strings.TrimSpace(in.CampaignID),
		SubmissionID:      strings.TrimSpace(in.SubmissionID),
		FileSizeBytes:     in.FileSizeBytes,
		SourceBucket:      in.SourceBucket,
		SourceKey:         in.SourceKey,
		DestinationBucket: in.DestinationBucket,
		DestinationKey:    in.DestinationKey,
		ChecksumMD5:       strings.TrimSpace(in.ChecksumMD5),
		StorageTier:       domain.TierGlacierDeepArchive,
		Status:            domain.FileStatusArchivedCold,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if s.lifecycle != nil {
		if err := s.lifecycle.Upsert(ctx, row); err != nil {
			return domain.LifecycleJob{}, err
		}
	}
	_ = s.appendAudit(ctx, domain.AuditRecord{
		AuditID:       uuid.NewString(),
		FileID:        row.FileID,
		CampaignID:    row.CampaignID,
		Action:        "archive",
		TriggeredBy:   actor.SubjectID,
		FileSizeBytes: row.FileSizeBytes,
		Reason:        "move_to_glacier",
		InitiatedAt:   now,
		CompletedAt:   now,
	})
	job := domain.LifecycleJob{
		JobID:     "job-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		FileID:    row.FileID,
		Status:    "in_progress",
		Message:   "move initiated",
		CreatedAt: now,
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, job)
	return job, nil
}

func (s *Service) ScheduleDeletion(ctx context.Context, actor Actor, in ScheduleDeletionInput) (domain.DeletionBatch, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.DeletionBatch{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.DeletionBatch{}, domain.ErrIdempotencyRequired
	}
	in.CampaignID = strings.TrimSpace(in.CampaignID)
	in.DeletionType = strings.TrimSpace(in.DeletionType)
	if in.CampaignID == "" || in.DeletionType != domain.DeletionTypeRawFiles || len(in.FileIDs) == 0 {
		return domain.DeletionBatch{}, domain.ErrInvalidInput
	}
	if in.DaysAfterClosure <= 0 {
		in.DaysAfterClosure = 30
	}
	cleanIDs := make([]string, 0, len(in.FileIDs))
	seen := map[string]struct{}{}
	for _, id := range in.FileIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		return domain.DeletionBatch{}, domain.ErrInvalidInput
	}
	in.FileIDs = cleanIDs

	requestHash := hashJSON(in)
	if raw, ok, err := s.getIdempotent(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DeletionBatch{}, err
	} else if ok {
		var out domain.DeletionBatch
		if json.Unmarshal(raw, &out) == nil {
			return out, nil
		}
	}
	if err := s.reserveIdempotency(ctx, actor.IdempotencyKey, requestHash); err != nil {
		return domain.DeletionBatch{}, err
	}

	now := s.nowFn()
	batch := domain.DeletionBatch{
		BatchID:          "bat-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8],
		CampaignID:       in.CampaignID,
		DeletionType:     in.DeletionType,
		FileIDs:          append([]string(nil), in.FileIDs...),
		DaysAfterClosure: in.DaysAfterClosure,
		FileCount:        len(in.FileIDs),
		ScheduledFor:     now.Add(time.Duration(in.DaysAfterClosure) * 24 * time.Hour),
		Status:           "scheduled",
		CreatedAt:        now,
	}
	if s.batches != nil {
		if err := s.batches.Create(ctx, batch); err != nil {
			return domain.DeletionBatch{}, err
		}
	}
	if s.lifecycle != nil {
		for _, fileID := range in.FileIDs {
			_ = s.lifecycle.Upsert(ctx, domain.LifecycleFile{
				FileID:      fileID,
				CampaignID:  in.CampaignID,
				StorageTier: domain.TierStandard,
				Status:      domain.FileStatusSoftDeleted,
				CreatedAt:   now,
				UpdatedAt:   now,
			})
			_ = s.appendAudit(ctx, domain.AuditRecord{
				AuditID:     uuid.NewString(),
				FileID:      fileID,
				CampaignID:  in.CampaignID,
				Action:      "soft_delete",
				TriggeredBy: actor.SubjectID,
				Reason:      "campaign_closure",
				InitiatedAt: now,
				CompletedAt: now,
			})
		}
	}
	_ = s.completeIdempotencyJSON(ctx, actor.IdempotencyKey, 202, batch)
	return batch, nil
}

func (s *Service) QueryDeletionAudit(ctx context.Context, actor Actor, in AuditQueryInput) (domain.AuditQueryResult, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.AuditQueryResult{}, domain.ErrUnauthorized
	}
	if s.audits == nil {
		return domain.AuditQueryResult{}, nil
	}
	return s.audits.Query(ctx, domain.AuditQuery{
		FileID:     strings.TrimSpace(in.FileID),
		CampaignID: strings.TrimSpace(in.CampaignID),
		Action:     strings.TrimSpace(in.Action),
		StartDate:  in.StartDate,
		EndDate:    in.EndDate,
		Limit:      in.Limit,
	})
}

func (s *Service) GetHealth(ctx context.Context) (domain.HealthReport, error) {
	_ = ctx
	now := s.nowFn()
	checks := map[string]domain.ComponentCheck{
		"policy_store":    {Name: "policy_store", Status: "healthy", LatencyMS: 5, LastChecked: now},
		"lifecycle_store": {Name: "lifecycle_store", Status: "healthy", LatencyMS: 8, LastChecked: now},
		"scheduler":       {Name: "scheduler", Status: "healthy", LatencyMS: 3, LastChecked: now},
	}
	return domain.HealthReport{
		Status:        "healthy",
		Timestamp:     now,
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		Version:       s.cfg.Version,
		Checks:        checks,
	}, nil
}

func (s *Service) GetCacheMetrics(ctx context.Context) (domain.MetricsSnapshot, error) {
	_ = ctx
	return domain.MetricsSnapshot{}, nil
}

func (s *Service) RecordHTTPMetric(ctx context.Context, in MetricObservation) {
	if s.metrics == nil {
		return
	}
	path := strings.TrimSpace(in.Path)
	if path == "" {
		path = "/unknown"
	}
	method := strings.ToUpper(strings.TrimSpace(in.Method))
	if method == "" {
		method = "GET"
	}
	status := strconv.Itoa(in.StatusCode)
	_ = s.metrics.IncCounter(ctx, "http_requests_total", map[string]string{
		"service": s.cfg.ServiceName,
		"method":  method,
		"path":    path,
		"status":  status,
	}, 1)
	_ = s.metrics.ObserveHistogram(ctx, "http_request_duration_seconds",
		map[string]string{"service": s.cfg.ServiceName, "method": method, "path": path},
		in.Duration.Seconds(), []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5})
}

func (s *Service) RenderPrometheusMetrics(ctx context.Context) (string, error) {
	if s.metrics == nil {
		return "# no metrics\n", nil
	}
	snap, err := s.metrics.Snapshot(ctx)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, c := range snap.Counters {
		if c.Name != "http_requests_total" {
			continue
		}
		if !strings.Contains(b.String(), "# TYPE http_requests_total counter") {
			b.WriteString("# TYPE http_requests_total counter\n")
		}
		b.WriteString(c.Name + formatLabels(c.Labels) + " " + strconv.FormatFloat(c.Value, 'f', -1, 64) + "\n")
	}
	for _, h := range snap.Histograms {
		if h.Name != "http_request_duration_seconds" {
			continue
		}
		if !strings.Contains(b.String(), "# TYPE http_request_duration_seconds histogram") {
			b.WriteString("# TYPE http_request_duration_seconds histogram\n")
		}
		for _, le := range sortedBucketKeys(h.Buckets) {
			lbl := copyMap(h.Labels)
			lbl["le"] = le
			b.WriteString(h.Name + "_bucket" + formatLabels(lbl) + " " + strconv.FormatFloat(h.Buckets[le], 'f', -1, 64) + "\n")
		}
		b.WriteString(h.Name + "_sum" + formatLabels(h.Labels) + " " + strconv.FormatFloat(h.Sum, 'f', -1, 64) + "\n")
		b.WriteString(h.Name + "_count" + formatLabels(h.Labels) + " " + strconv.FormatFloat(h.Count, 'f', -1, 64) + "\n")
	}
	if b.Len() == 0 {
		return "# no metrics yet\n", nil
	}
	return b.String(), nil
}

func (s *Service) appendAudit(ctx context.Context, row domain.AuditRecord) error {
	if s.audits == nil {
		return nil
	}
	return s.audits.Create(ctx, row)
}

func isAdminLike(actor Actor) bool {
	r := strings.ToLower(strings.TrimSpace(actor.Role))
	return r == "admin" || r == "sre" || r == "system"
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
	return append([]byte(nil), rec.ResponseBody...), true, nil
}

func (s *Service) reserveIdempotency(ctx context.Context, key, requestHash string) error {
	if s.idempotency == nil {
		return nil
	}
	return s.idempotency.Reserve(ctx, key, requestHash, s.nowFn().Add(s.cfg.IdempotencyTTL))
}

func (s *Service) completeIdempotencyJSON(ctx context.Context, key string, code int, payload any) error {
	if s.idempotency == nil || strings.TrimSpace(key) == "" {
		return nil
	}
	raw, _ := json.Marshal(payload)
	return s.idempotency.Complete(ctx, key, code, raw, s.nowFn())
}

func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("{")
	for i, k := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(k + "=\"")
		b.WriteString(strings.ReplaceAll(labels[k], "\"", "\\\""))
		b.WriteString("\"")
	}
	b.WriteString("}")
	return b.String()
}

func sortedBucketKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "+Inf" {
			return false
		}
		if keys[j] == "+Inf" {
			return true
		}
		fi, ei := strconv.ParseFloat(keys[i], 64)
		fj, ej := strconv.ParseFloat(keys[j], 64)
		if ei != nil || ej != nil {
			return keys[i] < keys[j]
		}
		return fi < fj
	})
	return keys
}
