package jsonrepo

import (
	"context"
	"fmt"
	"time"

	models "study-tracker-go/internal/model"
)

type AuthRepository struct{}

// CreateUser 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateUser(ctx context.Context, username string, email string, passwordHash string, emailVerified bool) (models.User, error) {
	return models.User{}, fmt.Errorf("JSON 模式不支持登录注册")
}

// DeleteUnverifiedUser 在存储层中删除、清理或撤销相应状态。
func (r *AuthRepository) DeleteUnverifiedUser(ctx context.Context, id int64) error {
	return nil
}

// FindUserByAccount 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error) {
	return models.AuthUser{}, fmt.Errorf("JSON 模式不支持登录注册")
}

// FindUserByID 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindUserByID(ctx context.Context, id int64) (models.AuthUser, error) {
	return models.AuthUser{}, fmt.Errorf("JSON 模式不支持登录注册")
}

// CreateEmailVerificationToken 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateEmailVerificationToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	return fmt.Errorf("JSON 模式不支持登录注册")
}

// ConsumeEmailVerificationToken 在存储层中完成本文件定义的局部处理。
func (r *AuthRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (models.AuthUser, error) {
	return models.AuthUser{}, fmt.Errorf("JSON 模式不支持登录注册")
}

// TouchLastLogin 在存储层中完成本文件定义的局部处理。
func (r *AuthRepository) TouchLastLogin(ctx context.Context, id int64) error {
	return nil
}

// CreateRefreshToken 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, userAgent string, ipAddress string, expiresAt time.Time) error {
	return fmt.Errorf("JSON 模式不支持登录注册")
}

// FindRefreshToken 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenHash string) (int64, time.Time, bool, error) {
	return 0, time.Time{}, true, fmt.Errorf("JSON 模式不支持登录注册")
}

// ConsumeRefreshToken 在存储层中执行当前数据访问或局部处理。
func (r *AuthRepository) ConsumeRefreshToken(ctx context.Context, tokenHash string) (int64, time.Time, error) {
	return 0, time.Time{}, fmt.Errorf("JSON 模式不支持登录注册")
}

// RevokeRefreshToken 在存储层中删除、清理或撤销相应状态。
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	return nil
}
