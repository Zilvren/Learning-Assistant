# Architecture

本项目采用 Go 后端分层架构，当前默认使用本地 JSON 存储，后续可扩展 PostgreSQL 服务端模式。

## Directory Layout

```text
api/handlers/          HTTP 请求处理层，只负责参数解析和响应输出
internal/service/      业务逻辑层，处理错题、复习、OCR、更新等核心流程
internal/repository/   数据访问层，当前实现为本地 JSON 文件存储
internal/model/        领域模型和请求/响应结构
internal/middleware/   Gin 中间件，例如安全头、CORS、后续 JWT 鉴权
pkg/config/            配置读取，支持命令行参数和环境变量
pkg/logger/            日志封装
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

上层可以依赖下层，下层不反向依赖上层。这样后续把 JSON 存储替换为 PostgreSQL 时，主要改动会集中在 `internal/repository`。

## Future PostgreSQL Mode

计划新增 PostgreSQL 实现时，可以保留当前 JSON 单机版，同时新增：

```text
internal/repository/postgres/
internal/repository/json/
```

服务层通过接口依赖 repository，运行时根据配置选择本地模式或服务端模式。

数据库表结构和迁移草案见 `docs/database.md` 与 `migrations/001_init_postgres.sql`，本地使用教程见 `docs/postgresql-tutorial.md`，Go 后端接入步骤见 `docs/postgresql-integration-tutorial.md`，更详细的小白版步骤见 `docs/postgresql-integration-beginner.md`，最终实现说明见 `docs/postgresql-repository-implementation.md`。
