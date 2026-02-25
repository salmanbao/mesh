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

	DatabaseURL      string
	RedisURL         string
	AuthGRPCURL      string
	KafkaBrokers     []string
	S3Bucket         string
	S3Region         string
	CloudFrontDomain string
	KMSKeyID         string
	ClamAVAddress    string

	MaxDBConns               int32
	KafkaConsumerGroup       string
	KafkaTopicUserRegistered string
	KafkaTopicUserDeleted    string
	KafkaTopicProfileUpdated string

	OutboxPollInterval   time.Duration
	OutboxBatchSize      int
	ConsumerPollInterval time.Duration

	ProfileCacheTTL      time.Duration
	UsernameCooldownDays int
	UsernameRedirectDays int
	MaxSocialLinks       int
	IdempotencyTTL       time.Duration
	EventDedupTTL        time.Duration

	EncryptionSeed string

	FeatureProfileCompletenessVisible   bool
	FeatureKYCReverificationInterval    time.Duration
	FeatureFollowerSyncHighEarnerHourly bool
	FeaturePayPalOwnershipVerification  bool
	FeatureAvatarManualRetry            bool
	KYCAnonymizeAfter                   time.Duration
	UsernameHistoryRetention            time.Duration
}

type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		PostgresURL              string   `yaml:"postgres_url"`
		RedisURL                 string   `yaml:"redis_url"`
		AuthGRPCURL              string   `yaml:"auth_grpc_url"`
		KafkaBrokers             []string `yaml:"kafka_brokers"`
		KafkaConsumerGroup       string   `yaml:"kafka_consumer_group"`
		KafkaTopicUserRegistered string   `yaml:"kafka_topic_user_registered"`
		KafkaTopicUserDeleted    string   `yaml:"kafka_topic_user_deleted"`
		KafkaTopicProfileUpdated string   `yaml:"kafka_topic_profile_updated"`
		S3Bucket                 string   `yaml:"s3_bucket"`
		S3Region                 string   `yaml:"s3_region"`
		CloudFrontDomain         string   `yaml:"cloudfront_domain"`
		KMSKeyID                 string   `yaml:"kms_key_id"`
		ClamAVAddress            string   `yaml:"clamav_address"`
	} `yaml:"dependencies"`
}

