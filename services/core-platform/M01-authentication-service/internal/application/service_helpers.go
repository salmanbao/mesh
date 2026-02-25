package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// recordFailure stores failed login context for audit and lockout policies.
func (s *Service) recordFailure(ctx context.Context, userID *uuid.UUID, req LoginRequest, reason string) {
	if err := s.loginAttempts.Insert(ctx, domain.LoginAttempt{
		UserID:        userID,
		AttemptAt:     s.nowFn(),
		IPAddress:     req.IPAddress,
		Status:        "FAILED",
		FailureReason: reason,
		DeviceName:    req.DeviceName,
		DeviceOS:      req.DeviceOS,
		UserAgent:     req.UserAgent,
	}); err != nil {
		slog.Default().WarnContext(ctx, "failed to persist login attempt",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "record_login_failure",
			"outcome", "failure",
			"reason", reason,
			"error", err,
		)
	}
}

// normalizeEmail canonicalizes and validates email format before persistence/comparison.
func normalizeEmail(email string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" {
		return "", fmt.Errorf("%w: email is required", domain.ErrInvalidInput)
	}
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return "", fmt.Errorf("%w: invalid email", domain.ErrInvalidInput)
	}
	return trimmed, nil
}

// hashRequest computes deterministic request fingerprint for idempotency conflict detection.
func hashRequest(req any) string {
	raw, _ := json.Marshal(req)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// hashToken stores one-way token fingerprints instead of raw secrets.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// randomHex returns a cryptographically random hex token.
func randomHex(bytesLen int) string {
	raw := make([]byte, bytesLen)
	_, _ = rand.Read(raw)
	return hex.EncodeToString(raw)
}

// randomBase32 returns a random base32 string suitable for human entry or URL usage.
func randomBase32(bytesLen int) string {
	raw := make([]byte, bytesLen)
	_, _ = rand.Read(raw)
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "=")
}

// randomDigits returns a zero-padded random numeric code.
func randomDigits(size int) string {
	if size <= 0 {
		size = 6
	}
	max := 1
	for i := 0; i < size; i++ {
		max *= 10
	}
	nRaw := make([]byte, 8)
	_, _ = rand.Read(nRaw)
	n := int(nRaw[0])<<24 | int(nRaw[1])<<16 | int(nRaw[2])<<8 | int(nRaw[3])
	if n < 0 {
		n = -n
	}
	value := n % max
	return fmt.Sprintf("%0*d", size, value)
}

// generatePKCEVerifierChallenge creates PKCE verifier and S256 challenge pair.
func generatePKCEVerifierChallenge() (string, string) {
	verifier := randomBase32(32)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

// buildRedirectWithFragment appends auth results to redirect fragment without mutating query params.
func buildRedirectWithFragment(redirectURI, fragment string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	if u.Path == "" {
		u.Path = "/"
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path = path.Clean(u.Path)
	}
	u.Fragment = fragment
	return u.String()
}

func hasDeprecatedOIDCRegisterFields(req RegisterRequest) bool {
	return strings.TrimSpace(req.Provider) != "" ||
		strings.TrimSpace(req.AuthorizationCode) != "" ||
		strings.TrimSpace(req.RedirectURI) != "" ||
		strings.TrimSpace(req.Nonce) != "" ||
		strings.TrimSpace(req.CodeVerifier) != ""
}

func (s *Service) enforceRateLimit(ctx context.Context, key string, threshold int, window time.Duration) error {
	if s.lockouts == nil || threshold <= 0 || window <= 0 {
		return nil
	}
	if strings.TrimSpace(key) == "" {
		return nil
	}

	state, err := s.lockouts.Get(ctx, key)
	if err == nil && state.LockedUntil != nil && state.LockedUntil.After(s.nowFn()) {
		return domain.ErrRateLimited
	}

	now := s.nowFn()
	updated, err := s.lockouts.RecordFailure(ctx, key, now, threshold, window)
	if err != nil {
		slog.Default().WarnContext(ctx, "rate-limit state unavailable",
			"service", "M01-Authentication-Service",
			"module", "application",
			"layer", "application",
			"operation", "rate_limit",
			"outcome", "warning",
			"key", key,
			"error", err,
		)
		return nil
	}
	if updated.LockedUntil != nil && updated.LockedUntil.After(now) {
		return domain.ErrRateLimited
	}
	return nil
}
