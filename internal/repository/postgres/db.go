package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	base "study-tracker-go/internal/repository"
	"study-tracker-go/migrations"
)

type Store struct {
	pool   *pgxpool.Pool
	userID int64
}

// NewPool 在存储层中创建所需对象并完成初始化。
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("TRACKER_DATABASE_URL 不能为空")
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}

	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := ApplyMigrations(ctx, pool, migrations.FS); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// EnsureLocalUser 在存储层中完成本文件定义的局部处理。
func EnsureLocalUser(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	return EnsureUser(ctx, pool, "local")
}

// EnsureUser 在存储层中完成本文件定义的局部处理。
func EnsureUser(ctx context.Context, pool *pgxpool.Pool, username string) (int64, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		username = "local"
	}
	var id int64
	err := pool.QueryRow(ctx, `
		SELECT id
		FROM users
		WHERE lower(username) = lower($1)
		  AND deleted_at IS NULL
		LIMIT 1
	`, username).Scan(&id)
	if err == nil {
		return id, nil
	}

	err = pool.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, status)
		VALUES ($1, $2, 'active')
		RETURNING id
	`, username, "import-only").Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// NewRepositories 在存储层中创建所需对象并完成初始化。
func NewRepositories(pool *pgxpool.Pool, userID int64) base.Repositories {
	store := &Store{pool: pool, userID: userID}
	return base.Repositories{
		Auth:      NewAuthRepository(pool),
		Subjects:  &SubjectRepository{store: store},
		Errors:    &ErrorRepository{store: store},
		Settings:  &SettingsRepository{store: store},
		Knowledge: &KnowledgeRepository{store: store},
		OCRTasks:  &OCRTaskRepository{store: store},
		Activity:  &ActivityRepository{store: store},
		Relations: &LearningRelationRepository{store: store},
		Backup:    &BackupRepository{store: store},
		Library:   &LibraryRepository{store: store},
	}
}

// UserID 在存储层中完成本文件定义的局部处理。
func (s *Store) UserID() int64 {
	return s.userID
}
