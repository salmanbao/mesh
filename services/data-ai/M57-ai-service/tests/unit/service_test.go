package unit

import (
	"context"
	"testing"

	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/application"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/domain"
)

func TestAnalyzeAndIdempotentReplay(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Predictions: repos.Predictions,
		BatchJobs:   repos.BatchJobs,
		Models:      repos.Models,
		Feedback:    repos.Feedback,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	actor := application.Actor{SubjectID: "user-1", Role: "user", IdempotencyKey: "idem-analyze-1"}

	row, err := svc.Analyze(context.Background(), actor, application.AnalyzeInput{Content: "normal content"})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if row.Label != "safe" || row.Flagged {
		t.Fatalf("unexpected prediction: %+v", row)
	}

	replay, err := svc.Analyze(context.Background(), actor, application.AnalyzeInput{Content: "normal content"})
	if err != nil {
		t.Fatalf("replay analyze: %v", err)
	}
	if replay.PredictionID != row.PredictionID {
		t.Fatalf("expected replay to reuse prediction id, got first=%s replay=%s", row.PredictionID, replay.PredictionID)
	}
}

func TestBatchAnalyzeAndStatus(t *testing.T) {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Predictions: repos.Predictions,
		BatchJobs:   repos.BatchJobs,
		Models:      repos.Models,
		Feedback:    repos.Feedback,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})

	job, err := svc.BatchAnalyze(context.Background(), application.Actor{SubjectID: "user-2", Role: "user", IdempotencyKey: "idem-batch-1"}, application.BatchAnalyzeInput{
		Items: []application.BatchItemInput{
			{ContentID: "c1", Content: "dmca takedown risk"},
			{ContentID: "c2", Content: "safe text"},
		},
	})
	if err != nil {
		t.Fatalf("batch analyze: %v", err)
	}
	if job.Status != domain.BatchStatusCompleted || job.CompletedCount != 2 {
		t.Fatalf("unexpected batch job: %+v", job)
	}

	got, err := svc.GetBatchStatus(context.Background(), application.Actor{SubjectID: "user-2", Role: "user"}, job.JobID)
	if err != nil {
		t.Fatalf("get batch status: %v", err)
	}
	if got.JobID != job.JobID || len(got.Predictions) != 2 {
		t.Fatalf("unexpected batch status response: %+v", got)
	}
}
