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

	AuthGRPCURL         string
	CatalogGRPCURL      string
	FeeEngineGRPCURL    string
	FinanceGRPCURL      string
	SubscriptionGRPCURL string

	KafkaBrokers       []string
	KafkaConsumerGroup string
	TopicPayoutPaid    string
	TopicPayoutFailed  string
	DLQTopic           string

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
		AuthGRPCURL         string   `yaml:"auth_grpc_url"`
		CatalogGRPCURL      string   `yaml:"catalog_grpc_url"`
		FeeEngineGRPCURL    string   `yaml:"fee_engine_grpc_url"`
		FinanceGRPCURL      string   `yaml:"finance_grpc_url"`
		SubscriptionGRPCURL string   `yaml:"subscription_grpc_url"`
		KafkaBrokers        []string `yaml:"kafka_brokers"`
		KafkaConsumerGroup  string   `yaml:"kafka_consumer_group"`
		TopicPayoutPaid     string   `yaml:"topic_payout_paid"`
		TopicPayoutFailed   string   `yaml:"topic_payout_failed"`
		TopicDLQ            string   `yaml:"topic_dlq"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:            "M05-Billing-Service",
		HTTPPort:             8080,
		GRPCPort:             9090,
		KafkaConsumerGroup:   "m05-billing-service",
		TopicPayoutPaid:      "payout.paid",
		TopicPayoutFailed:    "payout.failed",
		DLQTopic:             "billing-service.dlq",
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
		cfg.AuthGRPCURL = f.Dependencies.AuthGRPCURL
		cfg.CatalogGRPCURL = f.Dependencies.CatalogGRPCURL
		cfg.FeeEngineGRPCURL = f.Dependencies.FeeEngineGRPCURL
		cfg.FinanceGRPCURL = f.Dependencies.FinanceGRPCURL
		cfg.SubscriptionGRPCURL = f.Dependencies.SubscriptionGRPCURL
		if len(f.Dependencies.KafkaBrokers) > 0 {
			cfg.KafkaBrokers = trimNonEmpty(f.Dependencies.KafkaBrokers)
		}
		if f.Dependencies.KafkaConsumerGroup != "" {
			cfg.KafkaConsumerGroup = f.Dependencies.KafkaConsumerGroup
		}
		if f.Dependencies.TopicPayoutPaid != "" {
			cfg.TopicPayoutPaid = f.Dependencies.TopicPayoutPaid
		}
		if f.Dependencies.TopicPayoutFailed != "" {
			cfg.TopicPayoutFailed = f.Dependencies.TopicPayoutFailed
		}
		if f.Dependencies.TopicDLQ != "" {
			cfg.DLQTopic = f.Dependencies.TopicDLQ
		}
	}

	cfg.AuthGRPCURL = envOrDefault("AUTH_GRPC_URL", cfg.AuthGRPCURL)
	cfg.CatalogGRPCURL = envOrDefault("CATALOG_GRPC_URL", cfg.CatalogGRPCURL)
	cfg.FeeEngineGRPCURL = envOrDefault("FEE_ENGINE_GRPC_URL", cfg.FeeEngineGRPCURL)
	cfg.FinanceGRPCURL = envOrDefault("FINANCE_GRPC_URL", cfg.FinanceGRPCURL)
	cfg.SubscriptionGRPCURL = envOrDefault("SUBSCRIPTION_GRPC_URL", cfg.SubscriptionGRPCURL)
	cfg.KafkaBrokers = envCSV("KAFKA_BROKERS", cfg.KafkaBrokers)
	cfg.KafkaConsumerGroup = envOrDefault("KAFKA_CONSUMER_GROUP", cfg.KafkaConsumerGroup)
	cfg.TopicPayoutPaid = envOrDefault("KAFKA_TOPIC_PAYOUT_PAID", cfg.TopicPayoutPaid)
	cfg.TopicPayoutFailed = envOrDefault("KAFKA_TOPIC_PAYOUT_FAILED", cfg.TopicPayoutFailed)
	cfg.DLQTopic = envOrDefault("KAFKA_TOPIC_BILLING_DLQ", cfg.DLQTopic)
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
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
