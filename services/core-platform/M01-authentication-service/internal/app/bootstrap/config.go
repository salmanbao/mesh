package bootstrap

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the resolved runtime configuration for M01.
// It merges file defaults and environment overrides to support both local and deployed runs.
type Config struct {
	ServiceID string

	HTTPPort int
	GRPCPort int

	DatabaseURL string
	RedisURL    string

	JWTPrivateKeyPEM  string
	JWTPublicKeyPEM   string
	JWTKeyID          string
	AllowEphemeralJWT bool

	BcryptCost int

	TokenTTL           time.Duration
	SessionTTL         time.Duration
	SessionAbsoluteTTL time.Duration
	LockoutDuration    time.Duration
	FailedThreshold    int

	OIDCGoogleIssuerURL                       string
	OIDCGoogleClientID                        string
	OIDCGoogleClientSecret                    string
	OIDCGoogleScopes                          []string
	OIDCGoogleAllowedRedirectURIs             []string
	OIDCHTTPTimeout                           time.Duration
	RegisterOIDCFieldMode                     string
	OIDCAllowEmailLinking                     bool
	OIDCCompletionTokenTTL                    time.Duration
	RegisterRateLimitIPThreshold              int
	RegisterRateLimitIdentifierThreshold      int
	RegisterRateLimitWindow                   time.Duration
	OIDCAuthorizeRateLimitIPThreshold         int
	OIDCAuthorizeRateLimitIdentifierThreshold int
	OIDCAuthorizeRateLimitWindow              time.Duration

	MaxDBConns           int32
	OutboxPollInterval   time.Duration
	OutboxBatchSize      int
	OutboxClaimTTL       time.Duration
	OutboxMaxRetries     int
	OIDCRefreshInterval  time.Duration
	OIDCRefreshWindow    time.Duration
	OIDCRefreshBatchSize int
}

// configFile mirrors the YAML schema used by configs/default.yaml.
// It is intentionally separate from Config so runtime-only fields stay internal.
type configFile struct {
	Service struct {
		ID       string `yaml:"id"`
		HTTPPort int    `yaml:"http_port"`
		GRPCPort int    `yaml:"grpc_port"`
	} `yaml:"service"`
	Dependencies struct {
		PostgresURL string `yaml:"postgres_url"`
		RedisURL    string `yaml:"redis_url"`
	} `yaml:"dependencies"`
	OIDC struct {
		Google struct {
			IssuerURL           string   `yaml:"issuer_url"`
			ClientID            string   `yaml:"client_id"`
			ClientSecret        string   `yaml:"client_secret"`
			Scopes              []string `yaml:"scopes"`
			AllowedRedirectURIs []string `yaml:"allowed_redirect_uris"`
		} `yaml:"google"`
	} `yaml:"oidc"`
}

