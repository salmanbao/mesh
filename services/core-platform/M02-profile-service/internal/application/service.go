package application

import (
	"time"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
)

type Service struct {
	cfg               Config
	profiles          ports.ProfileRepository
	socialLinks       ports.SocialLinkRepository
	payoutMethods     ports.PayoutMethodRepository
	kyc               ports.KYCRepository
	stats             ports.ProfileStatsRepository
	reservedUsernames ports.ReservedUsernameRepository
	usernameHistory   ports.UsernameHistoryRepository
	reads             ports.ReadRepository
	outbox            ports.OutboxRepository
	eventDedup        ports.EventDedupRepository
	idempotency       ports.IdempotencyRepository
	authClient        ports.AuthClient
	cache             ports.Cache
	encryption        ports.Encryption
	nowFn             func() time.Time
}

type Dependencies struct {
	Config            Config
	Profiles          ports.ProfileRepository
	SocialLinks       ports.SocialLinkRepository
	PayoutMethods     ports.PayoutMethodRepository
	KYC               ports.KYCRepository
	Stats             ports.ProfileStatsRepository
	ReservedUsernames ports.ReservedUsernameRepository
	UsernameHistory   ports.UsernameHistoryRepository
	Reads             ports.ReadRepository
	Outbox            ports.OutboxRepository
	EventDedup        ports.EventDedupRepository
	Idempotency       ports.IdempotencyRepository
	AuthClient        ports.AuthClient
	Cache             ports.Cache
	Encryption        ports.Encryption
}

func NewService(deps Dependencies) *Service {
	cfg := deps.Config
	if cfg.ServiceName == "" {
		cfg.ServiceName = "M02-Profile-Service"
	}
	if cfg.ProfileCacheTTL <= 0 {
		cfg.ProfileCacheTTL = 5 * time.Minute
	}
	if cfg.UsernameCooldownDays <= 0 {
		cfg.UsernameCooldownDays = 365
	}
	if cfg.UsernameRedirectDays <= 0 {
		cfg.UsernameRedirectDays = 90
	}
	if cfg.MaxSocialLinks <= 0 {
		cfg.MaxSocialLinks = 5
	}
	if cfg.IdempotencyTTL <= 0 {
		cfg.IdempotencyTTL = 7 * 24 * time.Hour
	}
	if cfg.EventDedupTTL <= 0 {
		cfg.EventDedupTTL = 7 * 24 * time.Hour
	}
	if cfg.KYCAnonymizeAfter <= 0 {
		cfg.KYCAnonymizeAfter = 365 * 24 * time.Hour
	}
	if cfg.UsernameHistoryRetention <= 0 {
		cfg.UsernameHistoryRetention = 365 * 24 * time.Hour
	}

	return &Service{
		cfg:               cfg,
		profiles:          deps.Profiles,
		socialLinks:       deps.SocialLinks,
		payoutMethods:     deps.PayoutMethods,
		kyc:               deps.KYC,
		stats:             deps.Stats,
		reservedUsernames: deps.ReservedUsernames,
		usernameHistory:   deps.UsernameHistory,
		reads:             deps.Reads,
		outbox:            deps.Outbox,
		eventDedup:        deps.EventDedup,
		idempotency:       deps.Idempotency,
		authClient:        deps.AuthClient,
		cache:             deps.Cache,
		encryption:        deps.Encryption,
		nowFn:             time.Now().UTC,
	}
}
