package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
)

// Setup2FA enables or disables second-factor methods for the authenticated user.
// It enforces at least one remaining auth path to prevent accidental lockout.
func (s *Service) Setup2FA(ctx context.Context, jwtToken string, req TwoFASetupRequest) (TwoFASetupResponse, error) {
	claims, err := s.tokenSigner.ParseAndValidate(jwtToken)
	if err != nil {
		return TwoFASetupResponse{}, domain.ErrUnauthorized
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if action == "" || method == "" {
		return TwoFASetupResponse{}, fmt.Errorf("%w: action and method are required", domain.ErrInvalidInput)
	}
	if method != "sms" && method != "email" && method != "authenticator_app" && method != "totp" {
		return TwoFASetupResponse{}, fmt.Errorf("%w: unsupported method", domain.ErrInvalidInput)
	}
	if method == "totp" {
		method = "authenticator_app"
	}

	now := s.nowFn()
	switch action {
	case "enable":
		isPrimary := false
		enabledMethods, _ := s.mfa.ListEnabledMethods(ctx, claims.UserID)
		if len(enabledMethods) == 0 {
			isPrimary = true
		}
		if err := s.mfa.SetMethodEnabled(ctx, claims.UserID, method, true, isPrimary, now); err != nil {
			return TwoFASetupResponse{}, err
		}

		resp := TwoFASetupResponse{
			Method:  method,
			Enabled: true,
		}
		if method == "authenticator_app" {
			secret := randomBase32(20)
			if err := s.mfa.UpsertTOTPSecret(ctx, claims.UserID, []byte(secret), now); err != nil {
				return TwoFASetupResponse{}, err
			}
			resp.Secret = secret

			backupCodes := make([]string, 0, 10)
			backupHashes := make([]string, 0, 10)
			for i := 0; i < 10; i++ {
				code := strings.ToUpper(randomBase32(5))
				backupCodes = append(backupCodes, code)
				backupHashes = append(backupHashes, hashToken(code))
			}
			if err := s.mfa.ReplaceBackupCodes(ctx, claims.UserID, backupHashes, now); err != nil {
				return TwoFASetupResponse{}, err
			}
			resp.BackupCodes = backupCodes
		}
		return resp, nil

	case "disable":
		enabledMethods, err := s.mfa.ListEnabledMethods(ctx, claims.UserID)
		if err != nil {
			return TwoFASetupResponse{}, err
		}
		if len(enabledMethods) <= 1 {
			return TwoFASetupResponse{}, domain.ErrCannotUnlinkLastAuth
		}
		if err := s.mfa.SetMethodEnabled(ctx, claims.UserID, method, false, false, now); err != nil {
			return TwoFASetupResponse{}, err
		}
		return TwoFASetupResponse{
			Method:  method,
			Enabled: false,
		}, nil

	default:
		return TwoFASetupResponse{}, fmt.Errorf("%w: unsupported action", domain.ErrInvalidInput)
	}
}
