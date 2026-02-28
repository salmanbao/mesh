package bootstrap

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServiceID      string
	Version        string
	HTTPPort       int
	GRPCPort       int
	IdempotencyTTL time.Duration
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:      "M72-Webhook-Manager",
		Version:        "0.1.0",
		HTTPPort:       8080,
		GRPCPort:       9090,
		IdempotencyTTL: 7 * 24 * time.Hour,
	}
	if path != "" {
		if err := parseConfigFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.Version = envString("SERVICE_VERSION", cfg.Version)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	return cfg, nil
}

func parseConfigFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open config: %w", err)
	}
	defer file.Close()

	section := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") && !strings.Contains(strings.TrimSuffix(line, ":"), " ") {
			section = strings.TrimSuffix(line, ":")
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		switch section + "." + key {
		case "service.id":
			if value != "" {
				cfg.ServiceID = value
			}
		case "service.http_port":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.HTTPPort = v
			}
		case "service.grpc_port":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.GRPCPort = v
			}
		case "service.version":
			if value != "" {
				cfg.Version = value
			}
		case "runtime.idempotency_ttl_hours":
			if v, err := strconv.Atoi(value); err == nil && v > 0 {
				cfg.IdempotencyTTL = time.Duration(v) * time.Hour
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	return nil
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
