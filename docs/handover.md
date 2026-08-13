# Learning Assistant 项目交接手册

> 最后核对：2026-08-09。本文面向接手开发、部署和日常运维的人员；功能说明见根目录 `README.md`，深入源码阅读顺序见 `docs/project-reading-guide.md`。

## 1. 接手时先看这里

### 当前仓库状态

- 当前分支：`main`。
- 交接文档编写前的最新提交：`a54b106 feat(frontend): improve mobile experience`。
- GitHub 正确仓库地址：`git@github.com:Zilvren/Learning-Assistant.git`。
- 本地 `origin` 仍使用旧拼写 `Learning-Assitant`。GitHub 目前会重定向，但建议接手后执行：

  ```powershell
  git remote set-url origin git@github.com:Zilvren/Learning-Assistant.git
  ```

- 编写本文时工作区并非干净，以下移动端修复尚未提交：
  - `frontend/src/style.css`：移动抽屉宽度、图标与文字对齐、品牌和账户信息布局。
  - `frontend/src/styles/library.css`：触屏卡片不翻面、阅读页头部不裁切操作按钮、复习复选框尺寸修复。
- 本交接文档及 README 中的入口链接也需要一并提交。交接前先运行 `git status --short`，不要直接用 `git add .` 混入无关文件。

### 当前验证结果

交接时已执行并通过：

```powershell
cd frontend
npm test -- --run   # 11 个测试文件，40 项测试通过
npm run build

cd ..
go test ./...
go vet ./...
```

本地 Docker 已使用最新工作区源码重新构建，`app`、`db`、`caddy` 均为运行状态，`http://127.0.0.1/api/health` 返回 HTTP 200。

## 2. 项目定位与运行模式

Learning Assistant 是一个 Go + Vue 的学习资料库，核心能力包括资料库、Markdown 笔记、复习计划、学习记录、OCR、AI 学习助手、备份恢复和多用户认证。

项目支持两种存储模式：

| 模式 | 适用场景 | 认证 | 数据位置 |
| --- | --- | --- | --- |
| `json` | Windows 本机、单用户 | 不启用登录 | 运行目录下的 `data/` |
| `postgres` | Docker、服务器、多用户 | Cookie/JWT 登录 | PostgreSQL + `tracker_data` 卷 |

生产部署必须使用 PostgreSQL。不要把 JSON 模式直接暴露到公网，因为它没有用户隔离。

## 3. 架构速览

```text
浏览器
  -> Caddy（服务器入口，80/443）
  -> Go + Gin（API，同时提供内嵌的 Vue dist）
  -> Service（业务规则）
  -> Repository 接口
       -> jsonrepo（单机）
       -> postgres（多用户）
```

请求依赖方向：

```text
main.go
  -> service.App
  -> middleware
  -> api/handlers
  -> internal/service
  -> internal/repository
  -> internal/model
```

重要原则：

- Handler 只做参数解析和 HTTP 响应。
- Service 承担业务规则，不判断当前使用 JSON 还是 PostgreSQL。
- Repository 负责持久化；PostgreSQL Repository 必须按认证用户隔离。
- `main.go` 中的 `setupRepositories` 是存储模式组装入口。
- `registerRoutes` 是全局中间件和 API 路由入口。
- PostgreSQL 迁移位于 `migrations/*.sql`，由 `schema_migrations` 记录并在启动时自动执行。

## 4. 目录与关键入口

| 路径 | 用途 |
| --- | --- |
| `main.go` | 程序启动、依赖组装、中间件与路由注册 |
| `api/handlers/` | HTTP Handler |
| `internal/service/` | 认证、资料库、备份、OCR、复习、AI 学习助手等业务逻辑 |
| `internal/repository/interfaces.go` | Repository 接口集合 |
| `internal/repository/jsonrepo/` | JSON 存储实现 |
| `internal/repository/postgres/` | PostgreSQL 存储实现 |
| `internal/middleware/` | 请求 ID、认证、安全头、审计、恢复等中间件 |
| `pkg/config/config.go` | 环境变量、启动参数和配置校验 |
| `migrations/` | PostgreSQL 有序迁移，目前为 `001` 至 `007` |
| `frontend/src/router/index.js` | 前端路由和认证守卫 |
| `frontend/src/layouts/AppShell.vue` | 顶栏、桌面侧栏和移动抽屉 |
| `frontend/src/components/library/` | 资料库与笔记阅读/编辑页 |
| `frontend/src/style.css` | 全局、应用壳、设置、登录和响应式样式 |
| `frontend/src/styles/library.css` | 资料库、笔记、复习和首页概览样式 |
| `deploy/` | Compose、Caddy 和生产部署脚本 |
| `.github/workflows/ci.yml` | CI 和生产部署工作流 |
| `scripts/build-release.ps1` | Windows 发布包构建与 GitHub Release 上传 |

