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

type ActivateRequest struct {
	LicenseKey        string `json:"license_key"`
	DeviceID          string `json:"device_id"`
	DeviceFingerprint string `json:"device_fingerprint"`
}

type DeactivateRequest struct {
	LicenseKey string `json:"license_key"`
	DeviceID   string `json:"device_id"`
}

type ExportRequest struct {
	Format string `json:"format"`
}
