package domain

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

var (
	displayNamePattern = regexp.MustCompile(`^[a-zA-Z0-9 _-]{3,50}$`)
	usernamePattern    = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)
	bioPattern         = regexp.MustCompile(`^[a-zA-Z0-9 .,!?\-'"]*$`)
	ethAddressPattern  = regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`)
	btcAddressPattern  = regexp.MustCompile(`^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$`)
)

func NormalizeUsername(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func ValidateDisplayName(v string) error {
	trimmed := strings.TrimSpace(v)
	if !displayNamePattern.MatchString(trimmed) {
		return fmt.Errorf("%w: display_name must be 3-50 chars and contain only letters, numbers, spaces, hyphens, underscores", ErrInvalidInput)
	}
	return nil
}

func ValidateBio(v string) error {
	if len(v) > 200 {
		return fmt.Errorf("%w: bio must be <= 200 chars", ErrInvalidInput)
	}
	if v != "" && !bioPattern.MatchString(v) {
		return fmt.Errorf("%w: bio contains invalid characters", ErrInvalidInput)
	}
	return nil
}

func ValidateUsername(v string) error {
	if !usernamePattern.MatchString(strings.TrimSpace(v)) {
		return fmt.Errorf("%w: username must match ^[a-zA-Z0-9_]{3,30}$", ErrInvalidInput)
	}
	return nil
}

func ValidateSocialURL(platform, profileURL string) (string, error) {
	parsed, err := url.Parse(profileURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", fmt.Errorf("%w: invalid profile_url", ErrInvalidInput)
	}

	host := strings.ToLower(parsed.Host)
	path := strings.Trim(strings.ToLower(parsed.Path), "/")
	switch platform {
	case "tiktok":
		if !(host == "www.tiktok.com" || host == "tiktok.com") || !strings.HasPrefix(path, "@") {
			return "", fmt.Errorf("%w: invalid tiktok url", ErrInvalidInput)
		}
		return strings.TrimPrefix(path, "@"), nil
	case "instagram":
		if !(host == "www.instagram.com" || host == "instagram.com") || path == "" {
			return "", fmt.Errorf("%w: invalid instagram url", ErrInvalidInput)
		}
		return strings.Split(path, "/")[0], nil
	case "youtube":
		if !(host == "www.youtube.com" || host == "youtube.com") || path == "" {
			return "", fmt.Errorf("%w: invalid youtube url", ErrInvalidInput)
		}
		parts := strings.Split(path, "/")
		return parts[len(parts)-1], nil
	case "twitter":
		if !(host == "twitter.com" || host == "www.twitter.com" || host == "x.com" || host == "www.x.com") || path == "" {
			return "", fmt.Errorf("%w: invalid twitter url", ErrInvalidInput)
		}
		return strings.Split(path, "/")[0], nil
	case "linkedin":
		if !(host == "www.linkedin.com" || host == "linkedin.com") || !strings.HasPrefix(path, "in/") {
			return "", fmt.Errorf("%w: invalid linkedin url", ErrInvalidInput)
		}
		return strings.TrimPrefix(path, "in/"), nil
	case "website":
		return host, nil
	default:
		return "", fmt.Errorf("%w: unsupported social platform", ErrInvalidInput)
	}
}

func ValidatePayoutMethodInput(methodType, rawIdentifier string) error {
	switch methodType {
	case "stripe_connect":
		if strings.TrimSpace(rawIdentifier) == "" {
			return fmt.Errorf("%w: stripe_account_id is required", ErrInvalidInput)
		}
	case "paypal":
		if _, err := mail.ParseAddress(rawIdentifier); err != nil {
			return fmt.Errorf("%w: invalid paypal email", ErrInvalidInput)
		}
	case "usdc_polygon", "eth":
		if !ethAddressPattern.MatchString(rawIdentifier) {
			return fmt.Errorf("%w: invalid ethereum address", ErrInvalidInput)
		}
	case "btc":
		if !btcAddressPattern.MatchString(rawIdentifier) {
			return fmt.Errorf("%w: invalid bitcoin address", ErrInvalidInput)
		}
	default:
		return fmt.Errorf("%w: unsupported payout method", ErrInvalidInput)
	}
	return nil
}
