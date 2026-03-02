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
	HTTPPort  int
	GRPCPort  int

	CampaignGRPCURL string

	IdempotencyTTLHours int
	EventDedupTTLHours  int
	WorkerPollSeconds   int
	FullPipelineEnabled bool
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		CampaignGRPCURL string `yaml:"campaign_grpc_url"`
	} `yaml:"dependencies"`
	Pipeline struct {
		IdempotencyTTLHours int   `yaml:"idempotency_ttl_hours"`
		EventDedupTTLHours  int   `yaml:"event_dedup_ttl_hours"`
		WorkerPollSeconds   int   `yaml:"worker_poll_seconds"`
		FullPipelineEnabled *bool `yaml:"full_pipeline_enabled"`
	} `yaml:"pipeline"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:           "M06-Media-Processing-Pipeline",
		HTTPPort:            8080,
		GRPCPort:            9090,
		IdempotencyTTLHours: 168,
		EventDedupTTLHours:  168,
		WorkerPollSeconds:   2,
		FullPipelineEnabled: true,
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
		if f.Dependencies.CampaignGRPCURL != "" {
			cfg.CampaignGRPCURL = f.Dependencies.CampaignGRPCURL
		}
		if f.Pipeline.IdempotencyTTLHours > 0 {
			cfg.IdempotencyTTLHours = f.Pipeline.IdempotencyTTLHours
		}
		if f.Pipeline.EventDedupTTLHours > 0 {
			cfg.EventDedupTTLHours = f.Pipeline.EventDedupTTLHours
		}
		if f.Pipeline.WorkerPollSeconds > 0 {
			cfg.WorkerPollSeconds = f.Pipeline.WorkerPollSeconds
		}
		if f.Pipeline.FullPipelineEnabled != nil {
			cfg.FullPipelineEnabled = *f.Pipeline.FullPipelineEnabled
		}
	}
	cfg.CampaignGRPCURL = envOrDefault("CAMPAIGN_GRPC_URL", cfg.CampaignGRPCURL)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.IdempotencyTTLHours = envInt("IDEMPOTENCY_TTL_HOURS", cfg.IdempotencyTTLHours)
	cfg.EventDedupTTLHours = envInt("EVENT_DEDUP_TTL_HOURS", cfg.EventDedupTTLHours)
	cfg.WorkerPollSeconds = envInt("WORKER_POLL_SECONDS", cfg.WorkerPollSeconds)
	cfg.FullPipelineEnabled = envBool("FULL_PIPELINE_ENABLED", cfg.FullPipelineEnabled)
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
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "TRUE", "True", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "False", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

func (c Config) IdempotencyTTL() time.Duration {
	return time.Duration(c.IdempotencyTTLHours) * time.Hour
}

func (c Config) EventDedupTTL() time.Duration {
	return time.Duration(c.EventDedupTTLHours) * time.Hour
}

func (c Config) WorkerPollInterval() time.Duration {
	return time.Duration(c.WorkerPollSeconds) * time.Second
}