当前前端主要路由：

| 路径 | 页面 |
| --- | --- |
| `/login` | 登录/注册 |
| `/verify-email` | 邮箱验证 |
| `/` | 学习概览 |
| `/library/:folderId?` | 资料库 |
| `/library/items/:itemId` | 文件或 Markdown 笔记阅读/编辑 |
| `/trash/:folderId?` | 回收站 |
| `/review` | 今日复习 |
| `/ai` | AI 学习助手（DeepSeek） |
| `/settings` | 设置 |

旧的 `/errors/:id?` 现在重定向到资料库；视觉预览和 `design-preview` 测试路由已经删除，不应重新加入生产路由。

## 5. 本地开发

### 环境要求

- Go：以 `go.mod` 为准；Docker 构建当前使用 Go 1.26。
- Node.js：CI 和 Docker 使用 Node.js 22。
- npm：使用已提交的 `frontend/package-lock.json`。
- Docker Desktop：仅 Docker/PostgreSQL 联调时需要。

### 最省事的本机 JSON 模式

```powershell
.\start.bat
```

`start.ps1` 默认会执行 `npm install`、`npm run build`，然后运行 Go。8000 端口被占用时会向后寻找可用端口。

跳过前端重复构建：

```powershell
.\start.ps1 -SkipFrontendBuild
```

### 前后端热更新开发

终端 1：

```powershell
go run . -- --no-browser
```

终端 2：

```powershell
cd frontend
npm ci
npm run dev
```

访问 `http://127.0.0.1:5173`。Vite 会把 `/api` 代理到 `http://127.0.0.1:8000`。

### 本地 PostgreSQL 模式

已有 PostgreSQL 时：

```powershell
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:密码@localhost:5432/study_tracker?sslmode=disable"
$env:TRACKER_JWT_SECRET="至少32字符的固定随机密钥"
.\start-postgres.ps1 -DatabaseUrl $env:TRACKER_DATABASE_URL
```

