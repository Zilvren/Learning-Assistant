# PostgreSQL 接入小白版教程

这份文档是给第一次做 Go + PostgreSQL 分层改造的人看的。

你不需要一次性理解所有架构名词。按本文一步一步做，每完成一小步就运行一次项目，确认没有坏，再进入下一步。

## 0. 你现在要做的到底是什么

当前项目的数据大概是这样保存的：

```text
前端点击新增错题
  -> Go handler
  -> service.CreateError
  -> LoadJSON("errors.json")
  -> SaveJSON("errors.json")
  -> data/errors.json
```

现在要改成支持两种模式：

```text
JSON 模式：
前端 -> handler -> service -> repository interface -> JSON repository -> data/*.json

PostgreSQL 模式：
前端 -> handler -> service -> repository interface -> PostgreSQL repository -> PostgreSQL 数据库
```

重点是：

```text
service 不再直接读写 JSON
service 只调用接口
接口下面可以接 JSON，也可以接 PostgreSQL
```

## 1. 先不要一口气全改

小白最容易犯的错误是：想一次性把所有功能都迁移到 PostgreSQL。

不要这样做。

推荐顺序：

```text
第 1 阶段：只接 subjects 科目
第 2 阶段：再接 error_problems 错题
第 3 阶段：再接 settings 设置
第 4 阶段：最后接 OCR、附件、更新等边缘功能
```

为什么先做科目？

```text
subjects 最简单
只有字符串列表
没有复杂标签
没有复习算法
最适合练习 repository 接口
```

## 2. 开始前先做三件事

### 2.1 确认项目能跑

在项目根目录执行：

```powershell
go test ./...
go run .
```

如果这里本来就失败，先不要做 PostgreSQL 改造。

### 2.2 备份 data 目录

当前 JSON 数据在：

```text
C:\Users\Knock\Desktop\gotest\server-go\data
```

手动复制一份，例如：

```text
data-backup-before-postgres
```

### 2.3 确认 PostgreSQL 数据库已经建好

检查表：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "\dt"
```

能看到表之后再继续。

## 3. 第一步：新增数据库配置

目标：让项目知道自己应该使用 JSON 还是 PostgreSQL。

修改文件：

```text
pkg/config/config.go
```

找到：

```go
type Config struct {
    Host        string
    Port        int
    NoBrowser   bool
    GinMode     string
    FrontendDir string
}
```

改成：

```go
type Config struct {
    Host        string
    Port        int
    NoBrowser   bool
    GinMode     string
    FrontendDir string

    StorageDriver string
    DatabaseURL   string
}
```

然后在 `Load` 函数里的默认配置中加：

```go
StorageDriver: envString("TRACKER_STORAGE", "json"),
DatabaseURL:   envString("TRACKER_DATABASE_URL", ""),
```

也就是让配置支持这两个环境变量：

```text
TRACKER_STORAGE
TRACKER_DATABASE_URL
```

含义：

```text
TRACKER_STORAGE=json       使用本地 JSON
TRACKER_STORAGE=postgres   使用 PostgreSQL
```

本地默认值必须是 `json`，这样普通用户双击 exe 时不会被数据库影响。

### 做完这步怎么检查

运行：

```powershell
go test ./pkg/config
go test ./...
```

如果通过，说明配置没有写坏。

## 4. 第二步：安装 PostgreSQL 驱动

Go 连接 PostgreSQL 推荐用 `pgx`。

运行：

```powershell
go get github.com/jackc/pgx/v5
```

如果后面要用连接池，代码里会 import：

```go
github.com/jackc/pgx/v5/pgxpool
```

### 做完这步怎么检查

运行：

```powershell
go mod tidy
go test ./...
```

如果 `go.mod` 和 `go.sum` 更新了，是正常的。

## 5. 第三步：新增 PostgreSQL 连接池

目标：程序启动时连接数据库，并复用连接。

新增目录：

```text
internal/repository/postgres
```

新增文件：

```text
internal/repository/postgres/db.go
```

写入：

```go
package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
    if databaseURL == "" {
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

    return pool, nil
}
```

这段代码做了什么：

```text
检查 databaseURL 是否为空
解析 PostgreSQL 连接字符串
创建连接池
Ping 数据库，确认真的连得上
失败时关闭连接池
```

### 做完这步怎么检查

运行：

```powershell
go test ./...
```

现在只是新增文件，还没有真正用它。

## 6. 第四步：先定义 subjects 的 repository 接口

目标：让 service 不再关心科目从哪里来。

新增文件：

```text
internal/repository/interfaces.go
```

先只写科目接口：

```go
package repository

import "context"

