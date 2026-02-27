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
	AuthGRPCURL          string
	ProfileGRPCURL       string
	FraudGRPCURL         string
	ModerationGRPCURL    string
	ResolutionGRPCURL    string
	ReputationGRPCURL    string
	WebhookBearerToken   string
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
	Dependencies struct {
		AuthGRPCURL       string `yaml:"auth_grpc_url"`
		ProfileGRPCURL    string `yaml:"profile_grpc_url"`
		FraudGRPCURL      string `yaml:"fraud_grpc_url"`
		ModerationGRPCURL string `yaml:"moderation_grpc_url"`
		ResolutionGRPCURL string `yaml:"resolution_grpc_url"`
		ReputationGRPCURL string `yaml:"reputation_grpc_url"`
	} `yaml:"dependencies"`
	Security struct {
		WebhookBearerToken string `yaml:"webhook_bearer_token"`
	} `yaml:"security"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M36-Risk-Service",
		HTTPPort:             8080,
		GRPCPort:             9090,
		WebhookBearerToken:   "risk-webhook-secret",
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
		cfg.AuthGRPCURL = f.Dependencies.AuthGRPCURL
		cfg.ProfileGRPCURL = f.Dependencies.ProfileGRPCURL
		cfg.FraudGRPCURL = f.Dependencies.FraudGRPCURL
		cfg.ModerationGRPCURL = f.Dependencies.ModerationGRPCURL
		cfg.ResolutionGRPCURL = f.Dependencies.ResolutionGRPCURL
		cfg.ReputationGRPCURL = f.Dependencies.ReputationGRPCURL
		if f.Security.WebhookBearerToken != "" {
			cfg.WebhookBearerToken = f.Security.WebhookBearerToken
		}
	}
	cfg.AuthGRPCURL = envOrDefault("AUTH_GRPC_URL", cfg.AuthGRPCURL)
	cfg.ProfileGRPCURL = envOrDefault("PROFILE_GRPC_URL", cfg.ProfileGRPCURL)
	cfg.FraudGRPCURL = envOrDefault("FRAUD_GRPC_URL", cfg.FraudGRPCURL)
	cfg.ModerationGRPCURL = envOrDefault("MODERATION_GRPC_URL", cfg.ModerationGRPCURL)
	cfg.ResolutionGRPCURL = envOrDefault("RESOLUTION_GRPC_URL", cfg.ResolutionGRPCURL)
	cfg.ReputationGRPCURL = envOrDefault("REPUTATION_GRPC_URL", cfg.ReputationGRPCURL)
	cfg.WebhookBearerToken = envOrDefault("RISK_WEBHOOK_BEARER_TOKEN", cfg.WebhookBearerToken)
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
