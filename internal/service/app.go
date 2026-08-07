package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"study-tracker-go/internal/repository"
	postgresrepo "study-tracker-go/internal/repository/postgres"
	"study-tracker-go/pkg/config"
)

var (
	defaultMu    sync.RWMutex
	defaultRepos repository.Repositories
	appConfig    config.Config
	pgPool       *pgxpool.Pool
)

func Init(repos repository.Repositories) error {
	return InitApp(config.Config{}, repos, nil)
}

func InitApp(cfg config.Config, repos repository.Repositories, pool *pgxpool.Pool) error {
	if repos.Auth == nil || repos.Subjects == nil || repos.Errors == nil || repos.Settings == nil || repos.Knowledge == nil || repos.OCRTasks == nil || repos.Backup == nil || repos.Library == nil {
		return fmt.Errorf("repository 初始化不完整")
	}
	if err := validateEmailVerificationConfig(cfg); err != nil {
		return err
	}
	defaultMu.Lock()
	defaultRepos = repos
	appConfig = cfg
	pgPool = pool
	defaultMu.Unlock()
	return nil
}

func repositories(ctx context.Context) (repository.Repositories, error) {
	defaultMu.RLock()
	repos := defaultRepos
	cfg := appConfig
	pool := pgPool
	defaultMu.RUnlock()
	if repos.Auth == nil || repos.Subjects == nil || repos.Errors == nil || repos.Settings == nil || repos.Knowledge == nil || repos.OCRTasks == nil || repos.Backup == nil || repos.Library == nil {
		return repository.Repositories{}, fmt.Errorf("service 尚未初始化")
	}
	if cfg.AuthEnabled {
		userID, ok := UserIDFromContext(ctx)
		if !ok || userID <= 0 {
			return repository.Repositories{}, fmt.Errorf("未登录")
		}
		if pool == nil {
			return repository.Repositories{}, fmt.Errorf("PostgreSQL 连接池未初始化")
		}
		return postgresrepo.NewRepositories(pool, userID), nil
	}
	return repos, nil
}

func background() context.Context {
	return context.Background()
}

func AuthEnabled() bool {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return appConfig.AuthEnabled
}

func currentConfig() config.Config {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return appConfig
}

func authRepository() (repository.AuthRepository, error) {
	defaultMu.RLock()
	repo := defaultRepos.Auth
	defaultMu.RUnlock()
	if repo == nil {
		return nil, fmt.Errorf("认证服务尚未初始化")
	}
	return repo, nil
}

type userIDContextKey struct{}

func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

func UserIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(userIDContextKey{}).(int64)
	return value, ok
}
