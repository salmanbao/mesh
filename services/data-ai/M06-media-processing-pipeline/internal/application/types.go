package application

import (
	"time"

	"github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline/internal/ports"
)

type Config struct {
	ServiceName       string
	IdempotencyTTL    time.Duration
	EventDedupTTL     time.Duration
	QueuePollInterval time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateUploadInput struct {
	SubmissionID   string
	FileName       string
	MIMEType       string
	FileSize       int64
	ChecksumSHA256 string
}

type RetryAssetInput struct {
	AssetID string
}

type UploadResult struct {
	AssetID   string
	UploadURL string
	ExpiresIn int
}

type PreviewResult struct {
	PreviewURL string
	ExpiresAt  int64
}

type MetadataResult struct {
	AssetID         string
	ContentType     string
	FileSizeBytes   int64
	Width           int32
	Height          int32
	DurationSeconds float64
	Codec           string
}

type Service struct {
	cfg Config

	assets      ports.AssetRepository
	jobs        ports.JobRepository
	outputs     ports.OutputRepository
	thumbnails  ports.ThumbnailRepository
	watermarks  ports.WatermarkRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository

	campaign ports.CampaignReader
	queue    ports.JobQueue
	dlq      ports.DLQPublisher
	nowFn    func() time.Time
}

type Dependencies struct {
	Config Config

	Assets      ports.AssetRepository
	Jobs        ports.JobRepository
	Outputs     ports.OutputRepository
	Thumbnails  ports.ThumbnailRepository
	Watermarks  ports.WatermarkRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository

	Campaign ports.CampaignReader
	Queue    ports.JobQueue
	DLQ      ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M06-Media-Processing-Pipeline"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.QueuePollInterval <= 0 {
		cfg.QueuePollInterval = 2 * time.Second
	}
	return &Service{
		cfg:         cfg,
		assets:      deps.Assets,
		jobs:        deps.Jobs,
		outputs:     deps.Outputs,
		thumbnails:  deps.Thumbnails,
		watermarks:  deps.Watermarks,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		campaign:    deps.Campaign,
		queue:       deps.Queue,
		dlq:         deps.DLQ,
		nowFn:       time.Now().UTC,
	}
}
