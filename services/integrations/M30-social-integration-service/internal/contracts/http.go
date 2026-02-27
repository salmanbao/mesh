package contracts

import "encoding/json"

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

type ConnectAccountRequest struct {
	UserID    string `json:"user_id,omitempty"`
	Platform  string `json:"platform"`
	Handle    string `json:"handle,omitempty"`
	OAuthCode string `json:"oauth_code,omitempty"`
}

type ConnectAccountResponse struct {
	SocialAccountID string `json:"social_account_id"`
	UserID          string `json:"user_id"`
	Platform        string `json:"platform"`
	Handle          string `json:"handle"`
	Status          string `json:"status"`
	ConnectedAt     string `json:"connected_at"`
}

type SocialAccountItem struct {
	SocialAccountID string `json:"social_account_id"`
	UserID          string `json:"user_id"`
	Platform        string `json:"platform"`
	Handle          string `json:"handle,omitempty"`
	Status          string `json:"status"`
	ConnectedAt     string `json:"connected_at"`
	Source          string `json:"source"`
}

type ListAccountsResponse struct {
	Accounts []SocialAccountItem `json:"accounts"`
}

type ValidatePostRequest struct {
	UserID   string `json:"user_id,omitempty"`
	Platform string `json:"platform"`
	PostID   string `json:"post_id"`
}

type ValidatePostResponse struct {
	ValidationID string `json:"validation_id"`
	UserID       string `json:"user_id"`
	Platform     string `json:"platform"`
	PostID       string `json:"post_id"`
	IsValid      bool   `json:"is_valid"`
	Reason       string `json:"reason,omitempty"`
	ValidatedAt  string `json:"validated_at"`
}

type HealthResponse struct {
	Status        string          `json:"status"`
	Timestamp     string          `json:"timestamp"`
	UptimeSeconds int64           `json:"uptime_seconds"`
	Version       string          `json:"version"`
	Checks        json.RawMessage `json:"checks"`
}
