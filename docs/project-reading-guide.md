# 学习追踪器：项目阅读指南

这是一份面向开发者的阅读路线。目标不是一开始读完所有代码，而是在理解一个真实请求如何完成后，再逐层进入资料库、复习、认证和数据库实现。

建议先使用已经启动的本地环境：

- 应用首页：[http://127.0.0.1/](http://127.0.0.1/)
- 健康检查：[http://127.0.0.1/api/health](http://127.0.0.1/api/health)

当前本地环境是 PostgreSQL 模式。若想理解纯本地单机模式，可将 `TRACKER_STORAGE` 设为 `json` 后重新启动。

## 先建立全局认识

项目是一个“学习资料库 + 笔记 + 复习 + 错题 + OCR”的前后端应用：

- 前端：Vue 3 + Vue Router，负责页面、编辑器和交互。
- 后端：Go + Gin，负责 API、认证、业务规则与静态前端文件。
- 存储：同一套业务接口支持 JSON 本地存储和 PostgreSQL 多用户存储。
- 部署：Docker Compose 启动 PostgreSQL、Go 应用和 Caddy；生产镜像在构建时打包前端。

把它想成下面这条链路即可：

```text
浏览器 Vue 页面
  -> /api 请求
  -> Gin 路由与中间件
  -> Handler（解析 HTTP）
  -> Service（业务规则）
  -> Repository 接口
  -> JSON 文件 或 PostgreSQL
```

## 推荐阅读顺序

按这个顺序读，大约 2～3 次各 45 分钟即可建立完整地图。

| 阶段 | 阅读文件 | 想弄明白的问题 |
| --- | --- | --- |
| 1. 启动 | `main.go`、`embed.go` | 应用如何创建、路由如何注册、前端为何能由 Go 提供？ |
| 2. 依赖与请求 | `internal/service/app.go`、`internal/middleware/request.go`、`internal/middleware/auth.go` | 配置、仓储和用户 ID 如何进入一次请求？ |
| 3. 接口边界 | `api/handlers/response.go`、`api/handlers/library.go` | HTTP 参数如何转成业务调用，失败怎样返回？ |
| 4. 一条业务链路 | `internal/service/library_service.go`、`internal/repository/interfaces.go` | 资料库的规则与存储细节如何分离？ |
| 5. 两种存储 | `internal/repository/jsonrepo/`、`internal/repository/postgres/` | 同一接口为什么能写 JSON 或 PostgreSQL？ |
| 6. 前端 | `frontend/src/router/index.js`、`frontend/src/api/index.js`、`frontend/src/components/library/` | 页面路由和 API 调用怎样对应？ |
| 7. 数据演进 | `migrations/`、`internal/repository/postgres/migration.go` | 数据库如何安全地逐版本升级？ |
| 8. 质量保障 | `*_test.go`、`frontend/src/test/`、CI 配置 | 修改后怎样验证不会破坏原有功能？ |

不建议第一遍阅读就进入 `cmd/updater/` 或平台专用文件（`*_windows.go`、`*_unix.go`）；它们是功能分支，不是主干。

## 阅读前的术语表

| 术语 | 在本项目中的含义 | 第一次阅读时记住什么 |
| --- | --- | --- |
| Handler | `api/handlers/` 的 Gin 函数 | 只理解 HTTP：路径参数、JSON、文件上传、状态码。 |
| Service | `internal/service/` 的业务函数 | 放规则，不应该知道具体 SQL 或 JSON 文件名。 |
| Repository | `internal/repository/interfaces.go` 定义的存储契约 | 同一个接口同时有 JSON 和 PostgreSQL 实现。 |
| App | `service.App` | 启动时组装的一组不可变依赖，通过 request context 传递。 |
| context | Go 的请求上下文 | 传取消信号、当前 App 和当前 `user_id`，不是全局变量。 |
| Migration | `migrations/*.sql` | 数据结构的版本历史，每个文件只执行一次。 |
| Blob | `data/blobs/` 中按哈希保存的二进制内容 | 资料库索引和大文件内容分开存，备份时两者必须一起恢复。 |
| 热力图活动 | `user_activity_events` 的日聚合 | 由写入触发器记录，读取仪表盘本身不应制造活动。 |

## 从启动到页面：逐行理解主干

打开 [main.go](../main.go)，可以把启动过程分成六步：

```text
1. config.Load(os.Args[1:])
2. cfg.Validate()
3. setupRepositories(cfg)
4. service.NewApp(cfg, repos, pool)
5. registerRoutes(router, app)
6. router.RunListener(listener)
```

### 第 1、2 步：配置不是“到处读取环境变量”

[pkg/config/config.go](../pkg/config/config.go) 将环境变量和命令行参数收敛为一个 `Config`：

- `TRACKER_STORAGE=json|postgres` 决定存储实现；
- `TRACKER_DATABASE_URL` 只在 PostgreSQL 模式需要；
- `TRACKER_REQUIRE_POSTGRES=true` 防止服务器误降级到 JSON；
- `TRACKER_JWT_SECRET` 用于签发访问 Cookie；
- SMTP、公开 URL 和邮箱验证开关属于认证配置。

`Validate` 在网络监听前失败。这样部署时宁可服务起不来，也不会出现“以为在用数据库，实际上写进本地 JSON”的风险。

### 第 3 步：Repository 的装配点

`setupRepositories` 是唯一按存储模式分支的主要入口：

```text
json
  -> jsonrepo.NewRepositories()

postgres
  -> postgres.NewPool()
  -> ApplyMigrations()
  -> EnsureLocalUser()
  -> postgres.NewRepositories(pool, userID)
```

业务层之后只看到 `repository.Repositories`，不会再判断“我现在是 JSON 还是 PostgreSQL”。这正是分层能长期维护的关键。

### 第 4、5 步：App 和中间件

`service.NewApp` 只创建一次。`registerRoutes` 的全局中间件顺序很重要：

```text
Gin Logger
  -> RequestContext       写入 App、生成 X-Request-ID
  -> RequestAudit         在请求结束后记录写操作和失败请求
  -> Recovery             捕获 panic，转换为标准 500 响应
  -> SecurityHeaders
  -> LocalCORS
  -> CookieOriginGuard    PostgreSQL 模式下检查跨站写请求来源
  -> 路由/Handler
```

请留意：`RequestAudit` 放在业务路由外层，才能在请求结束后拿到最终状态码；`Recovery` 也必须放在路由外层，才能处理任何 Handler 或 Service 的 panic。

## 一次受认证资料库请求的完整链路

以“打开一个笔记”为例，建议同时在浏览器 Network 面板和代码中跟踪：

```text
1. LibraryItemPage.vue 调用 api.getLibraryContent(id)
2. api/index.js 发出 GET /api/library/items/:id/content，并携带 HttpOnly Cookie
3. AuthRequired 读取 tracker_access，校验 JWT
4. AuthRequired 将 user_id 写入 request context
5. GetLibraryContent 解析 id，并调用 service.ReadLibraryContent(ctx, id)
6. Service 通过 repositories(ctx) 得到当前用户的 LibraryRepository
7. PostgreSQL Repository 查询该用户的 library_items / library_versions
8. Handler 设置 ETag、Content-Type、Content-Disposition 并返回原始内容
```

其中第 6 步是多用户安全的核心：`repositories(ctx)` 没有拿到用户 ID 时会返回“未登录”，而不是悄悄使用一个默认用户。

### 自动保存与版本冲突

笔记保存请求是：

```text
PUT /api/library/items/:id/content
```

前端从响应头保存当前 `ETag` 版本；下次提交时把版本作为 `base_version` 带给后端。Service/Repository 用它做乐观并发控制：

- 版本一致：保存正文并生成新版本；
- 版本不一致：后端返回 `409 version_conflict`；
- 前端提示用户刷新或按冲突策略继续。

阅读这条链路时，重点找 `SaveLibraryContent`、`SaveContent` 和 `CurrentVersion`，不要只看编辑器组件。

### 图片与 Markdown

Markdown 编辑器的主入口是 [MarkdownEditor.vue](../frontend/src/components/MarkdownEditor.vue)，渲染入口是 [utils/markdown.js](../frontend/src/utils/markdown.js)。

- 编辑器把允许的图片处理为受控的 `data:image/...;base64,...`；
- 图片宽度和对齐信息放在 Markdown 图片 title 元数据中；
- `normalizeSafeDataImages` 只接受白名单图片 MIME 类型；
- `renderMd` 先保护公式和对齐块，再进行 Markdown 渲染，最后回填 HTML；
- 后端保存的仍是普通笔记正文，因此版本、备份和恢复不需要为图片另建一套协议。

## 资料库、回收站和备份：三个容易混淆的概念

### 资料库索引与内容

`LibraryItem` 是索引记录：名称、父目录、类型、标签、删除时间、当前版本、Blob 哈希等。正文或上传文件内容则按哈希独立保存为 Blob。

这样做的好处是：

- 列表页只读取轻量索引，不下载每篇笔记正文；
- 版本可以复用相同内容的 Blob；
- 备份可以检查索引引用的每个 Blob 是否存在；
- PostgreSQL 表保存结构化索引，而文件内容可以通过持久卷保留。

### 回收站不是“把子文件逐个显示”

删除文件夹时，资料库会把文件夹和子树视为一个回收站项目：列表中只显示该文件夹；恢复和永久删除则作用于整棵子树。阅读时查看：

```text
LibraryPage.vue: trashItem / restoreItem / purgeItem
library_service.go: TrashLibraryItem / RestoreLibraryItem / PurgeLibraryItem
LibraryRepository: Trash / Restore / Purge
```

如果 UI 出现“文件夹进入回收站后仍显示子文件”的问题，优先检查 Repository 的查询条件和 `trashed`/父节点过滤，而不是只改前端列表。

### ZIP 备份是可移植格式

备份根目录包含 `config.json`、`subjects.json`、`errors.json`、`knowledge.json`、`library.json` 和 `blobs/`。导入流程会：

1. 限制上传大小、条目数和压缩比，防止 ZIP 炸弹；
2. 校验 JSON、Blob 哈希与资料库引用；
3. 在导入前创建 `pre-import` 快照；
4. 导入 Repository 数据；
5. 在 JSON 模式额外写入 `library.json`，在 PostgreSQL 模式由数据库 Repository 写入索引。

因此导出包中看见多个 JSON 文件是正常的；它不是把整个 PostgreSQL 数据库文件直接压缩进去。

## 认证与邮箱验证的阅读顺序

认证需要同时读前后端，推荐顺序如下：

```text
前端：store/auth.js
  -> AuthPage.vue / VerifyEmailPage.vue
  -> api/index.js

后端：api/handlers/auth.go
  -> internal/service/auth_service.go
  -> internal/service/email_verification.go
  -> internal/repository/postgres/auth_repository.go
```

要记住以下规则：

- JSON 模式不启用账号体系；
- PostgreSQL 模式使用访问 Cookie + 刷新 Cookie；
- 浏览器 JavaScript 不能直接读取 HttpOnly Cookie；
- API 返回 401 时，前端只尝试刷新一次，再决定是否回到登录页；
- 开启邮箱验证时，注册会返回 `202 Accepted`，而非直接登录；
- SMTP 密码、JWT 密钥和 OCR Token 不应出现在前端代码、备份或 Git 仓库。

## 错误响应、日志与排错方法

所有 JSON API 错误都包含：

```json
{
  "detail": "给用户看的错误信息",
  "error": { "code": "给程序判断的稳定代码", "message": "同上" },
  "request_id": "本次请求的追踪编号"
}
```

排错时按这个顺序操作：

1. 浏览器 Network 面板记录 URL、状态码和 `X-Request-ID`；
2. 执行 `docker compose -f deploy/docker-compose.yml logs --tail=100 app`；
3. 用 request ID 搜索 `[AUDIT]` 或 `[ERROR]`；
4. 判断问题位于参数解析、认证、Service 规则、Repository 或数据库迁移；
5. 先写能重现问题的测试，再修生产代码。

不要把 500 响应里没有底层数据库错误当成“信息丢失”：这是刻意的安全策略。真实错误应只在服务器日志中查看。

## 如何阅读测试

后端使用 Go 标准测试，前端使用 Vitest。先从这些测试开始：

- `main_test.go`：路由认证策略、前端文件回退与端口选择；
- `internal/service/app_test.go`：请求 App 与 legacy App 的优先级；
- `internal/service/backup_service_test.go`：ZIP 备份恢复的可见性；
- `internal/repository/jsonrepo/concurrency_test.go`：JSON 锁与并发写入；
- `internal/repository/postgres/integration_test.go`：PostgreSQL 真实集成场景；
- `frontend/src/test/library.test.js`：资料库 UI 行为；
- `frontend/src/test/learning-heatmap.test.js`：热力图窗口和年份切换。

测试文件中每个 `Test...` 或 `it(...)` 的名字就是一个可执行的需求说明。阅读生产代码遇到不确定行为时，先搜索对应测试名字，通常比凭感觉修改更快。

## 建议的四次阅读安排

### 第一次：90 分钟，跑通主干

阅读 `main.go`、`app.go`、`request.go`、`router/index.js`、`api/index.js`。目标是能画出“浏览器请求为什么能到某个 Repository”。

### 第二次：90 分钟，掌握资料库

阅读 `LibraryPage.vue`、`LibraryItemPage.vue`、`library.go`、`library_service.go`、两个 `library_repository.go`。目标是解释新建、保存、回收、恢复和版本冲突。

### 第三次：60 分钟，掌握用户与数据

阅读认证链路、`db.go`、`migration.go`、`migrations/005` 和 `007`。目标是解释登录如何隔离数据、热力图为何不会由读取操作增加活动。

### 第四次：60 分钟，练习改一个小功能

给资料库卡片新增一个纯展示字段，完整走一遍“模型 → 接口 → JSON/PostgreSQL → Service → Handler → API → Vue → 测试”。不要跳过 PostgreSQL 实现或迁移；这样最能训练全栈闭环思维。

## 第一站：启动与路由

从 [main.go](../main.go) 开始。重点看四件事：

1. `config.Load` 和 `cfg.Validate`：读取环境变量/命令行参数，并在监听端口前拒绝错误配置。
2. `setupRepositories`：依据 `TRACKER_STORAGE` 选择 JSON 或 PostgreSQL Repository。
3. `service.NewApp`：创建不可变的应用依赖容器，持有配置、Repository 和 PostgreSQL 连接池。
4. `registerRoutes`：安装中间件，再把 `/api/...` 连接到 Handler。

[embed.go](../embed.go) 使用 Go 的 `embed` 把 `frontend/dist` 放进可执行文件。因此 Docker 或发布版只需要运行一个 Go 应用；前端的 Vue Router 页面由 `serveFrontend` 回退到 `index.html`。

### 练习：追踪一个请求

在浏览器打开资料库，然后用开发者工具观察：

```text
GET /api/library/items?parent_id=...
```

按下列顺序跳转代码：

```text
main.go: registerRoutes
  -> api/handlers/library.go: ListLibraryItems
  -> internal/service/library_service.go: ListLibrary
  -> internal/repository/interfaces.go: LibraryRepository.List
  -> jsonrepo/library_repository.go 或 postgres/library_repository.go
```

这一条链路理解后，项目大多数 API 的结构都是相同的。

## 第二站：应用依赖、认证与中间件

[internal/service/app.go](../internal/service/app.go) 是后端的依赖边界。

`App` 创建之后不再修改。HTTP 请求进入时，[RequestContext](../internal/middleware/request.go) 会把它写入 `context.Context`；Service 再从 context 中取得当前应用，而不是读取可变的包级全局变量。

在 PostgreSQL 模式下，请求还会经过 [AuthRequired](../internal/middleware/auth.go)：

```text
Cookie 中的 tracker_access
  -> ValidateAccessToken
  -> 得到当前 user_id
  -> user_id 写入 request context
  -> Service 为该用户创建 PostgreSQL Repository
```

这意味着不同用户使用相同接口时，Service 无须手写 `WHERE user_id = ...`；隔离逻辑收敛在 PostgreSQL Repository 的创建过程。

### 错误、请求 ID 与审计

请求中间件还负责：

- 生成 `X-Request-ID`，便于把浏览器报错和服务端日志对应起来；
- 捕获 panic，返回统一的 500 错误，不暴露堆栈给浏览器；
- 对 API 写操作和失败请求输出 `[AUDIT]` JSON 日志；
- 不记录 Cookie、Authorization、请求正文或查询参数，避免敏感信息进入日志。

所有 API 错误都经过 [internal/apierror](../internal/apierror/apierror.go)，结构如下：

```json
{
  "detail": "请求格式错误",
  "error": {
    "code": "invalid_request",
    "message": "请求格式错误"
  },
  "request_id": "..."
}
```

`detail` 保留给已有前端使用；新代码可优先依据稳定的 `error.code` 做分支处理。

## 第三站：业务层与 Repository 接口

[internal/repository/interfaces.go](../internal/repository/interfaces.go) 是非常值得先读的文件。它定义：

- `SubjectRepository`：科目；
- `ErrorRepository`：错题及复习；
- `LibraryRepository`：文件夹、笔记、文件、版本、回收站与资料复习；
- `AuthRepository`：账号、刷新令牌、邮箱验证；
- `BackupRepository`：导入/导出；
- `SettingsRepository`、`KnowledgeRepository`、`OCRTaskRepository`：设置、知识点和 OCR 任务。

Service 不知道 JSON 文件名、SQL 写法或表结构。它只调用接口并编排业务规则。例如可依次阅读：

- [library_service.go](../internal/service/library_service.go)：资料库、回收站、版本与复习；
- [error_service.go](../internal/service/error_service.go)：错题与间隔复习；
- [auth_service.go](../internal/service/auth_service.go)：注册、登录、刷新令牌与 Cookie；
- [backup_service.go](../internal/service/backup_service.go)：ZIP 备份导入导出与 Blob 校验；
- [activity_service.go](../internal/service/activity_service.go)：学习热力图查询。

读 Service 时先问三个问题：输入是否已验证？它调用了哪个 Repository？错误应由谁转换成 HTTP 状态码？

## 第四站：JSON 与 PostgreSQL 两个实现

### JSON 单机模式

阅读 `internal/repository/jsonrepo/`：

- `repositories.go`：组装整套 Repository；
- `library_repository.go`：资料库 JSON 索引与内容操作；
- `helpers.go` 和上层 `json_store.go`：读写、锁与原子替换；
- `concurrency_test.go`：并发读写的验证。

JSON 模式不要求登录，数据位于 `data/`。适合桌面单用户，但不应直接暴露到公网。

### PostgreSQL 多用户模式

阅读 `internal/repository/postgres/`：

- [db.go](../internal/repository/postgres/db.go)：创建 `pgxpool`、探活、执行迁移；
- [db.go](../internal/repository/postgres/db.go) 中的 `NewRepositories`：依据当前 `userID` 组装仓储；
- `library_repository.go`、`error_repository.go`：SQL 读写与用户隔离；
- `backup_repository.go`：数据库数据的备份导入导出；
- `integration_test.go`：需要真实 PostgreSQL 的测试入口。

一个重要约定：在 PostgreSQL 模式中，所有业务查询都必须按 `user_id` 隔离；新增功能时先检查 Repository 是否延续了这一点。

## 第五站：数据库迁移与学习记录

`migrations/` 中每一个 SQL 文件都是一个不可变版本。不要修改已经上线的旧迁移；需要变更表结构时新增下一个编号文件。

[migration.go](../internal/repository/postgres/migration.go) 会：

1. 创建 `schema_migrations` 账本；
2. 使用 PostgreSQL advisory lock 防止多实例并发迁移；
3. 仅执行尚未登记的 SQL；
4. 成功后记录版本。

与学习热力图有关的迁移值得重点阅读：

- `005_learning_activity.sql`：创建活动事件表和初始触发器；
- [007_activity_deduplication.sql](../migrations/007_activity_deduplication.sql)：把“同一资料/错题同一天的反复自动保存”去重，避免键入时刷出大量学习活动。

[activity_service.go](../internal/service/activity_service.go) 只是按年汇总 `user_activity_events`，不在页面加载时写入活动。因此看到异常活跃度时，应先检查触发器、活动事件表和写入来源，而不是热力图组件。

## 第六站：前端阅读路线

从这四个文件开始：

1. [main.js](../frontend/src/main.js)：挂载 Vue；
2. [router/index.js](../frontend/src/router/index.js)：页面路径与认证守卫；
3. [api/index.js](../frontend/src/api/index.js)：所有 `/api` 调用、刷新 Token 与错误处理；
4. [App.vue](../frontend/src/App.vue)：最外层路由出口和全局 Toast。

然后按功能读组件：

```text
首页/仪表盘：components/HomePage.vue
  -> components/dashboard/LearningHeatmap.vue
  -> components/dashboard/ReviewQueue.vue

资料库：components/library/LibraryPage.vue
  -> components/library/LibraryItemPage.vue
  -> components/MarkdownEditor.vue / MarkdownRenderer.vue

认证：components/AuthPage.vue / VerifyEmailPage.vue
  -> store/auth.js

设置与备份：components/SettingsPage.vue
```

`store/` 是轻量的模块级响应式状态，不使用 Pinia。`composables/` 放可复用的页面逻辑；`utils/` 放纯工具函数。样式主入口是 `style.css`，资料库专用样式在 `styles/library.css`。

## 如何安全地做一次功能改动

以“给资料库新增一个字段”为例：

1. 先修改 `internal/model/` 中的数据结构与请求结构；
2. 修改 `internal/repository/interfaces.go`，明确存储契约；
3. 同时实现 JSON 和 PostgreSQL Repository；
4. 若影响 PostgreSQL 表，新增迁移文件，不能重写旧迁移；
5. 在 `internal/service/` 中写规则，Handler 只解析参数和返回响应；
6. 在 `api/handlers/` 注册/调整接口，并使用 `respondProblem` 或 `respondError` 返回错误；
7. 在 `frontend/src/api/index.js` 增加 API 调用，再修改组件；
8. 补 Go 与 Vitest 测试，最后运行完整测试。

可用的常用命令：

```powershell
go test ./...
go vet ./...

cd frontend
npm test
npm run build

cd ..
docker compose -f deploy/docker-compose.yml ps
docker compose -f deploy/docker-compose.yml logs --tail=100 app
```

## 阅读时可暂时跳过的部分

- `cmd/updater/`：Windows 发布包的自更新程序；
- `scripts/build-release.ps1`：发布构建脚本；
- `docs/postgresql-*-tutorial.md`：数据库学习资料，适合主干读完后再看；
- `docs/diagrams/`：数据库关系图源文件与渲染产物。

## 最后用这份清单自测

读完主干后，你应该能回答：

1. 浏览器请求 `/api/library/items` 后，经过哪些层？
2. PostgreSQL 模式中，当前用户 ID 在什么位置写入请求上下文？
3. 为什么 Service 不直接写 SQL？
4. 新 SQL 迁移为什么不能修改已经上线的旧文件？
5. 500 错误为什么不会把数据库错误原样返回浏览器？
6. 热力图的“学习活动”究竟由什么数据表统计？
7. 为什么 Docker 启动后不需要单独启动一个前端服务器？

若这七题都能说明白，就已经掌握了项目的核心架构。接下来最适合深入的是 `LibraryRepository`、认证链路，或 PostgreSQL 迁移账本。
