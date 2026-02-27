package application

import (
	"time"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/ports"
)

type Config struct {
	ServiceName          string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	OutboxFlushBatchSize int
	WebhookBearerToken   string
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type FileDisputeInput struct {
	TransactionID string
	DisputeType   string
	Reason        string
	BuyerClaim    string
}

type SubmitEvidenceInput struct {
	Filename    string
	Description string
	FileURL     string
	SizeBytes   int64
	MimeType    string
}

type ChargebackInput struct {
	EventID          string
	EventType        string
	OccurredAt       string
	SourceService    string
	TraceID          string
	SchemaVersion    string
	PartitionKeyPath string
	PartitionKey     string
	Amount           float64
	ChargeID         string
	Currency         string
	DisputeReason    string
	SellerID         string
}

type Service struct {
	cfg Config

	riskProfiles ports.SellerRiskProfileRepository
	escrow       ports.SellerEscrowRepository
	disputes     ports.DisputeLogRepository
	evidence     ports.DisputeEvidenceRepository
	fraudFlags   ports.FraudPatternFlagRepository
	reserveLogs  ports.ReserveTriggerLogRepository
	debtLogs     ports.SellerDebtLogRepository
	suspensions  ports.SellerSuspensionLogRepository
	idempotency  ports.IdempotencyRepository
	eventDedup   ports.EventDedupRepository
	outbox       ports.OutboxRepository

	auth       ports.AuthReader
	profile    ports.ProfileReader
	fraud      ports.FraudReader
	moderation ports.ModerationReader
	resolution ports.ResolutionReader
	reputation ports.ReputationReader

	domainEvents ports.DomainPublisher
	analytics    ports.AnalyticsPublisher
	dlq          ports.DLQPublisher
	nowFn        func() time.Time
}

type Dependencies struct {
	Config       Config
	RiskProfiles ports.SellerRiskProfileRepository
	Escrow       ports.SellerEscrowRepository
	Disputes     ports.DisputeLogRepository
	Evidence     ports.DisputeEvidenceRepository
	FraudFlags   ports.FraudPatternFlagRepository
	ReserveLogs  ports.ReserveTriggerLogRepository
	DebtLogs     ports.SellerDebtLogRepository
	Suspensions  ports.SellerSuspensionLogRepository
	Idempotency  ports.IdempotencyRepository
	EventDedup   ports.EventDedupRepository
	Outbox       ports.OutboxRepository
	Auth         ports.AuthReader
	Profile      ports.ProfileReader
	Fraud        ports.FraudReader
	Moderation   ports.ModerationReader
	Resolution   ports.ResolutionReader
	Reputation   ports.ReputationReader
	DomainEvents ports.DomainPublisher
	Analytics    ports.AnalyticsPublisher
	DLQ          ports.DLQPublisher
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M36-Risk-Service"
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.OutboxFlushBatchSize <= 0 {
		cfg.OutboxFlushBatchSize = 100
	}
	if cfg.WebhookBearerToken == "" {
		cfg.WebhookBearerToken = "risk-webhook-secret"
	}
	return &Service{
		cfg:          cfg,
		riskProfiles: deps.RiskProfiles,
		escrow:       deps.Escrow,
		disputes:     deps.Disputes,
		evidence:     deps.Evidence,
		fraudFlags:   deps.FraudFlags,
		reserveLogs:  deps.ReserveLogs,
		debtLogs:     deps.DebtLogs,
		suspensions:  deps.Suspensions,
		idempotency:  deps.Idempotency,
		eventDedup:   deps.EventDedup,
		outbox:       deps.Outbox,
		auth:         deps.Auth,
		profile:      deps.Profile,
		fraud:        deps.Fraud,
		moderation:   deps.Moderation,
		resolution:   deps.Resolution,
		reputation:   deps.Reputation,
		domainEvents: deps.DomainEvents,
		analytics:    deps.Analytics,
		dlq:          deps.DLQ,
		nowFn:        func() time.Time { return time.Now().UTC() },
	}
}
