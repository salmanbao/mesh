package domain

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	AlertStatusFiring   = "firing"
	AlertStatusResolved = "resolved"

	IncidentStatusInvestigating = "investigating"
	IncidentStatusMitigated     = "mitigated"
	IncidentStatusResolved      = "resolved"
)

type AlertRule struct {
	RuleID          string          `json:"rule_id"`
	Name            string          `json:"name"`
	Query           string          `json:"query"`
	Threshold       float64         `json:"threshold"`
	DurationSeconds int             `json:"duration_seconds"`
	Service         string          `json:"service,omitempty"`
	Regex           string          `json:"regex,omitempty"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Severity        string          `json:"severity"`
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Alert struct {
	AlertID    string     `json:"alert_id"`
	RuleID     string     `json:"rule_id"`
	Status     string     `json:"status"`
	FiredAt    time.Time  `json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Incident struct {
	IncidentID string     `json:"incident_id"`
	AlertID    string     `json:"alert_id"`
	Service    string     `json:"service"`
	Severity   string     `json:"severity"`
	Status     string     `json:"status"`
	Assignee   string     `json:"assignee,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

type Silence struct {
	SilenceID string    `json:"silence_id"`
	RuleID    string    `json:"rule_id"`
	CreatedBy string    `json:"created_by"`
	Reason    string    `json:"reason"`
	StartAt   time.Time `json:"start_at"`
	EndAt     time.Time `json:"end_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Dashboard struct {
	DashboardID string          `json:"dashboard_id"`
	Name        string          `json:"name"`
	OwnerID     string          `json:"owner_id"`
	Layout      json.RawMessage `json:"layout"`
	Version     int             `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
}

type IncidentQuery struct {
	Status string
	Limit  int
}

type AuditLog struct {
	AuditID    string          `json:"audit_id"`
	ActorID    string          `json:"actor_id"`
	ActionType string          `json:"action_type"`
	ActionAt   time.Time       `json:"action_at"`
	IPAddress  string          `json:"ip_address,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

type AuditQuery struct {
	ActorID    string
	ActionType string
	Limit      int
}

type AuditQueryResult struct {
	Logs []AuditLog `json:"logs"`
}

type HealthReport struct {
	Status        string                    `json:"status"`
	Timestamp     time.Time                 `json:"timestamp"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version"`
	Checks        map[string]ComponentCheck `json:"checks"`
}

type ComponentCheck struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LatencyMS   int       `json:"latency_ms"`
	LastChecked time.Time `json:"last_checked"`
}

type MetricsSnapshot struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}

func IsValidSeverity(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case SeverityInfo, SeverityWarning, SeverityCritical:
		return true
	default:
		return false
	}
}

func NormalizeSeverity(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func IsValidAlertStatus(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case AlertStatusFiring, AlertStatusResolved:
		return true
	default:
		return false
	}
}

func NormalizeAlertStatus(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func IsValidIncidentStatus(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case IncidentStatusInvestigating, IncidentStatusMitigated, IncidentStatusResolved:
		return true
	default:
		return false
	}
}

func NormalizeIncidentStatus(v string) string { return strings.ToLower(strings.TrimSpace(v)) }
