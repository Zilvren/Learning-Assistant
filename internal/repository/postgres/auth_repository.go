package postgres

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	models "study-tracker-go/internal/model"
)

type AuthRepository struct {
	pool *pgxpool.Pool
}

// NewAuthRepository 在存储层中创建所需对象并完成初始化。
func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

// CreateUser 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateUser(ctx context.Context, username string, email string, passwordHash string, emailVerified bool) (models.User, error) {
	username = strings.TrimSpace(username)
	email = normalizeEmail(email)

	var user models.User
	var nullableEmail pgtype.Text
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, status, email_verified_at)
		VALUES ($1, nullif($2, ''), $3, 'active', CASE WHEN $4 THEN now() ELSE NULL END)
		RETURNING id, username, coalesce(email, ''), email_verified_at IS NOT NULL, status, created_at, updated_at
	`, username, email, passwordHash, emailVerified).Scan(
		&user.ID,
		&user.Username,
		&nullableEmail,
		&user.EmailVerified,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return models.User{}, err
	}
	user.Email = nullableEmail.String
	return user, nil
}

// DeleteUnverifiedUser 在存储层中删除、清理或撤销相应状态。
func (r *AuthRepository) DeleteUnverifiedUser(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1 AND email_verified_at IS NULL`, id)
	return err
}

// FindUserByAccount 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error) {
	account = strings.TrimSpace(account)
	var user models.AuthUser
	var email pgtype.Text
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, email_verified_at IS NOT NULL, password_hash, status
		FROM users
		WHERE deleted_at IS NULL
		  AND (lower(username) = lower($1) OR lower(email) = lower($1))
		LIMIT 1
	`, account).Scan(&user.ID, &user.Username, &email, &user.EmailVerified, &user.PasswordHash, &user.Status)
	if err != nil {
		return models.AuthUser{}, err
	}
	if email.Valid {
		user.Email = email.String
	}
	return user, nil
}

// FindUserByID 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindUserByID(ctx context.Context, id int64) (models.AuthUser, error) {
	var user models.AuthUser
	var email pgtype.Text
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, email_verified_at IS NOT NULL, password_hash, status
		FROM users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`, id).Scan(&user.ID, &user.Username, &email, &user.EmailVerified, &user.PasswordHash, &user.Status)
	if err != nil {
		return models.AuthUser{}, err
	}
	if email.Valid {
		user.Email = email.String
	}
	return user, nil
}

// CreateEmailVerificationToken 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateEmailVerificationToken(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	if _, err := r.pool.Exec(ctx, `
		DELETE FROM email_verification_tokens
		WHERE user_id = $1 AND consumed_at IS NULL
	`, userID); err != nil {
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// ConsumeEmailVerificationToken 在存储层中完成本文件定义的局部处理。
func (r *AuthRepository) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (models.AuthUser, error) {
	var user models.AuthUser
	var email pgtype.Text
	err := r.pool.QueryRow(ctx, `
		WITH verified_token AS (
			UPDATE email_verification_tokens
			SET consumed_at = now()
			WHERE token_hash = $1
			  AND consumed_at IS NULL
			  AND expires_at > now()
			RETURNING user_id
		)
		UPDATE users AS u
		SET email_verified_at = COALESCE(email_verified_at, now())
		FROM verified_token
		WHERE u.id = verified_token.user_id
		  AND u.deleted_at IS NULL
		RETURNING u.id, u.username, u.email, u.email_verified_at IS NOT NULL, u.password_hash, u.status
	`, tokenHash).Scan(&user.ID, &user.Username, &email, &user.EmailVerified, &user.PasswordHash, &user.Status)
	if err != nil {
		return models.AuthUser{}, err
	}
	if email.Valid {
		user.Email = email.String
	}
	return user, nil
}

// TouchLastLogin 在存储层中完成本文件定义的局部处理。
func (r *AuthRepository) TouchLastLogin(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

// CreateRefreshToken 在存储层中创建或更新相应状态。
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, userID int64, tokenHash string, userAgent string, ipAddress string, expiresAt time.Time) error {
	var ip interface{}
	if parsed := net.ParseIP(strings.TrimSpace(ipAddress)); parsed != nil {
		ip = parsed.String()
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, tokenHash, userAgent, ip, expiresAt)
	return err
}

// FindRefreshToken 在存储层中读取并整理所需数据。
func (r *AuthRepository) FindRefreshToken(ctx context.Context, tokenHash string) (int64, time.Time, bool, error) {
	var userID int64
	var expiresAt time.Time
	var revokedAt pgtype.Timestamptz
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, expires_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
		LIMIT 1
	`, tokenHash).Scan(&userID, &expiresAt, &revokedAt)
	if err != nil {
		return 0, time.Time{}, true, err
	}
	return userID, expiresAt, revokedAt.Valid, nil
}

// RevokeRefreshToken 在存储层中删除、清理或撤销相应状态。
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)
	return err
}

// normalizeEmail 在存储层中构造、编码或标准化数据。
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
