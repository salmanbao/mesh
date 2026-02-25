package unit

import (
	"testing"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/domain"
)

func TestValidateDisplayName(t *testing.T) {
	t.Parallel()

	if err := domain.ValidateDisplayName("Jane Clips"); err != nil {
		t.Fatalf("expected valid display name, got %v", err)
	}
	if err := domain.ValidateDisplayName("x"); err == nil {
		t.Fatalf("expected invalid display name error")
	}
}

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	if err := domain.ValidateUsername("jane_clips_22"); err != nil {
		t.Fatalf("expected valid username, got %v", err)
	}
	if err := domain.ValidateUsername("bad username"); err == nil {
		t.Fatalf("expected invalid username error")
	}
}

func TestValidatePayoutMethodInput(t *testing.T) {
	t.Parallel()

	if err := domain.ValidatePayoutMethodInput("paypal", "jane@example.com"); err != nil {
		t.Fatalf("expected valid paypal email, got %v", err)
	}
	if err := domain.ValidatePayoutMethodInput("usdc_polygon", "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb3"); err != nil {
		t.Fatalf("expected valid eth address, got %v", err)
	}
	if err := domain.ValidatePayoutMethodInput("btc", "not-a-btc-address"); err == nil {
		t.Fatalf("expected invalid btc address error")
	}
}
