package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureEmailVerificationSchema is intentionally idempotent so existing
// production databases can accept the auth update during a normal rollout.
func EnsureEmailVerificationSchema(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ`,
		`CREATE TABLE IF NOT EXISTS email_verification_tokens (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(128) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			consumed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_verification_tokens_user_active
			ON email_verification_tokens(user_id, created_at DESC)
			WHERE consumed_at IS NULL`,
		`UPDATE users
		SET email_verified_at = COALESCE(last_login_at, created_at)
		WHERE email IS NOT NULL AND email_verified_at IS NULL`,
	}
	for _, query := range queries {
		if _, err := pool.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}
