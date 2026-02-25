package postgres

import (
	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/ports"
	"gorm.io/gorm"
)

type Repositories struct {
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
}

func NewRepositories(db *gorm.DB) Repositories {
	core := &profileRepository{db: db}
	return Repositories{
		Profiles:          core,
		SocialLinks:       &socialLinkRepository{db: db},
		PayoutMethods:     &payoutMethodRepository{db: db},
		KYC:               &kycRepository{db: db},
		Stats:             &profileStatsRepository{db: db},
		ReservedUsernames: &reservedUsernameRepository{db: db},
		UsernameHistory:   &usernameHistoryRepository{db: db},
		Reads:             &readRepository{db: db},
		Outbox:            &outboxRepository{db: db},
		EventDedup:        &eventDedupRepository{db: db},
		Idempotency:       &idempotencyRepository{db: db},
	}
}