func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:                        "M02-Profile-Service",
		HTTPPort:                         8080,
		GRPCPort:                         9090,
		MaxDBConns:                       20,
		KafkaConsumerGroup:               "m02-profile-service",
		KafkaTopicUserRegistered:         "user.registered",
		KafkaTopicUserDeleted:            "user.deleted",
		KafkaTopicProfileUpdated:         "user.profile_updated",
		OutboxPollInterval:               2 * time.Second,
		OutboxBatchSize:                  100,
		ConsumerPollInterval:             2 * time.Second,
		ProfileCacheTTL:                  5 * time.Minute,
		UsernameCooldownDays:             365,
		UsernameRedirectDays:             90,
		MaxSocialLinks:                   5,
		IdempotencyTTL:                   7 * 24 * time.Hour,
		EventDedupTTL:                    7 * 24 * time.Hour,
		EncryptionSeed:                   "m02-default-seed",
		FeatureKYCReverificationInterval: 0,
		FeatureAvatarManualRetry:         true,
		KYCAnonymizeAfter:                365 * 24 * time.Hour,
		UsernameHistoryRetention:         365 * 24 * time.Hour,
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
		if f.Dependencies.PostgresURL != "" {
			cfg.DatabaseURL = f.Dependencies.PostgresURL
		}
		if f.Dependencies.RedisURL != "" {
			cfg.RedisURL = f.Dependencies.RedisURL
		}
		if f.Dependencies.AuthGRPCURL != "" {
			cfg.AuthGRPCURL = f.Dependencies.AuthGRPCURL
		}
		if len(f.Dependencies.KafkaBrokers) > 0 {
			cfg.KafkaBrokers = trimNonEmpty(f.Dependencies.KafkaBrokers)
		}
		if f.Dependencies.KafkaConsumerGroup != "" {
			cfg.KafkaConsumerGroup = f.Dependencies.KafkaConsumerGroup
		}
		if f.Dependencies.KafkaTopicUserRegistered != "" {
			cfg.KafkaTopicUserRegistered = f.Dependencies.KafkaTopicUserRegistered
		}
		if f.Dependencies.KafkaTopicUserDeleted != "" {
			cfg.KafkaTopicUserDeleted = f.Dependencies.KafkaTopicUserDeleted
		}
		if f.Dependencies.KafkaTopicProfileUpdated != "" {
			cfg.KafkaTopicProfileUpdated = f.Dependencies.KafkaTopicProfileUpdated
		}
		cfg.S3Bucket = f.Dependencies.S3Bucket
		cfg.S3Region = f.Dependencies.S3Region
		cfg.CloudFrontDomain = f.Dependencies.CloudFrontDomain
		cfg.KMSKeyID = f.Dependencies.KMSKeyID
		cfg.ClamAVAddress = f.Dependencies.ClamAVAddress
	}

	cfg.DatabaseURL = envOrDefault("DB_URL", envOrDefault("POSTGRES_URL", cfg.DatabaseURL))
	cfg.RedisURL = envOrDefault("REDIS_URL", cfg.RedisURL)
	cfg.AuthGRPCURL = envOrDefault("AUTH_GRPC_URL", envOrDefault("AUTH_SERVICE_GRPC_ADDR", cfg.AuthGRPCURL))
	cfg.KafkaBrokers = envCSV("KAFKA_BROKERS", cfg.KafkaBrokers)
	cfg.KafkaConsumerGroup = envOrDefault("KAFKA_CONSUMER_GROUP", cfg.KafkaConsumerGroup)
	cfg.KafkaTopicUserRegistered = envOrDefault("KAFKA_TOPIC_USER_REGISTERED", cfg.KafkaTopicUserRegistered)
	cfg.KafkaTopicUserDeleted = envOrDefault("KAFKA_TOPIC_USER_DELETED", cfg.KafkaTopicUserDeleted)
	cfg.KafkaTopicProfileUpdated = envOrDefault("KAFKA_TOPIC_PROFILE_UPDATED", cfg.KafkaTopicProfileUpdated)
	cfg.S3Bucket = envOrDefault("S3_BUCKET", cfg.S3Bucket)
	cfg.S3Region = envOrDefault("S3_REGION", cfg.S3Region)
	cfg.CloudFrontDomain = envOrDefault("CLOUDFRONT_DOMAIN", cfg.CloudFrontDomain)
	cfg.KMSKeyID = envOrDefault("KMS_KEY_ID", cfg.KMSKeyID)
	cfg.ClamAVAddress = envOrDefault("CLAMAV_ADDRESS", cfg.ClamAVAddress)
	cfg.EncryptionSeed = envOrDefault("ENCRYPTION_SEED", cfg.EncryptionSeed)
	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.MaxDBConns = int32(envInt("DB_MAX_CONNS", int(cfg.MaxDBConns)))
	cfg.OutboxPollInterval = time.Duration(envInt("OUTBOX_POLL_SECONDS", int(cfg.OutboxPollInterval.Seconds()))) * time.Second
	cfg.OutboxBatchSize = envInt("OUTBOX_BATCH_SIZE", cfg.OutboxBatchSize)
	cfg.ConsumerPollInterval = time.Duration(envInt("CONSUMER_POLL_SECONDS", int(cfg.ConsumerPollInterval.Seconds()))) * time.Second
	cfg.ProfileCacheTTL = time.Duration(envInt("PROFILE_CACHE_SECONDS", int(cfg.ProfileCacheTTL.Seconds()))) * time.Second
	cfg.UsernameCooldownDays = envInt("USERNAME_CHANGE_COOLDOWN_DAYS", cfg.UsernameCooldownDays)
	cfg.UsernameRedirectDays = envInt("USERNAME_REDIRECT_DAYS", cfg.UsernameRedirectDays)
	cfg.MaxSocialLinks = envInt("MAX_SOCIAL_LINKS", cfg.MaxSocialLinks)
	cfg.FeatureProfileCompletenessVisible = envBool("FEATURE_PROFILE_COMPLETENESS_VISIBLE", cfg.FeatureProfileCompletenessVisible)
	cfg.FeatureFollowerSyncHighEarnerHourly = envBool("FEATURE_FOLLOWER_SYNC_HIGH_EARNER_HOURLY", cfg.FeatureFollowerSyncHighEarnerHourly)
	cfg.FeaturePayPalOwnershipVerification = envBool("FEATURE_PAYPAL_OWNERSHIP_VERIFICATION", cfg.FeaturePayPalOwnershipVerification)
	cfg.FeatureAvatarManualRetry = envBool("FEATURE_AVATAR_MANUAL_RETRY", cfg.FeatureAvatarManualRetry)
	cfg.IdempotencyTTL = time.Duration(envInt("IDEMPOTENCY_TTL_HOURS", int(cfg.IdempotencyTTL.Hours()))) * time.Hour
	cfg.EventDedupTTL = time.Duration(envInt("EVENT_DEDUP_TTL_HOURS", int(cfg.EventDedupTTL.Hours()))) * time.Hour
	cfg.KYCAnonymizeAfter = time.Duration(envInt("KYC_ANONYMIZE_AFTER_DAYS", int(cfg.KYCAnonymizeAfter.Hours()/24))) * 24 * time.Hour
	cfg.UsernameHistoryRetention = time.Duration(envInt("USERNAME_HISTORY_RETENTION_DAYS", int(cfg.UsernameHistoryRetention.Hours()/24))) * 24 * time.Hour
	cfg.FeatureKYCReverificationInterval = time.Duration(envInt("KYC_REVERIFY_INTERVAL_DAYS", 0)) * 24 * time.Hour

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("missing DB_URL/POSTGRES_URL")
	}
	if cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("missing REDIS_URL")
	}
	if cfg.AuthGRPCURL == "" {
		return Config{}, fmt.Errorf("missing AUTH_GRPC_URL")
	}
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

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return fallback
	}
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
