package contracts

import "time"

type CreateTransactionRequest struct {
	UserID                string  `json:"user_id"`
	CampaignID            string  `json:"campaign_id"`
	ProductID             string  `json:"product_id"`
	Provider              string  `json:"provider"`
	ProviderTransactionID string  `json:"provider_transaction_id,omitempty"`
	Amount                float64 `json:"amount"`
	Currency              string  `json:"currency"`
	TrafficSource         string  `json:"traffic_source,omitempty"`
	UserTier              string  `json:"user_tier,omitempty"`
}

type CreateRefundRequest struct {
	TransactionID string  `json:"transaction_id"`
	UserID        string  `json:"user_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}

type ProviderWebhookRequest struct {
	WebhookID             string  `json:"webhook_id"`
	Provider              string  `json:"provider"`
	EventType             string  `json:"event_type"`
	ProviderEventID       string  `json:"provider_event_id"`
	ProviderTransactionID string  `json:"provider_transaction_id"`
	TransactionID         string  `json:"transaction_id"`
	UserID                string  `json:"user_id"`
	Amount                float64 `json:"amount"`
	Currency              string  `json:"currency"`
	Reason                string  `json:"reason"`
	ReceivedAt            string  `json:"received_at,omitempty"`
}

type Pagination struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Total  int `json:"total"`
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

type WebhookAccepted struct {
	WebhookID string    `json:"webhook_id"`
	Status    string    `json:"status"`
	Processed time.Time `json:"processed"`
}
