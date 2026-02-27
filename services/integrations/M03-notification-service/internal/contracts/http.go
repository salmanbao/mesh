package contracts

type SuccessResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorPayload struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type ErrorResponse struct {
	Status string       `json:"status"`
	Error  ErrorPayload `json:"error"`
}

type NotificationItem struct {
	NotificationID string            `json:"notification_id"`
	UserID         string            `json:"user_id"`
	Type           string            `json:"type"`
	Title          string            `json:"title"`
	Body           string            `json:"body"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      string            `json:"created_at"`
	ReadAt         string            `json:"read_at,omitempty"`
	ArchivedAt     string            `json:"archived_at,omitempty"`
}

type ListNotificationsResponse struct {
	Items       []NotificationItem `json:"items"`
	NextCursor  string             `json:"next_cursor,omitempty"`
	UnreadCount int                `json:"unread_count"`
}

type UnreadCountResponse struct {
	UnreadCount int `json:"unread_count"`
}

type MarkStateResponse struct {
	NotificationID string `json:"notification_id"`
	Status         string `json:"status"`
}

type BulkActionRequest struct {
	Action          string   `json:"action"`
	NotificationIDs []string `json:"notification_ids"`
}

type BulkActionResponse struct {
	Action    string `json:"action"`
	Processed int    `json:"processed"`
	Failed    int    `json:"failed"`
}

type PreferencesResponse struct {
	UserID            string   `json:"user_id"`
	EmailEnabled      bool     `json:"email_enabled"`
	PushEnabled       bool     `json:"push_enabled"`
	SMSEnabled        bool     `json:"sms_enabled"`
	InAppEnabled      bool     `json:"in_app_enabled"`
	QuietHoursEnabled bool     `json:"quiet_hours_enabled"`
	QuietHoursStart   string   `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd     string   `json:"quiet_hours_end,omitempty"`
	QuietHoursTZ      string   `json:"quiet_hours_timezone,omitempty"`
	Language          string   `json:"language,omitempty"`
	BatchingEnabled   bool     `json:"batching_enabled"`
	MutedTypes        []string `json:"muted_types,omitempty"`
	UpdatedAt         string   `json:"updated_at"`
}

type UpdatePreferencesRequest struct {
	EmailEnabled      *bool    `json:"email_enabled,omitempty"`
	PushEnabled       *bool    `json:"push_enabled,omitempty"`
	SMSEnabled        *bool    `json:"sms_enabled,omitempty"`
	InAppEnabled      *bool    `json:"in_app_enabled,omitempty"`
	QuietHoursEnabled *bool    `json:"quiet_hours_enabled,omitempty"`
	QuietHoursStart   string   `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd     string   `json:"quiet_hours_end,omitempty"`
	QuietHoursTZ      string   `json:"quiet_hours_timezone,omitempty"`
	Language          string   `json:"language,omitempty"`
	BatchingEnabled   *bool    `json:"batching_enabled,omitempty"`
	MutedTypes        []string `json:"muted_types,omitempty"`
}

type DeleteScheduledResponse struct {
	ScheduledID string `json:"scheduled_id"`
	Cancelled   bool   `json:"cancelled"`
}
