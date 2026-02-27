package domain

import (
	"math"
	"net"
	"strings"
	"time"
)

const (
	CanonicalEventClassDomain        = "domain"
	CanonicalEventClassAnalyticsOnly = "analytics_only"
	CanonicalEventClassOps           = "ops"
)

type ReferralEvent struct {
	EventID               string            `json:"event_id"`
	SourceEventType       string            `json:"source_event_type"`
	AffiliateID           string            `json:"affiliate_id,omitempty"`
	ReferralToken         string            `json:"referral_token,omitempty"`
	ReferrerID            string            `json:"referrer_id,omitempty"`
	UserID                string            `json:"user_id,omitempty"`
	ConversionID          string            `json:"conversion_id,omitempty"`
	OrderID               string            `json:"order_id,omitempty"`
	TransactionID         string            `json:"transaction_id,omitempty"`
	Amount                float64           `json:"amount,omitempty"`
	ClickIP               string            `json:"click_ip,omitempty"`
	UserAgent             string            `json:"user_agent,omitempty"`
	DeviceFingerprintID   string            `json:"device_fingerprint_id,omitempty"`
	DeviceFingerprintHash string            `json:"device_fingerprint_hash,omitempty"`
	FormFillTimeMS        int               `json:"form_fill_time_ms,omitempty"`
	MouseMovementCount    int               `json:"mouse_movement_count,omitempty"`
	KeyboardCPS           float64           `json:"keyboard_cps,omitempty"`
	Country               string            `json:"country,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
	OccurredAt            time.Time         `json:"occurred_at"`
	CreatedAt             time.Time         `json:"created_at"`
	RawPayload            []byte            `json:"-"`
}

type FraudDecision struct {
	DecisionID      string    `json:"decision_id"`
	EventID         string    `json:"event_id"`
	RiskScore       float64   `json:"risk_score"`
	Decision        string    `json:"decision"`
	RiskTier        string    `json:"risk_tier"`
	Flags           []string  `json:"flags"`
	ModelVersion    string    `json:"model_version"`
	PolicyVersion   string    `json:"policy_version"`
	ClusterID       string    `json:"cluster_id,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	SourceEventType string    `json:"source_event_type"`
}

type RiskPolicy struct {
	PolicyID     string            `json:"policy_id"`
	Name         string            `json:"name"`
	Region       string            `json:"region,omitempty"`
	CampaignType string            `json:"campaign_type,omitempty"`
	AffiliateID  string            `json:"affiliate_id,omitempty"`
	Threshold    float64           `json:"threshold"`
	ActionMap    map[string]string `json:"action_map"`
	IsActive     bool              `json:"is_active"`
	Version      string            `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
}

type DeviceFingerprint struct {
	DeviceFingerprintID string    `json:"device_fingerprint_id"`
	FingerprintHash     string    `json:"fingerprint_hash"`
	LastSeenIP          string    `json:"last_seen_ip"`
	DistinctIPCount     int       `json:"distinct_ip_count"`
	SeenCount           int       `json:"seen_count"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Cluster struct {
	ClusterID string    `json:"cluster_id"`
	Key       string    `json:"key"`
	Reason    string    `json:"reason"`
	Size      int       `json:"size"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DisputeCase struct {
	DisputeID   string     `json:"dispute_id"`
	DecisionID  string     `json:"decision_id"`
	EventID     string     `json:"event_id"`
	SubmittedBy string     `json:"submitted_by"`
	EvidenceURL string     `json:"evidence_url"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
}

type AuditLog struct {
	AuditID    string    `json:"audit_id"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Action     string    `json:"action"`
	Summary    string    `json:"summary"`
	TraceID    string    `json:"trace_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type MetricsSnapshot struct {
	FraudRate            float64        `json:"fraud_rate"`
	AttackVectors        []AttackVector `json:"attack_vectors"`
	RevenueProtectedUSD  float64        `json:"revenue_protected_usd"`
	ActiveInvestigations int            `json:"active_investigations"`
	AppealQueue          int            `json:"appeal_queue"`
	GeneratedAt          time.Time      `json:"generated_at"`
}

type AttackVector struct {
	Type    string  `json:"type"`
	Percent float64 `json:"percent"`
}

type ScoreRequest struct {
	EventID               string
	EventType             string
	AffiliateID           string
	ReferralToken         string
	ReferrerID            string
	UserID                string
	ConversionID          string
	OrderID               string
	Amount                float64
	ClickIP               string
	UserAgent             string
	DeviceFingerprintHash string
	DeviceFingerprintID   string
	FormFillTimeMS        int
	MouseMovementCount    int
	KeyboardCPS           float64
	Region                string
	CampaignType          string
	OccurredAt            time.Time
	TraceID               string
	RawPayload            []byte
}

func NormalizeDecision(score float64, threshold float64, flags []string) (decision string, tier string) {
	score = ClampScore(score)
	tier = RiskTier(score)
	decision = "allow"
	if score >= threshold || containsFlag(flags, "self_referral") {
		decision = "block"
	} else if score >= math.Max(0.5, threshold-0.2) || len(flags) > 0 {
		decision = "flag"
	}
	return decision, tier
}

func RiskTier(score float64) string {
	switch {
	case score >= 0.9:
		return "critical"
	case score >= 0.7:
		return "high"
	case score >= 0.4:
		return "medium"
	default:
		return "low"
	}
}

func ClampScore(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func ScoreReferral(req ScoreRequest, fp *DeviceFingerprint, clusterSize int) (float64, []string) {
	score := 0.05
	flags := make([]string, 0, 6)
	if req.FormFillTimeMS > 0 && req.FormFillTimeMS < 100 {
		score += 0.20
		flags = append(flags, "telemetry_fast_form_fill")
	}
	if req.MouseMovementCount == 0 {
		score += 0.15
		flags = append(flags, "bot_traffic")
	}
	if strings.Contains(strings.ToLower(req.UserAgent), "headless") {
		score += 0.20
		flags = append(flags, "bot_traffic")
	}
	if IsPrivateOrLoopbackIP(req.ClickIP) || strings.HasPrefix(req.ClickIP, "10.") {
		score += 0.05
	}
	if req.ReferrerID != "" && req.UserID != "" && req.ReferrerID == req.UserID {
		score += 0.50
		flags = append(flags, "self_referral")
	}
	if fp != nil && fp.DistinctIPCount >= 2 {
		score += 0.20
		flags = append(flags, "device_fingerprint_anomaly")
	}
	if clusterSize >= 3 {
		score += 0.20
		flags = append(flags, "cluster_suspected")
	}
	if req.Amount > 1000 {
		score += 0.08
	}
	return ClampScore(score), uniqStrings(flags)
}

func IsPrivateOrLoopbackIP(ipRaw string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipRaw))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func containsFlag(flags []string, target string) bool {
	for _, f := range flags {
		if f == target {
			return true
		}
	}
	return false
}

func uniqStrings(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
