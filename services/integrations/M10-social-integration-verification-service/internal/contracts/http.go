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
	Status    string       `json:"status"`
	Code      string       `json:"code,omitempty"`
	Message   string       `json:"message,omitempty"`
	RequestID string       `json:"request_id,omitempty"`
	Error     ErrorPayload `json:"error"`
}

type ConnectRequest struct {
	UserID string `json:"user_id"`
}

type ConnectResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}

type CallbackRequest struct {
	UserID string `json:"user_id"`
	Code   string `json:"code"`
	State  string `json:"state"`
	Handle string `json:"handle,omitempty"`
}

type CallbackResponse struct {
	SocialAccountID string `json:"social_account_id"`
	Provider        string `json:"provider,omitempty"`
	Status          string `json:"status"`
}

type SocialAccountResponse struct {
	SocialAccountID string `json:"social_account_id"`
	Provider        string `json:"provider"`
	Handle          string `json:"handle"`
	Status          string `json:"status"`
	ConnectedAt     string `json:"connected_at"`
}

type ListAccountsResponse struct {
	Accounts []SocialAccountResponse `json:"accounts"`
	Items    []SocialAccountResponse `json:"items,omitempty"`
}

type DisconnectResponse struct {
	SocialAccountID string `json:"social_account_id"`
	Status          string `json:"status"`
}

type PostValidationRequest struct {
	UserID   string `json:"user_id"`
	Platform string `json:"platform"`
	PostID   string `json:"post_id"`
}

type ComplianceViolationRequest struct {
	UserID   string `json:"user_id"`
	Platform string `json:"platform"`
	PostID   string `json:"post_id"`
	Reason   string `json:"reason"`
}

type FollowersSyncRequest struct {
	UserID        string `json:"user_id"`
	FollowerCount int    `json:"follower_count"`
}
