package contracts

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

type GetConfigResponse struct {
	Environment  string         `json:"environment"`
	ServiceScope string         `json:"service_scope"`
	Values       map[string]any `json:"values"`
}

type PatchConfigRequest struct {
	Environment  string `json:"environment"`
	ServiceScope string `json:"service_scope"`
	Value        any    `json:"value"`
	ValueType    string `json:"value_type"`
}

type PatchConfigResponse struct {
	Key          string `json:"key"`
	Environment  string `json:"environment"`
	ServiceScope string `json:"service_scope"`
	Version      int    `json:"version"`
}

type RolloutRuleRequest struct {
	Key        string `json:"key"`
	RuleType   string `json:"rule_type"`
	Percentage int    `json:"percentage,omitempty"`
	Role       string `json:"role,omitempty"`
	Tier       string `json:"tier,omitempty"`
}

type RolloutRuleResponse struct {
	RuleID    string `json:"rule_id"`
	Key       string `json:"key"`
	RuleType  string `json:"rule_type"`
	CreatedAt string `json:"created_at"`
}

type ImportConfigRequest struct {
	Environment  string             `json:"environment"`
	ServiceScope string             `json:"service_scope"`
	Entries      []PatchConfigEntry `json:"entries"`
}

type PatchConfigEntry struct {
	Key       string `json:"key"`
	ValueType string `json:"value_type"`
	Value     any    `json:"value"`
}

type ImportConfigResponse struct {
	AppliedCount int `json:"applied_count"`
}

type ExportConfigResponse struct {
	Version      int                   `json:"version"`
	GeneratedAt  string                `json:"generated_at"`
	Environment  string                `json:"environment"`
	ServiceScope string                `json:"service_scope"`
	Values       map[string]any        `json:"values"`
	Meta         map[string]ExportMeta `json:"meta"`
}

type ExportMeta struct {
	ValueType  string `json:"value_type"`
	UpdatedAt  string `json:"updated_at"`
	KeyVersion int    `json:"key_version"`
}

type RollbackRequest struct {
	Key          string `json:"key"`
	Environment  string `json:"environment"`
	ServiceScope string `json:"service_scope"`
	Version      int    `json:"version"`
}

type RollbackResponse struct {
	Key          string `json:"key"`
	Environment  string `json:"environment"`
	ServiceScope string `json:"service_scope"`
	Version      int    `json:"version"`
	RolledBackTo int    `json:"rolled_back_to"`
}

type AuditLogItem struct {
	AuditID      string `json:"audit_id"`
	ActionType   string `json:"action_type"`
	KeyName      string `json:"key_name"`
	ActorID      string `json:"actor_id"`
	Environment  string `json:"environment,omitempty"`
	ServiceScope string `json:"service_scope,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	ActionAt     string `json:"action_at"`
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
