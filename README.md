# 错题追踪器

本地优先的 Web 版错题管理工具，用来记录、复盘和定期复习错题。它把错题录入、Markdown/LaTeX 编辑、OCR 识别、标签筛选、艾宾浩斯复习提醒、PDF 导出、数据备份和自动更新放在同一个应用里。

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
- 当前复习间隔为：当天、1 天、2 天、4 天、7 天、15 天、30 天、60 天。
- 首页展示今日到期、逾期数量、薄弱科目和复习建议。

### OCR 识别

- 基于 MinerU API v4，将截图或图片识别为 Markdown。
- 支持公式识别和图片内嵌。
- 可在新增或编辑错题时选择 OCR 插入目标字段。
- 编辑弹窗中可直接粘贴截图，识别结果会追加到当前字段。

### PDF 导出

导出错题时可选择不同 PDF 样式：

- `详细复盘`：完整展示题目、错答、正解、错因。
- `紧凑打印`：压缩间距，适合大量错题省纸打印。
- `练习自测`：先输出题目和答题区，答案解析集中放在后面。

### 设置与安全

- 支持用户名设置。
- 支持 MinerU Token 配置和清除。
- 支持夜间模式。
- 支持数据备份和导入恢复。
- 支持检查 GitHub Releases 新版本；打包版可自动下载、替换并重启。

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

首次运行会在 `Tracker.exe` 同级目录创建：

```text
data/
  errors.json
  subjects.json
  config.json
```

这些文件就是你的本地错题数据。更新程序时不要删除 `data/`。

### 方式二：源码运行

需要：

- Python 3.10+
- Node.js

推荐运行：

```powershell
.\start.ps1
```

也可以手动运行：

```powershell
pip install -r requirements.txt
cd frontend
npm install
npm run build
cd ..
python run.py
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
5. 启动 `Updater.exe`。
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

## 常见问题

### 为什么设置页显示源码模式？

源码运行时没有 `Updater.exe`，所以只能检查更新，不能自动替换。只有打包后的 `Tracker.exe` 同级目录存在 `Updater.exe` 时，才会启用自动更新。

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

### 为什么 npm 提示 vulnerabilities？

这是前端依赖审计提示。它不一定影响本地运行或打包，但发布前可以单独评估是否需要升级依赖。

## 技术栈

| 层       | 技术                          |
| -------- | ----------------------------- |
| 前端     | Vue 3 + Vite                  |
| 后端     | FastAPI + Uvicorn             |
| Markdown | markdown-it                   |
| 公式渲染 | KaTeX                         |
| OCR      | MinerU API v4                 |
| 数据存储 | 本地 JSON 文件                |
| 打包     | PyInstaller                   |
| 更新     | GitHub Releases + Updater.exe |

## License

MIT
