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
	Version              string
	HTTPPort             int
	GRPCPort             int
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
		Version  string `yaml:"version"`
	} `yaml:"service"`
	Runtime struct {
		IdempotencyTTLHours int `yaml:"idempotency_ttl_hours"`
		EventDedupTTLHours  int `yaml:"event_dedup_ttl_hours"`
		ConsumerPollSeconds int `yaml:"consumer_poll_seconds"`
	} `yaml:"runtime"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M79-Monitoring-Service",
		Version:              "0.1.0",
		HTTPPort:             8080,
		GRPCPort:             9090,
		IdempotencyTTL:       7 * 24 * time.Hour,
		EventDedupTTL:        7 * 24 * time.Hour,
		ConsumerPollInterval: 2 * time.Second,
	}
	if raw, err := os.ReadFile(path); err == nil {
		var f configFile
		if err := yaml.Unmarshal(raw, &f); err != nil {
			return Config{}, fmt.Errorf("parse config file: %w", err)
		}
		if f.Service.ID != "" {
			cfg.ServiceID = f.Service.ID
		}
		if f.Service.Version != "" {
			cfg.Version = f.Service.Version
		}
		if f.Service.HTTPPort > 0 {
			cfg.HTTPPort = f.Service.HTTPPort
		}
		if f.Service.GRPCPort > 0 {
			cfg.GRPCPort = f.Service.GRPCPort
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
	}
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.Version = envString("SERVICE_VERSION", cfg.Version)
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
