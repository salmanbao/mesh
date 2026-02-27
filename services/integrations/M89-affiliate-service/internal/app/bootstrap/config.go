package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceID            string
	HTTPPort             int
	GRPCPort             int
	PublicBaseURL        string
	CommissionRate       float64
	PayoutThreshold      float64
	ReferralCookieTTL    time.Duration
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
	OutboxFlushBatchSize int
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Affiliate struct {
		PublicBaseURL   string  `yaml:"public_base_url"`
		CommissionRate  float64 `yaml:"commission_rate"`
		PayoutThreshold float64 `yaml:"payout_threshold"`
	} `yaml:"affiliate"`
	Runtime struct {
		ReferralCookieTTLHours int `yaml:"referral_cookie_ttl_hours"`
		IdempotencyTTLHours    int `yaml:"idempotency_ttl_hours"`
		EventDedupTTLHours     int `yaml:"event_dedup_ttl_hours"`
		ConsumerPollSeconds    int `yaml:"consumer_poll_seconds"`
		OutboxFlushBatchSize   int `yaml:"outbox_flush_batch_size"`
	} `yaml:"runtime"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M89-Affiliate-Service",
		HTTPPort:             8080,
		GRPCPort:             9090,
		PublicBaseURL:        "https://platform.com",
		CommissionRate:       0.10,
		PayoutThreshold:      100.0,
		ReferralCookieTTL:    30 * 24 * time.Hour,
		IdempotencyTTL:       7 * 24 * time.Hour,
		EventDedupTTL:        7 * 24 * time.Hour,
		ConsumerPollInterval: 2 * time.Second,
		OutboxFlushBatchSize: 100,
	}
	if raw, err := os.ReadFile(path); err == nil {
		var f configFile
		if err := yaml.Unmarshal(raw, &f); err != nil {
			return Config{}, fmt.Errorf("parse config file: %w", err)
		}
		if f.Service.ID != "" {
			cfg.ServiceID = f.Service.ID
		}
		if f.Service.HTTPPort > 0 {
			cfg.HTTPPort = f.Service.HTTPPort
		}
		if f.Service.GRPCPort > 0 {
			cfg.GRPCPort = f.Service.GRPCPort
		}
		if f.Affiliate.PublicBaseURL != "" {
			cfg.PublicBaseURL = f.Affiliate.PublicBaseURL
		}
		if f.Affiliate.CommissionRate > 0 {
			cfg.CommissionRate = f.Affiliate.CommissionRate
		}
		if f.Affiliate.PayoutThreshold > 0 {
			cfg.PayoutThreshold = f.Affiliate.PayoutThreshold
		}
		if f.Runtime.ReferralCookieTTLHours > 0 {
			cfg.ReferralCookieTTL = time.Duration(f.Runtime.ReferralCookieTTLHours) * time.Hour
		}
		if f.Runtime.IdempotencyTTLHours > 0 {
			cfg.IdempotencyTTL = time.Duration(f.Runtime.IdempotencyTTLHours) * time.Hour
		}
		if f.Runtime.EventDedupTTLHours > 0 {
			cfg.EventDedupTTL = time.Duration(f.Runtime.EventDedupTTLHours) * time.Hour
		}
		if f.Runtime.ConsumerPollSeconds > 0 {
			cfg.ConsumerPollInterval = time.Duration(f.Runtime.ConsumerPollSeconds) * time.Second
		}
		if f.Runtime.OutboxFlushBatchSize > 0 {
			cfg.OutboxFlushBatchSize = f.Runtime.OutboxFlushBatchSize
		}
	}
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.PublicBaseURL = envString("PUBLIC_BASE_URL", cfg.PublicBaseURL)
	cfg.CommissionRate = envFloat("COMMISSION_RATE", cfg.CommissionRate)
	cfg.PayoutThreshold = envFloat("PAYOUT_THRESHOLD", cfg.PayoutThreshold)
	cfg.ReferralCookieTTL = time.Duration(envInt("REFERRAL_COOKIE_TTL_HOURS", int(cfg.ReferralCookieTTL.Hours()))) * time.Hour
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second
	cfg.OutboxFlushBatchSize = envInt("OUTBOX_FLUSH_BATCH_SIZE", cfg.OutboxFlushBatchSize)
	return cfg, nil
}

func envInt(name string, fallback int) int {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			return v
		}
	}
	return fallback
}

func envFloat(name string, fallback float64) float64 {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.ParseFloat(raw, 64); err == nil {
			return v
		}
	}
	return fallback
}

func envString(name, fallback string) string {
	if raw := os.Getenv(name); raw != "" {
		return raw
	}
	return fallback
}