也可以直接使用项目 Docker：

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
docker compose -f deploy/docker-compose.yml ps
```

## 6. Docker：重启、重建与前端生效规则

这是本项目最容易误判的地方。Vue 生产文件会在 Dockerfile 中先构建，再复制到 Go 构建上下文并嵌入最终可执行文件。Compose 没有把宿主机 `frontend/dist` 挂载到 app 容器。

| 命令 | 是否重新构建前端 | 用途 |
| --- | --- | --- |
| `docker compose restart app` | 否 | 只重启旧容器 |
| `docker compose up -d app` | 通常否 | 使用现有镜像重新创建/启动 |
| `npm run build` | 只更新宿主机 `frontend/dist` | 不会自动进入正在运行的 Docker app |
| `docker compose up -d --build app` | 是 | 重新构建 Vue、Go 和 app 镜像 |

在项目根目录执行：

```powershell
docker compose -f deploy/docker-compose.yml up -d --build app
```

或者进入 `deploy`：

```powershell
cd deploy
docker compose up -d --build app
```

构建后检查：

```powershell
docker compose -f deploy/docker-compose.yml ps
(Invoke-WebRequest -UseBasicParsing http://127.0.0.1/api/health).StatusCode
```

浏览器仍显示旧界面时，先确认 app 的创建时间已经变化，再按 `Ctrl + Shift + R` 强制刷新。

## 7. 配置与密钥

常用变量：

| 变量 | 默认值/说明 |
| --- | --- |
| `TRACKER_STORAGE` | `json`；生产使用 `postgres` |
| `TRACKER_DATABASE_URL` | PostgreSQL 模式必填 |
| `TRACKER_REQUIRE_POSTGRES` | 生产为 `true`，防止错误降级到 JSON |
| `TRACKER_JWT_SECRET` | PostgreSQL 模式至少 32 字符，生产必须固定 |
| `TRACKER_REGISTRATION_ENABLED` | 首次建号后应改为 `false` |
| `TRACKER_ACCESS_TOKEN_TTL` | 默认 15 分钟 |
| `TRACKER_REFRESH_TOKEN_TTL` | 默认 30 天 |
| `TRACKER_COOKIE_SECURE` | HTTPS 后设为 `true`；HTTPS 公网 URL 也会自动启用 |
| `TRACKER_ENCRYPTION_KEY` | 可选的独立敏感数据加密密钥 |
| `TRACKER_DATA_DIR` | 自定义数据目录 |
| `TRACKER_AUTO_BACKUP` | JSON 本地模式自动备份开关，默认 `true` |
| `TRACKER_AUTO_BACKUP_INTERVAL` | 自动备份检查周期，默认 `24h`，不得小于 `1h` |
| `TRACKER_AUTO_BACKUP_KEEP` | 自动恢复点保留数量，默认 `14`，范围 `1-365` |
| `TRACKER_FRONTEND_DIR` | 默认 `frontend/dist`，主要用于源码运行 |
| `TRACKER_HOST` / `TRACKER_PORT` | 默认 `127.0.0.1:8000` |
| `TRACKER_NO_BROWSER` | 容器中为 `true` |
| `TRACKER_EMAIL_VERIFICATION_ENABLED` | 是否启用邮箱验证 |
| `TRACKER_PUBLIC_URL` | 邮件验证链接使用的 HTTPS 公网地址 |
| `TRACKER_SMTP_*` | SMTP 主机、端口、账号、授权码、发件人和 TLS 模式 |
| `DEEPSEEK_API_KEY` | AI 学习助手的可选环境变量 Key；设置页保存的加密 Key 优先 |
| `DEEPSEEK_MODEL` | 可选模型名，默认 `deepseek-chat` |
| `GIN_MODE` | 生产使用 `release` |

安全要求：

- 不提交 `deploy/.env`，服务器权限保持 `chmod 600 deploy/.env`。
- `POSTGRES_PASSWORD` 与 `TRACKER_JWT_SECRET` 使用两个不同的随机值。
- PostgreSQL 5432 不映射到公网。
- HTTP IP 测试阶段不要使用复用的重要密码。
- 配置域名和 HTTPS 后再启用 `TRACKER_COOKIE_SECURE=true` 与邮箱验证。

## 8. 测试与提交前检查

完整检查：

```powershell
cd frontend
npm ci
npm test -- --run
npm run build

cd ..
go test ./...
go vet ./...
git diff --check
git status --short
```

当前 GitHub Actions 会执行前端安装与构建、Go 测试、Go vet、Tracker 和 Updater 构建，但没有执行 `npm test`。在 CI 补上前端测试之前，本地提交前必须手工运行。

提交时只暂存本次文件。例如：

```powershell
git add -- frontend/src/style.css frontend/src/styles/library.css docs/handover.md README.md
git commit -m "fix(frontend): polish mobile navigation and reader actions"
git push origin main
```

推送 `main` 会触发生产流水线，不能把它当作仅备份代码的无副作用操作。

## 9. CI/CD 与生产发布

`.github/workflows/ci.yml` 对 `main` 的 push 执行：

1. Windows runner 安装 Go 和 Node.js 22。
2. 构建前端，执行 Go test 和 vet。
3. 构建 `Tracker.exe` 与 `Updater.exe`。
4. Ubuntu runner 构建带提交 SHA 标签的生产镜像。
5. 镜像作为短期 artifact 交给生产 self-hosted runner。
6. `deploy/deploy-from-runner.sh` 将代码同步到 `/opt/Learning-Assistant`，保留服务器 `.env` 和数据卷。
7. 使用 `docker compose up -d --no-build app` 切换到 CI 已验证的镜像。
8. 最多等待约 60 秒检查 `/api/health`；失败时输出 app 日志并使部署失败。

生产服务器常用检查：

```bash
cd /opt/Learning-Assistant/deploy
docker compose ps
docker compose logs --tail=100 app
docker compose logs --tail=100 db
docker compose logs --tail=100 caddy
curl -i http://127.0.0.1/api/health
```

不要直接在容器内改代码，重建后修改会丢失。

## 10. 数据、备份与恢复

### JSON 模式

主要数据位于 `data/`：

- `subjects.json`
- `errors.json`
- `config.json`
- `knowledge.json`
- `library.json`
- `blobs/`
- `backups/`

更新或迁移时不要删除整个 `data/`。

### Docker/PostgreSQL 模式

关键卷：

| 卷 | 内容 |
| --- | --- |
| `deploy_postgres_data` | PostgreSQL 数据库 |
| `deploy_tracker_data` | app 本地数据、附件或辅助文件 |
| `deploy_caddy_data` | Caddy 证书和运行数据 |
| `deploy_caddy_config` | Caddy 配置状态 |

数据库备份：

```bash
cd /opt/Learning-Assistant/deploy
docker compose exec -T db pg_dump -U study_tracker study_tracker | gzip > study-tracker-$(date +%F).sql.gz
```

重要警告：正式数据存在后不要执行 `docker compose down -v`，这会删除 Compose 数据卷。普通 `docker compose up -d --build app` 不会删除数据库。

应用设置页也支持 ZIP 导出和导入。导入前会生成 `pre-import` 快照；自动更新前会生成 `pre-update` 快照。

## 11. 日志与排障顺序

### 页面打不开

1. `docker compose ps` 查看三个容器。
2. 请求 `/api/health`。
3. 查看 `app` 日志，再看 `db` 与 `caddy`。
4. 确认 80/443 端口和服务器安全组。

### 修改前端后页面没变化

1. 确认访问的是 5173 开发服务还是 80 端口 Docker。
2. Docker 页面必须执行 `docker compose up -d --build app`。
3. 查看 `docker compose ps` 中 app 的创建时间。
4. 强制刷新浏览器。

### API 报错

- 记录响应中的 `X-Request-ID` 或 JSON `request_id`。
- 在服务端日志搜索相同请求 ID。
- `[AUDIT]` 记录方法、路径、状态、耗时、IP 和用户 ID，但不应记录 Cookie、Authorization 或正文。

### 数据库 unhealthy

```bash
docker compose logs --tail=100 db
```

不要在未备份时删除卷。新增表或字段必须追加新的有序迁移文件，不能回改已经在生产执行过的迁移。

## 12. 已知未修复风险

以下问题已经在代码复核中确认，接手后应优先处理：

### P1：刷新令牌存在并发重放竞态

- 位置：`internal/service/auth_service.go` 的 `RefreshLogin`，以及 `internal/repository/postgres/auth_repository.go` 的 `RevokeRefreshToken`。
- 当前流程是“查询令牌 -> 查询用户 -> 撤销令牌 -> 签发新令牌”，不是一个原子操作。
- PostgreSQL 的撤销更新即使影响 0 行也返回成功；两个并发刷新请求可能都签发新令牌。
- 建议：在事务中使用条件更新并检查影响行数，或 `UPDATE ... WHERE revoked_at IS NULL ... RETURNING`，只有成功消费令牌的请求才能继续签发。

### P1：大资料库备份导出可能造成内存峰值过高

- 位置：`internal/service/backup_service.go` 的 `loadBackupData` 与 `encodeBackupZip`。
- 当前先把全部 blob 读入 `map[string][]byte`，再用 `bytes.Buffer` 生成完整 ZIP，原始附件和 ZIP 会同时驻留内存。
- 单文件允许达到 200MB，Compose 中的 `GOMEMLIMIT=384MiB` 只是软限制，进程仍可能被系统终止。
- 建议：使用 `zip.Writer` 直接流式写响应或受控临时文件，blob 逐个 `io.Copy`，并增加备份总量配额。

### P1：JSON 备份恢复不是跨文件原子提交

- 位置：`internal/repository/jsonrepo/backup_repository.go` 的 `Import`，以及 `internal/service/backup_service.go` 中后续单独保存 `library.json` 的逻辑。
- 多个 JSON 文件依次原子替换，但整个恢复没有事务回滚；磁盘满、权限或 rename 失败时可能留下新旧数据混合。
- 建议：把完整快照写入 staging 目录后一次切换，或增加 journal 和失败自动回滚；`library.json` 必须纳入同一次提交。

### P2：CI 未运行前端单元测试

- `.github/workflows/ci.yml` 目前只有 `npm run build`，没有 `npm test -- --run`。
- 建议在构建前后增加 Vitest 步骤，防止前端回归直接进入生产部署。

### P3：本地 Git remote 仍是旧拼写

- 旧地址目前由 GitHub 重定向，推送可成功，但会持续出现 repository moved 警告。
- 按本文开头命令更新 `origin`。

## 13. Windows 发布包

构建本地发布包：

```powershell
.\scripts\build-release.ps1 -Version 2026.08.09-2100 -Clean
```

产物包括：

- `dist/Tracker.exe`
- `dist/Updater.exe`
- `dist/release/Tracker.zip`

上传 GitHub Release：

```powershell
gh auth login
.\scripts\build-release.ps1 -Version 2026.08.09-2100 -Clean -Upload
```

根目录 `version.json` 当前版本为 `2026.08.07-1107`。发布脚本会为发布包生成对应版本文件；发布前确认版本、Tag 与 Release 资产一致。

## 14. 推荐接手顺序

1. 更新 `origin` 到正确仓库地址。
2. 检查并提交当前移动端修复和本文档。
3. 在 CI 中加入前端测试。
4. 先修刷新令牌并发消费问题。
5. 将备份导出改为流式，并为 JSON 恢复增加事务式回滚。
6. 在生产服务器验证定时数据库备份和恢复演练。
7. 完成域名、HTTPS、Secure Cookie 后再启用邮箱验证。

## 15. 相关文档

- 功能和普通用户说明：`README.md`
- 源码阅读路线：`docs/project-reading-guide.md`
- 架构说明：`docs/architecture.md`
- 生产部署：`docs/deployment.md`
- 数据库设计：`docs/database.md`
- PostgreSQL 教程：`docs/postgresql-tutorial.md`
- PostgreSQL 接入说明：`docs/postgresql-integration-tutorial.md`
