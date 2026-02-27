package contracts

import "encoding/json"

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type CreateAlertRuleRequest struct {
	Name            string  `json:"name"`
	Query           string  `json:"query"`
	Threshold       float64 `json:"threshold"`
	DurationSeconds int     `json:"duration_seconds"`
	Severity        string  `json:"severity"`
	Enabled         bool    `json:"enabled"`
	Service         string  `json:"service,omitempty"`
	Regex           string  `json:"regex,omitempty"`
}

type CreateAlertRuleResponse struct {
	RuleID string `json:"rule_id"`
}

type AlertRuleItem struct {
	RuleID          string  `json:"rule_id"`
	Name            string  `json:"name"`
	Query           string  `json:"query"`
	Threshold       float64 `json:"threshold"`
	DurationSeconds int     `json:"duration_seconds"`
	Severity        string  `json:"severity"`
	Enabled         bool    `json:"enabled"`
	Service         string  `json:"service,omitempty"`
	Regex           string  `json:"regex,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type ListAlertRulesResponse struct {
	Items []AlertRuleItem `json:"items"`
}

type IncidentItem struct {
	IncidentID string `json:"incident_id"`
	AlertID    string `json:"alert_id"`
	Service    string `json:"service"`
	Severity   string `json:"severity"`
	Status     string `json:"status"`
	Assignee   string `json:"assignee,omitempty"`
	CreatedAt  string `json:"created_at"`
	ResolvedAt string `json:"resolved_at,omitempty"`
}

type ListIncidentsResponse struct {
	Items []IncidentItem `json:"items"`
}

type CreateSilenceRequest struct {
	RuleID  string `json:"rule_id"`
	Reason  string `json:"reason"`
	StartAt string `json:"start_at"`
	EndAt   string `json:"end_at"`
}

type CreateSilenceResponse struct {
	SilenceID string `json:"silence_id"`
}

type AuditLogItem struct {
	AuditID    string          `json:"audit_id"`
	ActorID    string          `json:"actor_id"`
	ActionType string          `json:"action_type"`
	ActionAt   string          `json:"action_at"`
	IPAddress  string          `json:"ip_address,omitempty"`
	Details    json.RawMessage `json:"details,omitempty"`
}

type AuditQueryResponse struct {
	Logs []AuditLogItem `json:"logs"`
}

type ServiceMetricsResponse struct {
	Hits            int64 `json:"hits"`
	Misses          int64 `json:"misses"`
	Evictions       int64 `json:"evictions"`
	MemoryUsedBytes int64 `json:"memory_used_bytes"`
}
