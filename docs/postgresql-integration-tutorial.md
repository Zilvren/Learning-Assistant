# PostgreSQL 接入实现教程

本文档说明如何把当前 Go 版 Study Tracker 从本地 JSON 存储，逐步改造成支持 PostgreSQL 的分层架构。

目标不是一次性把所有代码改完，而是按步骤完成：

```text
1. 新增数据库配置
2. 新增 PostgreSQL 连接池
3. 定义 repository 接口
4. 保留 JSON repository
5. 新增 PostgreSQL repository
6. service 层只依赖接口
7. 通过配置切换 json/postgres
8. 写 JSON 数据导入 PostgreSQL 的迁移工具
```

## 1. 当前代码的问题

当前 service 层直接调用 JSON 工具函数：

```go
store.LoadJSON("errors.json", &errors)
store.SaveJSON("errors.json", errors)
```

这会导致：

```text
service 层知道数据来自 JSON 文件
PostgreSQL 接入时需要改很多业务代码
测试时不好替换数据源
后续多用户、事务、分页、搜索都不好扩展
```

目标结构应该变成：

```text
handler
  -> service
      -> repository interface
          -> json repository
          -> postgres repository
```

service 只关心“我要保存错题”，不关心“保存到 JSON 还是 PostgreSQL”。

## 2. 推荐目录结构

建议最后形成这个结构：

```text
api/handlers/
internal/service/
internal/repository/
  interfaces.go
  json/
    subject_repository.go
    error_repository.go
    settings_repository.go
  postgres/
    db.go
    subject_repository.go
    error_repository.go
    settings_repository.go
internal/model/
pkg/config/
cmd/import-json/
  main.go
```

其中：

```text
internal/repository/interfaces.go     定义接口
internal/repository/json/             JSON 实现
internal/repository/postgres/         PostgreSQL 实现
cmd/import-json/                      JSON 导入 PostgreSQL 工具
```

## 3. 新增数据库配置

修改：

```text
pkg/config/config.go
```

给 `Config` 增加字段：

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

推荐环境变量：

```text
TRACKER_STORAGE=json
TRACKER_DATABASE_URL=postgres://study_tracker_app:password@localhost:5432/study_tracker?sslmode=disable
```

默认值建议：

```go
StorageDriver: envString("TRACKER_STORAGE", "json"),
DatabaseURL:   envString("TRACKER_DATABASE_URL", ""),
```

这样本地双击运行时仍然默认使用 JSON，不会破坏已有用户数据。

配置含义：

```text
TRACKER_STORAGE=json       使用本地 JSON 文件
TRACKER_STORAGE=postgres   使用 PostgreSQL
```

## 4. 新增 PostgreSQL 连接池

推荐使用 `pgx`：

```powershell
go get github.com/jackc/pgx/v5
```

新增文件：

```text
internal/repository/postgres/db.go
```

示例结构：

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

为什么用连接池：

```text
避免每次请求都重新连接数据库
控制最大连接数
提升并发访问性能
方便统一关闭资源
```

## 5. 定义 repository 接口

新增文件：

```text
internal/repository/interfaces.go
```

建议先从核心业务开始，不要一口气抽所有接口。

第一阶段只抽：

```text
SubjectRepository
ErrorRepository
SettingsRepository
```

示例：

```go
package repository

import (
    "context"

    models "study-tracker-go/internal/model"
)

type SubjectRepository interface {
    List(ctx context.Context) ([]string, error)
    Exists(ctx context.Context, name string) (bool, error)
    Create(ctx context.Context, name string) ([]string, error)
    Delete(ctx context.Context, name string) ([]string, error)
}

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

type SettingsRepository interface {
    Load(ctx context.Context) (models.Config, error)
    Save(ctx context.Context, cfg models.Config) error
}

type Repositories struct {
    Subjects SubjectRepository
    Errors   ErrorRepository
    Settings SettingsRepository
}
```

这个接口设计的含义是：

```text
service 调用接口
JSON 实现接口
PostgreSQL 也实现接口
配置决定用哪个实现
```

## 6. 保留 JSON repository

当前的：

```text
internal/repository/json_store.go
```

只是底层 JSON 工具函数，不是真正的业务 repository。

建议把它移动或保留为工具，再新增：

```text
internal/repository/json/subject_repository.go
internal/repository/json/error_repository.go
internal/repository/json/settings_repository.go
```

JSON repository 可以继续复用原来的读写逻辑：

```go
LoadJSON("subjects.json", &subjects)
SaveJSON("subjects.json", subjects)
```

注意：

```text
不要删除 JSON 存储
不要破坏现有 data/*.json
不要让 PostgreSQL 改造影响普通双击用户
```

第一阶段最重要的是保持兼容。

## 7. 新增 PostgreSQL repository

新增目录：

```text
internal/repository/postgres/
```

### SubjectRepository

文件：

```text
internal/repository/postgres/subject_repository.go
```

职责：

```text
查询 subjects 表
新增科目
软删除科目
判断科目是否存在
```

核心 SQL 示例：

```sql
SELECT name
FROM subjects
WHERE user_id = $1
  AND deleted_at IS NULL
ORDER BY sort_order, id;
```

