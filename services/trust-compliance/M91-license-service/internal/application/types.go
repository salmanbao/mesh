package application

import "time"

type Config struct {
	ServiceName    string
	Version        string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
	ClientIP       string
}

type ActivateInput struct {
	LicenseKey        string `json:"license_key"`
	DeviceID          string `json:"device_id"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type DeactivateInput struct {
	LicenseKey string `json:"license_key"`
	DeviceID   string `json:"device_id"`
}

type ExportInput struct {
	Format string `json:"format"`
}
