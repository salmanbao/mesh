package application

import (
	"time"

	"github.com/viralforge/mesh/services/integrations/M52-delivery-service/internal/ports"
)

type Config struct {
	ServiceName          string
	PublicBaseURL        string
	DefaultTokenTTL      time.Duration
	DefaultMaxDownloads  int
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type UpsertProductFileInput struct {
	ProductID   string
	FileID      string
	FileName    string
	ContentType string
	SizeBytes   int64
	Status      string
}

type GetDownloadLinkInput struct {
	ProductID     string
	TokenTTLHours int
	MaxDownloads  int
}

type DownloadRequest struct {
	Token       string
	IPAddress   string
	RangeHeader string
}

type RevokeLinksInput struct {
	ProductID string
	UserID    string
	Reason    string
}

type RevokeLinksResult struct {
	ProductID    string
	UserID       string
	RevokedCount int
	RevokedAt    time.Time
}

type DownloadLinkResult struct {
	Token              string
	DownloadURL        string
	ExpiresAt          time.Time
	ExpiresInHours     int
	DownloadsRemaining int
	SingleUse          bool
	ProductName        string
	FileCount          int
	TotalSizeMB        float64
}

type DownloadResult struct {
	ProductID          string
	FileID             string
	FileName           string
	ContentType        string
	BytesTotal         int64
	DownloadsRemaining int
}

type Service struct {
	cfg         Config
	files       ports.ProductFileRepository
	tokens      ports.DownloadTokenRepository
	downloads   ports.DownloadEventRepository
	revocations ports.DownloadRevocationAuditRepository
	idempotency ports.IdempotencyRepository
	eventDedup  ports.EventDedupRepository
	nowFn       func() time.Time
}

type Dependencies struct {
	Config      Config
	Files       ports.ProductFileRepository
	Tokens      ports.DownloadTokenRepository
	Downloads   ports.DownloadEventRepository
	Revocations ports.DownloadRevocationAuditRepository
	Idempotency ports.IdempotencyRepository
	EventDedup  ports.EventDedupRepository
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M52-Delivery-Service"
	}
	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "http://localhost:8080"
	}
	if cfg.DefaultTokenTTL <= 0 {
		cfg.DefaultTokenTTL = 24 * time.Hour
	}
	if cfg.DefaultMaxDownloads <= 0 {
		cfg.DefaultMaxDownloads = 5
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.ConsumerPollInterval <= 0 {
		cfg.ConsumerPollInterval = 2 * time.Second
	}
	return &Service{
		cfg:         cfg,
		files:       deps.Files,
		tokens:      deps.Tokens,
		downloads:   deps.Downloads,
		revocations: deps.Revocations,
		idempotency: deps.Idempotency,
		eventDedup:  deps.EventDedup,
		nowFn:       func() time.Time { return time.Now().UTC() },
	}
}
