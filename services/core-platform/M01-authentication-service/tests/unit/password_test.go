package unit

import (
	"testing"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		password  string
		wantError bool
	}{
		{name: "valid", password: "StrongPass123!", wantError: false},
		{name: "too short", password: "Ab1!", wantError: true},
		{name: "no symbol", password: "StrongPass1234", wantError: true},
		{name: "weak pattern", password: "Password123!", wantError: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := domain.ValidatePassword(tc.password)
			if tc.wantError && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
		})
	}
}
