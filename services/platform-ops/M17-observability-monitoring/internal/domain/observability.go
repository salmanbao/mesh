package domain

import "time"

const (
	StatusHealthy   = "healthy"
	StatusDegraded  = "degraded"
	StatusUnhealthy = "unhealthy"
)

type ComponentCheck struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Critical         bool              `json:"critical"`
	LatencyMS        int               `json:"latency_ms,omitempty"`
	BrokersConnected int               `json:"brokers_connected,omitempty"`
	Error            string            `json:"error,omitempty"`
	LastChecked      time.Time         `json:"last_checked"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds,omitempty"`
	Version       string                    `json:"version,omitempty"`
	Checks        map[string]ComponentCheck `json:"checks"`
}

func NormalizeStatus(v string) string {
	switch v {
	case StatusHealthy, StatusDegraded, StatusUnhealthy:
		return v
	default:
		return ""
	}
}
