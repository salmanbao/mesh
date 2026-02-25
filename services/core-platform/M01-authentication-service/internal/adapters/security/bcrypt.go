package security

import (
	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher implements password hashing via bcrypt.
// Cost is configurable so security/performance can be tuned by environment.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a bcrypt-based hasher with default fallback cost.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost <= 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (h *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
