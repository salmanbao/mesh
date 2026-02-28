package domain

import "time"

const (
	BatchStatusPending   = "pending"
	BatchStatusCompleted = "completed"
)

type Prediction struct {
	PredictionID string    `json:"prediction_id"`
	UserID       string    `json:"user_id"`
	ContentID    string    `json:"content_id,omitempty"`
	ContentHash  string    `json:"content_hash"`
	ModelID      string    `json:"model_id"`
	ModelVersion string    `json:"model_version"`
	Label        string    `json:"label"`
	Confidence   float64   `json:"confidence"`
	Flagged      bool      `json:"flagged"`
	CreatedAt    time.Time `json:"created_at"`
}

type BatchJob struct {
	JobID          string       `json:"job_id"`
	UserID         string       `json:"user_id"`
	Status         string       `json:"status"`
	ModelID        string       `json:"model_id"`
	ModelVersion   string       `json:"model_version"`
	RequestedCount int          `json:"requested_count"`
	CompletedCount int          `json:"completed_count"`
	CreatedAt      time.Time    `json:"created_at"`
	CompletedAt    *time.Time   `json:"completed_at,omitempty"`
	StatusURL      string       `json:"status_url"`
	PredictionIDs  []string     `json:"prediction_ids"`
	Predictions    []Prediction `json:"predictions,omitempty"`
}

type Model struct {
	ModelID     string    `json:"model_id"`
	Version     string    `json:"version"`
	DisplayName string    `json:"display_name"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
}

type FeedbackLog struct {
	FeedbackID   string    `json:"feedback_id"`
	PredictionID string    `json:"prediction_id"`
	UserID       string    `json:"user_id"`
	Feedback     string    `json:"feedback"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditLog struct {
	EventID    string            `json:"event_id"`
	EventType  string            `json:"event_type"`
	ActorID    string            `json:"actor_id"`
	EntityID   string            `json:"entity_id"`
	OccurredAt time.Time         `json:"occurred_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}
