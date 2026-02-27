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
	AffiliateGRPCURL     string
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
	OutboxFlushBatchSize int
	DefaultThreshold     float64
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		AffiliateGRPCURL string `yaml:"affiliate_grpc_url"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{ServiceID: "M96-Referral-Fraud-Detection-Service", HTTPPort: 8080, GRPCPort: 9090, IdempotencyTTL: 7 * 24 * time.Hour, EventDedupTTL: 7 * 24 * time.Hour, ConsumerPollInterval: 2 * time.Second, OutboxFlushBatchSize: 100, DefaultThreshold: 0.8}
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
		cfg.AffiliateGRPCURL = f.Dependencies.AffiliateGRPCURL
	}
	cfg.AffiliateGRPCURL = envOrDefault("AFFILIATE_GRPC_URL", cfg.AffiliateGRPCURL)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second
	cfg.OutboxFlushBatchSize = envInt("OUTBOX_FLUSH_BATCH_SIZE", cfg.OutboxFlushBatchSize)
	return cfg, nil
}
func envOrDefault(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
