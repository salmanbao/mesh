package bootstrap

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceID               string
	HTTPPort                int
	GRPCPort                int
	ClippingToolOwnerAPIURL string
	IdempotencyStorePath    string
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		ClippingToolOwnerAPIURL string `yaml:"clipping_tool_owner_api_url"`
	} `yaml:"dependencies"`
	Persistence struct {
		IdempotencyStorePath string `yaml:"idempotency_store_path"`
	} `yaml:"persistence"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		GRPCPort:                9090,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
		IdempotencyStorePath:    "data/m25-admin-model-deploy-idempotency.json",
	}
	if raw, err := os.ReadFile(path); err == nil {
		var fileCfg configFile
		if err := yaml.Unmarshal(raw, &fileCfg); err != nil {
			return Config{}, fmt.Errorf("parse config file: %w", err)
		}
		if fileCfg.Service.ID != "" {
			cfg.ServiceID = fileCfg.Service.ID
		}
		if fileCfg.Service.HTTPPort > 0 {
			cfg.HTTPPort = fileCfg.Service.HTTPPort
		}
		if fileCfg.Service.GRPCPort > 0 {
			cfg.GRPCPort = fileCfg.Service.GRPCPort
		}
		if value := strings.TrimSpace(fileCfg.Dependencies.ClippingToolOwnerAPIURL); value != "" && !strings.Contains(value, "${") {
			cfg.ClippingToolOwnerAPIURL = value
		}
		if value := strings.TrimSpace(fileCfg.Persistence.IdempotencyStorePath); value != "" && !strings.Contains(value, "${") {
			cfg.IdempotencyStorePath = value
		}
	}

	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.ClippingToolOwnerAPIURL = envOrDefault("M24_CLIPPING_TOOL_OWNER_API_URL", cfg.ClippingToolOwnerAPIURL)
	cfg.IdempotencyStorePath = envOrDefault("M25_IDEMPOTENCY_STORE_PATH", cfg.IdempotencyStorePath)
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.ServiceID == "" {
		return fmt.Errorf("service.id is required")
	}
	if cfg.HTTPPort <= 0 {
		return fmt.Errorf("service.http_port must be positive")
	}
	ownerURL := strings.TrimSpace(cfg.ClippingToolOwnerAPIURL)
	if ownerURL == "" {
		return fmt.Errorf("dependencies.clipping_tool_owner_api_url is required")
	}
	storePath := strings.TrimSpace(cfg.IdempotencyStorePath)
	if storePath == "" {
		return fmt.Errorf("persistence.idempotency_store_path is required")
	}
	if runtimeModeAllowsFallback() {
		return nil
	}
	rawOwnerURL := strings.TrimSpace(os.Getenv("M24_CLIPPING_TOOL_OWNER_API_URL"))
	if rawOwnerURL == "" {
		return fmt.Errorf("M24_CLIPPING_TOOL_OWNER_API_URL is required in production runtime")
	}
	if _, err := parseAbsoluteURL(rawOwnerURL); err != nil {
		return fmt.Errorf("M24_CLIPPING_TOOL_OWNER_API_URL must be an absolute URL")
	}
	if _, err := parseAbsoluteURL(ownerURL); err != nil {
		return fmt.Errorf("dependencies.clipping_tool_owner_api_url must be an absolute URL")
	}
	rawStorePath := strings.TrimSpace(os.Getenv("M25_IDEMPOTENCY_STORE_PATH"))
	if rawStorePath == "" {
		return fmt.Errorf("M25_IDEMPOTENCY_STORE_PATH is required in production runtime")
	}
	if !filepath.IsAbs(rawStorePath) {
		return fmt.Errorf("M25_IDEMPOTENCY_STORE_PATH must be an absolute path")
	}
	if !filepath.IsAbs(storePath) {
		return fmt.Errorf("persistence.idempotency_store_path must be an absolute path")
	}
	return nil
}

func runtimeModeAllowsFallback() bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("M25_RUNTIME_MODE")))
	switch mode {
	case "", "dev", "development", "local", "test", "testing":
		return true
	default:
		return false
	}
}

func parseAbsoluteURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid absolute url")
	}
	return parsed, nil
}

func envOrDefault(name string, fallback string) string {
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
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}
