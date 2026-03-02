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

	AuthGRPCURL       string
	CampaignGRPCURL   string
	VotingGRPCURL     string
	TrackingGRPCURL   string
	SubmissionGRPCURL string

	KafkaBrokers               []string
	KafkaConsumerGroup         string
	TopicSubmissionAutoApprove string
	TopicSubmissionCancelled   string
	TopicSubmissionVerified    string
	TopicSubmissionViewLocked  string
	TopicTrackingUpdated       string
	TopicRewardCalculated      string
	TopicRewardPayoutEligible  string
	DLQTopic                   string

	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration
	ConsumerPollInterval time.Duration

	EnableDomainEventConsumption bool
	EnablePayoutEligibleEmission bool
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		AuthGRPCURL                 string   `yaml:"auth_grpc_url"`
		CampaignGRPCURL             string   `yaml:"campaign_grpc_url"`
		VotingGRPCURL               string   `yaml:"voting_grpc_url"`
		TrackingGRPCURL             string   `yaml:"tracking_grpc_url"`
		SubmissionGRPCURL           string   `yaml:"submission_grpc_url"`
		KafkaBrokers                []string `yaml:"kafka_brokers"`
		KafkaConsumerGroup          string   `yaml:"kafka_consumer_group"`
		TopicSubmissionAutoApproved string   `yaml:"topic_submission_auto_approved"`
		TopicSubmissionCancelled    string   `yaml:"topic_submission_cancelled"`
		TopicSubmissionVerified     string   `yaml:"topic_submission_verified"`
		TopicSubmissionViewLocked   string   `yaml:"topic_submission_view_locked"`
		TopicTrackingUpdated        string   `yaml:"topic_tracking_metrics_updated"`
		TopicRewardCalculated       string   `yaml:"topic_reward_calculated"`
		TopicRewardPayoutEligible   string   `yaml:"topic_reward_payout_eligible"`
		TopicDLQ                    string   `yaml:"topic_dlq"`
	} `yaml:"dependencies"`
	FeatureFlags struct {
		EnableDomainEventConsumption *bool `yaml:"enable_domain_event_consumption"`
		EnablePayoutEligibleEmission *bool `yaml:"enable_payout_eligible_emission"`
	} `yaml:"feature_flags"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:                    "M41-Reward-Engine",
		HTTPPort:                     8080,
		GRPCPort:                     9090,
		KafkaConsumerGroup:           "m41-reward-engine",
		TopicSubmissionAutoApprove:   "submission.auto_approved",
		TopicSubmissionCancelled:     "submission.cancelled",
		TopicSubmissionVerified:      "submission.verified",
		TopicSubmissionViewLocked:    "submission.view_locked",
		TopicTrackingUpdated:         "tracking.metrics.updated",
		TopicRewardCalculated:        "reward.calculated",
		TopicRewardPayoutEligible:    "reward.payout_eligible",
		DLQTopic:                     "reward-engine.dlq",
		IdempotencyTTL:               7 * 24 * time.Hour,
		EventDedupTTL:                7 * 24 * time.Hour,
		ConsumerPollInterval:         2 * time.Second,
		EnableDomainEventConsumption: true,
		EnablePayoutEligibleEmission: true,
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
		cfg.VotingGRPCURL = f.Dependencies.VotingGRPCURL
		cfg.TrackingGRPCURL = f.Dependencies.TrackingGRPCURL
		cfg.SubmissionGRPCURL = f.Dependencies.SubmissionGRPCURL
		if len(f.Dependencies.KafkaBrokers) > 0 {
			cfg.KafkaBrokers = trimNonEmpty(f.Dependencies.KafkaBrokers)
		}
		if f.Dependencies.KafkaConsumerGroup != "" {
			cfg.KafkaConsumerGroup = f.Dependencies.KafkaConsumerGroup
		}
		if f.Dependencies.TopicSubmissionAutoApproved != "" {
			cfg.TopicSubmissionAutoApprove = f.Dependencies.TopicSubmissionAutoApproved
		}
		if f.Dependencies.TopicSubmissionCancelled != "" {
			cfg.TopicSubmissionCancelled = f.Dependencies.TopicSubmissionCancelled
		}
		if f.Dependencies.TopicSubmissionVerified != "" {
			cfg.TopicSubmissionVerified = f.Dependencies.TopicSubmissionVerified
		}
		if f.Dependencies.TopicSubmissionViewLocked != "" {
			cfg.TopicSubmissionViewLocked = f.Dependencies.TopicSubmissionViewLocked
		}
		if f.Dependencies.TopicTrackingUpdated != "" {
			cfg.TopicTrackingUpdated = f.Dependencies.TopicTrackingUpdated
		}
		if f.Dependencies.TopicRewardCalculated != "" {
			cfg.TopicRewardCalculated = f.Dependencies.TopicRewardCalculated
		}
		if f.Dependencies.TopicRewardPayoutEligible != "" {
			cfg.TopicRewardPayoutEligible = f.Dependencies.TopicRewardPayoutEligible
		}
		if f.Dependencies.TopicDLQ != "" {
			cfg.DLQTopic = f.Dependencies.TopicDLQ
		}
		if f.FeatureFlags.EnableDomainEventConsumption != nil {
			cfg.EnableDomainEventConsumption = *f.FeatureFlags.EnableDomainEventConsumption
		}
		if f.FeatureFlags.EnablePayoutEligibleEmission != nil {
			cfg.EnablePayoutEligibleEmission = *f.FeatureFlags.EnablePayoutEligibleEmission
		}
	}

	cfg.AuthGRPCURL = envOrDefault("AUTH_GRPC_URL", cfg.AuthGRPCURL)
	cfg.CampaignGRPCURL = envOrDefault("CAMPAIGN_GRPC_URL", cfg.CampaignGRPCURL)
	cfg.VotingGRPCURL = envOrDefault("VOTING_GRPC_URL", cfg.VotingGRPCURL)
	cfg.TrackingGRPCURL = envOrDefault("TRACKING_GRPC_URL", cfg.TrackingGRPCURL)
	cfg.SubmissionGRPCURL = envOrDefault("SUBMISSION_GRPC_URL", cfg.SubmissionGRPCURL)
	cfg.KafkaBrokers = envCSV("KAFKA_BROKERS", cfg.KafkaBrokers)
	cfg.KafkaConsumerGroup = envOrDefault("KAFKA_CONSUMER_GROUP", cfg.KafkaConsumerGroup)
	cfg.TopicSubmissionAutoApprove = envOrDefault("KAFKA_TOPIC_SUBMISSION_AUTO_APPROVED", cfg.TopicSubmissionAutoApprove)
	cfg.TopicSubmissionCancelled = envOrDefault("KAFKA_TOPIC_SUBMISSION_CANCELLED", cfg.TopicSubmissionCancelled)
	cfg.TopicSubmissionVerified = envOrDefault("KAFKA_TOPIC_SUBMISSION_VERIFIED", cfg.TopicSubmissionVerified)
	cfg.TopicSubmissionViewLocked = envOrDefault("KAFKA_TOPIC_SUBMISSION_VIEW_LOCKED", cfg.TopicSubmissionViewLocked)
	cfg.TopicTrackingUpdated = envOrDefault("KAFKA_TOPIC_TRACKING_METRICS_UPDATED", cfg.TopicTrackingUpdated)
	cfg.TopicRewardCalculated = envOrDefault("KAFKA_TOPIC_REWARD_CALCULATED", cfg.TopicRewardCalculated)
	cfg.TopicRewardPayoutEligible = envOrDefault("KAFKA_TOPIC_REWARD_PAYOUT_ELIGIBLE", cfg.TopicRewardPayoutEligible)
	cfg.DLQTopic = envOrDefault("KAFKA_TOPIC_REWARD_DLQ", cfg.DLQTopic)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second
	cfg.EnableDomainEventConsumption = envBool("ENABLE_DOMAIN_EVENT_CONSUMPTION", cfg.EnableDomainEventConsumption)
	cfg.EnablePayoutEligibleEmission = envBool("ENABLE_PAYOUT_ELIGIBLE_EMISSION", cfg.EnablePayoutEligibleEmission)

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

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
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
