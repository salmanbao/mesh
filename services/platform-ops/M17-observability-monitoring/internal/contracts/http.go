package contracts

type SuccessResponse struct {
	Status string `json:"status"`
	Data   any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type HealthCheckResponse struct {
	Status        string                       `json:"status"`
	Timestamp     string                       `json:"timestamp"`
	UptimeSeconds int64                        `json:"uptime_seconds,omitempty"`
	Version       string                       `json:"version,omitempty"`
	Checks        map[string]ComponentResponse `json:"checks"`
}

type ComponentResponse struct {
	Status           string            `json:"status"`
	LatencyMS        int               `json:"latency_ms,omitempty"`
	BrokersConnected int               `json:"brokers_connected,omitempty"`
	LastChecked      string            `json:"last_checked,omitempty"`
	Error            string            `json:"error,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type UpsertComponentRequest struct {
	Status           string            `json:"status"`
	Critical         *bool             `json:"critical,omitempty"`
	LatencyMS        *int              `json:"latency_ms,omitempty"`
	BrokersConnected *int              `json:"brokers_connected,omitempty"`
	Error            string            `json:"error,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type UpsertComponentResponse struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Critical    bool   `json:"critical"`
	LastChecked string `json:"last_checked"`
}

type ComponentsListResponse struct {
	Items []NamedComponentResponse `json:"items"`
}

type NamedComponentResponse struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Critical         bool              `json:"critical"`
	LatencyMS        int               `json:"latency_ms,omitempty"`
	BrokersConnected int               `json:"brokers_connected,omitempty"`
	LastChecked      string            `json:"last_checked,omitempty"`
	Error            string            `json:"error,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}