新增科目：

```sql
INSERT INTO subjects (user_id, name, sort_order)
VALUES ($1, $2, $3);
```

软删除科目：

```sql
UPDATE subjects
SET deleted_at = now()
WHERE user_id = $1
  AND name = $2
  AND deleted_at IS NULL;
```

### ErrorRepository

文件：

```text
internal/repository/postgres/error_repository.go
```

职责：

```text
新增错题
查询错题列表
更新错题
删除错题
复习错题
查询标签
```

注意当前 `model.ErrorProblem` 使用的是科目名称：

```go
Subject string `json:"subject"`
```

而数据库中错题表保存的是：

```text
subject_id
```

所以 PostgreSQL repository 需要做转换：

```text
科目名称 -> subjects.id
subjects.id -> 科目名称
```

查询错题时使用 JOIN：

```sql
SELECT
  p.id,
  s.name AS subject,
  p.title,
  p.question,
  p.wrong_answer,
  p.correct_answer,
  p.reason,
  p.review_count,
  p.review_stage,
  p.next_review_at,
  p.last_reviewed_at,
  p.created_at
FROM error_problems p
LEFT JOIN subjects s ON s.user_id = p.user_id AND s.id = p.subject_id
WHERE p.user_id = $1
  AND p.deleted_at IS NULL
ORDER BY p.created_at DESC;
```

标签可以通过：

```text
tags
error_problem_tags
```

两张表维护。

第一版也可以先简化：错题主表先接 PostgreSQL，标签稍后再拆。真正毕业设计里建议拆成多对多表。

## 8. service 层只依赖接口

当前 service 是包级函数：

```go
func CreateError(req models.AddErrorRequest) (models.ErrorProblem, error)
```

后续建议改成结构体：

```go
type ErrorService struct {
    errors   repository.ErrorRepository
    subjects repository.SubjectRepository
}

func NewErrorService(errors repository.ErrorRepository, subjects repository.SubjectRepository) *ErrorService {
    return &ErrorService{
        errors: errors,
        subjects: subjects,
    }
}
```

方法变成：

```go
func (s *ErrorService) Create(ctx context.Context, req models.AddErrorRequest) (models.ErrorProblem, error) {
    exists, err := s.subjects.Exists(ctx, req.Subject)
    if err != nil {
        return models.ErrorProblem{}, err
    }
    if !exists {
        return models.ErrorProblem{}, fmt.Errorf("无效科目")
    }

    return s.errors.Create(ctx, req)
}
```

这样 service 不再知道数据来自 JSON 还是 PostgreSQL。

### 兼容旧 handler 的过渡方案

如果不想一次性大改 handler，可以先保留包级函数，但内部委托给默认 service：

```go
var defaultErrorService *ErrorService

func InitErrorService(s *ErrorService) {
    defaultErrorService = s
}

func CreateError(req models.AddErrorRequest) (models.ErrorProblem, error) {
    return defaultErrorService.Create(context.Background(), req)
}
```

这不是最终最优雅的写法，但迁移成本低。

更长期的写法是：

```text
handler 持有 service
service 持有 repository interface
main.go 负责组装依赖
```

## 9. 通过配置切换 json/postgres

在 `main.go` 中根据配置组装 repository。

伪代码：

```go
cfg := config.Load(os.Args[1:])

var repos repository.Repositories

switch cfg.StorageDriver {
case "json":
    repos = jsonrepo.NewRepositories("data")

case "postgres":
    pool, err := postgres.NewPool(context.Background(), cfg.DatabaseURL)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    repos = postgres.NewRepositories(pool, 1)

default:
    log.Fatalf("未知存储类型: %s", cfg.StorageDriver)
}

services := service.NewServices(repos)
```

这里的 `1` 是早期接入 PostgreSQL 时为了方便演示使用的默认用户 ID。

当前最终实现已经加入登录注册：JSON 模式免登录，PostgreSQL 模式从认证中间件解析当前用户 ID。分阶段练习时，第一阶段仍然可以先固定：

```text
user_id = 1
```

但正式代码中不要长期固定它，应从请求上下文中拿真实登录用户 ID。

## 10. JSON 导入 PostgreSQL 工具

新增命令：

```text
cmd/import-json/main.go
```

