package domain

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	ValueTypeString    = "string"
	ValueTypeNumber    = "number"
	ValueTypeBoolean   = "boolean"
	ValueTypeJSON      = "json"
	ValueTypeEncrypted = "encrypted"

	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvProduction  = "production"

	GlobalServiceScope = "global"

	RuleTypePercentage = "percentage"
	RuleTypeRole       = "role"
	RuleTypeTier       = "tier"
)

type ConfigKey struct {
	KeyID       string    `json:"key_id"`
	KeyName     string    `json:"key_name"`
	ValueType   string    `json:"value_type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastVersion int       `json:"last_version"`
}

type ConfigValue struct {
	ValueID        string          `json:"value_id"`
	KeyID          string          `json:"key_id"`
	Environment    string          `json:"environment"`
	ServiceScope   string          `json:"service_scope"`
	ValueJSON      json.RawMessage `json:"value_json,omitempty"`
	ValueEncrypted string          `json:"value_encrypted,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type ConfigVersion struct {
	VersionID     string          `json:"version_id"`
	VersionNumber int             `json:"version_number"`
	KeyID         string          `json:"key_id"`
	KeyName       string          `json:"key_name"`
	Environment   string          `json:"environment"`
	ServiceScope  string          `json:"service_scope"`
	OldValue      json.RawMessage `json:"old_value,omitempty"`
	NewValue      json.RawMessage `json:"new_value,omitempty"`
	ChangedBy     string          `json:"changed_by"`
	ChangedAt     time.Time       `json:"changed_at"`
}

type RolloutRule struct {
	RuleID    string          `json:"rule_id"`
	KeyID     string          `json:"key_id"`
	KeyName   string          `json:"key_name"`
	RuleType  string          `json:"rule_type"`
	RuleValue json.RawMessage `json:"rule_value"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type AuditLog struct {
	AuditID      string          `json:"audit_id"`
	ActionType   string          `json:"action_type"`
	KeyID        string          `json:"key_id"`
	KeyName      string          `json:"key_name"`
	ActorID      string          `json:"actor_id"`
	Environment  string          `json:"environment,omitempty"`
	ServiceScope string          `json:"service_scope,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	ChangeDetail json.RawMessage `json:"change_details,omitempty"`
	ActionAt     time.Time       `json:"action_at"`
}

type AuditQuery struct {
	KeyName      string
	Environment  string
	ServiceScope string
	ActorID      string
	Limit        int
}

type AuditQueryResult struct {
	Logs []AuditLog `json:"logs"`
}

type ExportSnapshot struct {
	Version      int                   `json:"version"`
	GeneratedAt  time.Time             `json:"generated_at"`
	Environment  string                `json:"environment"`
	ServiceScope string                `json:"service_scope"`
	Values       map[string]any        `json:"values"`
	Meta         map[string]ExportMeta `json:"meta"`
}

type ExportMeta struct {
	ValueType  string    `json:"value_type"`
	UpdatedAt  time.Time `json:"updated_at"`
	KeyVersion int       `json:"key_version"`
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

func IsValidValueType(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case ValueTypeString, ValueTypeNumber, ValueTypeBoolean, ValueTypeJSON, ValueTypeEncrypted:
		return true
	default:
		return false
	}
}

func NormalizeValueType(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func IsValidEnvironment(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case EnvDevelopment, EnvStaging, EnvProduction:
		return true
	default:
		return false
	}
}

func NormalizeEnvironment(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func NormalizeServiceScope(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return GlobalServiceScope
	}
	return strings.ToLower(v)
}

func IsValidRuleType(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case RuleTypePercentage, RuleTypeRole, RuleTypeTier:
		return true
	default:
		return false
	}
}
