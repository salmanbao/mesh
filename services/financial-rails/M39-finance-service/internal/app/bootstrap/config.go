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

	AuthGRPCURL           string
	CampaignGRPCURL       string
	ContentLibraryGRPCURL string
	EscrowGRPCURL         string
	FeeEngineGRPCURL      string
	ProductGRPCURL        string

	KafkaBrokers            []string
	KafkaConsumerGroup      string
	TopicTransactionSuccess string
	TopicTransactionFailed  string
	TopicTransactionRefund  string
	DLQTopic                string

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
		AuthGRPCURL           string   `yaml:"auth_grpc_url"`
		CampaignGRPCURL       string   `yaml:"campaign_grpc_url"`
		ContentLibraryGRPCURL string   `yaml:"content_library_grpc_url"`
		EscrowGRPCURL         string   `yaml:"escrow_grpc_url"`
		FeeEngineGRPCURL      string   `yaml:"fee_engine_grpc_url"`
		ProductGRPCURL        string   `yaml:"product_grpc_url"`
		KafkaBrokers          []string `yaml:"kafka_brokers"`
		KafkaConsumerGroup    string   `yaml:"kafka_consumer_group"`
		TopicTransactionOK    string   `yaml:"topic_transaction_succeeded"`
		TopicTransactionFail  string   `yaml:"topic_transaction_failed"`
		TopicTransactionRfd   string   `yaml:"topic_transaction_refunded"`
		TopicDLQ              string   `yaml:"topic_dlq"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:               "M39-Finance-Service",
		HTTPPort:                8080,
		GRPCPort:                9090,
		KafkaConsumerGroup:      "m39-finance-service",
		TopicTransactionSuccess: "transaction.succeeded",
		TopicTransactionFailed:  "transaction.failed",
		TopicTransactionRefund:  "transaction.refunded",
		DLQTopic:                "finance-service.dlq",
		IdempotencyTTL:          7 * 24 * time.Hour,
		EventDedupTTL:           7 * 24 * time.Hour,
		ConsumerPollInterval:    2 * time.Second,
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
		cfg.CampaignGRPCURL = f.Dependencies.CampaignGRPCURL
		cfg.ContentLibraryGRPCURL = f.Dependencies.ContentLibraryGRPCURL
		cfg.EscrowGRPCURL = f.Dependencies.EscrowGRPCURL
		cfg.FeeEngineGRPCURL = f.Dependencies.FeeEngineGRPCURL
		cfg.ProductGRPCURL = f.Dependencies.ProductGRPCURL
		if len(f.Dependencies.KafkaBrokers) > 0 {
			cfg.KafkaBrokers = trimNonEmpty(f.Dependencies.KafkaBrokers)
		}
		if f.Dependencies.KafkaConsumerGroup != "" {
			cfg.KafkaConsumerGroup = f.Dependencies.KafkaConsumerGroup
		}
		if f.Dependencies.TopicTransactionOK != "" {
			cfg.TopicTransactionSuccess = f.Dependencies.TopicTransactionOK
		}
		if f.Dependencies.TopicTransactionFail != "" {
			cfg.TopicTransactionFailed = f.Dependencies.TopicTransactionFail
		}
		if f.Dependencies.TopicTransactionRfd != "" {
			cfg.TopicTransactionRefund = f.Dependencies.TopicTransactionRfd
		}
		if f.Dependencies.TopicDLQ != "" {
			cfg.DLQTopic = f.Dependencies.TopicDLQ
		}
	}

	cfg.AuthGRPCURL = envOrDefault("AUTH_GRPC_URL", cfg.AuthGRPCURL)
	cfg.CampaignGRPCURL = envOrDefault("CAMPAIGN_GRPC_URL", cfg.CampaignGRPCURL)
	cfg.ContentLibraryGRPCURL = envOrDefault("CONTENT_LIBRARY_GRPC_URL", cfg.ContentLibraryGRPCURL)
	cfg.EscrowGRPCURL = envOrDefault("ESCROW_GRPC_URL", cfg.EscrowGRPCURL)
	cfg.FeeEngineGRPCURL = envOrDefault("FEE_ENGINE_GRPC_URL", cfg.FeeEngineGRPCURL)
	cfg.ProductGRPCURL = envOrDefault("PRODUCT_GRPC_URL", cfg.ProductGRPCURL)
	cfg.KafkaBrokers = envCSV("KAFKA_BROKERS", cfg.KafkaBrokers)
	cfg.KafkaConsumerGroup = envOrDefault("KAFKA_CONSUMER_GROUP", cfg.KafkaConsumerGroup)
	cfg.TopicTransactionSuccess = envOrDefault("KAFKA_TOPIC_TRANSACTION_SUCCEEDED", cfg.TopicTransactionSuccess)
	cfg.TopicTransactionFailed = envOrDefault("KAFKA_TOPIC_TRANSACTION_FAILED", cfg.TopicTransactionFailed)
	cfg.TopicTransactionRefund = envOrDefault("KAFKA_TOPIC_TRANSACTION_REFUNDED", cfg.TopicTransactionRefund)
	cfg.DLQTopic = envOrDefault("KAFKA_TOPIC_FINANCE_DLQ", cfg.DLQTopic)
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
