package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type CreateHoldRequest struct {
	CampaignID string  `json:"campaign_id"`
	CreatorID  string  `json:"creator_id"`
	Amount     float64 `json:"amount"`
}

type ReleaseHoldRequest struct {
	EscrowID string  `json:"escrow_id"`
	Amount   float64 `json:"amount"`
}

type RefundHoldRequest struct {
	EscrowID string   `json:"escrow_id"`
	Amount   *float64 `json:"amount,omitempty"`
}

type HoldResponse struct {
	EscrowID         string  `json:"escrow_id"`
	CampaignID       string  `json:"campaign_id"`
	CreatorID        string  `json:"creator_id"`
	Status           string  `json:"status"`
	OriginalAmount   float64 `json:"original_amount"`
	RemainingAmount  float64 `json:"remaining_amount"`
	ReleasedAmount   float64 `json:"released_amount"`
	RefundedAmount   float64 `json:"refunded_amount"`
	EventDelivery    string  `json:"event_delivery,omitempty"`
}

type WalletBalanceResponse struct {
	CampaignID       string  `json:"campaign_id"`
	HeldBalance      float64 `json:"held_balance"`
	ReleasedBalance  float64 `json:"released_balance"`
	RefundedBalance  float64 `json:"refunded_balance"`
	NetEscrowBalance float64 `json:"net_escrow_balance"`
}