// LoadConfig resolves configuration in priority order: defaults -> file -> env.
// This order keeps local bootstrap simple while allowing environment-specific overrides.
func LoadConfig(path string) (Config, error) {
	cfg := Config{
		ServiceID:                            "M01-Authentication-Service",
		HTTPPort:                             8080,
		GRPCPort:                             9090,
		JWTKeyID:                             "m01-auth-key-1",
		AllowEphemeralJWT:                    true,
		BcryptCost:                           12,
		TokenTTL:                             24 * time.Hour,
		SessionTTL:                           30 * 24 * time.Hour,
		SessionAbsoluteTTL:                   90 * 24 * time.Hour,
		LockoutDuration:                      30 * time.Minute,
		FailedThreshold:                      5,
		OIDCGoogleIssuerURL:                  "https://accounts.google.com",
		OIDCGoogleScopes:                     []string{"openid", "email", "profile"},
		OIDCHTTPTimeout:                      8 * time.Second,
		RegisterOIDCFieldMode:                "reject",
		OIDCAllowEmailLinking:                true,
		OIDCCompletionTokenTTL:               10 * time.Minute,
		RegisterRateLimitIPThreshold:         20,
		RegisterRateLimitIdentifierThreshold: 6,
		RegisterRateLimitWindow:              time.Minute,
		OIDCAuthorizeRateLimitIPThreshold:    30,
		OIDCAuthorizeRateLimitIdentifierThreshold: 10,
		OIDCAuthorizeRateLimitWindow:              time.Minute,
		MaxDBConns:                                20,
		OutboxPollInterval:                        2 * time.Second,
		OutboxBatchSize:                           100,
		OutboxClaimTTL:                            30 * time.Second,
		OutboxMaxRetries:                          5,
		OIDCRefreshInterval:                       time.Hour,
		OIDCRefreshWindow:                         24 * time.Hour,
		OIDCRefreshBatchSize:                      100,
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
		if f.OIDC.Google.IssuerURL != "" {
			cfg.OIDCGoogleIssuerURL = f.OIDC.Google.IssuerURL
		}
		if f.OIDC.Google.ClientID != "" {
			cfg.OIDCGoogleClientID = f.OIDC.Google.ClientID
		}
		if f.OIDC.Google.ClientSecret != "" {
			cfg.OIDCGoogleClientSecret = f.OIDC.Google.ClientSecret
		}
		if len(f.OIDC.Google.Scopes) > 0 {
			cfg.OIDCGoogleScopes = f.OIDC.Google.Scopes
		}
		if len(f.OIDC.Google.AllowedRedirectURIs) > 0 {
			cfg.OIDCGoogleAllowedRedirectURIs = f.OIDC.Google.AllowedRedirectURIs
		}
	}

	cfg.DatabaseURL = envOrDefault("DB_URL", envOrDefault("POSTGRES_URL", cfg.DatabaseURL))
	cfg.RedisURL = envOrDefault("REDIS_URL", cfg.RedisURL)
	cfg.JWTPrivateKeyPEM = envOrDefault("JWT_PRIVATE_KEY_PEM", cfg.JWTPrivateKeyPEM)
	cfg.JWTPublicKeyPEM = envOrDefault("JWT_PUBLIC_KEY_PEM", cfg.JWTPublicKeyPEM)
	cfg.JWTKeyID = envOrDefault("JWT_KEY_ID", cfg.JWTKeyID)
	cfg.AllowEphemeralJWT = envBool("JWT_ALLOW_EPHEMERAL", cfg.AllowEphemeralJWT)
	cfg.OIDCGoogleIssuerURL = envOrDefault("OIDC_GOOGLE_ISSUER_URL", cfg.OIDCGoogleIssuerURL)
	cfg.OIDCGoogleClientID = envOrDefault("OIDC_GOOGLE_CLIENT_ID", cfg.OIDCGoogleClientID)
	cfg.OIDCGoogleClientSecret = envOrDefault("OIDC_GOOGLE_CLIENT_SECRET", cfg.OIDCGoogleClientSecret)
	cfg.OIDCGoogleScopes = envCSV("OIDC_GOOGLE_SCOPES", cfg.OIDCGoogleScopes)
	cfg.OIDCGoogleAllowedRedirectURIs = envCSV("OIDC_GOOGLE_ALLOWED_REDIRECT_URIS", cfg.OIDCGoogleAllowedRedirectURIs)
	cfg.RegisterOIDCFieldMode = strings.ToLower(strings.TrimSpace(envOrDefault("REGISTER_OIDC_FIELD_MODE", cfg.RegisterOIDCFieldMode)))
	cfg.OIDCAllowEmailLinking = envBool("OIDC_ALLOW_EMAIL_LINKING", cfg.OIDCAllowEmailLinking)
	cfg.RegisterRateLimitIPThreshold = envInt("REGISTER_RATE_LIMIT_IP_THRESHOLD", cfg.RegisterRateLimitIPThreshold)
	cfg.RegisterRateLimitIdentifierThreshold = envInt("REGISTER_RATE_LIMIT_IDENTIFIER_THRESHOLD", cfg.RegisterRateLimitIdentifierThreshold)
	cfg.OIDCAuthorizeRateLimitIPThreshold = envInt("OIDC_AUTHORIZE_RATE_LIMIT_IP_THRESHOLD", cfg.OIDCAuthorizeRateLimitIPThreshold)
	cfg.OIDCAuthorizeRateLimitIdentifierThreshold = envInt("OIDC_AUTHORIZE_RATE_LIMIT_IDENTIFIER_THRESHOLD", cfg.OIDCAuthorizeRateLimitIdentifierThreshold)

	cfg.HTTPPort = envInt("HTTP_PORT", cfg.HTTPPort)
	cfg.GRPCPort = envInt("GRPC_PORT", cfg.GRPCPort)
	cfg.BcryptCost = envInt("BCRYPT_ROUNDS", cfg.BcryptCost)
	cfg.FailedThreshold = envInt("FAILED_LOGIN_THRESHOLD", cfg.FailedThreshold)
	cfg.MaxDBConns = int32(envInt("DB_MAX_CONNS", int(cfg.MaxDBConns)))

	cfg.TokenTTL = time.Duration(envInt("TOKEN_EXPIRY_HOURS", int(cfg.TokenTTL.Hours()))) * time.Hour
	cfg.SessionTTL = time.Duration(envInt("SESSION_EXPIRY_DAYS", int(cfg.SessionTTL.Hours()/24))) * 24 * time.Hour
	cfg.SessionAbsoluteTTL = time.Duration(envInt("SESSION_ABSOLUTE_DAYS", int(cfg.SessionAbsoluteTTL.Hours()/24))) * 24 * time.Hour
	cfg.LockoutDuration = time.Duration(envInt("ACCOUNT_LOCKOUT_MINUTES", int(cfg.LockoutDuration.Minutes()))) * time.Minute
	cfg.OIDCHTTPTimeout = time.Duration(envInt("OIDC_HTTP_TIMEOUT_SECONDS", int(cfg.OIDCHTTPTimeout.Seconds()))) * time.Second
	cfg.OutboxPollInterval = time.Duration(envInt("OUTBOX_POLL_SECONDS", int(cfg.OutboxPollInterval.Seconds()))) * time.Second
	cfg.OutboxBatchSize = envInt("OUTBOX_BATCH_SIZE", cfg.OutboxBatchSize)
	cfg.OutboxClaimTTL = time.Duration(envInt("OUTBOX_CLAIM_TTL_SECONDS", int(cfg.OutboxClaimTTL.Seconds()))) * time.Second
	cfg.OutboxMaxRetries = envInt("OUTBOX_MAX_RETRIES", cfg.OutboxMaxRetries)
	cfg.OIDCRefreshInterval = time.Duration(envInt("OIDC_REFRESH_INTERVAL_SECONDS", int(cfg.OIDCRefreshInterval.Seconds()))) * time.Second
	cfg.OIDCRefreshWindow = time.Duration(envInt("OIDC_REFRESH_WINDOW_HOURS", int(cfg.OIDCRefreshWindow.Hours()))) * time.Hour
	cfg.OIDCRefreshBatchSize = envInt("OIDC_REFRESH_BATCH_SIZE", cfg.OIDCRefreshBatchSize)
	cfg.OIDCCompletionTokenTTL = time.Duration(envInt("OIDC_COMPLETION_TOKEN_TTL_SECONDS", int(cfg.OIDCCompletionTokenTTL.Seconds()))) * time.Second
	cfg.RegisterRateLimitWindow = time.Duration(envInt("REGISTER_RATE_LIMIT_WINDOW_SECONDS", int(cfg.RegisterRateLimitWindow.Seconds()))) * time.Second
	cfg.OIDCAuthorizeRateLimitWindow = time.Duration(envInt("OIDC_AUTHORIZE_RATE_LIMIT_WINDOW_SECONDS", int(cfg.OIDCAuthorizeRateLimitWindow.Seconds()))) * time.Second

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("missing DB_URL/POSTGRES_URL")
	}
	if cfg.RedisURL == "" {
		return Config{}, fmt.Errorf("missing REDIS_URL")
	}
	if (cfg.JWTPrivateKeyPEM == "" || cfg.JWTPublicKeyPEM == "") && !cfg.AllowEphemeralJWT {
		return Config{}, fmt.Errorf("missing JWT_PRIVATE_KEY_PEM or JWT_PUBLIC_KEY_PEM")
	}

	return cfg, nil
}

// envOrDefault returns an env var when present, otherwise the provided fallback.
func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

// envInt parses integer env vars with safe fallback on empty/invalid values.
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

// envBool parses common boolean env forms while keeping a deterministic fallback.
func envBool(name string, fallback bool) bool {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "TRUE", "yes", "YES":
		return true
	case "0", "false", "FALSE", "no", "NO":
		return false
	default:
		return fallback
	}
}

// envCSV parses comma-separated env vars and removes empty segments.
func envCSV(name string, fallback []string) []string {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parts := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	if len(parts) == 0 {
		return fallback
	}
	return parts
}
