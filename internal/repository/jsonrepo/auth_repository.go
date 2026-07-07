package jsonrepo

import (
	"context"
	"fmt"
	"time"

	models "study-tracker-go/internal/model"
)

type AuthRepository struct{}

func (r *AuthRepository) CreateUser(ctx context.Context, username string, email string, passwordHash string) (models.User, error) {
	return models.User{}, fmt.Errorf("JSON 模式不支持登录注册")
}

func (r *AuthRepository) FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error) {
	return models.AuthUser{}, fmt.Errorf("JSON 模式不支持登录注册")
}

func (r *AuthRepository) FindUserByID(ctx context.Context, id int64) (models.AuthUser, error) {
	return models.AuthUser{}, fmt.Errorf("JSON 模式不支持登录注册")
}

func (r *AuthRepository) TouchLastLogin(ctx context.Context, id int64) error {
	return nil
}

func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, userAgent string, ipAddress string, expiresAt time.Time) error {
	return fmt.Errorf("JSON 模式不支持登录注册")
}

func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenHash string) (int64, time.Time, bool, error) {
	return 0, time.Time{}, true, fmt.Errorf("JSON 模式不支持登录注册")
}

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}
