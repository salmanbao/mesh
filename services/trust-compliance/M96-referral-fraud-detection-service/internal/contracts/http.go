package contracts

type ScoreReferralRequest struct {
	EventID               string            `json:"event_id"`
	AffiliateID           string            `json:"affiliate_id"`
	ReferralToken         string            `json:"referral_token"`
	ReferrerID            string            `json:"referrer_id"`
	UserID                string            `json:"user_id"`
	ClickIP               string            `json:"click_ip"`
	UserAgent             string            `json:"user_agent"`
	DeviceFingerprintHash string            `json:"device_fingerprint_hash"`
	FormFillTimeMS        int               `json:"form_fill_time_ms"`
	MouseMovementCount    int               `json:"mouse_movement_count"`
	KeyboardCPS           float64           `json:"keyboard_cps"`
	Amount                float64           `json:"amount"`
	Region                string            `json:"region"`
	CampaignType          string            `json:"campaign_type"`
	OccurredAt            string            `json:"occurred_at"`
	Metadata              map[string]string `json:"metadata"`
}

type SubmitDisputeRequest struct {
	DecisionID  string `json:"decision_id"`
	SubmittedBy string `json:"submitted_by"`
	EvidenceURL string `json:"evidence_url"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}