运行方式建议：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:password@localhost:5432/study_tracker?sslmode=disable" --user-id 1
```

工具职责：

```text
1. 读取 data/subjects.json
2. 读取 data/errors.json
3. 确保 users 表存在默认用户
4. 导入 subjects
5. 导入 error_problems
6. 导入 tags
7. 导入 error_problem_tags
8. 全过程使用事务
```

推荐参数：

```text
--data-dir       JSON 数据目录，默认 data
--database-url   PostgreSQL 连接字符串
--user-id        导入到哪个用户，默认 1
--dry-run        只检查，不写入
```

导入流程伪代码：

```go
func main() {
    parseFlags()
    loadSubjectsJSON()
    loadErrorsJSON()

    tx := pool.Begin(ctx)
    defer tx.Rollback(ctx)

    ensureDefaultUser(tx, userID)
    importSubjects(tx, userID, subjects)
    importErrors(tx, userID, errors)
    importTags(tx, userID, errors)

    tx.Commit(ctx)
}
```

### 数据映射规则

JSON 科目：

```json
["数学", "英语"]
```

导入到：

```text
subjects.name
subjects.user_id
subjects.sort_order
```

JSON 错题：

```json
{
  "id": 1,
  "subject": "数学",
  "title": "函数单调性",
  "question": "...",
  "wrong": "...",
  "correct": "...",
  "reason": "...",
  "tags": ["函数"],
  "reason_tags": ["概念不清"],
  "created": "2026-06-28 12:00:00",
  "review_count": 0,
  "last_review": null,
  "review_stage": 0,
  "next_review": "2026-06-29"
}
```

导入到：

```text
error_problems.subject_id
error_problems.title
error_problems.question
error_problems.wrong_answer
error_problems.correct_answer
error_problems.reason
error_problems.review_count
error_problems.review_stage
error_problems.next_review_at
error_problems.last_reviewed_at
error_problems.created_at
```

标签导入：

```text
tags.tag_type = 'question'   对应 tags
tags.tag_type = 'reason'     对应 reason_tags
```

错题和标签关系导入：

```text
error_problem_tags.error_problem_id
error_problem_tags.tag_id
```

## 11. 建议迁移顺序

不要一口气迁移所有功能，推荐顺序：

```text
第 1 步：配置 + pgx 连接池
第 2 步：SubjectRepository 接口和 JSON/PostgreSQL 双实现
第 3 步：SubjectService 改成依赖接口
第 4 步：ErrorRepository 接口和 JSON/PostgreSQL 双实现
第 5 步：ErrorService 改成依赖接口
第 6 步：配置切换 json/postgres
第 7 步：JSON 导入 PostgreSQL 工具
第 8 步：settings、OCR、attachments 继续迁移
```

这样每一步都能运行和测试。

## 12. 每一步要验证什么

### 配置验证

```powershell
go test ./pkg/config
```

确认：

```text
TRACKER_STORAGE 默认是 json
TRACKER_DATABASE_URL 可以读取
非法 storage 能报错或回退
```

### 连接池验证

```powershell
go test ./internal/repository/postgres
```

确认：

```text
database_url 为空时报错
数据库不可用时报错
数据库可用时 Ping 成功
```

### JSON repository 验证

```powershell
go test ./internal/repository/json
```

确认：

```text
旧 data/*.json 格式可以正常读取
新增/删除科目不破坏 JSON
错题增删改查行为和原来一致
```

### PostgreSQL repository 验证

```powershell
go test ./internal/repository/postgres
```

确认：

```text
新增科目成功
重复科目失败或被正确处理
新增错题成功
查询错题可以带出科目名称
删除错题使用软删除或符合预期
```

### 整体验证

JSON 模式：

```powershell
$env:TRACKER_STORAGE="json"
go test ./...
go run .
```

PostgreSQL 模式：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:password@localhost:5432/study_tracker?sslmode=disable"
go test ./...
go run .
```

## 13. 面试和毕设可以怎么讲

可以这样描述你的设计：

```text
项目最初是单机 JSON 存储，适合快速启动和离线使用。
后来我把数据访问抽象成 repository interface。
service 层只依赖接口，不依赖具体存储。
底层同时提供 JSON repository 和 PostgreSQL repository。
运行时通过配置选择 storage driver。
这样既保留了本地桌面应用的易用性，又支持后续多用户、云同步和统计分析。
```

技术亮点：

```text
分层架构
依赖倒置
PostgreSQL 连接池
JSON/PostgreSQL 双存储
数据迁移工具
事务导入
多用户数据隔离
Repository 模式
```

## 14. 最容易踩的坑

### 一开始就把所有功能迁移到 PostgreSQL

不要这样做。先迁移科目和错题，确保主流程跑通。

### service 层继续调用 LoadJSON

这说明抽象没有完成。service 应该调用接口，而不是调用 JSON 工具。

### 删除 JSON 模式

不要删。JSON 模式是桌面应用的优势，也方便调试和用户离线使用。

### 直接用 postgres 超级用户跑项目

开发时可以，长期不推荐。应该创建 `study_tracker_app` 这类专用用户。

### 忘记事务

JSON 导入 PostgreSQL 时必须使用事务。否则导入一半失败，会留下脏数据。

### 忘记 user_id

PostgreSQL 设计是多用户模式。所有业务表写入时都要带 `user_id`。

早期练习时可以先固定 `user_id = 1`，当前最终实现应从登录态解析真实用户 ID。

## 15. 推荐最终验收标准

完成接入后，至少满足：

```text
默认不设置环境变量时，应用继续使用 JSON
设置 TRACKER_STORAGE=postgres 后，应用使用 PostgreSQL
科目新增/删除正常
错题新增/查询/更新/删除/复习正常
JSON 导入工具可以把现有 data 数据导入 PostgreSQL
go test ./... 通过
打包后的 exe 在 JSON 模式下仍可直接双击使用
```

这套改造完成后，你的项目就不只是“会用数据库”，而是能体现比较完整的 Go 后端工程能力。
