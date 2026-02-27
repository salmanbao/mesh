package contracts

type RecommendationQuery struct {
	Role    string `json:"role,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Segment string `json:"segment,omitempty"`
}

type RecommendationFeedbackRequest struct {
	EventID          string                     `json:"event_id"`
	EventType        string                     `json:"event_type"`
	OccurredAt       string                     `json:"occurred_at"`
	SourceService    string                     `json:"source_service"`
	TraceID          string                     `json:"trace_id"`
	SchemaVersion    string                     `json:"schema_version"`
	PartitionKeyPath string                     `json:"partition_key_path"`
	PartitionKey     string                     `json:"partition_key"`
	Data             RecommendationFeedbackData `json:"data"`
}

type RecommendationFeedbackData struct {
	EntityID string `json:"entity_id"`
}

type RecommendationOverrideRequest struct {
	OverrideType string  `json:"override_type"`
	EntityID     string  `json:"entity_id"`
	Scope        string  `json:"scope"`
	ScopeValue   string  `json:"scope_value"`
	Multiplier   float64 `json:"multiplier"`
	Reason       string  `json:"reason"`
	EndDate      string  `json:"end_date,omitempty"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}
