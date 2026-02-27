package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceID string

	HTTPPort int
	GRPCPort int

	ProfileGRPCURL      string
	BillingGRPCURL      string
	ContentGRPCURL      string
	EscrowGRPCURL       string
	OnboardingGRPCURL   string
	FinanceGRPCURL      string
	RewardGRPCURL       string
	GamificationGRPCURL string
	AnalyticsGRPCURL    string
	ProductGRPCURL      string

	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	DashboardCacheTTL    time.Duration
	ConsumerPollInterval time.Duration
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		ProfileGRPCURL      string `yaml:"profile_grpc_url"`
		BillingGRPCURL      string `yaml:"billing_grpc_url"`
		ContentGRPCURL      string `yaml:"content_grpc_url"`
		EscrowGRPCURL       string `yaml:"escrow_grpc_url"`
		OnboardingGRPCURL   string `yaml:"onboarding_grpc_url"`
		FinanceGRPCURL      string `yaml:"finance_grpc_url"`
		RewardGRPCURL       string `yaml:"reward_grpc_url"`
		GamificationGRPCURL string `yaml:"gamification_grpc_url"`
		AnalyticsGRPCURL    string `yaml:"analytics_grpc_url"`
		ProductGRPCURL      string `yaml:"product_grpc_url"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M55-Dashboard-Service",
		HTTPPort:             8080,
		GRPCPort:             9090,
		IdempotencyTTL:       7 * 24 * time.Hour,
		EventDedupTTL:        7 * 24 * time.Hour,
		DashboardCacheTTL:    5 * time.Minute,
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
		cfg.ProfileGRPCURL = f.Dependencies.ProfileGRPCURL
		cfg.BillingGRPCURL = f.Dependencies.BillingGRPCURL
		cfg.ContentGRPCURL = f.Dependencies.ContentGRPCURL
		cfg.EscrowGRPCURL = f.Dependencies.EscrowGRPCURL
		cfg.OnboardingGRPCURL = f.Dependencies.OnboardingGRPCURL
		cfg.FinanceGRPCURL = f.Dependencies.FinanceGRPCURL
		cfg.RewardGRPCURL = f.Dependencies.RewardGRPCURL
		cfg.GamificationGRPCURL = f.Dependencies.GamificationGRPCURL
		cfg.AnalyticsGRPCURL = f.Dependencies.AnalyticsGRPCURL
		cfg.ProductGRPCURL = f.Dependencies.ProductGRPCURL
	}

	cfg.ProfileGRPCURL = envOrDefault("PROFILE_GRPC_URL", cfg.ProfileGRPCURL)
	cfg.BillingGRPCURL = envOrDefault("BILLING_GRPC_URL", cfg.BillingGRPCURL)
	cfg.ContentGRPCURL = envOrDefault("CONTENT_GRPC_URL", cfg.ContentGRPCURL)
	cfg.EscrowGRPCURL = envOrDefault("ESCROW_GRPC_URL", cfg.EscrowGRPCURL)
	cfg.OnboardingGRPCURL = envOrDefault("ONBOARDING_GRPC_URL", cfg.OnboardingGRPCURL)
	cfg.FinanceGRPCURL = envOrDefault("FINANCE_GRPC_URL", cfg.FinanceGRPCURL)
	cfg.RewardGRPCURL = envOrDefault("REWARD_GRPC_URL", cfg.RewardGRPCURL)
	cfg.GamificationGRPCURL = envOrDefault("GAMIFICATION_GRPC_URL", cfg.GamificationGRPCURL)
	cfg.AnalyticsGRPCURL = envOrDefault("ANALYTICS_GRPC_URL", cfg.AnalyticsGRPCURL)
	cfg.ProductGRPCURL = envOrDefault("PRODUCT_GRPC_URL", cfg.ProductGRPCURL)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.DashboardCacheTTL = time.Duration(envInt("DASHBOARD_CACHE_TTL_SECONDS", int(cfg.DashboardCacheTTL.Seconds()))) * time.Second
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

func envCSV(name string, fallback []string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	items := strings.Split(raw, ",")
	return trimNonEmpty(items)
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