type SubjectRepository interface {
    List(ctx context.Context) ([]string, error)
    Exists(ctx context.Context, name string) (bool, error)
    Create(ctx context.Context, name string) ([]string, error)
    Delete(ctx context.Context, name string) ([]string, error)
}
```

这是什么意思？

```text
List     获取所有科目
Exists   判断科目是否存在
Create   新增科目
Delete   删除科目
```

service 以后只调用这些方法，不直接调用 `LoadJSON`。

### 为什么要有 context.Context

你先简单理解成：

```text
context.Context 用来控制请求生命周期
以后数据库查询超时、取消请求、传用户信息都会用到它
```

Go 后端项目里 repository 方法通常都会带 `ctx context.Context`。

## 7. 第五步：写 JSON 版 SubjectRepository

目标：先让旧 JSON 模式继续正常工作。

新增目录：

```text
internal/repository/json
```

新增文件：

```text
internal/repository/json/subject_repository.go
```

示例结构：

```go
package json

import (
    "context"
    "fmt"
    "strings"

    base "study-tracker-go/internal/repository"
)

type SubjectRepository struct{}

func NewSubjectRepository() *SubjectRepository {
    return &SubjectRepository{}
}

func (r *SubjectRepository) List(ctx context.Context) ([]string, error) {
    var subjects []string
    if err := base.LoadJSON("subjects.json", &subjects); err != nil {
        return nil, err
    }
    if subjects == nil {
        subjects = []string{}
    }
    return subjects, nil
}

func (r *SubjectRepository) Exists(ctx context.Context, name string) (bool, error) {
    subjects, err := r.List(ctx)
    if err != nil {
        return false, err
    }
    for _, s := range subjects {
        if s == name {
            return true, nil
        }
    }
    return false, nil
}

func (r *SubjectRepository) Create(ctx context.Context, name string) ([]string, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return nil, fmt.Errorf("科目名称不能为空")
    }

    subjects, err := r.List(ctx)
    if err != nil {
        return nil, err
    }

    for _, s := range subjects {
        if s == name {
            return nil, fmt.Errorf("科目已存在")
        }
    }

    subjects = append(subjects, name)
    if err := base.SaveJSON("subjects.json", subjects); err != nil {
        return nil, err
    }
    return subjects, nil
}

func (r *SubjectRepository) Delete(ctx context.Context, name string) ([]string, error) {
    subjects, err := r.List(ctx)
    if err != nil {
        return nil, err
    }

    found := false
    remaining := []string{}
    for _, s := range subjects {
        if s == name {
            found = true
            continue
        }
        remaining = append(remaining, s)
    }

    if !found {
        return nil, fmt.Errorf("科目不存在")
    }

    if err := base.SaveJSON("subjects.json", remaining); err != nil {
        return nil, err
    }
    return remaining, nil
}
```

注意：

```text
这里仍然使用 data/subjects.json
只是把 JSON 读写包装成了接口实现
```

## 8. 第六步：写 PostgreSQL 版 SubjectRepository

新增文件：

```text
internal/repository/postgres/subject_repository.go
```

示例结构：

```go
package postgres

import (
    "context"
    "fmt"
    "strings"

    "github.com/jackc/pgx/v5/pgxpool"
)

type SubjectRepository struct {
    pool   *pgxpool.Pool
    userID int64
}

func NewSubjectRepository(pool *pgxpool.Pool, userID int64) *SubjectRepository {
    return &SubjectRepository{
        pool:   pool,
        userID: userID,
    }
}

func (r *SubjectRepository) List(ctx context.Context) ([]string, error) {
    rows, err := r.pool.Query(ctx, `
        SELECT name
        FROM subjects
        WHERE user_id = $1
          AND deleted_at IS NULL
        ORDER BY sort_order, id
    `, r.userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    subjects := []string{}
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            return nil, err
        }
        subjects = append(subjects, name)
    }
    return subjects, rows.Err()
}

func (r *SubjectRepository) Exists(ctx context.Context, name string) (bool, error) {
    var exists bool
    err := r.pool.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1
            FROM subjects
            WHERE user_id = $1
              AND name = $2
              AND deleted_at IS NULL
        )
    `, r.userID, name).Scan(&exists)
    return exists, err
}

func (r *SubjectRepository) Create(ctx context.Context, name string) ([]string, error) {
    name = strings.TrimSpace(name)
    if name == "" {
        return nil, fmt.Errorf("科目名称不能为空")
    }

    _, err := r.pool.Exec(ctx, `
        INSERT INTO subjects (user_id, name, sort_order)
        VALUES ($1, $2, 0)
    `, r.userID, name)
    if err != nil {
        return nil, err
    }

    return r.List(ctx)
}

func (r *SubjectRepository) Delete(ctx context.Context, name string) ([]string, error) {
    tag, err := r.pool.Exec(ctx, `
        UPDATE subjects
        SET deleted_at = now()
        WHERE user_id = $1
          AND name = $2
          AND deleted_at IS NULL
    `, r.userID, name)
    if err != nil {
        return nil, err
    }
    if tag.RowsAffected() == 0 {
        return nil, fmt.Errorf("科目不存在")
    }

    return r.List(ctx)
}
```

