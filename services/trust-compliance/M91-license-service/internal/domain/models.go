package domain

import "time"

type License struct {
	ID             string    `json:"id"`
	LicenseKey     string    `json:"license_key"`
	ProductID      string    `json:"product_id"`
	TransactionID  string    `json:"transaction_id"`
	UserID         string    `json:"user_id,omitempty"`
	Model          string    `json:"model"`
	MaxActivations int       `json:"max_activations"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Activation struct {
	ID                string    `json:"id"`
	LicenseID         string    `json:"license_id"`
	DeviceID          string    `json:"device_id"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	IPHash            string    `json:"ip_hash"`
	ActivatedAt       time.Time `json:"activated_at"`
	DeactivatedAt     time.Time `json:"deactivated_at,omitempty"`
	Status            string    `json:"status"`
}

type Revocation struct {
	ID        string    `json:"id"`
	LicenseID string    `json:"license_id"`
	Reason    string    `json:"reason"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductConfig struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	Model          string `json:"model"`
	MaxActivations int    `json:"max_activations"`
}

type ExportRecord struct {
	Format      string    `json:"format"`
	GeneratedAt time.Time `json:"generated_at"`
	Rows        []License `json:"rows"`
}

type IdempotencyRecord struct {
	Key         string
	RequestHash string
	Response    []byte
	ExpiresAt   time.Time
}
