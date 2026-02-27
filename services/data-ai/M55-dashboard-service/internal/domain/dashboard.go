package domain

import (
	"strings"
	"time"
)

type DashboardRole string

type WidgetStatus string

const (
	RoleCreator   DashboardRole = "creator"
	RoleClipper   DashboardRole = "clipper"
	RoleDeveloper DashboardRole = "developer"
	RoleAdmin     DashboardRole = "admin"
	RoleBrand     DashboardRole = "brand"
)

const (
	WidgetStatusOK          WidgetStatus = "ok"
	WidgetStatusStale       WidgetStatus = "stale"
	WidgetStatusUnavailable WidgetStatus = "unavailable"
)

type Widget struct {
	WidgetID string                 `json:"widget_id"`
	Status   WidgetStatus           `json:"status"`
	Source   string                 `json:"source"`
	Data     map[string]interface{} `json:"data"`
}

type Dashboard struct {
	UserID          string            `json:"user_id"`
	Role            DashboardRole     `json:"role"`
	DateRange       string            `json:"date_range"`
	Timezone        string            `json:"timezone"`
	GeneratedAt     time.Time         `json:"generated_at"`
	Widgets         map[string]Widget `json:"widgets"`
	DegradedWidgets []string          `json:"degraded_widgets,omitempty"`
}

type LayoutItem struct {
	WidgetID string `json:"widget_id"`
	Position int    `json:"position"`
	Size     string `json:"size"`
	Visible  bool   `json:"visible"`
}

type DashboardLayout struct {
	LayoutID      string       `json:"layout_id"`
	UserID        string       `json:"user_id"`
	DeviceType    string       `json:"device_type"`
	LayoutVersion int          `json:"layout_version"`
	Items         []LayoutItem `json:"items"`
	LastUpdatedAt time.Time    `json:"last_updated_at"`
}

type CustomView struct {
	ViewID           string        `json:"view_id"`
	UserID           string        `json:"user_id"`
	ViewName         string        `json:"view_name"`
	Role             DashboardRole `json:"role"`
	WidgetIDs        []string      `json:"widget_ids"`
	DateRangeDefault string        `json:"date_range_default"`
	IsDefault        bool          `json:"is_default"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

type UserPreference struct {
	PrefID           string    `json:"pref_id"`
	UserID           string    `json:"user_id"`
	Timezone         string    `json:"timezone"`
	DefaultDateRange string    `json:"default_date_range"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CacheInvalidation struct {
	InvalidationID  string    `json:"invalidation_id"`
	UserID          string    `json:"user_id"`
	TriggerEvent    string    `json:"trigger_event"`
	AffectedWidgets []string  `json:"affected_widgets"`
	InvalidatedAt   time.Time `json:"invalidated_at"`
}

func NormalizeRole(raw string) DashboardRole {
	role := DashboardRole(strings.ToLower(strings.TrimSpace(raw)))
	switch role {
	case RoleCreator, RoleClipper, RoleDeveloper, RoleAdmin, RoleBrand:
		return role
	default:
		return RoleCreator
	}
}

func ValidateLayout(layout DashboardLayout) error {
	if strings.TrimSpace(layout.UserID) == "" {
		return ErrInvalidInput
	}
	device := strings.ToLower(strings.TrimSpace(layout.DeviceType))
	if device != "web" && device != "mobile" {
		return ErrInvalidInput
	}
	if layout.LayoutVersion <= 0 {
		return ErrInvalidInput
	}
	if len(layout.Items) == 0 {
		return ErrInvalidInput
	}
	seen := make(map[string]struct{}, len(layout.Items))
	for _, item := range layout.Items {
		if strings.TrimSpace(item.WidgetID) == "" {
			return ErrInvalidInput
		}
		if _, ok := seen[item.WidgetID]; ok {
			return ErrConflict
		}
		seen[item.WidgetID] = struct{}{}
		if item.Position < 0 {
			return ErrInvalidInput
		}
	}
	return nil
}

func ValidateCustomView(view CustomView) error {
	if strings.TrimSpace(view.UserID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(view.ViewName) == "" {
		return ErrInvalidInput
	}
	if len(view.ViewName) > 120 {
		return ErrInvalidInput
	}
	if len(view.WidgetIDs) == 0 {
		return ErrInvalidInput
	}
	return nil
}
