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
	EmbedBaseURL         string
	CacheTTL             time.Duration
	PerIPLimitPerHour    int
	PerEmbedLimitPerHour int
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
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{ServiceID: "M66-Embed-Service", HTTPPort: 8080, GRPCPort: 9090, EmbedBaseURL: "https://embed.platform.com", CacheTTL: 5 * time.Minute, PerIPLimitPerHour: 1000, PerEmbedLimitPerHour: 100, IdempotencyTTL: 7 * 24 * time.Hour, EventDedupTTL: 7 * 24 * time.Hour, ConsumerPollInterval: 2 * time.Second}
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
	}
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.EmbedBaseURL = envString("EMBED_BASE_URL", cfg.EmbedBaseURL)
	cfg.CacheTTL = time.Duration(envInt("CACHE_TTL_SECONDS", int(cfg.CacheTTL.Seconds()))) * time.Second
	cfg.PerIPLimitPerHour = envInt("RATE_LIMIT_PER_IP", cfg.PerIPLimitPerHour)
	cfg.PerEmbedLimitPerHour = envInt("RATE_LIMIT_PER_EMBED", cfg.PerEmbedLimitPerHour)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second
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
func envString(name, fallback string) string {
	if raw := os.Getenv(name); raw != "" {
		return raw
	}
	return fallback
}
