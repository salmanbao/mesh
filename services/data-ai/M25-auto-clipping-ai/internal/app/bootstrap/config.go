package bootstrap

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServiceID               string
	HTTPPort                int
	GRPCPort                int
	ClippingToolOwnerAPIURL string
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
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:               "M25-Auto-Clipping-AI",
		HTTPPort:                8086,
		GRPCPort:                9090,
		ClippingToolOwnerAPIURL: "http://m24-clipping-tool-service:8080",
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
		if fileCfg.Dependencies.ClippingToolOwnerAPIURL != "" {
			cfg.ClippingToolOwnerAPIURL = fileCfg.Dependencies.ClippingToolOwnerAPIURL
		}
	}

	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.ClippingToolOwnerAPIURL = envOrDefault("M24_CLIPPING_TOOL_OWNER_API_URL", cfg.ClippingToolOwnerAPIURL)
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.ServiceID == "" {
		return fmt.Errorf("service.id is required")
	}
	if cfg.HTTPPort <= 0 {
		return fmt.Errorf("service.http_port must be positive")
	}
	if cfg.ClippingToolOwnerAPIURL == "" {
		return fmt.Errorf("dependencies.clipping_tool_owner_api_url is required")
	}
	return nil
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