这一段是 PostgreSQL 接入早期的简化写法。当前最终实现已经加入登录注册：JSON 模式免登录，PostgreSQL 模式从认证中间件解析当前用户 ID。

如果你只是跟着教程分阶段练习，第一阶段可以临时统一使用：

```text
user_id = 1
```

真正项目代码中不要长期固定 `user_id = 1`，应从请求上下文读取登录用户 ID。

## 9. 第七步：改 SubjectService

当前：

```text
internal/service/subject_service.go
```

里面直接调用：

```go
store.LoadJSON(...)
store.SaveJSON(...)
```

目标是改成：

```go
type SubjectService struct {
    repo repository.SubjectRepository
}
```

示例：

```go
package service

import (
    "context"

    "study-tracker-go/internal/repository"
)

type SubjectService struct {
    repo repository.SubjectRepository
}

func NewSubjectService(repo repository.SubjectRepository) *SubjectService {
    return &SubjectService{repo: repo}
}

func (s *SubjectService) GetAllSubjects(ctx context.Context) ([]string, error) {
    return s.repo.List(ctx)
}

func (s *SubjectService) AddSubject(ctx context.Context, name string) ([]string, error) {
    return s.repo.Create(ctx, name)
}

func (s *SubjectService) SubjectExists(ctx context.Context, name string) (bool, error) {
    return s.repo.Exists(ctx, name)
}

func (s *SubjectService) DeleteSubject(ctx context.Context, name string) ([]string, error) {
    return s.repo.Delete(ctx, name)
}
```

### 小白过渡写法

如果你不想马上改所有 handler，可以先保留旧函数名。

例如：

```go
var defaultSubjectService *SubjectService

func InitSubjectService(s *SubjectService) {
    defaultSubjectService = s
}

func GetAllSubjects() ([]string, error) {
    return defaultSubjectService.GetAllSubjects(context.Background())
}

func AddSubject(name string) ([]string, error) {
    return defaultSubjectService.AddSubject(context.Background(), name)
}

func SubjectExists(name string) bool {
    ok, err := defaultSubjectService.SubjectExists(context.Background(), name)
    return err == nil && ok
}

func DeleteSubject(name string) ([]string, error) {
    return defaultSubjectService.DeleteSubject(context.Background(), name)
}
```

这样 handler 暂时不用大改。

做完这步，虽然代码结构变了，但 JSON 模式应该还和以前一样。

### 做完这步怎么检查

```powershell
go test ./...
go run .
```

然后在页面里测试：

```text
新增科目
删除科目
刷新页面
确认 data/subjects.json 正常变化
```

## 10. 第八步：在 main.go 里根据配置选择 repository

目标：

```text
TRACKER_STORAGE=json       使用 JSON repository
TRACKER_STORAGE=postgres   使用 PostgreSQL repository
```

伪代码：

```go
cfg := config.Load(os.Args[1:])

switch cfg.StorageDriver {
case "json":
    subjectRepo := jsonrepo.NewSubjectRepository()
    service.InitSubjectService(service.NewSubjectService(subjectRepo))

case "postgres":
    pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    subjectRepo := postgres.NewSubjectRepository(pool, 1)
    service.InitSubjectService(service.NewSubjectService(subjectRepo))

default:
    log.Fatalf("未知存储类型: %s", cfg.StorageDriver)
}
```

注意 import 命名可能需要别名：

```go
jsonrepo "study-tracker-go/internal/repository/json"
postgresrepo "study-tracker-go/internal/repository/postgres"
```

因为 Go 标准库也有 `encoding/json`，所以业务包叫 `jsonrepo` 更清楚。

## 11. 第九步：测试 JSON 模式

清空当前 PowerShell 里的 PostgreSQL 配置：

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
Remove-Item Env:TRACKER_DATABASE_URL -ErrorAction SilentlyContinue
```

运行：

```powershell
go run .
```

因为默认是：

```text
TRACKER_STORAGE=json
```

所以应该继续使用本地 JSON。

测试：

```text
新增科目
刷新页面
关闭程序
重新打开
科目还在
```

## 12. 第十步：测试 PostgreSQL 模式

设置环境变量：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
```

运行：

```powershell
go run .
```

测试：

```text
新增科目
刷新页面
确认页面能看到新科目
```

