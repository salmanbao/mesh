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
	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/domain"
)

func (s *Service) GetCreatorDashboard(ctx context.Context, actor Actor, input DashboardInput) (domain.CreatorDashboard, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.CreatorDashboard{}, domain.ErrUnauthorized
	}
	targetUser := coalesceUserID(actor, strings.TrimSpace(input.UserID))
	if normalizeRole(actor.Role) != "admin" && targetUser != actor.SubjectID {
		return domain.CreatorDashboard{}, domain.ErrForbidden
	}

	from, to, err := parseDateRange(input.DateFrom, input.DateTo, s.nowFn())
	if err != nil {
		return domain.CreatorDashboard{}, err
	}

	// Owner-API reads declared in dependencies and shared data surface.
	_, _ = s.voting.GetVoteSummary(ctx, targetUser, from, to)
	_, _ = s.social.GetSocialSummary(ctx, targetUser)
	_, _ = s.tracking.GetTrackingSummary(ctx, targetUser, from, to)
	_, _ = s.submission.GetSubmissionSummary(ctx, targetUser, from, to)
	_, _ = s.finance.GetFinanceSummary(ctx, targetUser, from, to)

	dashboard, err := s.warehouse.GetCreatorDashboard(ctx, targetUser, from, to)
	if err != nil {
		return domain.CreatorDashboard{}, err
	}
	dashboard.UserID = targetUser
	dashboard.DateFrom = from.Format("2006-01-02")
	dashboard.DateTo = to.Format("2006-01-02")
	dashboard.GeneratedAt = s.nowFn()
	return dashboard, nil
}

func (s *Service) GetAdminFinancialReport(ctx context.Context, actor Actor, input FinancialReportInput) (domain.FinancialReport, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.FinancialReport{}, domain.ErrUnauthorized
	}
	if normalizeRole(actor.Role) != "admin" {
		return domain.FinancialReport{}, domain.ErrForbidden
	}
	from, to, err := parseDateRange(input.DateFrom, input.DateTo, s.nowFn())
	if err != nil {
		return domain.FinancialReport{}, err
	}
	report, err := s.warehouse.GetFinancialReport(ctx, from, to)
	if err != nil {
		return domain.FinancialReport{}, err
	}
	report.DateFrom = from.Format("2006-01-02")
	report.DateTo = to.Format("2006-01-02")
	report.GeneratedAt = s.nowFn()
	return report, nil
}

func (s *Service) RequestExport(ctx context.Context, actor Actor, input ExportInput) (domain.ExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportJob{}, domain.ErrUnauthorized
	}
	if strings.TrimSpace(actor.IdempotencyKey) == "" {
		return domain.ExportJob{}, domain.ErrIdempotencyRequired
	}
	reportType := domain.NormalizeReportType(input.ReportType)
	if err := domain.ValidateReportType(reportType); err != nil {
		return domain.ExportJob{}, err
	}
	if err := domain.ValidateExportFormat(input.Format); err != nil {
		return domain.ExportJob{}, err
	}
	if input.Filters == nil {
		input.Filters = map[string]string{}
	}

	now := s.nowFn()
	requestHash := hashPayload(input)
	existing, err := s.idempotency.Get(ctx, actor.IdempotencyKey, now)
	if err != nil {
		return domain.ExportJob{}, err
	}
	if existing != nil {
		if existing.RequestHash != requestHash {
			_ = s.publishDLQIdempotencyConflict(ctx, actor.IdempotencyKey, actor.RequestID)
			return domain.ExportJob{}, domain.ErrIdempotencyConflict
		}
		var cached domain.ExportJob
		if err := json.Unmarshal(existing.ResponseBody, &cached); err != nil {
			return domain.ExportJob{}, err
		}
		return cached, nil
	}
	if err := s.idempotency.Reserve(ctx, actor.IdempotencyKey, requestHash, now.Add(s.cfg.IdempotencyTTL)); err != nil {
		return domain.ExportJob{}, err
	}

	exportJob := domain.ExportJob{
		ExportID:       uuid.NewString(),
		UserID:         actor.SubjectID,
		ReportType:     reportType,
		Format:         strings.ToLower(strings.TrimSpace(input.Format)),
		DateFrom:       strings.TrimSpace(input.DateFrom),
		DateTo:         strings.TrimSpace(input.DateTo),
		Filters:        cloneFilters(input.Filters),
		Status:         domain.ExportStatusQueued,
		IdempotencyKey: actor.IdempotencyKey,
		CreatedAt:      now,
	}
	if exportJob.DateFrom == "" || exportJob.DateTo == "" {
		from, to, _ := parseDateRange("", "", now)
		exportJob.DateFrom = from.Format("2006-01-02")
		exportJob.DateTo = to.Format("2006-01-02")
	}
	if err := s.exports.Create(ctx, exportJob); err != nil {
		return domain.ExportJob{}, err
	}

	readyAt := s.nowFn()
	exportJob.Status = domain.ExportStatusReady
	exportJob.ReadyAt = &readyAt
	exportJob.DownloadURL = fmt.Sprintf("https://exports.viralforge.local/%s.%s", exportJob.ExportID, exportJob.Format)
	if err := s.exports.Update(ctx, exportJob); err != nil {
		return domain.ExportJob{}, err
	}

	encoded, err := json.Marshal(exportJob)
	if err != nil {
		return domain.ExportJob{}, err
	}
	if err := s.idempotency.Complete(ctx, actor.IdempotencyKey, 202, encoded, s.nowFn()); err != nil {
		return domain.ExportJob{}, err
	}
	return exportJob, nil
}

func (s *Service) GetExport(ctx context.Context, actor Actor, exportID string) (domain.ExportJob, error) {
	if strings.TrimSpace(actor.SubjectID) == "" {
		return domain.ExportJob{}, domain.ErrUnauthorized
	}
	job, err := s.exports.GetByID(ctx, strings.TrimSpace(exportID))
	if err != nil {
		return domain.ExportJob{}, err
	}
	if normalizeRole(actor.Role) != "admin" && job.UserID != actor.SubjectID {
		return domain.ExportJob{}, domain.ErrForbidden
	}
	return job, nil
}

func cloneFilters(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func parseDateRange(dateFrom, dateTo string, now time.Time) (time.Time, time.Time, error) {
	const layout = "2006-01-02"
	if strings.TrimSpace(dateFrom) == "" || strings.TrimSpace(dateTo) == "" {
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
		start := end.AddDate(0, 0, -30)
		return start, end, nil
	}
	from, err := time.Parse(layout, dateFrom)
	if err != nil {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	to, err := time.Parse(layout, dateTo)
	if err != nil {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	if from.After(to) {
		return time.Time{}, time.Time{}, domain.ErrInvalidInput
	}
	return from, to.Add(23*time.Hour + 59*time.Minute + 59*time.Second), nil
}

func hashPayload(value interface{}) string {
	blob, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(blob)
	return hex.EncodeToString(sum[:])
}
