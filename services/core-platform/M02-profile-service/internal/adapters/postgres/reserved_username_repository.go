package postgres

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

type reservedUsernameRepository struct {
	db *gorm.DB
}

func (r *reservedUsernameRepository) IsReserved(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&reservedUsernameModel{}).
		Where("username = ?", strings.ToLower(strings.TrimSpace(username))).
		Count(&count).Error
	return count > 0, err
}
