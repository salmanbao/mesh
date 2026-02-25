package postgres

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/domain"
	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/ports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type oidcRepository struct {
	db *gorm.DB
}

func (r *oidcRepository) FindUserByIssuerSubject(ctx context.Context, issuer, subject string) (uuid.UUID, error) {
	var rec oauthConnectionModel
	if err := r.db.WithContext(ctx).
		Where("issuer = ?", issuer).
		Where("subject = ?", subject).
		Where("status = 'ACTIVE'").
		Take(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return uuid.Nil, domain.ErrNotFound
		}
		return uuid.Nil, err
	}
	return rec.UserID, nil
}

func (r *oidcRepository) UpsertConnection(ctx context.Context, userID uuid.UUID, issuer, subject, provider, email string, isPrimary bool, now time.Time) error {
	rec := oauthConnectionModel{
		UserID:         userID,
		Issuer:         issuer,
		Subject:        subject,
		Provider:       provider,
		ProviderUserID: subject,
		Email:          email,
		EmailAtLinkTime: email,
		LinkedAt:       now,
		LastLoginAt:    now,
		IsPrimary:      isPrimary,
		Status:         "ACTIVE",
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "issuer"},
			{Name: "subject"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"user_id":         rec.UserID,
			"provider":        rec.Provider,
			"provider_user_id": rec.ProviderUserID,
			"email":           rec.Email,
			"email_at_link_time": rec.EmailAtLinkTime,
			"is_primary":      rec.IsPrimary,
			"status":          "ACTIVE",
			"linked_at":       rec.LinkedAt,
			"last_login_at":   rec.LastLoginAt,
		}),
	}).Create(&rec).Error
}

func (r *oidcRepository) CountConnections(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&oauthConnectionModel{}).
		Where("user_id = ?", userID).
		Where("status = 'ACTIVE'").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func (r *oidcRepository) DeleteConnection(ctx context.Context, userID uuid.UUID, provider string) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&oauthConnectionModel{}).
		Where("user_id = ?", userID).
		Where("provider = ?", provider).
		Where("status = 'ACTIVE'").
		Update("status", "REVOKED")
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *oidcRepository) UpsertToken(
	ctx context.Context,
	userID uuid.UUID,
	provider, accessToken, refreshToken string,
	expiresAt *time.Time,
	now time.Time,
) error {
	updates := map[string]any{
		"access_token": accessToken,
		"expires_at":   expiresAt,
		"updated_at":   now,
	}
	if strings.TrimSpace(refreshToken) != "" {
		updates["refresh_token"] = refreshToken
	}

	res := r.db.WithContext(ctx).
		Model(&oauthTokenModel{}).
		Where("user_id = ?", userID).
		Where("provider = ?", provider).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}

	rec := oauthTokenModel{
		UserID:      userID,
		Provider:    provider,
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if strings.TrimSpace(refreshToken) != "" {
		rec.RefreshToken = &refreshToken
	}
	return r.db.WithContext(ctx).Create(&rec).Error
}

func (r *oidcRepository) ListTokensExpiringBefore(ctx context.Context, before time.Time, limit int) ([]ports.OIDCTokenRecord, error) {
	type row struct {
		UserID       uuid.UUID  `gorm:"column:user_id"`
		Provider     string     `gorm:"column:provider"`
		AccessToken  string     `gorm:"column:access_token"`
		RefreshToken *string    `gorm:"column:refresh_token"`
		ExpiresAt    *time.Time `gorm:"column:expires_at"`
	}

	var rows []row
	query := r.db.WithContext(ctx).
		Table("oauth_tokens t").
		Select("t.user_id, t.provider, t.access_token, t.refresh_token, t.expires_at").
		Joins("JOIN oauth_connections c ON c.user_id = t.user_id AND c.provider = t.provider").
		Where("c.status = 'ACTIVE'").
		Where("t.refresh_token IS NOT NULL AND t.refresh_token <> ''").
		Where("t.expires_at IS NULL OR t.expires_at <= ?", before).
		Order("t.expires_at ASC NULLS FIRST, t.updated_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]ports.OIDCTokenRecord, 0, len(rows))
	for _, row := range rows {
		refreshToken := ""
		if row.RefreshToken != nil {
			refreshToken = *row.RefreshToken
		}
		result = append(result, ports.OIDCTokenRecord{
			UserID:       row.UserID,
			Provider:     row.Provider,
			AccessToken:  row.AccessToken,
			RefreshToken: refreshToken,
			ExpiresAt:    row.ExpiresAt,
		})
	}
	return result, nil
}

func (r *oidcRepository) UpdateConnectionStatus(ctx context.Context, userID uuid.UUID, provider, status string, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&oauthConnectionModel{}).
		Where("user_id = ?", userID).
		Where("provider = ?", provider).
		Updates(map[string]any{
			"status":    strings.ToUpper(strings.TrimSpace(status)),
			"linked_at": now,
		}).Error
}
