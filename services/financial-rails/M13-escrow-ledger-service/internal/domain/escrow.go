package domain

import "time"

const (
	HoldStatusActive         = "active"
	HoldStatusPartialRelease = "partial_release"
	HoldStatusFullyReleased  = "fully_released"
	HoldStatusRefunded       = "refunded"
)

type EscrowHold struct {
	EscrowID          string
	CampaignID        string
	CreatorID         string
	OriginalAmount    float64
	ReleasedAmount    float64
	RefundedAmount    float64
	RemainingAmount   float64
	Status            string
	HeldAt            time.Time
	UpdatedAt         time.Time
}

type LedgerEntry struct {
	EntryID      string
	EscrowID     string
	CampaignID   string
	EntryType    string
	Amount       float64
	OccurredAt   time.Time
}

type WalletBalance struct {
	CampaignID         string
	HeldBalance        float64
	ReleasedBalance    float64
	RefundedBalance    float64
	NetEscrowBalance   float64
	CalculatedAt       time.Time
}
