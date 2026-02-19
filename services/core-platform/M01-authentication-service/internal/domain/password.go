package domain

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	minPasswordLength = 12
	maxPasswordLength = 128
)

// ValidatePassword enforces baseline M01 password policy.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLength {
		return fmt.Errorf("%w: password must be at least %d characters", ErrInvalidInput, minPasswordLength)
	}
	if len(password) > maxPasswordLength {
		return fmt.Errorf("%w: password must be <= %d characters", ErrInvalidInput, maxPasswordLength)
	}

	var (
		hasUpper bool
		hasLower bool
		hasDigit bool
		hasPunct bool
	)

	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasPunct = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit || !hasPunct {
		return fmt.Errorf("%w: password must include upper, lower, digit, and symbol", ErrInvalidInput)
	}

	lowered := strings.ToLower(password)
	for _, banned := range []string{"password", "qwerty", "123456", "letmein"} {
		if strings.Contains(lowered, banned) {
			return fmt.Errorf("%w: password includes weak pattern", ErrInvalidInput)
		}
	}

	return nil
}
