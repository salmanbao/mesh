package domain

import "time"

type Notification struct {
	NotificationID  string            `json:"notification_id"`
	UserID          string            `json:"user_id"`
	Type            string            `json:"type"`
	Title           string            `json:"title"`
	Body            string            `json:"body"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	SourceEventID   string            `json:"source_event_id,omitempty"`
	SourceEventType string            `json:"source_event_type,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	ReadAt          *time.Time        `json:"read_at,omitempty"`
	ArchivedAt      *time.Time        `json:"archived_at,omitempty"`
}

type NotificationFilter struct {
	Type     string
	Status   string
	Page     int
	PageSize int
}

func (n Notification) IsUnread() bool   { return n.ReadAt == nil && n.ArchivedAt == nil }
func (n Notification) IsArchived() bool { return n.ArchivedAt != nil }
func (n Notification) IsRead() bool     { return n.ReadAt != nil }

func (n *Notification) MarkRead(at time.Time) {
	if n.ReadAt == nil {
		t := at.UTC()
		n.ReadAt = &t
	}
}

func (n *Notification) Archive(at time.Time) {
	if n.ArchivedAt == nil {
		t := at.UTC()
		n.ArchivedAt = &t
	}
}

type Preferences struct {
	UserID            string    `json:"user_id"`
	EmailEnabled      bool      `json:"email_enabled"`
	PushEnabled       bool      `json:"push_enabled"`
	SMSEnabled        bool      `json:"sms_enabled"`
	InAppEnabled      bool      `json:"in_app_enabled"`
	QuietHoursEnabled bool      `json:"quiet_hours_enabled"`
	QuietHoursStart   string    `json:"quiet_hours_start,omitempty"`
	QuietHoursEnd     string    `json:"quiet_hours_end,omitempty"`
	QuietHoursTZ      string    `json:"quiet_hours_timezone,omitempty"`
	Language          string    `json:"language,omitempty"`
	BatchingEnabled   bool      `json:"batching_enabled"`
	MutedTypes        []string  `json:"muted_types,omitempty"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func DefaultPreferences(userID string, now time.Time) Preferences {
	return Preferences{
		UserID:            userID,
		EmailEnabled:      true,
		PushEnabled:       true,
		SMSEnabled:        false,
		InAppEnabled:      true,
		QuietHoursEnabled: false,
		MutedTypes:        []string{},
		Language:          "en-US",
		QuietHoursTZ:      "UTC",
		BatchingEnabled:   false,
		UpdatedAt:         now.UTC(),
	}
}
