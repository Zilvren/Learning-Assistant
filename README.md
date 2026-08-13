# 学习追踪器

本地优先的个人学习资料库，用来管理文件夹、Markdown 笔记、学习文件和错题复习。

应用默认优先运行在本机：

```text
http://127.0.0.1:8000
```

如果 8000 端口已被占用，程序会自动改用后续可用端口，并打开对应地址。

默认使用本地 JSON 存储，数据保存在程序目录下的 `data/` 文件夹中，不需要账号，也不会把错题上传到远程服务器。

如果你需要多用户、登录注册或服务端部署，可以切换到 PostgreSQL 存储模式。

服务器 Docker 部署、更新 Bug、备份与安全检查请见：[服务器部署与日常维护指南](docs/deployment.md)。

准备接手开发或运维本项目，请先阅读：[项目交接手册](docs/handover.md)。

想从源码理解项目，可按：[项目阅读指南](docs/project-reading-guide.md) 的顺序阅读。

## 快速开始

下载 `Tracker.zip`，解压后双击：

```text
Tracker.exe
```

程序启动后会自动打开浏览器。如果没有自动打开，可以手动访问：

```text
http://127.0.0.1:8000
```

如果启动窗口提示 8000 已被占用，请按窗口里显示的新端口访问。

首次运行会在程序同级目录创建或使用：

```text
data/
  errors.json
  subjects.json
  config.json
  library.json
  blobs/
```

更新程序时不要删除 `data/`。

## 源码运行

已经安装 Go 和 Node.js 后，可以直接运行：

```powershell
.\start.bat
```

也可以手动启动：

```powershell
cd frontend
npm install
npm run build
cd ..
go run .
```

默认仍使用 JSON 本地模式。源码模式没有 `Updater.exe` 时，可以检查更新，但不能执行自动替换。

## 主要功能

### 个人资料库

- 建立任意层级的文件夹，统一整理笔记、错题和学习文件。
- 支持 Markdown、纯文本、图片、PDF 与常见 Office 文档，单文件最大 200MB。
- Markdown 笔记自动保存，并保留最近 50 个检查点版本。
- 支持搜索、网格/列表视图、拖拽移动、复制、置顶和回收站恢复。
- 在资料库首页输入至少两个字，可跨笔记正文、文件信息和错题内容检索，并显示命中片段。
- DOCX、XLSX、PPTX 支持安全的只读文本预览；原文件仍可下载。
- 笔记与错题可双向关联，复习时可快速跳回相关知识点。
- 删除内容进入回收站，默认保留 30 天；移入回收站的文件夹只显示为一个项目，恢复或永久删除会同时作用于其中全部内容。

### 仪表盘

- 查看今日到期和逾期复习数量。
- 查看当前薄弱科目和复习建议。
- 在“今日优先复习”中点击错题即可查看完整题目、错解、正解和错因。
- 标记复习后会自动计算下一次复习日期。
- “学习记录”以全年热力图展示学习活动，支持横向滑动浏览、切换年份和查看指定日期；当年默认定位在今天靠右的位置。
- 可设定每日复习、专注和笔记目标，使用专注计时器，并查看最近 7 天学习汇总。
- 统一复习收件箱会合并到期笔记与错题；可按“忘了 / 有点难 / 掌握 / 很轻松”调节下一次复习间隔。

### 错题管理

- 新增、编辑、删除错题。
- 每道错题包含科目、标题、题目、错答、正解、错因。
- 支持 Markdown、LaTeX、代码块、图片和公式块。
- 支持题目标签和错因标签。
- 可以按科目、关键词、题目标签、错因标签筛选。

### 科目与标签

- 可在侧边栏“管理科目”中增删科目。
- 点击错题卡片上的标签可快速筛选同类错题。
- 首次使用时，如果还没有科目，应用会引导你先创建一个科目。

### OCR 识别

- 基于 MinerU API v4，将截图或图片识别为 Markdown。
- 支持公式识别和图片内嵌。
- 可在新增或编辑错题时选择 OCR 插入目标字段。
- 编辑弹窗中可直接粘贴截图，识别结果会追加到当前字段。
- 上传后会先加入后台任务队列；即使离开页面也会继续执行，失败任务可在设置中心重试。

### AI 学习助手 🤖

- 在侧栏打开“AI 助手”，可让 DeepSeek 根据你的问题分析关联的笔记、纯文本、Office 文档文本预览、错题和学习进度。
- 每次请求只发送与问题关联、长度受限的文本上下文；不会发送图片或 PDF 原件、附件二进制内容、密码或 API Key。
- 回答会列出本次参考的资料或错题，可一键回到对应内容核对。

