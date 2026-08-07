# Changelog

所有重要变更都会记录在这里。

格式参考 Keep a Changelog，版本号使用项目发布版本。

## [Unreleased]

## [2026.08.07-1107] - 2026-08-07

### Fixed

- 修正自动更新所使用的 GitHub 仓库标识，确保发布包可从当前项目获取后续更新。

## [2026.08.07-1034] - 2026-08-07

### Added

- 新增 Go 后端分层架构：`api/handlers`、`internal/service`、`internal/repository`、`internal/model`、`internal/middleware`、`pkg/config`、`pkg/logger`。
- 新增 `docs/architecture.md`，说明项目分层、依赖方向和后续 PostgreSQL 扩展思路。
- 新增端口配置能力，支持通过命令行参数和环境变量指定启动端口与 host。
- 新增 PostgreSQL 存储模式，支持通过 `TRACKER_STORAGE=json|postgres` 在 JSON 本地模式和 PostgreSQL 模式之间切换。
- 新增 `pgx/pgxpool` PostgreSQL 连接池、数据库迁移脚本 `migrations/001_init_postgres.sql` 和 PostgreSQL repository 实现。
- 新增登录注册功能，仅在 PostgreSQL 模式启用，支持注册、登录、刷新登录状态、退出登录和获取当前用户信息。
- 新增认证中间件，支持 HttpOnly Cookie、access token、refresh token 和写操作来源校验。
- 新增 JSON 数据导入 PostgreSQL 工具 `cmd/import-json`，支持 `--dry-run` 和 `--replace`。
- 新增 PostgreSQL 模式下的备份导入导出能力，继续兼容原 JSON zip 备份格式。
- 新增 `.env.example`，提供 JSON/PostgreSQL、JWT、端口和 OCR Token 环境变量示例。
- 新增 GitHub Actions CI，自动执行前端构建、Go 测试、`go vet`、`Tracker.exe` 和 `Updater.exe` 构建。
- 新增 PostgreSQL、repository 实现、数据库设计和项目教程相关文档。

### Changed

- 将原 `handlers/`、`service/`、`store/`、`models/` 迁移到更标准的 Go 项目目录。
- service 层改为依赖 repository interface，业务逻辑不再直接读写 JSON 文件。
- PostgreSQL 模式下按登录用户 `user_id` 隔离科目、错题、设置、知识点、OCR 任务和备份导入导出数据。
- 前端请求统一携带 Cookie，并在收到 401 时尝试刷新登录状态后重试请求。
- 前端新增登录/注册页面；JSON 模式继续免登录，PostgreSQL 认证模式未登录时进入登录页。
- README 补充 PostgreSQL 模式、登录注册、环境变量和 JSON 导入 PostgreSQL 说明。
- TUTORIAL 和 PostgreSQL 文档更新为当前登录注册实现，移除“当前没有登录 API”的过时描述。
- 优化仪表盘和错题列表分隔条拖动体验。
- 调整 OCR 上传限制为 MinerU API 最大限制 200MB。
- 优化白天模式配色，降低纯白背景带来的刺眼感。
- 延长并优化日夜主题切换过渡，减少切换闪烁。

### Fixed

- 修复 Markdown 原始 HTML 可能被执行的问题。
- 修复错题接口 JSON 解析失败后仍继续执行业务逻辑的问题。
- 修复 8000 端口被占用时启动失败的问题，程序会自动切换到后续可用端口。
- 修复更新器在部分 Windows 环境下启动临时 updater 需要提权时失败的问题。
- 修复上传前 `tmp/` 临时目录未被忽略的问题，避免浏览器缓存、渲染临时文件进入 Git。
- 修复文档与当前登录注册实现不一致的问题。

### Security

- 禁用 Markdown 原始 HTML 渲染，降低 XSS 风险，同时保留 KaTeX 公式渲染。
- 为 OCR 上传、OCR 结果下载和备份导入增加大小限制。
- 新增基础安全响应头中间件。
- 密码使用 bcrypt 哈希保存，不保存明文密码。
- refresh token 只在数据库中保存 SHA-256 哈希，退出登录时撤销当前 refresh token。
- access token 和 refresh token 使用 HttpOnly Cookie 保存，前端不写入 `localStorage`。
- `.env` 和 `.env.local` 默认忽略，避免本地数据库密码、JWT 密钥和 OCR Token 被提交。

## [2026.06.28-1319] - 2026-06-28

### Added

- 新增 Go 原生自动更新链路，支持检查 GitHub Release、下载 `Tracker.zip`、备份数据、启动 `Updater.exe` 并重启应用。
- 新增 Go 原生更新器 `cmd/updater`，支持等待主程序退出、解压更新包、替换程序文件、失败回滚和写入更新日志。
- 新增发布脚本 `scripts/build-release.ps1`，可构建 `Tracker.exe`、`Updater.exe` 并生成 `Tracker.zip`。
- 新增根目录 `version.json`，用于记录版本、仓库、更新包名称和主程序名称。
- 新增源码运行脚本 `start.bat` / `start.ps1`。

### Changed

- 将项目主程序对外发布名称统一为 `Tracker.exe`。
- 前端源码完整保留在 `frontend/`，不再只依赖已有构建产物。
- README 调整为面向用户的使用文档，并保留技术栈说明。
- 侧边栏入口从“每日推送”改为“仪表盘”。
- 仪表盘“今日优先复习”支持点击整道题打开详情。

### Fixed

- 修复源码模式缺少 `Updater.exe` 时自动更新按钮状态不清晰的问题。
- 修复双击旧开发期 exe 访问错误路径时容易看到“接口不存在”的说明问题。
- 修复仪表盘待复习题目过多时页面需要整体下移才能看到设置入口的问题。
- 修复自动更新下载包格式校验和更新前数据备份流程。

### Security

- 自动更新前会生成 `data/backups/pre-update-*.zip` 数据备份。
- 更新包解压时校验路径，避免 zip 路径穿越。
- 更新器跳过 `data/`、`.git/`、`__pycache__/`，避免覆盖本地数据和开发目录。
