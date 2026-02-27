package domain

import (
	"strings"
	"time"
)

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

const (
	RoleClipper = "clipper"
	RoleCreator = "creator"
	RoleBuyer   = "buyer"
	RoleAdmin   = "admin"
)

const (
	EntityTypeCampaign   = "campaign"
	EntityTypeCreator    = "creator"
	EntityTypeSubmission = "submission"
)

const (
	OverrideTypePromoteCampaign  = "promote_campaign"
	OverrideTypeSuppressCampaign = "suppress_campaign"
	OverrideTypePromoteCreator   = "promote_creator"
	OverrideTypeSuppressCreator  = "suppress_creator"
)

const (
	FeedbackEventClick      = "recommendation.click"
	FeedbackEventSubmission = "recommendation.submission"
	FeedbackEventIgnore     = "recommendation.ignore"
	FeedbackEventDismiss    = "recommendation.dismiss"
)

const (
	EventRecommendationGenerated        = "recommendation.generated"
	EventRecommendationFeedbackRecorded = "recommendation.feedback_recorded"
	EventRecommendationOverrideApplied  = "recommendation.override_applied"
)

const (
	ABVariantControl        = "control"
	ABVariantMLDriven       = "ml_driven"
	ABVariantHybrid         = "hybrid"
	ABVariantDiversityFirst = "diversity_first"
)

type Factor struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type CampaignSnapshot struct {
	CampaignID   string  `json:"campaign_id"`
	Title        string  `json:"title"`
	CreatorID    string  `json:"creator_id"`
	RewardRate   float64 `json:"reward_rate"`
	Platform     string  `json:"platform"`
	Category     string  `json:"category,omitempty"`
	ApprovalRate float64 `json:"approval_rate,omitempty"`
}

type Recommendation struct {
	RecommendationID    string            `json:"recommendation_id"`
	BatchID             string            `json:"batch_id,omitempty"`
	UserID              string            `json:"user_id,omitempty"`
	Role                string            `json:"role,omitempty"`
	EntityID            string            `json:"entity_id"`
	EntityType          string            `json:"entity_type"`
	Score               float64           `json:"score"`
	Position            int               `json:"position"`
	Reason              string            `json:"reason"`
	ConfidenceScore     float64           `json:"confidence_score"`
	ConfidenceLevel     string            `json:"confidence_level"`
	ContributingFactors []Factor          `json:"contributing_factors"`
	Campaign            *CampaignSnapshot `json:"campaign,omitempty"`
	ModelVersion        string            `json:"model_version,omitempty"`
	ComputedAt          time.Time         `json:"computed_at,omitempty"`
}

type RecommendationMeta struct {
	ComputedAt         time.Time `json:"computed_at"`
	CacheHit           bool      `json:"cache_hit"`
	ModelVersion       string    `json:"model_version"`
	RecommendationMode string    `json:"recommendation_mode,omitempty"`
}

type RecommendationResponse struct {
	Recommendations []Recommendation   `json:"recommendations"`
	Meta            RecommendationMeta `json:"meta"`
}

type FeedbackRecord struct {
	FeedbackID       string    `json:"feedback_id"`
	RecommendationID string    `json:"recommendation_id"`
	UserID           string    `json:"user_id"`
	EventType        string    `json:"event_type"`
	EntityID         string    `json:"entity_id"`
	OccurredAt       time.Time `json:"occurred_at"`
	CreatedAt        time.Time `json:"created_at"`
	SourceService    string    `json:"source_service"`
	TraceID          string    `json:"trace_id"`
	SchemaVersion    string    `json:"schema_version"`
	IdempotencyKey   string    `json:"idempotency_key"`
}

type RecommendationOverride struct {
	OverrideID   string     `json:"override_id"`
	OverrideType string     `json:"override_type"`
	EntityID     string     `json:"entity_id"`
	Scope        string     `json:"scope"`
	ScopeValue   string     `json:"scope_value"`
	Multiplier   float64    `json:"multiplier"`
	Reason       string     `json:"reason"`
	StartAt      time.Time  `json:"start_at"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by"`
	Active       bool       `json:"active"`
}

type RecommendationBatch struct {
	BatchID         string           `json:"batch_id"`
	UserID          string           `json:"user_id"`
	Role            string           `json:"role"`
	Recommendations []Recommendation `json:"recommendations"`
	ModelVersion    string           `json:"model_version"`
	ComputedAt      time.Time        `json:"computed_at"`
	CacheHit        bool             `json:"cache_hit"`
}

type ABTestAssignment struct {
	AssignmentID string    `json:"assignment_id"`
	TestID       string    `json:"test_id"`
	UserID       string    `json:"user_id"`
	Variant      string    `json:"variant"`
	AssignedAt   time.Time `json:"assigned_at"`
}

type RecommendationModel struct {
	ModelID   string    `json:"model_id"`
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NormalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "clipper":
		return RoleClipper
	case "creator":
		return RoleCreator
	case "buyer":
		return RoleBuyer
	case "admin":
		return RoleAdmin
	default:
		return ""
	}
}

func NormalizeFeedbackEvent(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case FeedbackEventClick:
		return FeedbackEventClick
	case FeedbackEventSubmission:
		return FeedbackEventSubmission
	case FeedbackEventIgnore:
		return FeedbackEventIgnore
	case FeedbackEventDismiss:
		return FeedbackEventDismiss
	default:
		return ""
	}
}

func NormalizeOverrideType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case OverrideTypePromoteCampaign:
		return OverrideTypePromoteCampaign
	case OverrideTypeSuppressCampaign:
		return OverrideTypeSuppressCampaign
	case OverrideTypePromoteCreator:
		return OverrideTypePromoteCreator
	case OverrideTypeSuppressCreator:
		return OverrideTypeSuppressCreator
	default:
		return ""
	}
}

func CanonicalEventClass(eventType string) string {
	switch eventType {
	case EventRecommendationGenerated, EventRecommendationFeedbackRecorded, EventRecommendationOverrideApplied:
		return CanonicalEventClassDomain
	default:
		return ""
	}
}

func CanonicalPartitionKeyPath(eventType string) string {
	switch eventType {
	case EventRecommendationGenerated, EventRecommendationFeedbackRecorded:
		return "data.user_id"
	case EventRecommendationOverrideApplied:
		return "data.override_id"
	default:
		return ""
	}
}

func IsModuleInternalEvent(eventType string) bool {
	return CanonicalEventClass(eventType) != ""
}

func ConfidenceLevel(score float64) string {
	switch {
	case score >= 0.85:
		return "High"
	case score >= 0.65:
		return "Medium"
	default:
		return "Low"
	}
}

func Clamp(min, v, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
