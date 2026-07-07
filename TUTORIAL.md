# 错题追踪器 Go + Vue 项目教程

本文是一份面向开发者和初学者的项目教程，用来帮助你从零理解、运行、修改、测试和发布当前的“错题追踪器”项目。

这份文档和 `README.md` 的定位不同：

| 文档 | 面向对象 | 主要内容 |
| --- | --- | --- |
| `README.md` | 普通用户 | 软件是什么、怎么双击使用、常见功能 |
| `TUTORIAL.md` | 开发者和毕设维护者 | 怎么跑源码、怎么看架构、怎么改后端、怎么改前端、怎么切 PostgreSQL、怎么打包 |
| `TUTORIAL.review.md` | 历史参考 | 旧阶段的详细评审和迁移记录，不作为当前主教程 |

本文会尽量遵循一个原则：

```text
每一步只做一件事。
做完以后立刻验证。
出错时先看错误在哪一层。
```

## 目录

- [1. 项目是什么](#1-项目是什么)
- [2. 开发环境准备](#2-开发环境准备)
- [3. 第一次运行源码](#3-第一次运行源码)
- [4. 目录结构和分层架构](#4-目录结构和分层架构)
- [5. 后端启动流程](#5-后端启动流程)
- [6. API 路由总览](#6-api-路由总览)
- [7. 前端结构和构建流程](#7-前端结构和构建流程)
- [8. JSON 本地存储模式](#8-json-本地存储模式)
- [9. PostgreSQL 存储模式](#9-postgresql-存储模式)
- [10. JSON 数据导入 PostgreSQL](#10-json-数据导入-postgresql)
- [11. 核心业务流程](#11-核心业务流程)
- [12. OCR 功能流程](#12-ocr-功能流程)
- [13. 备份和恢复](#13-备份和恢复)
- [14. 自动更新机制](#14-自动更新机制)
- [15. 如何开发一个新功能](#15-如何开发一个新功能)
- [16. 测试和验证](#16-测试和验证)
- [17. 打包发布](#17-打包发布)
- [18. Git 和提交注意事项](#18-git-和提交注意事项)
- [19. 常见问题排查](#19-常见问题排查)
- [20. 后续演进方向](#20-后续演进方向)
- [21. 源码阅读路线](#21-源码阅读路线)
- [22. 前后端联调排错方法](#22-前后端联调排错方法)
- [附录 A. 常用命令速查](#附录-a-常用命令速查)
- [附录 B. 常用环境变量](#附录-b-常用环境变量)
- [附录 C. 数据文件和目录说明](#附录-c-数据文件和目录说明)

## 1. 项目是什么

本项目是一个本地优先的错题管理系统。它的目标不是只做一个“记事本”，而是围绕学习复盘提供一条完整链路：

```text
记录错题 -> 标注科目和标签 -> 复习提醒 -> 仪表盘统计 -> OCR 辅助录入 -> 备份恢复 -> 自动更新
```

当前主要功能包括：

| 功能 | 说明 |
| --- | --- |
| 仪表盘 | 展示今日待复习、逾期复习、薄弱错题、知识点 |
| 错题管理 | 新增、编辑、删除、筛选错题 |
| 科目管理 | 创建和删除科目 |
| 标签管理 | 通过题目标签和错因标签筛选错题 |
| Markdown 渲染 | 题目、答案、错因支持 Markdown |
| 公式渲染 | 前端通过 KaTeX 渲染公式 |
| OCR | 使用 MinerU API 将图片识别成 Markdown |
| 备份恢复 | 导出和导入 zip 备份 |
| 自动更新 | 发布包通过 GitHub Releases 下载 `Tracker.zip` 并使用 `Updater.exe` 替换程序 |
| 双存储模式 | 默认 JSON 单机模式，也支持 PostgreSQL 模式 |

当前技术栈：

| 层 | 技术 |
| --- | --- |
| 前端 | Vue 3、Vite |
| 后端 | Go、Gin |
| 数据访问 | repository 接口 |
| 本地存储 | JSON 文件 |
| 数据库模式 | PostgreSQL、pgx、pgxpool |
| Markdown | markdown-it、markdown-it-mark |
| 公式 | KaTeX |
| OCR | MinerU API v4 |
| 更新 | GitHub Releases、Go Updater |

## 2. 开发环境准备

### 2.1 需要安装什么

源码运行至少需要：

| 工具 | 用途 | 是否必需 |
| --- | --- | --- |
| Go | 运行后端、构建 exe、运行测试 | 必需 |
| Node.js 和 npm | 构建 Vue 前端 | 必需 |
| Git | 查看版本、提交代码、推送 GitHub | 建议 |
| PostgreSQL | 使用数据库模式时需要 | 可选 |
| GitHub CLI `gh` | 自动创建 GitHub Release 时需要 | 可选 |

### 2.2 检查 Go

在 PowerShell 中执行：

```powershell
go version
```

能看到版本号就说明 Go 可用。

项目 `go.mod` 中声明：

```text
go 1.26.4
```

如果你本机 Go 版本不同，通常小版本差异问题不大。但如果构建失败，优先升级到项目要求附近的版本。

### 2.3 检查 Node.js 和 npm

```powershell
node -v
npm -v
```

能看到版本号即可。前端项目在：

```text
frontend/
```

如果 `npm install` 很慢，通常是网络或 npm 源问题，可以先单独进入前端目录排查。

### 2.4 检查 PostgreSQL

只有要使用 PostgreSQL 模式时才需要这一步。

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

这里开头的 `&` 是 PowerShell 的调用运算符。因为路径里有空格，PowerShell 需要用 `&` 来执行这个字符串路径。

如果你只想用默认 JSON 模式，不需要安装或启动 PostgreSQL。

## 3. 第一次运行源码

### 3.1 推荐方式：双击 `start.bat`

在项目根目录双击：

```text
start.bat
```

它会调用：

```text
start.ps1
```

`start.ps1` 会做这些事情：

```text
1. 检查 go 是否存在。
2. 检查 npm 是否存在。
3. 进入 frontend/ 执行 npm install。
4. 执行 npm run build 生成 frontend/dist。
5. 检查 8000 端口是否可用。
6. 如果 8000 被占用，自动找 8001、8002 等后续端口。
7. 执行 go run . -- --port 实际端口。
8. 默认自动打开浏览器。
```

启动成功后，浏览器会打开类似地址：

```text
http://127.0.0.1:8000
```

如果 8000 被占用，窗口会提示：

```text
Port 8000 is already in use. Starting on port 8001 instead.
```

此时应访问提示里的新端口。

### 3.2 PowerShell 手动启动

在项目根目录执行：

```powershell
.\start.ps1
```

如果你已经构建过前端，不想每次都执行 `npm install` 和 `npm run build`，可以执行：

```powershell
.\start.ps1 -SkipFrontendBuild
```

如果不想自动打开浏览器：

```powershell
.\start.ps1 -NoBrowser
```

如果想指定端口：

```powershell
.\start.ps1 -Port 8010
```

### 3.3 直接用 Go 启动

如果你确认 `frontend/dist` 已经存在，可以直接执行：

```powershell
go run .
```

指定端口：

```powershell
go run . -- --port 8010
```

不自动打开浏览器：

```powershell
go run . -- --no-browser
```

### 3.4 启动后如何验证

打开浏览器访问：

```text
http://127.0.0.1:8000/
```

再访问健康检查：

```text
http://127.0.0.1:8000/api/health
```

正确返回：

```json
{"status":"ok"}
```

再访问版本接口：

```text
http://127.0.0.1:8000/api/version
```

能看到版本、仓库、更新包名等信息。

### 3.5 为什么访问 `/api` 会显示“接口不存在”

后端只注册了明确的接口，例如：

```text
/api/health
/api/errors
/api/subjects
/api/version
```

如果直接访问：

```text
http://127.0.0.1:8000/api
```

或者访问不存在的：

```text
http://127.0.0.1:8000/api/xxx
```

后端会返回：

```json
{"detail":"接口不存在"}
```

这是正常现象。进入应用页面应该访问根路径：

```text
http://127.0.0.1:8000/
```

## 4. 目录结构和分层架构

当前项目采用 Go 后端分层架构。核心目录如下：

```text
server-go/
  api/
    handlers/              HTTP 处理层
  internal/
    service/               业务逻辑层
    repository/            数据访问接口和实现
      jsonrepo/            JSON 文件实现
      postgres/            PostgreSQL 实现
    model/                 请求、响应、领域模型
    middleware/            中间件
  pkg/
    config/                配置读取
    logger/                日志封装
  cmd/
    updater/               自动更新器
    import-json/           JSON 导入 PostgreSQL 工具
  frontend/                Vue 前端源码
  migrations/              PostgreSQL 建表 SQL
  scripts/                 构建和发布脚本
  data/                    本地数据，已被 .gitignore 忽略
  dist/                    构建产物，已被 .gitignore 忽略
```

### 4.1 分层依赖方向

项目的依赖方向是：

```mermaid
flowchart TD
  A["main.go<br/>启动和组装"] --> B["api/handlers<br/>HTTP 参数和响应"]
  B --> C["internal/service<br/>业务逻辑"]
  C --> D["internal/repository<br/>接口"]
  D --> E["jsonrepo<br/>JSON 文件"]
  D --> F["postgres<br/>PostgreSQL"]
  C --> G["internal/model<br/>数据结构"]
```

简单理解：

| 层 | 它负责什么 | 不应该负责什么 |
| --- | --- | --- |
| handler | 解析 HTTP 请求、返回 JSON 响应 | 不写复杂业务规则 |
| service | 处理业务流程，例如复习、OCR、备份、更新 | 不关心数据到底存在 JSON 还是 PostgreSQL |
| repository | 读写数据 | 不处理页面逻辑 |
| model | 定义结构体 | 不写业务流程 |
| middleware | 安全头、CORS 等请求中间处理 | 不写具体业务 |

### 4.2 为什么要这样分层

如果没有分层，handler 里会混在一起：

```text
解析请求
校验字段
计算复习日期
读写 JSON
拼响应
处理数据库错误
```

代码会很快变乱。

分层以后，每一层只处理自己的事情。比如以后从 JSON 换成 PostgreSQL，service 不需要大改，因为它依赖的是 repository 接口。

### 4.3 当前 repository 接口

接口定义在：

```text
internal/repository/interfaces.go
```

当前已有：

| 接口 | 负责 |
| --- | --- |
| `SubjectRepository` | 科目列表、创建、删除、替换 |
| `ErrorRepository` | 错题创建、查询、更新、删除、复习、标签 |
| `SettingsRepository` | 用户名、MinerU Token 等设置 |
| `KnowledgeRepository` | 仪表盘知识点 |
| `OCRTaskRepository` | OCR 任务记录 |
| `BackupRepository` | 备份导出和导入 |

repository 的组装结构是：

```go
type Repositories struct {
    Subjects  SubjectRepository
    Errors    ErrorRepository
    Settings  SettingsRepository
    Knowledge KnowledgeRepository
    OCRTasks  OCRTaskRepository
    Backup    BackupRepository
}
```

这意味着 service 层只需要拿到 `Repositories`，不用知道底层是哪种存储。

### 4.4 关键文件速查

刚开始读项目时，不建议从所有文件一起看。可以先按下面顺序读。

| 文件 | 先看原因 |
| --- | --- |
| `main.go` | 看程序如何启动、如何注册路由、如何选择存储 |
| `pkg/config/config.go` | 看环境变量和命令行参数怎么生效 |
| `internal/repository/interfaces.go` | 看业务层到底依赖哪些数据接口 |
| `internal/service/app.go` | 看 service 如何保存全局 repository 容器 |
| `internal/service/error_service.go` | 看错题核心业务 |
| `api/handlers/errors.go` | 看 HTTP 请求如何进入错题业务 |
| `frontend/src/api/index.js` | 看前端如何调用后端 |
| `frontend/src/components/ErrorList.vue` | 看错题列表和编辑交互 |

读代码时可以使用这个顺序：

```text
先看 main.go 的路由
再看对应 api/handlers 文件
再看 handler 调用哪个 service
再看 service 调用哪个 repository 接口
最后看 JSON 或 PostgreSQL repository 实现
```

例如要理解“新增错题”，阅读路线是：

```text
frontend/src/api/index.js
  -> api.addError(data)

main.go
  -> POST /api/errors

api/handlers/errors.go
  -> CreateError

internal/service/error_service.go
  -> CreateError 业务逻辑

internal/repository/interfaces.go
  -> ErrorRepository.Create

internal/repository/jsonrepo/repositories.go
  -> JSON 模式保存

internal/repository/postgres/error_repository.go
  -> PostgreSQL 模式保存
```

### 4.5 当前文件职责表

后端 handler：

| 文件 | 职责 |
| --- | --- |
| `api/handlers/subjects.go` | 科目增删查 |
| `api/handlers/errors.go` | 错题增删改查、复习、标签 |
| `api/handlers/daily.go` | 仪表盘数据 |
| `api/handlers/settings.go` | Token 和用户名设置 |
| `api/handlers/backup.go` | 备份导出和导入 |
| `api/handlers/ocr.go` | OCR 图片上传 |
| `api/handlers/update.go` | 版本检查和自动更新 |

后端 service：

| 文件 | 职责 |
| --- | --- |
| `internal/service/app.go` | 初始化 service 层使用的 repository 容器 |
| `internal/service/subject_service.go` | 科目业务 |
| `internal/service/error_service.go` | 错题业务和复习逻辑 |
| `internal/service/daily_service.go` | 仪表盘统计、薄弱错题和知识点 |
| `internal/service/settings_service.go` | 设置保存与读取 |
| `internal/service/backup_service.go` | 备份 zip 的导出和导入 |
| `internal/service/ocr_service.go` | MinerU OCR 调用 |
| `internal/service/update_service.go` | GitHub Release 检查、下载更新包、启动更新器 |
| `internal/service/update_process_windows.go` | Windows 下启动更新器 |
| `internal/service/update_process_other.go` | 非 Windows 下的兼容实现 |

数据访问层：

| 文件 | 职责 |
| --- | --- |
| `internal/repository/interfaces.go` | repository 接口定义 |
| `internal/repository/jsonrepo/repositories.go` | JSON 文件读写实现 |
| `internal/repository/postgres/db.go` | PostgreSQL 连接池和默认用户 |
| `internal/repository/postgres/subject_repository.go` | PostgreSQL 科目实现 |
| `internal/repository/postgres/error_repository.go` | PostgreSQL 错题、标签、复习记录实现 |
| `internal/repository/postgres/settings_repository.go` | PostgreSQL 设置实现 |
| `internal/repository/postgres/knowledge_repository.go` | PostgreSQL 知识点实现 |
| `internal/repository/postgres/ocr_task_repository.go` | PostgreSQL OCR 任务实现 |
| `internal/repository/postgres/backup_repository.go` | PostgreSQL 备份导入导出实现 |
| `internal/repository/postgres/util.go` | PostgreSQL 辅助函数 |

前端：

| 文件 | 职责 |
| --- | --- |
| `frontend/src/App.vue` | 应用主布局和页面切换 |
| `frontend/src/components/HomePage.vue` | 仪表盘 |
| `frontend/src/components/ErrorList.vue` | 错题列表、筛选、编辑弹窗 |
| `frontend/src/components/Sidebar.vue` | 侧边栏、科目、设置入口 |
| `frontend/src/components/MarkdownEditor.vue` | Markdown 编辑器 |
| `frontend/src/components/MarkdownRenderer.vue` | Markdown 和公式渲染 |
| `frontend/src/store/subjects.js` | 科目状态 |
| `frontend/src/store/settings.js` | 设置状态 |
| `frontend/src/utils/markdown.js` | Markdown 安全渲染配置 |
| `frontend/src/style.css` | 全局样式和主题 |

## 5. 后端启动流程

后端入口是：

```text
main.go
```

启动流程可以概括为：

```mermaid
sequenceDiagram
  participant User as 用户
  participant Start as start.ps1
  participant Main as main.go
  participant Repo as repository
  participant Service as service
  participant Gin as Gin 路由
  User->>Start: 双击 start.bat 或执行 start.ps1
  Start->>Start: 构建前端并查找可用端口
  Start->>Main: go run . -- --port 端口
  Main->>Main: 读取配置
  Main->>Repo: 根据 TRACKER_STORAGE 选择存储实现
  Main->>Service: service.Init(repos)
  Main->>Gin: 注册 API 和前端静态资源
  Gin-->>User: 打开浏览器访问应用
```

### 5.1 配置读取

配置读取在：

```text
pkg/config/config.go
```

支持环境变量和命令行参数。

常用命令行参数：

| 参数 | 作用 |
| --- | --- |
| `--port 8010` | 指定端口 |
| `--port=8010` | 指定端口 |
| `--host 127.0.0.1` | 指定监听地址 |
| `--no-browser` | 不自动打开浏览器 |

常用环境变量：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `TRACKER_HOST` | `127.0.0.1` | 服务监听地址 |
| `TRACKER_PORT` | `8000` | 默认端口 |
| `TRACKER_NO_BROWSER` | `false` | 是否不自动打开浏览器 |
| `TRACKER_STORAGE` | `json` | 存储模式 |
| `TRACKER_DATABASE_URL` | 空 | PostgreSQL 连接字符串 |
| `GIN_MODE` | 空 | Gin 运行模式 |

### 5.2 存储实现选择

`main.go` 中的 `setupRepositories` 会根据配置选择存储：

```text
TRACKER_STORAGE=json       使用 internal/repository/jsonrepo
TRACKER_STORAGE=postgres   使用 internal/repository/postgres
```

默认不设置环境变量时就是 JSON 模式。

### 5.3 前端静态资源

前端构建产物在：

```text
frontend/dist/
```

Go 通过 `embed.go` 把它嵌入程序。

如果请求不是 `/api/*`，后端会尝试从嵌入的前端文件中返回页面。这样打包成一个 `Tracker.exe` 后，用户不需要单独启动前端服务。

## 6. API 路由总览

当前路由注册在：

```text
main.go
```

### 6.1 健康检查

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/health` | 检查后端是否启动 |

### 6.2 科目接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/subjects` | 获取科目列表 |
| POST | `/api/subjects` | 新增科目 |
| DELETE | `/api/subjects/:name` | 删除科目 |

新增科目请求示例：

```json
{
  "name": "数学"
}
```

### 6.3 错题接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/errors` | 获取错题列表 |
| POST | `/api/errors` | 新增错题 |
| PUT | `/api/errors/:id` | 更新错题 |
| DELETE | `/api/errors/:id` | 删除错题 |
| PUT | `/api/errors/:id/review` | 标记复习 |
| GET | `/api/tags` | 获取标签 |

`GET /api/errors` 支持筛选参数：

| 参数 | 说明 |
| --- | --- |
| `subject` | 科目 |
| `keyword` | 关键词 |
| `tag` | 题目标签 |
| `reason_tag` | 错因标签 |

新增错题请求示例：

```json
{
  "subject": "数学",
  "title": "导数单调性",
  "question": "题目内容",
  "wrong": "错误答案",
  "correct": "正确答案",
  "reason": "忽略定义域",
  "tags": ["导数", "函数"],
  "reason_tags": ["审题", "定义域"]
}
```

### 6.4 仪表盘接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/daily-push` | 获取仪表盘数据 |

这个接口虽然路径里还叫 `daily-push`，但前端展示文案已经按“仪表盘”理解。后续如果想统一命名，可以新增 `/api/dashboard`，但要保留旧接口一段时间，避免前端或旧版本断掉。

### 6.5 设置接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/settings/token` | 获取 MinerU Token 状态 |
| PUT | `/api/settings/token` | 保存 MinerU Token |
| DELETE | `/api/settings/token` | 删除 MinerU Token |
| PUT | `/api/settings/username` | 保存用户名 |

### 6.6 备份接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/backup/export` | 导出备份 zip |
| POST | `/api/backup/import` | 导入备份 zip |

### 6.7 OCR 接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/ocr` | 上传图片并返回识别后的 Markdown |

### 6.8 更新接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/version` | 获取本地版本和更新能力 |
| GET | `/api/update/check` | 检查 GitHub Release 是否有新版本 |
| POST | `/api/update/apply` | 下载并应用更新 |

检查更新可以带参数：

```text
/api/update/check?force=true
```

## 7. 前端结构和构建流程

前端目录：

```text
frontend/
  src/
    api/
    components/
    store/
    App.vue
  package.json
  vite.config.js
```

### 7.1 前端技术

| 技术 | 用途 |
| --- | --- |
| Vue 3 | 构建界面 |
| Vite | 开发和构建工具 |
| markdown-it | Markdown 渲染 |
| markdown-it-mark | Markdown 高亮扩展 |
| KaTeX | 数学公式渲染 |

### 7.2 API 封装

前端请求封装在：

```text
frontend/src/api/index.js
```

这里把后端路径包装成函数，例如：

```text
api.getSubjects()
api.addError(data)
api.reviewError(id)
api.checkUpdate(force)
```

开发新页面时，不建议在组件里到处手写 `fetch('/api/xxx')`。更推荐先在 `api/index.js` 增加函数，再让组件调用。

### 7.3 构建前端

单独构建前端：

```powershell
cd frontend
npm install
npm run build
cd ..
```

构建完成后会生成：

```text
frontend/dist/
```

后端打包时会嵌入这个目录。

### 7.4 前端开发模式

如果只调前端，可以进入：

```powershell
cd frontend
npm run dev
```

但注意，Vite 开发服务器只负责前端页面，后端 API 仍然需要 Go 服务运行。

更简单的方式是使用项目根目录的：

```powershell
.\start.ps1
```

它会先构建前端再运行后端。

## 8. JSON 本地存储模式

### 8.1 默认就是 JSON 模式

不设置任何环境变量时：

```powershell
go run .
```

或：

```powershell
.\start.ps1
```

都会使用 JSON 模式。

JSON 模式会读写：

```text
data/
  errors.json
  subjects.json
  config.json
  knowledge.json
```

### 8.2 每个 JSON 文件存什么

| 文件 | 内容 |
| --- | --- |
| `data/errors.json` | 错题列表 |
| `data/subjects.json` | 科目列表 |
| `data/config.json` | 用户名、MinerU Token 等设置 |
| `data/knowledge.json` | 仪表盘知识点 |

### 8.3 为什么默认使用 JSON

原因有三个：

```text
1. 普通用户双击就能用，不需要数据库。
2. 本地数据透明，方便备份和排查。
3. 毕设阶段可以先展示完整功能，再逐步扩展 PostgreSQL 和多用户。
```

### 8.4 JSON 模式适合什么场景

适合：

```text
个人单机使用
课程演示
毕设功能演示
不想配置数据库的普通用户
```

不适合：

```text
多人同时使用
云端同步
复杂权限控制
大量数据并发写入
```

这些就是 PostgreSQL 模式后续要解决的问题。

## 9. PostgreSQL 存储模式

PostgreSQL 模式是为了后续服务端化、多用户和实习项目展示准备的。当前系统已经有 PostgreSQL repository 实现，但默认不启用。

### 9.1 为什么使用 pgx

项目使用：

```text
github.com/jackc/pgx/v5
```

原因：

| 原因 | 说明 |
| --- | --- |
| 原生 PostgreSQL 支持好 | 比通用 `database/sql` 更贴近 PostgreSQL |
| 自带连接池 | `pgxpool` 适合 Web 服务长期运行 |
| 支持 PostgreSQL 特性 | `RETURNING`、`jsonb`、事务等更自然 |
| 性能和生态成熟 | Go PostgreSQL 项目常用选择 |

### 9.2 建库

进入 PostgreSQL：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres
```

创建数据库和用户示例：

```sql
CREATE DATABASE study_tracker;
CREATE USER study_tracker_app WITH PASSWORD '你的密码';
GRANT ALL PRIVILEGES ON DATABASE study_tracker TO study_tracker_app;
```

连接到数据库：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker
```

执行建表脚本：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker -f .\migrations\001_init_postgres.sql
```

### 9.3 启用 PostgreSQL 模式

在 PowerShell 里设置环境变量：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go run .
```

或者用启动脚本：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
.\start.ps1 -SkipFrontendBuild
```

### 9.4 如何切回 JSON 模式

当前 PowerShell 窗口执行：

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
Remove-Item Env:TRACKER_DATABASE_URL -ErrorAction SilentlyContinue
go run .
```

也可以新开一个 PowerShell 窗口，因为上面设置的是当前窗口临时环境变量。

### 9.5 PostgreSQL 模式的用户是什么

PostgreSQL 表结构支持多用户，很多表都有：

```text
user_id
```

当前项目有两种运行方式：

```text
JSON 模式：本地单机免登录，直接读写 data/*.json
PostgreSQL 模式：启用登录注册，按登录用户的 user_id 隔离数据
```

因此 PostgreSQL repository 不能再固定使用某个默认用户，而是要从请求上下文中取出当前登录用户 ID，再创建对应的 repository。

### 9.6 PostgreSQL 表大致对应关系

| JSON 数据 | PostgreSQL 表 |
| --- | --- |
| `subjects.json` | `subjects` |
| `errors.json` | `error_problems`、`tags`、`error_problem_tags`、`review_records` |
| `config.json` | `user_settings` |
| `knowledge.json` | `knowledge_items` |
| OCR 临时状态 | `ocr_tasks` |
| 未来附件 | `attachments` |
| 登录刷新令牌 | `refresh_tokens` |

当前 `attachments` 仍是附件持久化预留表；`refresh_tokens` 已用于登录状态刷新和退出登录时撤销令牌。

### 9.7 PostgreSQL E-R 图

下面是当前 PostgreSQL 表之间的主要关系。为了让图更容易看，省略了一些时间字段和 metadata 字段。

```mermaid
erDiagram
  users ||--|| user_settings : has
  users ||--o{ subjects : owns
  users ||--o{ error_problems : owns
  users ||--o{ tags : owns
  users ||--o{ review_records : owns
  users ||--o{ knowledge_items : owns
  users ||--o{ ocr_tasks : owns
  users ||--o{ attachments : owns
  users ||--o{ refresh_tokens : owns

  subjects ||--o{ error_problems : classifies
  subjects ||--o{ knowledge_items : classifies
  error_problems ||--o{ review_records : has
  error_problems ||--o{ error_problem_tags : has
  tags ||--o{ error_problem_tags : links
  error_problems ||--o{ attachments : has
  ocr_tasks ||--o{ attachments : has

  users {
    bigint id PK
    varchar username
    varchar email
    text password_hash
    varchar status
  }

  subjects {
    bigint id PK
    bigint user_id FK
    varchar name
    integer sort_order
  }

  error_problems {
    bigint id PK
    bigint user_id FK
    bigint subject_id FK
    varchar title
    text question
    text wrong_answer
    text correct_answer
    text reason
    integer review_count
    date next_review_at
  }

  tags {
    bigint id PK
    bigint user_id FK
    varchar name
    varchar tag_type
  }

  error_problem_tags {
    bigint user_id FK
    bigint error_problem_id FK
    bigint tag_id FK
  }

  review_records {
    bigint id PK
    bigint user_id FK
    bigint error_problem_id FK
    integer review_no
    varchar result
  }

  user_settings {
    bigint user_id PK
    varchar display_name
    text mineru_token_cipher
  }

  knowledge_items {
    bigint id PK
    bigint user_id FK
    bigint subject_id FK
    text content
  }

  ocr_tasks {
    bigint id PK
    bigint user_id FK
    varchar provider
    varchar status
    text result_markdown
  }
```

### 9.8 为什么错题表要和用户表关联

在 PostgreSQL 模式中，错题表 `error_problems` 有：

```text
user_id
```

并且外键关联：

```text
users(id)
```

原因是以后要支持多用户。同一张错题表里会存很多用户的数据，如果没有 `user_id`，系统就不知道哪道题属于谁。

JSON 模式没有登录系统，数据天然属于当前本机用户。PostgreSQL 模式已经接入登录注册，请求进入业务接口前会由认证中间件解析当前用户，并把 `user_id` 写入请求上下文。

### 9.9 为什么 users 表有 password_hash

`users.password_hash` 用于保存登录注册时的密码哈希。

注意，它不应该保存明文密码。当前实现使用 bcrypt，将用户密码转换为哈希后再写入数据库：

```text
密码 + 随机盐 -> bcrypt 哈希 -> password_hash
```

登录时不会取出明文密码，而是用 bcrypt 对用户输入的密码和数据库中的 `password_hash` 做比对。

### 9.10 JSON 字段和 PostgreSQL 字段映射

错题在前端和 JSON 中是：

```json
{
  "id": 1,
  "subject": "数学",
  "title": "导数单调性",
  "question": "题目",
  "wrong": "错误答案",
  "correct": "正确答案",
  "reason": "错因",
  "tags": ["导数"],
  "reason_tags": ["审题"],
  "review_count": 0,
  "next_review": "2026-07-03"
}
```

PostgreSQL 中会拆开：

| 前端/JSON 字段 | PostgreSQL 字段或表 |
| --- | --- |
| `id` | `error_problems.id` |
| `subject` | `subjects.name`，写入时转成 `error_problems.subject_id` |
| `title` | `error_problems.title` |
| `question` | `error_problems.question` |
| `wrong` | `error_problems.wrong_answer` |
| `correct` | `error_problems.correct_answer` |
| `reason` | `error_problems.reason` |
| `tags` | `tags` + `error_problem_tags`，`tag_type='question'` |
| `reason_tags` | `tags` + `error_problem_tags`，`tag_type='reason'` |
| `review_count` | `error_problems.review_count` |
| `last_review` | `error_problems.last_reviewed_at` |
| `next_review` | `error_problems.next_review_at` |

这样设计的原因是：前端希望拿到简单对象，但数据库需要更规范的表结构，方便查询和统计。

## 10. JSON 数据导入 PostgreSQL

如果你之前已经在 JSON 模式下积累了数据，可以用导入工具迁移到 PostgreSQL。

工具入口：

```text
cmd/import-json/main.go
```

### 10.1 先 dry-run

dry-run 只统计，不写数据库：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --dry-run
```

你会看到类似：

```text
准备导入：subjects=3 errors=20 knowledge_items=5
dry-run 已完成，未写入数据库
```

如果这里报错，先不要真正导入。常见原因是：

```text
数据库连接字符串写错
数据库不存在
建表脚本没有执行
PostgreSQL 服务没有启动
```

### 10.2 确认后导入

如果目标 PostgreSQL 里没有旧数据：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
```

如果目标用户已经有数据，需要覆盖：

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --replace
```

`--replace` 会替换当前默认本地用户的数据。执行前建议先备份。

### 10.3 导入后验证

启用 PostgreSQL 模式运行：

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go run .
```

打开页面后检查：

```text
科目是否存在
错题是否存在
标签是否正常显示
仪表盘是否能看到复习数据
备份导出是否正常
```

也可以在 psql 中查：

```sql
SELECT id, username FROM users;
SELECT id, name FROM subjects;
SELECT id, title FROM error_problems ORDER BY id DESC LIMIT 10;
```

## 11. 核心业务流程

### 11.1 新增错题流程

```mermaid
sequenceDiagram
  participant UI as Vue 页面
  participant API as api/index.js
  participant Handler as api/handlers
  participant Service as internal/service
  participant Repo as repository
  participant Store as JSON 或 PostgreSQL
  UI->>API: 调用 addError(data)
  API->>Handler: POST /api/errors
  Handler->>Handler: 解析请求体
  Handler->>Service: CreateError
  Service->>Service: 校验科目、补时间、计算初始复习信息
  Service->>Repo: Errors.Create
  Repo->>Store: 写入 errors.json 或 error_problems
  Store-->>Repo: 返回保存结果
  Repo-->>Service: 返回错题
  Service-->>Handler: 返回业务结果
  Handler-->>UI: JSON 响应
```

这个流程体现了分层架构的好处：

```text
前端不关心数据存在哪里。
handler 不关心 JSON 和 PostgreSQL 的细节。
service 只处理业务规则。
repository 只处理存储读写。
```

### 11.2 复习错题流程

点击“已复习”后，前端调用：

```text
PUT /api/errors/:id/review
```

后端会更新：

```text
review_count
review_stage
last_review
next_review
```

在 PostgreSQL 模式下，还会插入：

```text
review_records
```

这对后续做统计分析很重要。比如以后可以统计：

```text
哪类错因复习最多
哪个科目遗忘最多
复习间隔是否合理
一周学习趋势如何
```

### 11.3 标签流程

错题有两类标签：

| 标签 | 字段 | 例子 |
| --- | --- | --- |
| 题目标签 | `tags` | 导数、函数、阅读理解 |
| 错因标签 | `reason_tags` | 审题、粗心、公式记错 |

JSON 模式下，这些标签直接存在每道错题中。

PostgreSQL 模式下，它们拆成：

```text
tags
error_problem_tags
```

并通过：

```text
tag_type = question
tag_type = reason
```

区分题目标签和错因标签。

### 11.4 仪表盘数据流程

仪表盘对应接口：

```text
GET /api/daily-push
```

它的业务重点不是简单返回所有错题，而是从错题中计算：

```text
总错题数
今日已复习数
今日到期数
逾期复习数
薄弱错题
知识点
复习建议
```

流程如下：

```mermaid
sequenceDiagram
  participant UI as HomePage.vue
  participant API as api.getDailyPush
  participant Handler as daily handler
  participant Service as daily service
  participant Repo as repository
  UI->>API: 加载仪表盘
  API->>Handler: GET /api/daily-push
  Handler->>Service: GetDailyPush
  Service->>Repo: 查询错题和知识点
  Repo-->>Service: 返回业务数据
  Service->>Service: 计算 due/overdue/weak/advice
  Service-->>Handler: DailyPushResult
  Handler-->>UI: JSON
```

如果仪表盘排版或数据不满意，通常要同时看：

| 问题 | 主要文件 |
| --- | --- |
| 数据不对 | `internal/service/daily_service.go` |
| 接口没返回 | `api/handlers/daily.go` |
| 页面排版不对 | `frontend/src/components/HomePage.vue` |
| 样式不对 | `frontend/src/style.css` |

### 11.5 备份导入流程

备份导入要特别小心，因为它会覆盖数据。

流程：

```mermaid
flowchart TD
  A["用户选择备份 zip"] --> B["POST /api/backup/import"]
  B --> C["handler 检查上传文件"]
  C --> D["service 校验 zip 大小和内容"]
  D --> E["先创建 pre-import 备份"]
  E --> F["repository.Import"]
  F --> G{"存储模式"}
  G --> H["JSON: 覆盖 data/*.json"]
  G --> I["PostgreSQL: 事务内替换当前用户数据"]
  H --> J["返回导入成功"]
  I --> J
```

导入类功能的原则：

```text
先校验，再备份，再覆盖。
```

不要为了省事直接覆盖原文件或原数据库数据。

## 12. OCR 功能流程

OCR 用于把图片识别成 Markdown，当前接入的是 MinerU API v4。

### 12.1 使用前配置 Token

打开应用设置页，保存 MinerU Token。

后端保存位置：

| 模式 | 保存位置 |
| --- | --- |
| JSON | `data/config.json` |
| PostgreSQL | `user_settings.mineru_token_cipher` |

当前 token 仍按旧版等价方式保存。后续如果要增强安全，可以增加 Windows DPAPI 或应用级加密。

### 12.2 OCR 调用流程

```mermaid
sequenceDiagram
  participant UI as 前端编辑弹窗
  participant Handler as OCR handler
  participant Service as OCR service
  participant MinerU as MinerU API
  participant Repo as OCRTaskRepository
  UI->>Handler: POST /api/ocr 上传图片
  Handler->>Service: OCRImage
  Service->>Repo: 创建 pending 任务
  Service->>MinerU: 上传文件并提交识别
  MinerU-->>Service: 返回 batch_id 和 task_id
  Service->>Repo: 更新 processing
  Service->>MinerU: 轮询识别结果
  MinerU-->>Service: 返回 Markdown
  Service->>Repo: 更新 succeeded
  Service-->>Handler: 返回 markdown
  Handler-->>UI: 插入当前字段
```

如果 OCR 失败，PostgreSQL 模式下会把失败状态写入 `ocr_tasks`，方便以后排查。

### 12.3 OCR 上传限制

后端会限制上传大小，防止特别大的文件占满内存。前端使用时建议上传截图或普通图片，不要上传超大扫描件。

如果遇到失败，优先检查：

```text
MinerU Token 是否正确
网络是否能访问 MinerU
图片大小是否过大
返回错误是否来自 MinerU
```

## 13. 备份和恢复

### 13.1 导出备份

接口：

```text
GET /api/backup/export
```

前端设置页会调用这个接口并下载 zip。

zip 中包含：

```text
errors.json
subjects.json
config.json
knowledge.json
```

### 13.2 导入备份

接口：

```text
POST /api/backup/import
```

导入前会自动生成恢复点：

```text
data/backups/pre-import-*.zip
```

### 13.3 JSON 和 PostgreSQL 模式下的区别

| 模式 | 导出 | 导入 |
| --- | --- | --- |
| JSON | 读取 `data/*.json` 后打 zip | 解压并覆盖 `data/*.json` |
| PostgreSQL | 从数据库组装出同样结构的 zip | 解析 zip 后用事务替换当前用户业务数据 |

这个设计的好处是：备份文件格式不随存储模式变化。也就是说，JSON 模式导出的备份，以后可以导入 PostgreSQL 模式。

## 14. 自动更新机制

自动更新只针对打包版。源码模式可以检查更新，但如果没有 `Updater.exe`，不能自动替换程序。

### 14.1 版本文件

根目录有：

```text
version.json
```

发布包中也会包含：

```text
version.json
```

典型内容：

```json
{
  "version": "0.0.0-dev",
  "repo": "Zilvren/Learning-Assitant",
  "asset_name": "Tracker.zip",
  "app_exe": "Tracker.exe"
}
```

### 14.2 检查更新

接口：

```text
GET /api/update/check?force=true
```

后端会访问：

```text
https://api.github.com/repos/Zilvren/Learning-Assitant/releases/latest
```

并寻找名称等于：

```text
Tracker.zip
```

的 release asset。

### 14.3 应用更新

接口：

```text
POST /api/update/apply
```

流程：

```mermaid
flowchart TD
  A["用户点击立即更新"] --> B["后端检查是否支持自动更新"]
  B --> C["下载 Tracker.zip 到 data/updates"]
  C --> D["校验 zip 格式"]
  D --> E["创建 data/backups/pre-update-*.zip"]
  E --> F["复制 Updater.exe 为临时 updater-run-*.exe"]
  F --> G["启动临时 updater"]
  G --> H["Tracker.exe 延迟退出"]
  H --> I["Updater 等待主程序退出"]
  I --> J["备份旧程序到 rollback-*"]
  J --> K["替换 Tracker.exe、Updater.exe、version.json 等文件"]
  K --> L["保留 data/"]
  L --> M["启动新版 Tracker.exe --no-browser"]
```

### 14.4 更新日志

更新器日志：

```text
data/updates/update.log
```

如果更新失败，先看这个文件。

常见问题：

| 问题 | 可能原因 |
| --- | --- |
| 下载失败 | GitHub 连接不稳定、代理只对浏览器生效、Go 程序未走代理 |
| 需要管理员权限 | 当前目录权限不足或 exe 被系统保护 |
| 版本号没变 | 实际启动的是另一个目录的旧 `Tracker.exe` |
| 自动更新不可用 | 当前目录没有 `Updater.exe` |

## 15. 如何开发一个新功能

新功能不要一上来就写前端。推荐按下面顺序做。

### 15.1 第一步：定义数据

看是否需要修改：

```text
internal/model/models.go
```

例如要给错题增加 `difficulty`，需要先想清楚：

```text
前端请求字段叫什么
后端 JSON 响应字段叫什么
JSON 文件里怎么保存
PostgreSQL 表里是否已有字段
旧数据没有这个字段时默认值是什么
```

### 15.2 第二步：改 repository 接口

接口在：

```text
internal/repository/interfaces.go
```

如果只是改已有错题字段，可能不需要新增接口方法。

如果是新资源，例如“学习计划”，可能需要新增：

```text
PlanRepository
```

### 15.3 第三步：实现 JSON repository

JSON 实现在：

```text
internal/repository/jsonrepo/
```

这里要保证默认单机模式继续可用。

验证方法：

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
go test ./...
go run .
```

然后在页面操作，看 `data/*.json` 是否正确变化。

### 15.4 第四步：实现 PostgreSQL repository

PostgreSQL 实现在：

```text
internal/repository/postgres/
```

如果需要新字段，先改：

```text
migrations/001_init_postgres.sql
```

然后实现对应 repository 方法。

如果要兼容旧数据库，后续还需要新增版本化迁移文件，例如：

```text
migrations/002_add_xxx.sql
```

### 15.5 第五步：写 service 业务逻辑

service 在：

```text
internal/service/
```

这里写业务规则。例如：

```text
新增错题时补 created 时间
复习时计算 next_review
OCR 成功后更新任务状态
导入备份时做格式校验
```

service 不应该直接读写 `data/*.json`，也不应该直接写 SQL。

### 15.6 第六步：写 handler

handler 在：

```text
api/handlers/
```

它负责：

```text
读取路径参数
读取 query 参数
解析 JSON body
调用 service
返回 JSON 或文件响应
```

handler 不要堆太多业务代码。

### 15.7 第七步：注册路由

路由在：

```text
main.go
```

新增接口后要在 `registerRoutes` 中注册。

### 15.8 第八步：改前端 API 封装

前端请求封装：

```text
frontend/src/api/index.js
```

先加 API 函数，再在组件里调用。

### 15.9 第九步：改前端页面

组件在：

```text
frontend/src/components/
```

状态管理在：

```text
frontend/src/store/
```

修改后执行：

```powershell
cd frontend
npm run build
cd ..
go run .
```

### 15.10 第十步：测试

至少执行：

```powershell
go test ./...
go build -o dist/Tracker.exe .
```

如果涉及前端：

```powershell
cd frontend
npm run build
cd ..
```

如果涉及 PostgreSQL：

```powershell
$env:TEST_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go test ./internal/repository/postgres
```

## 16. 测试和验证

### 16.1 后端单元测试

执行：

```powershell
go test ./...
```

默认测试不依赖 PostgreSQL。

### 16.2 PostgreSQL 集成测试

只有设置了 `TEST_DATABASE_URL` 才运行：

```powershell
$env:TEST_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go test ./internal/repository/postgres
```

集成测试主要验证：

```text
默认用户创建
科目 CRUD
错题 CRUD
标签组装
复习记录
设置保存
知识点保存
OCR task 状态
备份导入导出
```

### 16.3 构建后端

```powershell
go build -o dist/Tracker.exe .
```

构建更新器：

```powershell
go build -o dist/Updater.exe ./cmd/updater
```

### 16.4 构建前端

```powershell
cd frontend
npm install
npm run build
cd ..
```

### 16.5 手动验收清单

每次大改后建议按下面检查：

| 检查项 | 怎么验证 |
| --- | --- |
| 页面能打开 | 访问 `http://127.0.0.1:8000/` |
| 健康检查正常 | 访问 `/api/health` |
| 科目能新增 | 页面新增科目后刷新仍存在 |
| 错题能新增 | 新增错题后列表出现 |
| 筛选正常 | 按科目、标签、关键词筛选 |
| 复习正常 | 点击复习后下次复习日期变化 |
| 设置能保存 | 保存用户名或 Token 后刷新仍存在 |
| 备份能导出 | 设置页下载 zip |
| 备份能导入 | 导入后数据恢复 |
| OCR 能提示错误 | 未配置 Token 时能看到正确提示 |
| 更新接口正常 | `/api/version` 有返回 |

## 17. 打包发布

发布脚本在：

```text
scripts/build-release.ps1
```

### 17.1 本地打包

执行：

```powershell
.\scripts\build-release.ps1 -Version 2026.07.02-0001
```

它会：

```text
1. 构建前端。
2. 构建 dist/Tracker.exe。
3. 构建 dist/Updater.exe。
4. 生成发布目录 dist/release/Tracker-版本号。
5. 写入 version.json。
6. 写入 README-release.txt。
7. 压缩为 dist/release/Tracker.zip。
```

输出重点看：

```text
dist/Tracker.exe
dist/Updater.exe
dist/release/Tracker.zip
```

### 17.2 清理后打包

```powershell
.\scripts\build-release.ps1 -Version 2026.07.02-0001 -Clean
```

`-Clean` 会删除旧的 `dist/` 再重新构建。

### 17.3 上传 GitHub Release

脚本支持：

```powershell
.\scripts\build-release.ps1 -Version 2026.07.02-0001 -Upload
```

但只有在你明确要发布 Release 时才使用。它依赖 GitHub CLI：

```powershell
gh auth login
```

如果只是本地构建或写代码，不要加 `-Upload`。

### 17.4 发布包应该包含什么

`Tracker.zip` 中应该包含：

```text
Tracker.exe
Updater.exe
version.json
README-release.txt
```

不要把开发目录里的：

```text
data/
dist/
frontend/node_modules/
push.bat
push.ps1
TUTORIAL.review.md
```

放入发布包。

## 18. Git 和提交注意事项

当前 `.gitignore` 已经忽略：

```text
/data/
/dist/
/frontend/node_modules/
/push.bat
/push.ps1
/TUTORIAL.review.md
/TUTORIAL.review.md.bak
*.exe
*.zip
*.log
```

这些文件不要提交到 GitHub。

### 18.1 查看当前改动

```powershell
git status --short
```

### 18.2 查看将要提交的文件

```powershell
git diff --stat
```

### 18.3 提交前建议检查

```powershell
go test ./...
cd frontend
npm run build
cd ..
```

### 18.4 不要误提交的内容

尤其注意：

| 文件或目录 | 为什么不要提交 |
| --- | --- |
| `data/` | 里面是个人数据、Token、错题 |
| `dist/` | 构建产物，不是源码 |
| `frontend/node_modules/` | 依赖目录，非常大 |
| `push.bat` / `push.ps1` | 本地上传脚本，不适合作为项目源码 |
| `TUTORIAL.review.md` | 历史参考和评审文件，不作为正式文档 |
| `*.exe` | 可执行文件应放 Release，不放源码仓库 |

## 19. 常见问题排查

### 19.1 双击后显示“接口不存在”

原因通常是访问了：

```text
http://127.0.0.1:8000/api
```

正确入口是：

```text
http://127.0.0.1:8000/
```

`/api/*` 是给前端调用的接口，不是应用页面入口。

### 19.2 8000 端口被占用

现在项目已经有端口 fallback：

```text
8000 不可用 -> 尝试 8001 -> 8002 -> 最多尝试 20 个端口
```

启动窗口会提示实际端口。

如果你想手动查看谁占用端口：

```powershell
netstat -ano | findstr :8000
```

杀掉对应进程：

```powershell
taskkill /PID 进程ID /F
```

### 19.3 `npm install` 很慢或失败

优先检查网络。

可以单独进入前端目录重试：

```powershell
cd frontend
npm install
npm run build
cd ..
```

如果是缓存问题，可以尝试：

```powershell
npm cache verify
```

### 19.4 `frontend/dist/index.html not found`

说明前端还没构建。

执行：

```powershell
cd frontend
npm install
npm run build
cd ..
go run .
```

或者直接：

```powershell
.\start.ps1
```

### 19.5 PostgreSQL 提示数据库不存在

例如：

```text
database "study_tracker" does not exist
```

说明还没创建数据库。

执行：

```sql
CREATE DATABASE study_tracker;
```

再连接：

```powershell
& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -d study_tracker
```

### 19.6 PostgreSQL 模式启动失败

检查三件事：

```text
1. PostgreSQL 服务是否启动。
2. TRACKER_DATABASE_URL 是否正确。
3. migrations/001_init_postgres.sql 是否执行过。
```

临时切回 JSON：

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
Remove-Item Env:TRACKER_DATABASE_URL -ErrorAction SilentlyContinue
go run .
```

### 19.7 GitHub 更新下载慢，但浏览器很快

浏览器可能走了代理，但 Go 程序不一定走同一个代理。

常见情况：

```text
浏览器代理插件只对浏览器生效。
系统代理没有设置。
PowerShell 环境变量没有设置 HTTPS_PROXY。
```

可以在启动程序前设置：

```powershell
$env:HTTPS_PROXY="http://127.0.0.1:你的代理端口"
$env:HTTP_PROXY="http://127.0.0.1:你的代理端口"
.\Tracker.exe
```

端口要按你自己的代理软件实际端口填写。

### 19.8 更新器提示需要管理员权限

可能原因：

```text
程序放在需要管理员权限的目录。
杀毒软件拦截了临时 updater。
旧 exe 仍被占用。
Windows 对下载文件加了限制。
```

建议把程序放到普通用户目录，例如：

```text
C:\Users\你的用户名\Desktop\Tracker
```

不要放到：

```text
C:\Program Files
```

## 20. 后续演进方向

如果你把这个项目作为毕设和 Go 后端实习项目，后续可以按这个顺序继续增强。

### 20.1 登录注册和 JWT 鉴权

当前数据库已经有：

```text
users
refresh_tokens
```

其中 `users.password_hash` 用于保存 bcrypt 密码哈希，`refresh_tokens` 用于保存刷新令牌哈希。

当前已经实现：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/auth/register` | 注册 |
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/refresh` | 刷新 access token |
| POST | `/api/auth/logout` | 退出登录 |
| GET | `/api/auth/me` | 获取当前用户 |

PostgreSQL 模式下，repository 不再使用默认 `local` 用户，而是使用认证中间件解析出的真实 `user_id`。

### 20.2 多用户 PostgreSQL 模式

当前 PostgreSQL 表已经按 `user_id` 隔离。

当前请求流程是：

```text
handler 从登录态拿 user_id
service 带着 user_id 调用 repository
repository 查询时强制加 user_id 条件
```

### 20.3 云端同步

可以做两种方向：

| 方向 | 说明 |
| --- | --- |
| 单服务端模式 | 所有数据直接写 PostgreSQL |
| 本地加同步模式 | 本地仍可离线使用，联网后同步 |

如果只是毕设，单服务端模式更容易讲清楚。

### 20.4 对象存储和附件

以后图片、PDF、OCR 原图可以进入：

```text
attachments
```

文件本体可以放：

```text
本地磁盘
MinIO
阿里云 OSS
腾讯云 COS
```

数据库只保存文件元信息和 `storage_key`。

### 20.5 复习算法优化

现在复习更偏基础间隔复习。

后续可以加入：

```text
错题难度
遗忘次数
错因权重
科目薄弱程度
最近学习时间
```

然后动态计算下一次复习时间。

### 20.6 数据统计分析

可以增加：

```text
科目错题趋势
错因分布
复习完成率
一周学习热力图
高频知识点
```

这部分对毕设展示非常有帮助，也适合面试讲项目亮点。

### 20.7 Docker 部署

服务端化以后可以加入：

```text
Dockerfile
docker-compose.yml
PostgreSQL 容器
后端容器
Nginx 反向代理
```

这样能体现部署能力。

## 21. 源码阅读路线

如果你是为了毕设答辩或实习面试准备项目，建议不要只会说“我用了 Go 和 Vue”。更重要的是能讲清楚：请求如何流动、数据如何落库、为什么这样分层、遇到错误怎么定位。

### 21.1 第一轮：先跑起来

目标：确认项目能运行。

执行：

```powershell
.\start.ps1
```

验证：

```text
页面能打开
/api/health 返回 ok
能新增一个科目
能新增一道错题
```

这一轮不要急着改代码，只确认环境没问题。

### 21.2 第二轮：读启动链路

阅读：

```text
start.ps1
main.go
pkg/config/config.go
embed.go
```

你要能回答：

```text
为什么双击 start.bat 能启动？
为什么 8000 被占用会换端口？
为什么一个 Tracker.exe 就能显示前端页面？
为什么访问不存在的 /api 会返回接口不存在？
```

### 21.3 第三轮：读错题链路

阅读：

```text
frontend/src/components/ErrorList.vue
frontend/src/api/index.js
api/handlers/errors.go
internal/service/error_service.go
internal/repository/interfaces.go
internal/repository/jsonrepo/repositories.go
internal/repository/postgres/error_repository.go
```

你要能回答：

```text
新增错题时前端传了哪些字段？
handler 做了哪些参数解析？
service 做了哪些业务处理？
JSON 模式怎么保存？
PostgreSQL 模式怎么保存标签？
复习后 next_review 怎么变化？
```

### 21.4 第四轮：读存储切换

阅读：

```text
main.go
pkg/config/config.go
internal/repository/interfaces.go
internal/repository/jsonrepo/repositories.go
internal/repository/postgres/db.go
```

重点看：

```text
TRACKER_STORAGE=json 时如何创建 JSON repository
TRACKER_STORAGE=postgres 时如何创建 PostgreSQL repository
service 为什么不用知道具体存储类型
```

这部分是面试和毕设都比较好讲的亮点。

### 21.5 第五轮：读 PostgreSQL repository

阅读：

```text
migrations/001_init_postgres.sql
internal/repository/postgres/db.go
internal/repository/postgres/subject_repository.go
internal/repository/postgres/error_repository.go
internal/repository/postgres/backup_repository.go
```

你要能讲：

```text
为什么每张业务表都有 user_id？
为什么错题表 subject_id 要关联 subjects？
为什么标签拆成 tags 和 error_problem_tags？
为什么备份导出仍然输出 JSON zip？
为什么 users.password_hash 不保存明文密码？
```

### 21.6 第六轮：读 OCR 和更新

OCR：

```text
api/handlers/ocr.go
internal/service/ocr_service.go
internal/repository/postgres/ocr_task_repository.go
```

自动更新：

```text
api/handlers/update.go
internal/service/update_service.go
internal/service/update_process_windows.go
cmd/updater/main.go
scripts/build-release.ps1
version.json
```

你要能讲：

```text
OCR 为什么要保存任务状态？
自动更新为什么要先备份 data？
为什么 Updater.exe 要作为独立程序？
为什么 updater 运行前要复制成 updater-run-*.exe？
为什么源码模式通常不能自动更新？
```

### 21.7 一句话讲项目架构

面试或答辩时，可以这样讲：

```text
这个项目前端使用 Vue 3 和 Vite，后端使用 Go 和 Gin。
后端按照 handler、service、repository、model 分层，
service 只依赖 repository 接口，因此默认可以使用 JSON 本地存储，
也可以通过环境变量切换到 PostgreSQL。
项目还实现了 OCR、备份恢复和基于 GitHub Release 的自动更新。
```

不要只背这一段，真正答辩时老师可能会追问某个细节，所以前面的源码路线要至少走一遍。

## 22. 前后端联调排错方法

联调时不要凭感觉乱改。推荐按“浏览器 -> 前端 API -> 后端 handler -> service -> repository -> 数据”的顺序查。

### 22.1 第一步：看浏览器请求

打开浏览器开发者工具，进入 Network，操作页面后看请求。

重点看：

```text
请求路径是否正确
请求方法是 GET/POST/PUT/DELETE 哪个
状态码是 200、400、404 还是 500
响应 JSON 里有没有 detail
```

如果状态码是 404：

```text
检查 main.go 是否注册了这个路由。
检查前端 api/index.js 路径是否写错。
```

如果状态码是 400：

```text
通常是请求参数不对、字段缺失、文件太大或业务校验失败。
```

如果状态码是 500：

```text
通常是后端内部错误，例如数据库失败、文件读写失败、外部服务失败。
```

### 22.2 第二步：看前端 API 封装

前端 API 在：

```text
frontend/src/api/index.js
```

例如错题列表：

```text
getErrors
addError
updateError
deleteError
reviewError
```

如果页面按钮点了没反应，先看组件有没有调用这些函数。

### 22.3 第三步：看 handler

handler 常见问题：

| 问题 | 表现 | 处理 |
| --- | --- | --- |
| body 解析失败 | 返回 400 | 检查前端传的 JSON |
| id 转数字失败 | 返回 400 | 检查路径参数 |
| 上传文件为空 | 返回 400 | 检查 FormData 或 Blob |
| service 返回错误 | 返回 400 或 500 | 继续看 service |

handler 的职责是“接请求”和“回响应”，不要在 handler 里做很复杂的业务。

### 22.4 第四步：看 service

service 是业务层。比如：

```text
新增错题时科目是否存在
复习时间如何计算
OCR Token 是否存在
备份导入前是否生成恢复点
更新前是否检查 Updater.exe
```

如果功能逻辑不符合预期，优先看 service。

### 22.5 第五步：看 repository

如果业务逻辑是对的，但数据没有保存，检查 repository。

JSON 模式看：

```text
internal/repository/jsonrepo/repositories.go
data/*.json
```

PostgreSQL 模式看：

```text
internal/repository/postgres/
```

并在 psql 里查表：

```sql
SELECT * FROM subjects ORDER BY id DESC LIMIT 5;
SELECT * FROM error_problems ORDER BY id DESC LIMIT 5;
SELECT * FROM tags ORDER BY id DESC LIMIT 10;
```

### 22.6 第六步：判断是哪一层的问题

可以用这个表快速判断：

| 现象 | 优先检查 |
| --- | --- |
| 页面样式错位 | `frontend/src/style.css`、对应 Vue 组件 |
| 按钮没有发请求 | Vue 组件事件绑定 |
| 请求路径不对 | `frontend/src/api/index.js` |
| 404 接口不存在 | `main.go` 路由 |
| 400 参数错误 | handler 解析和前端请求体 |
| 500 内部错误 | service、repository、数据库、文件权限 |
| JSON 文件没变化 | JSON repository 或数据目录 |
| PostgreSQL 表没变化 | 连接字符串、repository SQL、事务 |
| 打包后页面不是最新 | 是否重新 `npm run build`，是否重新 `go build` |

### 22.7 改完以后如何确认没有破坏旧功能

每次改完至少做：

```powershell
go test ./...
cd frontend
npm run build
cd ..
go build -o dist/Tracker.exe .
```

然后手动检查：

```text
新增科目
新增错题
编辑错题
删除错题
复习错题
导出备份
导入备份
设置用户名
设置页版本信息
```

## 附录 A. 常用命令速查

### 启动

```powershell
.\start.ps1
.\start.ps1 -SkipFrontendBuild
.\start.ps1 -NoBrowser
.\start.ps1 -Port 8010
go run .
go run . -- --port 8010
```

### 前端

```powershell
cd frontend
npm install
npm run build
npm run dev
cd ..
```

### 后端测试和构建

```powershell
go test ./...
go build -o dist/Tracker.exe .
go build -o dist/Updater.exe ./cmd/updater
```

### 发布包

```powershell
.\scripts\build-release.ps1 -Version 2026.07.02-0001
.\scripts\build-release.ps1 -Version 2026.07.02-0001 -Clean
```

### PostgreSQL 模式

```powershell
$env:TRACKER_STORAGE="postgres"
$env:TRACKER_DATABASE_URL="postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable"
go run .
```

### 切回 JSON 模式

```powershell
Remove-Item Env:TRACKER_STORAGE -ErrorAction SilentlyContinue
Remove-Item Env:TRACKER_DATABASE_URL -ErrorAction SilentlyContinue
go run .
```

### JSON 导入 PostgreSQL

```powershell
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --dry-run
go run ./cmd/import-json --data-dir data --database-url "postgres://study_tracker_app:你的密码@localhost:5432/study_tracker?sslmode=disable" --replace
```

### Git 查看

```powershell
git status --short
git diff --stat
```

## 附录 B. 常用环境变量

| 环境变量 | 示例 | 说明 |
| --- | --- | --- |
| `TRACKER_HOST` | `127.0.0.1` | 后端监听地址 |
| `TRACKER_PORT` | `8000` | 默认端口 |
| `TRACKER_NO_BROWSER` | `true` | 不自动打开浏览器 |
| `TRACKER_STORAGE` | `json` 或 `postgres` | 存储模式 |
| `TRACKER_DATABASE_URL` | `postgres://...` | PostgreSQL 连接字符串 |
| `TRACKER_FRONTEND_DIR` | `frontend/dist` | 前端构建目录配置 |
| `GIN_MODE` | `release` | Gin 运行模式 |
| `TEST_DATABASE_URL` | `postgres://...` | PostgreSQL 集成测试连接字符串 |
| `HTTP_PROXY` | `http://127.0.0.1:7890` | HTTP 代理 |
| `HTTPS_PROXY` | `http://127.0.0.1:7890` | HTTPS 代理 |

## 附录 C. 数据文件和目录说明

| 路径 | 是否提交 Git | 说明 |
| --- | --- | --- |
| `data/errors.json` | 否 | 本地错题 |
| `data/subjects.json` | 否 | 本地科目 |
| `data/config.json` | 否 | 用户名和 Token |
| `data/knowledge.json` | 否 | 仪表盘知识点 |
| `data/backups/` | 否 | 导入和更新前备份 |
| `data/updates/` | 否 | 更新包、更新日志、临时 updater |
| `dist/Tracker.exe` | 否 | 构建出的主程序 |
| `dist/Updater.exe` | 否 | 构建出的更新器 |
| `dist/release/Tracker.zip` | 否 | 发布包，上传 Release 用 |
| `frontend/dist/` | 通常可以提交或随项目策略决定 | 当前 Go 会嵌入该目录 |
| `frontend/node_modules/` | 否 | npm 依赖目录 |
| `push.bat` | 否 | 本地推送脚本 |
| `push.ps1` | 否 | 本地推送脚本 |
| `TUTORIAL.review.md` | 否 | 历史评审参考 |

如果你不确定一个文件该不该提交，先执行：

```powershell
git status --short
```

再结合 `.gitignore` 判断。不要把个人数据、Token、构建产物和本地脚本提交到 GitHub。