然后用 SQL 查数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -c "SELECT id, user_id, name, deleted_at FROM subjects ORDER BY id;"
```

如果数据库里出现新科目，说明 SubjectRepository 的 PostgreSQL 模式成功了。

## 13. 第十一步：再迁移错题 ErrorRepository

科目跑通后，再做错题。

先扩展：

```text
internal/repository/interfaces.go
```

增加：

```go
type ErrorFilter struct {
    Subject   string
    Keyword   string
    Tag       string
    ReasonTag string
}

type ErrorRepository interface {
    Create(ctx context.Context, req models.AddErrorRequest) (models.ErrorProblem, error)
    List(ctx context.Context, filter ErrorFilter) ([]models.ErrorProblem, error)
    Update(ctx context.Context, id int, req models.UpdateErrorRequest) error
    Delete(ctx context.Context, id int) error
    Review(ctx context.Context, id int) (models.ErrorProblem, error)
    ListTags(ctx context.Context) ([]string, error)
}
```

错题比科目复杂，因为：

```text
JSON 里 subject 是科目名称
PostgreSQL 里 error_problems 存 subject_id
JSON 里 tags 是字符串数组
PostgreSQL 里 tags 是独立表
```

所以 PostgreSQL 版 ErrorRepository 要做转换：

```text
新增错题时：
subject name -> 查 subjects.id -> 插入 error_problems

查询错题时：
error_problems.subject_id -> JOIN subjects -> 返回 subject name
```

第一版可以先做：

```text
Create
List
Update
Delete
Review
```

标签可以先保存在 `metadata` 里，等主流程稳定后再拆到 `tags` 和 `error_problem_tags`。

毕业设计最终版建议拆成独立标签表。

## 14. 第十二步：写 JSON 导入 PostgreSQL 工具

当 PostgreSQL 模式跑通后，再写导入工具。

新增目录：

```text
cmd/import-json
```

新增文件：

```text
cmd/import-json/main.go
```

运行方式：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --user-id 1
```

这个工具要做：

```text
读取 data/subjects.json
读取 data/errors.json
连接 PostgreSQL
开启事务
确保 users 表有默认用户
导入 subjects
导入 error_problems
导入 tags
提交事务
```

### 为什么要事务

事务的作用：

```text
全部成功 -> 提交
中间失败 -> 回滚
```

否则可能出现：

```text
科目导入成功
错题导入一半失败
数据库里留下半截脏数据
```

导入工具里应该有：

```go
tx, err := pool.Begin(ctx)
if err != nil {
    return err
}
defer tx.Rollback(ctx)

// import...

return tx.Commit(ctx)
```

## 15. 每一步完成后都要做的检查

每完成一小步，至少运行：

```powershell
go test ./...
```

如果改了 repository：

```powershell
go run .
```

如果改了前端相关接口：

```powershell
npm run build
```

如果只改后端，不一定每次都要构建前端。

## 16. 你应该怎么判断自己没有改坏

JSON 模式必须满足：

```text
不设置任何环境变量
go run .
页面能打开
原来的 data 数据还在
新增科目正常
新增错题正常
```

PostgreSQL 模式必须满足：

```text
设置 TRACKER_STORAGE=postgres
设置 TRACKER_DATABASE_URL
go run .
页面能打开
新增科目会写入 subjects 表
新增错题会写入 error_problems 表
```

导入工具必须满足：

```text
导入前先备份 data
导入失败不会留下半截数据
导入后 subjects 数量正确
导入后 error_problems 数量正确
```

## 17. 推荐你真正动手的顺序

照这个顺序做，最稳：

```text
1. 改 pkg/config/config.go，加 TRACKER_STORAGE 和 TRACKER_DATABASE_URL
2. go test ./...
3. go get github.com/jackc/pgx/v5
4. 新增 internal/repository/postgres/db.go
5. go test ./...
6. 新增 internal/repository/interfaces.go，只写 SubjectRepository
7. 新增 internal/repository/json/subject_repository.go
8. 改 internal/service/subject_service.go
9. main.go 里先接 JSON repository
10. go test ./...
11. go run .，确认 JSON 模式科目正常
12. 新增 internal/repository/postgres/subject_repository.go
13. main.go 里接 postgres repository
14. 设置 TRACKER_STORAGE=postgres 后 go run .
15. 确认科目写入 PostgreSQL
16. 再开始做 ErrorRepository
17. 最后做 cmd/import-json
```

## 18. 小白重点理解

你现在最应该理解这句话：

```text
接口不是为了炫技，而是为了让 service 不关心数据到底存在哪里。
```

以前：

```text
service -> JSON 文件
```

现在：

```text
service -> repository 接口 -> JSON 或 PostgreSQL
```

这就是这次改造的核心。