## AI 配置

1. 在 [DeepSeek 开放平台](https://platform.deepseek.com/api_keys) 创建 API Key。
2. 打开应用“设置” → “AI 学习助手”，粘贴并保存 Key。
3. 从侧栏进入“AI 助手”即可开始聊天。

Key 会加密存储，页面只显示是否已配置。设置页可选择 `deepseek-v4-flash`（默认、更快）或 `deepseek-v4-pro`（更强）；已保存的选择优先于环境变量 `DEEPSEEK_MODEL`。也可以通过环境变量 `DEEPSEEK_API_KEY` 提供 Key。

## OCR 配置

1. 前往 [MinerU](https://mineru.net) 获取 API Token。
2. 打开应用左下角“设置”。
3. 将 Token 粘贴到 `MinerU Token` 输入框并保存。
4. 添加或编辑错题时使用 `OCR 插入`。

未配置 Token 时，OCR 功能会提示先完成设置。

## 数据备份与恢复

数据默认位于：

```text
data/
```

常见文件：

- `errors.json`：错题列表。
- `subjects.json`：科目列表。
- `config.json`：用户名、MinerU Token、DeepSeek API Key 等配置；敏感 Key 以加密形式保存且不会出现在导出备份中。
- `knowledge.json`：可选，自定义每日知识点。
- `library.json`：资料库的文件夹、笔记和文件索引。
- `blobs/`：资料库上传文件及 Markdown 笔记内容。
- `backups/`：导入、更新前以及每日自动生成的恢复点；默认保留最近 14 份每日备份。
- `updates/`：自动更新下载包和更新日志。

在应用“设置”中可以：

- 点击“备份数据”下载 zip 备份包。
- 点击“导入备份”恢复备份。

导入备份会覆盖当前对应数据。覆盖前程序会自动生成：

```text
data/backups/pre-import-*.zip
```

自动更新前也会生成：

```text
data/backups/pre-update-*.zip
```

本地 JSON 模式默认开启每日自动备份。可按需调整：

```powershell
$env:TRACKER_AUTO_BACKUP="true"       # 设为 false 可关闭
$env:TRACKER_AUTO_BACKUP_INTERVAL="24h"
$env:TRACKER_AUTO_BACKUP_KEEP="14"
```

## 登录与 PostgreSQL 模式

JSON 本地模式默认免登录，适合单机自用。

PostgreSQL 模式会启用登录注册，所有业务数据按用户隔离。启动前需要准备 PostgreSQL 数据库，并设置环境变量：

```powershell
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
$env:TRACKER_JWT_SECRET="请换成一段固定的强随机密钥"
.\start-postgres.ps1 -DatabaseUrl $env:TRACKER_DATABASE_URL
```

该脚本会强制使用 PostgreSQL；连接配置缺失或错误时不会回退到 JSON 模式。`TRACKER_DATABASE_URL` 必须是包含主机和数据库名的 `postgres://` 或 `postgresql://` 地址，生产环境的 `TRACKER_JWT_SECRET` 至少应为 32 个字符。数据库迁移会在应用启动时从内嵌的 SQL 文件自动、按版本执行。

相关环境变量：

| 变量 | 说明 |
| ---- | ---- |
| `TRACKER_STORAGE` | `json` 或 `postgres`，默认 `json` |
| `TRACKER_DATABASE_URL` | PostgreSQL 连接字符串，`postgres` 模式必填 |
| `TRACKER_REQUIRE_POSTGRES` | 为 `true` 时禁止服务意外以 JSON 模式启动；Docker 部署已默认开启 |
| `TRACKER_DATA_DIR` | JSON 数据与本地 Blob、恢复点的目录，默认 `data` |
| `TRACKER_AUTO_BACKUP` | JSON 本地模式是否自动创建每日恢复点，默认 `true` |
| `TRACKER_AUTO_BACKUP_INTERVAL` | 自动备份检查周期，默认 `24h`，最小 `1h` |
| `TRACKER_AUTO_BACKUP_KEEP` | 保留的每日自动恢复点数量，默认 `14` |
| `TRACKER_JWT_SECRET` | 登录 Cookie/JWT 签名密钥，PostgreSQL 模式至少 32 个字符；生产环境必须设置固定强密钥 |
| `TRACKER_EMAIL_VERIFICATION_ENABLED` | 设为 `true` 后，新用户必须先验证邮箱才可登录 |
| `TRACKER_PUBLIC_URL` | 生产环境对外访问地址，例如 `https://study.example.com`，用于生成验证链接 |
| `TRACKER_SMTP_HOST` / `TRACKER_SMTP_PORT` | SMTP 主机与端口；端口默认 `465` |
| `TRACKER_SMTP_USERNAME` / `TRACKER_SMTP_PASSWORD` | SMTP 登录账号和授权码；两者需同时设置 |
| `TRACKER_SMTP_FROM` | 发件人邮箱地址，启用邮箱验证时必填 |
| `TRACKER_SMTP_TLS_MODE` | `implicit`（默认）、`starttls` 或 `none`；使用 SMTP 账号时必须启用 TLS |
| `TRACKER_HOST` | 服务监听地址，默认 `127.0.0.1` |
| `TRACKER_PORT` | 服务端口，默认 `8000` |
| `GIN_MODE` | Gin 运行模式，例如 `release` |

在 PostgreSQL 模式下，在设置页选择已导出的 ZIP 备份即可恢复资料库、笔记版本、附件、错题、标签和设置。系统会先创建 `pre-import` 快照，恢复完成后自动打开资料库。

服务器端自动化场景也可以使用命令行导入：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --dry-run
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --replace
```

`--dry-run` 只预览导入数量，`--replace` 会替换当前用户的数据。

### 启用注册邮箱验证（可选）

邮箱验证仅在 PostgreSQL 模式下生效。配置可用 SMTP 后设置：

```powershell
$env:TRACKER_EMAIL_VERIFICATION_ENABLED="true"
$env:TRACKER_PUBLIC_URL="https://你的域名"
$env:TRACKER_SMTP_HOST="smtp.example.com"
$env:TRACKER_SMTP_PORT="465"
$env:TRACKER_SMTP_USERNAME="你的邮箱账号"
$env:TRACKER_SMTP_PASSWORD="邮箱授权码"
$env:TRACKER_SMTP_FROM="你的邮箱账号"
$env:TRACKER_SMTP_TLS_MODE="implicit"
```

如果不启用邮箱验证，无需配置 SMTP。生产环境应使用能从公网访问的 `TRACKER_PUBLIC_URL`，否则邮件中的验证链接无法打开。

## 接口排错与审计日志

每次请求都会返回 `X-Request-ID`。若接口报错，请一并提供这个值和服务端对应的 `[AUDIT]` / `[ERROR]` 日志，便于定位问题。API 错误同时兼容旧字段 `detail`，并提供稳定的 `error.code`、`error.message` 与 `request_id`；5xx 错误不会把数据库连接或内部堆栈暴露给浏览器。

## 自动更新

打包版会从 GitHub Releases 检查最新版本。更新流程：

1. 打开“设置”。
2. 点击“检查更新”。
3. 如果发现新版本，点击“立即更新”。
4. 应用会自动备份数据、下载更新包、替换程序并重启。
5. 页面会等待新版程序启动后自动刷新。

更新日志位于：

```text
data/updates/update.log
```

## 常见问题

### 为什么设置页显示源码模式？

这表示当前运行目录中没有可用于自动替换的 `Updater.exe`。发布包正常解压后，`Tracker.exe` 和 `Updater.exe` 应该位于同一目录。

### 为什么双击后显示接口不存在？

请确认浏览器访问的是：

```text
http://127.0.0.1:8000/
```

如果访问 `/api` 或不存在的 `/api/*` 路径，后端会返回“接口不存在”。

### 为什么更新后版本号没变？

先查看：

```text
data/updates/update.log
```

如果没有出现 `Update installed successfully`，说明更新器没有完成替换。常见原因是旧程序还在运行，或者实际启动的是另一个目录里的 `Tracker.exe`。

### 为什么解压后看到旧数据？

`data/` 是本地数据目录，自动更新和手动解压都不会随便覆盖它。如果要完全干净测试，请解压到一个全新的文件夹。

## 技术栈

| 层       | 技术                         |
| -------- | ---------------------------- |
| 前端     | Vue 3 + Vite                 |
| 后端     | Go + Gin                     |
| Markdown | markdown-it                  |
| 公式渲染 | KaTeX                        |
| OCR      | MinerU API v4                |
| 数据存储 | 本地 JSON 文件 / PostgreSQL  |
| 数据库驱动 | pgx / pgxpool              |
| 认证     | HttpOnly Cookie + refresh token |
| 更新     | GitHub Releases + Go Updater |

## License

MIT
