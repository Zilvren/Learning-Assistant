package service

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"

	"study-tracker-go/internal/repository"
	postgresrepo "study-tracker-go/internal/repository/postgres"
	"study-tracker-go/pkg/config"
)

// App owns the process dependencies used by services. It is immutable after
// construction, so requests can safely share it without package-level mutable
// state. PostgreSQL repositories are still created per request user.
type App struct {
	config config.Config
	repos  repository.Repositories
	pool   *pgxpool.Pool
}

// NewApp 校验 Repository 是否齐全和功能配置是否自洽，再创建一次启动周期内保持不变的应用依赖容器。
func NewApp(cfg config.Config, repos repository.Repositories, pool *pgxpool.Pool) (*App, error) {
	if !completeRepositories(repos) {
		return nil, fmt.Errorf("repository 初始化不完整")
	}
	if err := validateEmailVerificationConfig(cfg); err != nil {
		return nil, err
	}
	return &App{config: cfg, repos: repos, pool: pool}, nil
}

// Config 返回 App 的配置副本，供路由和 Service 读取运行模式，而不暴露可变内部状态。
func (a *App) Config() config.Config {
	if a == nil {
		return config.Config{}
	}
	return a.config
}

// AuthEnabled 表示当前是否为启用 Cookie/JWT 与用户隔离的 PostgreSQL 登录模式。
func (a *App) AuthEnabled() bool {
	return a != nil && a.config.AuthEnabled
}

// repositories 在 JSON 模式返回共享仓储，在登录模式依据请求 context 中的 userID 创建隔离的 PostgreSQL 仓储。
func (a *App) repositories(ctx context.Context) (repository.Repositories, error) {
	if a == nil || !completeRepositories(a.repos) {
		return repository.Repositories{}, fmt.Errorf("service 尚未初始化")
	}
	if a.config.AuthEnabled {
		userID, ok := UserIDFromContext(ctx)
		if !ok || userID <= 0 {
			return repository.Repositories{}, fmt.Errorf("未登录")
		}
		if a.pool == nil {
			return repository.Repositories{}, fmt.Errorf("PostgreSQL 连接池未初始化")
		}
		return postgresrepo.NewRepositories(a.pool, userID), nil
	}
	return a.repos, nil
}

// authRepository 返回跨用户查询所需的认证仓储；它不携带某个业务用户的范围。
func (a *App) authRepository() (repository.AuthRepository, error) {
	if a == nil || a.repos.Auth == nil {
		return nil, fmt.Errorf("认证服务尚未初始化")
	}
	return a.repos.Auth, nil
}

// completeRepositories 防止应用在缺少某一业务仓储的半初始化状态下开始处理请求。
func completeRepositories(repos repository.Repositories) bool {
	return repos.Auth != nil && repos.Subjects != nil && repos.Errors != nil && repos.Settings != nil &&
		repos.Knowledge != nil && repos.OCRTasks != nil && repos.Backup != nil && repos.Library != nil
}

type appContextKey struct{}

// ContextWithApp 把本次请求要使用的 App 放入标准 context，供 Handler 之后的 Service 读取。
func ContextWithApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appContextKey{}, app)
}

// AppFromContext 从请求 context 取出 App，并区分“未注入”与“App 为空”。
func AppFromContext(ctx context.Context) (*App, bool) {
	app, ok := ctx.Value(appContextKey{}).(*App)
	return app, ok && app != nil
}

// legacyApp only exists for package-level compatibility in desktop helpers and
// older unit tests. HTTP requests use ContextWithApp instead.
var legacyApp atomic.Pointer[App]

// Init 是旧桌面辅助与测试的兼容入口；新的 HTTP 启动路径应使用 NewApp 和 RequestContext。
func Init(repos repository.Repositories) error {
	return InitApp(config.Config{}, repos, nil)
}

// InitApp 为旧调用方创建并保存 legacy App；它不参与正常 HTTP 请求的依赖传递。
func InitApp(cfg config.Config, repos repository.Repositories, pool *pgxpool.Pool) error {
	app, err := NewApp(cfg, repos, pool)
	if err != nil {
		return err
	}
	legacyApp.Store(app)
	return nil
}

// DefaultApp 返回兼容层保存的 App，仅供无请求 context 的旧辅助逻辑和测试使用。
func DefaultApp() *App {
	return legacyApp.Load()
}

// appFor 优先使用请求注入的 App，只有非 HTTP 兼容场景才回退到 legacy App。
func appFor(ctx context.Context) (*App, error) {
	if app, ok := AppFromContext(ctx); ok {
		return app, nil
	}
	if app := DefaultApp(); app != nil {
		return app, nil
	}
	return nil, fmt.Errorf("service 尚未初始化")
}

// repositories 是 Service 使用的统一仓储入口，负责把当前 context 转交给对应 App。
func repositories(ctx context.Context) (repository.Repositories, error) {
	app, err := appFor(ctx)
	if err != nil {
		return repository.Repositories{}, err
	}
	return app.repositories(ctx)
}

// currentConfig 从当前 App 读取运行配置，避免 Service 直接依赖包级配置变量。
func currentConfig(ctx context.Context) (config.Config, error) {
	app, err := appFor(ctx)
	if err != nil {
		return config.Config{}, err
	}
	return app.config, nil
}

// authRepository 是 Service 访问认证数据的统一入口，并在 App 未初始化时返回明确错误。
func authRepository(ctx context.Context) (repository.AuthRepository, error) {
	app, err := appFor(ctx)
	if err != nil {
		return nil, err
	}
	return app.authRepository()
}

// background 为没有 HTTP 请求的后台兼容流程创建 context，并尽可能附带 legacy App。
func background() context.Context {
	ctx := context.Background()
	if app := DefaultApp(); app != nil {
		return ContextWithApp(ctx, app)
	}
	return ctx
}

type userIDContextKey struct{}

// ContextWithUserID 在认证完成后把当前用户 ID 写入请求 context，供 PostgreSQL Repository 隔离数据。
func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserIDFromContext 读取认证中间件写入的用户 ID。
func UserIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(userIDContextKey{}).(int64)
	return value, ok
}
