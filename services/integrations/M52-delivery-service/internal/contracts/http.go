package contracts

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type UpsertProductFileRequest struct {
	FileID      string `json:"file_id,omitempty"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Status      string `json:"status,omitempty"`
}

type ProductFileResponse struct {
	FileID      string `json:"file_id"`
	ProductID   string `json:"product_id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type DownloadLinkResponse struct {
	DownloadURL        string  `json:"download_url"`
	ExpiresAt          string  `json:"expires_at"`
	ExpiresInHours     int     `json:"expires_in_hours"`
	DownloadsRemaining int     `json:"downloads_remaining"`
	SingleUse          bool    `json:"single_use"`
	ProductName        string  `json:"product_name"`
	FileCount          int     `json:"file_count"`
	TotalSizeMB        float64 `json:"total_size_mb"`
	Token              string  `json:"token"`
}

type DownloadMetadataResponse struct {
	ProductID          string `json:"product_id"`
	FileID             string `json:"file_id"`
	FileName           string `json:"file_name"`
	ContentType        string `json:"content_type"`
	BytesTotal         int64  `json:"bytes_total"`
	DownloadsRemaining int    `json:"downloads_remaining"`
}

type RevokeLinksRequest struct {
	ProductID string `json:"product_id"`
	UserID    string `json:"user_id"`
	Reason    string `json:"reason"`
}

type RevokeLinksResponse struct {
	ProductID      string `json:"product_id"`
	UserID         string `json:"user_id"`
	RevokedCount   int    `json:"revoked_count"`
	RevocationTime string `json:"revocation_time"`
}
