package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceID string

	HTTPPort int
	GRPCPort int

	VotingGRPCURL     string
	SocialGRPCURL     string
	TrackingGRPCURL   string
	SubmissionGRPCURL string
	FinanceGRPCURL    string

	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		VotingGRPCURL     string `yaml:"voting_grpc_url"`
		SocialGRPCURL     string `yaml:"social_grpc_url"`
		TrackingGRPCURL   string `yaml:"tracking_grpc_url"`
		SubmissionGRPCURL string `yaml:"submission_grpc_url"`
		FinanceGRPCURL    string `yaml:"finance_grpc_url"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M54-Analytics-Service",
		HTTPPort:             8080,
		GRPCPort:             9090,
		IdempotencyTTL:       7 * 24 * time.Hour,
		EventDedupTTL:        7 * 24 * time.Hour,
		ConsumerPollInterval: 2 * time.Second,
	}

	raw, err := os.ReadFile(path)
	if err == nil {
		var f configFile
		if unmarshalErr := yaml.Unmarshal(raw, &f); unmarshalErr != nil {
			return Config{}, fmt.Errorf("parse config file: %w", unmarshalErr)
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
		cfg.VotingGRPCURL = f.Dependencies.VotingGRPCURL
		cfg.SocialGRPCURL = f.Dependencies.SocialGRPCURL
		cfg.TrackingGRPCURL = f.Dependencies.TrackingGRPCURL
		cfg.SubmissionGRPCURL = f.Dependencies.SubmissionGRPCURL
		cfg.FinanceGRPCURL = f.Dependencies.FinanceGRPCURL
	}

	cfg.VotingGRPCURL = envOrDefault("VOTING_GRPC_URL", cfg.VotingGRPCURL)
	cfg.SocialGRPCURL = envOrDefault("SOCIAL_GRPC_URL", cfg.SocialGRPCURL)
	cfg.TrackingGRPCURL = envOrDefault("TRACKING_GRPC_URL", cfg.TrackingGRPCURL)
	cfg.SubmissionGRPCURL = envOrDefault("SUBMISSION_GRPC_URL", cfg.SubmissionGRPCURL)
	cfg.FinanceGRPCURL = envOrDefault("FINANCE_GRPC_URL", cfg.FinanceGRPCURL)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
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
