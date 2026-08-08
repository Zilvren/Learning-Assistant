package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationLockID int64 = 741902338

// ApplyMigrations applies each embedded SQL migration exactly once. Existing
// Docker-initialized databases predate the migration ledger, so their current
// schema is detected and recorded before only newer migrations are executed.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationFS fs.FS) error {
	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return err
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			versions = append(versions, entry.Name())
		}
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return fmt.Errorf("未找到数据库迁移文件")
	}

	if _, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	if err = bootstrapMigrationLedger(ctx, pool); err != nil {
		return err
	}
	for _, version := range versions {
		if err = applyMigration(ctx, pool, migrationFS, version); err != nil {
			return fmt.Errorf("应用迁移 %s 失败: %w", version, err)
		}
	}
	return nil
}

func bootstrapMigrationLedger(ctx context.Context, pool *pgxpool.Pool) error {
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&count); err != nil || count != 0 {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return err
	}
	if err = markExistingMigration(ctx, tx, "001_init_postgres.sql", `SELECT to_regclass('public.users') IS NOT NULL`); err != nil {
		return err
	}
	if err = markExistingMigration(ctx, tx, "002_library.sql", `SELECT to_regclass('public.library_items') IS NOT NULL`); err != nil {
		return err
	}
	if err = markExistingMigration(ctx, tx, "003_reviewable_notes.sql", `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='library_items' AND column_name='review_enabled')`); err != nil {
		return err
	}
	if err = markExistingMigration(ctx, tx, "004_email_verification.sql", `SELECT to_regclass('public.email_verification_tokens') IS NOT NULL`); err != nil {
		return err
	}
	if err = markExistingMigration(ctx, tx, "005_learning_activity.sql", `SELECT to_regclass('public.user_activity_events') IS NOT NULL`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func markExistingMigration(ctx context.Context, tx pgx.Tx, version, query string) error {
	var exists bool
	if err := tx.QueryRow(ctx, query).Scan(&exists); err != nil || !exists {
		return err
	}
	_, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1) ON CONFLICT DO NOTHING`, version)
	return err
}

func applyMigration(ctx context.Context, pool *pgxpool.Pool, migrationFS fs.FS, version string) error {
	sql, err := fs.ReadFile(migrationFS, version)
	if err != nil {
		return err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationLockID); err != nil {
		return err
	}
	var applied bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, version).Scan(&applied); err != nil {
		return err
	}
	if applied {
		return tx.Commit(ctx)
	}
	if _, err = tx.Exec(ctx, unwrapMigrationTransaction(string(sql))); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES($1)`, version); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func unwrapMigrationTransaction(sql string) string {
	lines := strings.Split(strings.TrimSpace(sql), "\n")
	first, last := -1, -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			first = i
			break
		}
	}
	for i := len(lines) - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			last = i
			break
		}
	}
	if first >= 0 && strings.EqualFold(strings.TrimSpace(lines[first]), "BEGIN;") {
		lines[first] = ""
	}
	if last >= 0 && strings.EqualFold(strings.TrimSpace(lines[last]), "COMMIT;") {
		lines[last] = ""
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
