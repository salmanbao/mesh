package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
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

type ValidatePostRequest struct {
	UserID   string `json:"user_id"`
	Platform string `json:"platform"`
	PostURL  string `json:"post_url"`
}

type ValidatePostResponse struct {
	Valid         bool   `json:"valid"`
	Platform      string `json:"platform"`
	NormalizedURL string `json:"normalized_url"`
	Reason        string `json:"reason,omitempty"`
}

type RegisterPostRequest struct {
	UserID             string `json:"user_id"`
	Platform           string `json:"platform"`
	PostURL            string `json:"post_url"`
	DistributionItemID string `json:"distribution_item_id,omitempty"`
	CampaignID         string `json:"campaign_id,omitempty"`
}

type TrackedPostResponse struct {
	TrackedPostID      string `json:"tracked_post_id"`
	UserID             string `json:"user_id"`
	Platform           string `json:"platform"`
	PostURL            string `json:"post_url"`
	DistributionItemID string `json:"distribution_item_id,omitempty"`
	CampaignID         string `json:"campaign_id,omitempty"`
	Status             string `json:"status"`
	ValidationStatus   string `json:"validation_status"`
	LastPolledAt       string `json:"last_polled_at,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

type MetricSnapshotResponse struct {
	SnapshotID    string `json:"snapshot_id"`
	TrackedPostID string `json:"tracked_post_id"`
	Platform      string `json:"platform"`
	Views         int    `json:"views"`
	Likes         int    `json:"likes"`
	Shares        int    `json:"shares"`
	Comments      int    `json:"comments"`
	PolledAt      string `json:"polled_at"`
}

type MetricsListResponse struct {
	TrackedPostID string                   `json:"tracked_post_id"`
	LastPolledAt  string                   `json:"last_polled_at,omitempty"`
	Snapshots     []MetricSnapshotResponse `json:"snapshots"`
}
