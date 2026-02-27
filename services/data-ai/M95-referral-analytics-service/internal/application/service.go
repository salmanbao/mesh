package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/data-ai/M95-referral-analytics-service/internal/domain"
)

func (s *Service) GetFunnel(ctx context.Context, actor Actor, input DateRangeInput) (domain.FunnelReport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.FunnelReport{}, domain.ErrUnauthorized
	}
	from, to, err := parseDateRange(input.StartDate, input.EndDate, s.nowFn())
	if err != nil {
		return domain.FunnelReport{}, err
	}
	if s.affiliate != nil {
		_, _ = s.affiliate.GetAffiliateSummary(ctx, actor.SubjectID, from, to)
	}
	report, err := s.warehouse.GetFunnel(ctx, from, to)
	if err != nil {
		return domain.FunnelReport{}, err
	}
	report.StartDate = from.Format("2006-01-02")
	report.EndDate = to.Format("2006-01-02")
	report.DataFreshnessAt = s.nowFn()
	return report, nil
}

func (s *Service) GetLeaderboard(ctx context.Context, actor Actor, input LeaderboardInput) (domain.LeaderboardReport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.LeaderboardReport{}, domain.ErrUnauthorized
	}
	period := domain.NormalizePeriod(input.Period)
	if err := domain.ValidatePeriod(period); err != nil {
		return domain.LeaderboardReport{}, err
	}
	if s.affiliate != nil {
		_, _ = s.affiliate.GetAffiliateSummary(ctx, actor.SubjectID, time.Time{}, time.Time{})
	}
	report, err := s.warehouse.GetLeaderboard(ctx, period, s.nowFn())
	if err != nil {
		return domain.LeaderboardReport{}, err
	}
	report.Period = period
	report.GeneratedAt = s.nowFn()
	return report, nil
}

func (s *Service) GetCohortRetention(ctx context.Context, actor Actor, input CohortInput) (domain.CohortRetentionReport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CohortRetentionReport{}, domain.ErrUnauthorized
	}
	from, to, err := parseDateRange(input.CohortStart, input.CohortEnd, s.nowFn())
	if err != nil {
		return domain.CohortRetentionReport{}, err
	}
	report, err := s.warehouse.GetCohortRetention(ctx, from, to)
	if err != nil {
		return domain.CohortRetentionReport{}, err
	}
	report.CohortStart = from.Format("2006-01-02")
	report.CohortEnd = to.Format("2006-01-02")
	report.GeneratedAt = s.nowFn()
	return report, nil
}

func (s *Service) GetGeo(ctx context.Context, actor Actor, input DateRangeInput) (domain.GeoPerformanceReport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.GeoPerformanceReport{}, domain.ErrUnauthorized
	}
	from, to, err := parseDateRange(input.StartDate, input.EndDate, s.nowFn())
	if err != nil {
		return domain.GeoPerformanceReport{}, err
	}
	report, err := s.warehouse.GetGeo(ctx, from, to)
	if err != nil {
		return domain.GeoPerformanceReport{}, err
	}
	report.StartDate = from.Format("2006-01-02")
	report.EndDate = to.Format("2006-01-02")
	report.GeneratedAt = s.nowFn()
	return report, nil
}

func (s *Service) GetForecast(ctx context.Context, actor Actor, input LeaderboardInput) (domain.PayoutForecast, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.PayoutForecast{}, domain.ErrUnauthorized
	}
	period := domain.NormalizePeriod(input.Period)
	if err := domain.ValidatePeriod(period); err != nil {
		return domain.PayoutForecast{}, err
	}
	forecast, err := s.warehouse.GetPayoutForecast(ctx, period, s.nowFn())
	if err != nil {
		return domain.PayoutForecast{}, err
	}
	forecast.Period = period
	forecast.GeneratedAt = s.nowFn()
	return forecast, nil
}

func (s *Service) RequestExport(ctx context.Context, actor Actor, input ExportInput) (domain.ReferralExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ReferralExportJob{}, domain.ErrUnauthorized
	}
	if normalizeRole(actor.Role) != "admin" && normalizeRole(actor.Role) != "analyst" {
		return domain.ReferralExportJob{}, domain.ErrForbidden
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ReferralExportJob{}, domain.ErrIdempotencyRequired
	}
	if err := domain.ValidateExportType(input.ExportType); err != nil {
		return domain.ReferralExportJob{}, err
	}
	if err := domain.ValidatePeriod(input.Period); err != nil {
		return domain.ReferralExportJob{}, err
	}
	if err := domain.ValidateExportFormat(input.Format); err != nil {
		return domain.ReferralExportJob{}, err
	}
	if input.Filters == nil {
		input.Filters = map[string]string{}
	}

	now := s.nowFn()
	requestHash := hashPayload(input)
	if existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now); err != nil {
		return domain.ReferralExportJob{}, err
	} else if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.ReferralExportJob{}, domain.ErrIdempotencyConflict
		}
		var cached domain.ReferralExportJob
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.ReferralExportJob{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.ReferralExportJob{}, err
	}

	job := domain.ReferralExportJob{
		ID:             uuid.NewString(),
		RequestedBy:    actor.SubjectID,
		ExportType:     domain.NormalizeExportType(input.ExportType),
		Format:         domain.NormalizeExportFormat(input.Format),
		Period:         domain.NormalizePeriod(input.Period),
		Filters:        cloneFilters(input.Filters),
		Status:         domain.ExportStatusQueued,
		IdempotencyKey: actor.IdempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.exports.Create(ctx, job); err != nil {
		return domain.ReferralExportJob{}, err
	}
	processing := s.nowFn()
	job.Status = domain.ExportStatusProcessing
	job.UpdatedAt = processing
	_ = s.exports.Update(ctx, job)
	completed := s.nowFn()
	job.Status = domain.ExportStatusCompleted
	job.OutputURI = fmt.Sprintf("s3://referral-analytics-exports/%s.%s", job.ID, job.Format)
	job.CompletedAt = &completed
	job.UpdatedAt = completed
	if err := s.exports.Update(ctx, job); err != nil {
		return domain.ReferralExportJob{}, err
	}
	blob, err := json.Marshal(job)
	if err != nil {
		return domain.ReferralExportJob{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 202, blob, s.nowFn()); err != nil {
		return domain.ReferralExportJob{}, err
	}
	return job, nil
}

func (s *Service) GetExport(ctx context.Context, actor Actor, id string) (domain.ReferralExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ReferralExportJob{}, domain.ErrUnauthorized
	}
	job, err := s.exports.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return domain.ReferralExportJob{}, err
	}
	role := normalizeRole(actor.Role)
	if role != "admin" && role != "analyst" && job.RequestedBy != actor.SubjectID {
		return domain.ReferralExportJob{}, domain.ErrForbidden
	}
	return job, nil
}

func cloneFilters(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func parseDateRange(startDate, endDate string, now time.Time) (time.Time, time.Time, error) {
	const layout = "2006-01-02"
	if strings.TrimSpace(startDate) == "" || strings.TrimSpace(endDate) == "" {
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
		start := end.AddDate(0, 0, -30)
		return start, end, nil
	}
	start, err := time.Parse(layout, startDate)
	if err != nil {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	end, err := time.Parse(layout, endDate)
	if err != nil {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	if start.After(end) {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	return start.UTC(), end.UTC().Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
}

func hashPayload(v interface{}) string {
	blob, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
