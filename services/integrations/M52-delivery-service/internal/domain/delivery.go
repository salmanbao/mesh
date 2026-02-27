package domain

import "time"

type ProductFile struct {
	FileID      string    `json:"file_id"`
	ProductID   string    `json:"product_id"`
	FileName    string    `json:"file_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DownloadToken struct {
	TokenID        string     `json:"token_id"`
	Token          string     `json:"token"`
	ProductID      string     `json:"product_id"`
	UserID         string     `json:"user_id"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	DownloadCount  int        `json:"download_count"`
	MaxDownloads   int        `json:"max_downloads"`
	SingleUse      bool       `json:"single_use"`
	Revoked        bool       `json:"revoked"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	LastDownloadAt *time.Time `json:"last_download_at,omitempty"`
}

type DownloadEvent struct {
	DownloadID      string    `json:"download_id"`
	TokenID         string    `json:"token_id"`
	ProductID       string    `json:"product_id"`
	UserID          string    `json:"user_id,omitempty"`
	IPAddress       string    `json:"ip_address"`
	Timestamp       time.Time `json:"timestamp"`
	DownloadStatus  string    `json:"download_status"`
	BytesTotal      int64     `json:"bytes_total"`
	BytesDownloaded int64     `json:"bytes_downloaded"`
	DurationMillis  int64     `json:"duration_millis"`
}

type DownloadRevocationAudit struct {
	RevocationID string    `json:"revocation_id"`
	TokenID      string    `json:"token_id"`
	ProductID    string    `json:"product_id"`
	UserID       string    `json:"user_id"`
	RevokedAt    time.Time `json:"revoked_at"`
	Reason       string    `json:"reason"`
	RevokedBy    string    `json:"revoked_by"`
}
