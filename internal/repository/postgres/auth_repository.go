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

func NewAuthRepository(pool *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{pool: pool}
}

func (r *AuthRepository) CreateUser(ctx context.Context, username string, email string, passwordHash string) (models.User, error) {
	username = strings.TrimSpace(username)
	email = normalizeEmail(email)

	var user models.User
	var nullableEmail pgtype.Text
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, email, password_hash, status)
		VALUES ($1, nullif($2, ''), $3, 'active')
		RETURNING id, username, coalesce(email, ''), status, created_at, updated_at
	`, username, email, passwordHash).Scan(
		&user.ID,
		&user.Username,
		&nullableEmail,
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

func (r *AuthRepository) FindUserByAccount(ctx context.Context, account string) (models.AuthUser, error) {
	account = strings.TrimSpace(account)
	var user models.AuthUser
	var email pgtype.Text
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, status
		FROM users
		WHERE deleted_at IS NULL
		  AND (lower(username) = lower($1) OR lower(email) = lower($1))
		LIMIT 1
	`, account).Scan(&user.ID, &user.Username, &email, &user.PasswordHash, &user.Status)
	if err != nil {
		return models.AuthUser{}, err
	}
	if email.Valid {
		user.Email = email.String
	}
	return user, nil
}

func (r *AuthRepository) FindUserByID(ctx context.Context, id int64) (models.AuthUser, error) {
	var user models.AuthUser
	var email pgtype.Text
	err := r.pool.QueryRow(ctx, `
		SELECT id, username, email, password_hash, status
		FROM users
		WHERE id = $1
		  AND deleted_at IS NULL
		LIMIT 1
	`, id).Scan(&user.ID, &user.Username, &email, &user.PasswordHash, &user.Status)
	if err != nil {
		return models.AuthUser{}, err
	}
	if email.Valid {
		user.Email = email.String
	}
	return user, nil
}

func (r *AuthRepository) TouchLastLogin(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

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

func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
