package application

import "time"

type Config struct {
	ServiceName    string
	Version        string
	IdempotencyTTL time.Duration
}

type Actor struct {
	SubjectID      string
	Role           string
	RequestID      string
	IdempotencyKey string
}

type CreateConfigInput struct {
	Provider     string            `json:"provider"`
	Config       map[string]any    `json:"config"`
	HeaderRules  map[string]string `json:"header_rules,omitempty"`
	SignedURLTTL int               `json:"signed_url_ttl_seconds,omitempty"`
}

type PurgeInput struct {
	Scope  string `json:"scope"`
	Target string `json:"target"`
}
