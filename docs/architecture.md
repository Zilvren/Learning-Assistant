# Architecture

本项目采用 Go 后端分层架构，同时支持本地 JSON 存储和 PostgreSQL 服务端存储。默认使用 JSON 模式；通过环境变量切换为 PostgreSQL 模式后，会启用登录注册、Cookie/JWT 认证和按用户隔离的数据访问。

## Directory Layout

```text
api/handlers/          HTTP 请求处理层，只负责参数解析和响应输出
internal/service/      业务逻辑层，处理错题、复习、OCR、更新等核心流程
internal/repository/   Repository 接口、JSON 公共存储工具及两种存储实现
  jsonrepo/            本地 JSON 文件存储实现
  postgres/            PostgreSQL 存储实现，所有业务数据按用户隔离
internal/model/        领域模型和请求/响应结构
internal/middleware/   Gin 中间件，包括安全头、CORS、Cookie 来源检查和认证
pkg/config/            配置读取，支持命令行参数和环境变量
pkg/logger/            日志封装
cmd/import-json/       显式执行的旧 JSON 导入工具（正常启动不会读取或迁移 data/）
cmd/updater/           独立更新器入口
frontend/              Vue 前端源码
scripts/               构建和发布脚本
```

## Dependency Direction

```text
main
  -> api/handlers
    -> internal/service
      -> internal/repository
      -> internal/model
```

上层可以依赖下层，下层不反向依赖上层。Service 通过 Repository 接口工作；在 PostgreSQL 模式中，认证中间件把用户 ID 写入请求上下文，Service 为该用户创建隔离的 Repository。

## Storage Modes

### JSON 模式

JSON 是默认模式，适合本机单用户使用，不启用登录。数据保存在运行目录的 `data/` 文件夹中。

同一数据目录的访问由进程内读写锁和跨进程文件锁共同保护。所有“读取—修改—写回”操作在一个写事务中完成，JSON 文件通过同目录临时文件原子替换，多个 Tracker 实例共享 `data/` 时也不会互相覆盖。备份导出和导入使用同一目录级锁，以保证导出快照的一致性。

```powershell
go run .
```

也可以显式指定：

```powershell
$env:TRACKER_STORAGE="json"
go run .
```

### PostgreSQL 模式

PostgreSQL 模式已经实现，适合多用户或服务端部署。该模式会启用登录注册，并通过认证中间件将当前用户 ID 传入 Service；Service 再为该用户创建 Repository，确保科目、错题、设置、知识点、OCR 任务和备份数据相互隔离。

```powershell
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:password@localhost:5432/study_tracker?sslmode=disable"
.\start-postgres.ps1 -DatabaseUrl $env:TRACKER_DATABASE_URL
```

该启动脚本会设置 `TRACKER_STORAGE=postgres` 和 `TRACKER_REQUIRE_POSTGRES=true`；配置不完整时拒绝降级为 JSON。应用会使用内嵌的 `migrations/*.sql` 和 `schema_migrations` 账本按版本自动升级数据库，启动本身不会再写入活动记录或重复创建业务数据。

旧 JSON 数据不会被自动读取或迁移。确有迁移需要时，使用 ZIP 备份恢复，或明确执行 `cmd/import-json`。

数据库设计见 `docs/database.md`，初始化脚本见 `migrations/001_init_postgres.sql`，本地使用教程见 `docs/postgresql-tutorial.md`，接入过程见 `docs/postgresql-integration-tutorial.md` 和 `docs/postgresql-integration-beginner.md`，最终实现说明见 `docs/postgresql-repository-implementation.md`。
