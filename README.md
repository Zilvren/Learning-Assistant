# 错题追踪器

本地优先的个人学习资料库，用来管理文件夹、Markdown 笔记、学习文件和错题复习。

应用默认优先运行在本机：

```text
http://127.0.0.1:8000
```

如果 8000 端口已被占用，程序会自动改用后续可用端口，并打开对应地址。

默认使用本地 JSON 存储，数据保存在程序目录下的 `data/` 文件夹中，不需要账号，也不会把错题上传到远程服务器。

如果你需要多用户、登录注册或服务端部署，可以切换到 PostgreSQL 存储模式。

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
- 删除内容进入回收站，默认保留 30 天。

### 仪表盘

- 查看今日到期和逾期复习数量。
- 查看当前薄弱科目和复习建议。
- 在“今日优先复习”中点击错题即可查看完整题目、错解、正解和错因。
- 标记复习后会自动计算下一次复习日期。

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
- `config.json`：用户名、MinerU Token 等配置。
- `knowledge.json`：可选，自定义每日知识点。
- `backups/`：导入或更新前自动生成的恢复点。
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

## 登录与 PostgreSQL 模式

JSON 本地模式默认免登录，适合单机自用。

PostgreSQL 模式会启用登录注册，所有业务数据按用户隔离。启动前需要准备 PostgreSQL 数据库，并设置环境变量：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
$env:TRACKER_JWT_SECRET="请换成一段固定的强随机密钥"
go run .
```

相关环境变量：

| 变量 | 说明 |
| ---- | ---- |
| `TRACKER_STORAGE` | `json` 或 `postgres`，默认 `json` |
| `TRACKER_DATABASE_URL` | PostgreSQL 连接字符串，`postgres` 模式必填 |
| `TRACKER_JWT_SECRET` | 登录 Cookie/JWT 签名密钥，生产环境必须设置固定强密钥 |
| `TRACKER_HOST` | 服务监听地址，默认 `127.0.0.1` |
| `TRACKER_PORT` | 服务端口，默认 `8000` |
| `GIN_MODE` | Gin 运行模式，例如 `release` |

JSON 数据导入 PostgreSQL：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --dry-run
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --replace
```

`--dry-run` 只预览导入数量，`--replace` 会替换当前用户的数据。

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
