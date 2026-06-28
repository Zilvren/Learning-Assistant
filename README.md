# 错题追踪器 Go 版

本地优先的 Web 版错题管理工具，用来记录、复盘和定期复习错题。Go 版后端使用 Gin，前端沿用 Vue 3 + Vite，发布时会把构建后的前端静态文件嵌入到 `Tracker.exe` 中。

应用默认运行在本机：

```text
http://127.0.0.1:8000
```

数据保存在程序目录下的 `data/` 文件夹中，不需要账号系统，也不会把错题上传到远程服务器。

## 主要功能

### 错题管理

- 新增、编辑、删除错题。
- 每道错题包含科目、标题、题目、错答、正解、错因。
- 支持 Markdown、LaTeX、代码块、图片和公式块。
- 错题 ID 使用当前最大 ID 加 1 生成，删除旧错题后不会复用旧编号。
- 首次启动时如果没有科目，会自动引导创建科目。

### 标签与筛选

- 自定义科目，可在侧边栏“管理科目”中增删。
- 支持题目标签和错因标签。
- 列表可按科目、关键词、题目标签、错因标签筛选。
- 点击错题卡片上的标签可快速切换筛选条件。

### 艾宾浩斯复习

- 新增错题默认当天进入复习队列。
- 标记已复习后，自动计算下一次复习日期。
- 首页展示今日到期、逾期数量、薄弱错题和复习建议。

### OCR 识别

- 基于 MinerU API v4，将截图或图片识别为 Markdown。
- 支持公式识别和图片内嵌。
- 可在新增或编辑错题时选择 OCR 插入目标字段。
- 编辑弹窗中可直接粘贴截图，识别结果会追加到当前字段。

### 设置与安全

- 支持用户名设置。
- 支持 MinerU Token 配置和清除。
- 支持夜间模式。
- 支持数据备份和导入恢复。
- 打包版支持从 GitHub Releases 检查新版本、下载更新包、自动替换并重启。

## 快速开始

### 方式一：运行发布包

下载 `Tracker.zip`，解压后双击：

```text
Tracker.exe
```

程序启动后会自动打开浏览器。如果浏览器没有自动打开，手动访问：

```text
http://127.0.0.1:8000
```

首次运行会在 `Tracker.exe` 同级目录创建或使用：

```text
data/
  errors.json
  subjects.json
  config.json
```

这些文件就是你的本地错题数据。更新程序时不要删除 `data/`。

### 方式二：源码运行

需要：

- Go
- Node.js 和 npm

推荐双击：

```text
start.bat
```

也可以手动运行：

```powershell
cd frontend
npm install
npm run build
cd ..
go run .
```

如果只想启动后端且不自动打开浏览器：

```powershell
.\start.bat -NoBrowser
```

如果前端已经构建过，可以跳过前端构建：

```powershell
.\start.bat -SkipFrontendBuild
```

## 前端开发

前端源码位于：

```text
frontend/
```

开发模式：

```powershell
cd frontend
npm install
npm run dev
```

Vite 默认地址：

```text
http://127.0.0.1:5173
```

开发模式下 `/api` 会代理到：

```text
http://127.0.0.1:8000
```

因此前端开发时需要同时启动 Go 后端：

```powershell
go run . -- --no-browser
```

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

## 自动更新

打包版会从 GitHub Releases 检查最新版本：

```text
Zilvren/Learning-Assitant
```

要求最新 Release 中存在资源：

```text
Tracker.zip
```

更新流程：

1. 设置页点击“检查更新”。
2. 如果发现新版本，点击“立即更新”。
3. 后端下载 `Tracker.zip` 到 `data/updates/`。
4. 更新前自动备份本地数据。
5. 主程序复制并启动 `Updater.exe` 临时副本。
6. 旧 `Tracker.exe` 退出。
7. `Updater.exe` 替换程序文件，但跳过 `data/`。
8. 新版 `Tracker.exe` 静默启动。
9. 原网页等待后端恢复，然后自动刷新。

更新日志位于：

```text
data/updates/update.log
```

如果更新后版本号不对，优先检查：

- `data/updates/update.log`
- 程序目录下的 `version.json`
- GitHub Release 中 `Tracker.zip` 内的 `version.json`

## 打包发布

本地打包：

```powershell
.\scripts\build-release.ps1
```

指定版本：

```powershell
.\scripts\build-release.ps1 -Version 2026.06.28-1200
```

脚本会：

1. 安装并构建前端。
2. 构建 `dist/Tracker.exe`。
3. 构建 `dist/Updater.exe`。
4. 写入发布包内的 `version.json`。
5. 生成 `dist/release/Tracker.zip`。

如果已登录 GitHub CLI，可以上传 Release：

```powershell
.\scripts\build-release.ps1 -Version 2026.06.28-1200 -Upload
```

## 常见问题

### 为什么设置页显示源码模式？

源码运行时通常没有同级 `Updater.exe`，所以只能检查更新，不能自动替换。只有打包后的 `Tracker.exe` 同级目录存在 `Updater.exe` 时，才会启用自动更新。

### 为什么双击后显示接口不存在？

请确认浏览器访问的是：

```text
http://127.0.0.1:8000/
```

如果访问 `/api` 或不存在的 `/api/*` 路径，后端会返回“接口不存在”。

### 为什么更新后版本号没变？

先看：

```text
data/updates/update.log
```

确认日志中是否出现：

```text
Payload version ...
Installed version ...
Update installed successfully
```

如果没有，说明 updater 没有完成替换。常见原因是旧程序还占用文件，或者解压到了旧目录但实际启动的是另一个目录里的 `Tracker.exe`。

### 为什么解压后看到旧数据？

`data/` 是本地数据目录，自动更新和手动解压都不应该随便覆盖它。如果要完全干净测试，请解压到一个全新的文件夹。

## 技术栈

| 层       | 技术                          |
| -------- | ----------------------------- |
| 前端     | Vue 3 + Vite                  |
| 后端     | Go + Gin                      |
| Markdown | markdown-it                   |
| 公式渲染 | KaTeX                         |
| OCR      | MinerU API v4                 |
| 数据存储 | 本地 JSON 文件                |
| 打包     | Go build + PowerShell         |
| 更新     | GitHub Releases + Go Updater  |

## License

MIT
