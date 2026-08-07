# PostgreSQL Repository 开发文档

本文档记录 Study Tracker Go 版的 PostgreSQL repository 实现方式。当前应用支持两种存储模式：

```text
TRACKER_STORAGE=json       默认，本地 JSON 单机模式
TRACKER_STORAGE=postgres   PostgreSQL 模式
```

## 1. 为什么用 pgx

本项目使用 `github.com/jackc/pgx/v5` 连接 PostgreSQL。

选择原因：

- `pgx` 是 Go 生态里 PostgreSQL 支持最完整的主流驱动。
- `pgxpool` 自带连接池，适合 Web 后端长期运行。
- 原生支持 PostgreSQL 特性，例如 `RETURNING`、`jsonb`、事务、批量查询。
- 不需要先绕一层 `database/sql`，类型能力和错误信息更贴近 PostgreSQL。

## 2. 存储模式配置

默认不设置任何环境变量时：

```powershell
go run .
```

应用使用 JSON 模式，继续读写：

```text
data/errors.json
data/subjects.json
data/config.json
data/knowledge.json
```

启用 PostgreSQL 模式：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://postgres:knockknock@localhost:5432/study_tracker?sslmode=disable"
go run .
```

如果 `TRACKER_STORAGE=postgres` 但没有设置 `TRACKER_DATABASE_URL`，程序会启动失败并输出明确错误。

## 3. 当前分层结构

核心依赖方向：

```text
api/handlers
  -> internal/service
      -> internal/repository interfaces
          -> internal/repository/jsonrepo
          -> internal/repository/postgres
```

service 层不再直接调用：

```go
LoadJSON(...)
SaveJSON(...)
```

而是调用 repository interface。

这样做的好处：

```text
同一套业务逻辑可以跑 JSON
同一套业务逻辑也可以跑 PostgreSQL
以后加 MySQL、SQLite 或云同步时也不会大改 handler
```

## 4. Repository 覆盖范围

当前已覆盖：

```text
SubjectRepository     科目
ErrorRepository       错题、标签、复习记录
SettingsRepository    OCR token、用户名
KnowledgeRepository   仪表盘知识点
OCRTaskRepository     OCR 任务记录
BackupRepository      备份导入导出
```

暂时只作为 schema 预留或认证辅助：

```text
attachments
refresh_tokens
```

原因：

```text
当前应用没有附件持久化功能
refresh_tokens 已用于登录刷新令牌，attachments 仍等待后续附件持久化功能接入
```

## 5. 用户隔离和认证

PostgreSQL 表是多用户结构，业务表都有：

```text
user_id
```

当前最终实现中，JSON 模式保持本地免登录；PostgreSQL 模式启用登录注册。认证中间件会解析 HttpOnly Cookie 中的登录状态，并把当前用户 ID 写入请求上下文。

```text
业务 handler -> service -> repository -> 使用当前请求的 user_id
```

因此 PostgreSQL repository 不应固定使用某个默认用户，而是根据当前请求的 `user_id` 访问数据。这样用户 A 和用户 B 的科目、错题、设置、OCR 任务和备份导入导出都能隔离。

## 6. 错题数据映射

前端模型里错题使用科目名称：

```json
{
  "subject": "数学"
}
```

数据库里错题使用外键：

```text
error_problems.subject_id
```

PostgreSQL repository 负责转换：

```text
新增错题：subject name -> subjects.id
查询错题：subjects.id -> subject name
```

标签映射：

```text
tags              标签表
error_problem_tags 错题和标签关联表
```

普通标签：

```text
tag_type = question
```

错因标签：

```text
tag_type = reason
```

复习时会同时：

```text
更新 error_problems.review_count / next_review_at
插入 review_records
```

## 7. 备份导入导出

HTTP API 不变：

```text
GET  /api/backup/export
POST /api/backup/import
```

JSON 模式：

```text
从 data/*.json 导出 zip
导入 zip 后覆盖 data/*.json
```

PostgreSQL 模式：

```text
从数据库导出同样结构的 zip
导入 zip 后用事务替换当前用户的数据
```

zip 里仍然是：

```text
errors.json
subjects.json
config.json
knowledge.json
```

这样可以保持旧版备份兼容。

## 8. JSON 导入 PostgreSQL

预览导入数量：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --dry-run
```

确认没问题后覆盖导入：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --replace
```

默认不加 `--replace` 时，如果 PostgreSQL 目标用户已有数据，会中止导入，避免重复写入。

## 9. 测试方式

普通测试不依赖 PostgreSQL：

```powershell
go test ./...
```

PostgreSQL 集成测试需要设置：

```powershell
$env:TEST_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go test ./internal/repository/postgres
```

集成测试会：

```text
创建临时测试用户
新增科目
新增错题
写入标签
写入复习记录
保存设置
保存知识点
记录 OCR task
导出备份
删除临时测试用户
```

## 10. 验收清单

JSON 模式：

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
Remove-Item Env:TRACKER_DATABASE_URL -ErrorAction SilentlyContinue
go run .
```

确认：

```text
页面正常打开
新增科目写入 data/subjects.json
新增错题写入 data/errors.json
备份导出正常
OCR token 仍保存到 config.json
```

PostgreSQL 模式：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go run .
```

确认：

```text
页面正常打开
新增科目写入 subjects
新增错题写入 error_problems
标签写入 tags / error_problem_tags
复习后 review_records 新增记录
备份导出得到同样的 zip 格式
OCR 后 ocr_tasks 有记录
```
