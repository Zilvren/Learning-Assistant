# 错题追踪器

一个本地优先的 Web 版错题管理工具。它把错题录入、Markdown + LaTeX 编辑、OCR 识别、标签筛选、每日复习提醒和打印导出放在同一个工作台里，适合长期整理自己的错题本。

数据默认保存在程序目录下的 `data/` 文件夹中，不依赖账号系统。

## 功能概览

### 错题管理

- 支持新增、编辑、删除错题。
- 每道错题包含科目、标题、题目、错答、正解、错因。
- 题目、错答、正解、错因均支持 Markdown 和 LaTeX。
- 内置编辑工具栏，支持加粗、斜体、删除线、标题、列表、链接、图片、代码块、行内公式和公式块。
- 支持 `\xcancel{}` 公式删除线，方便标记错误推导。
- OCR 生成的 base64 图片可在编辑器中折叠，避免大段图片数据影响编辑。

### 科目与标签

- 支持自定义科目，可在侧边栏“管理科目”中增删。
- 错题可设置题目标签和错因标签，标签用空格分隔。
- 列表支持按科目、标题/题目、题目标签、错因标签筛选。
- 点击错题卡片上的标签可快速切换到对应筛选。

### OCR 识别

- 基于 MinerU API v4，将截图或图片识别为 Markdown。
- 支持公式识别和图片内嵌。
- 可在添加或编辑错题时选择 OCR 插入目标字段。
- 支持在编辑弹窗中直接粘贴截图，识别结果会追加到当前字段。

### 复习与导出

- 首页展示每日复习建议、错题总数、今日到期、逾期复习和当前薄弱科目。
- 新增错题默认当天进入复习队列。
- 标记已复习后，会按艾宾浩斯遗忘曲线间隔自动计算下一次复习日期。
- 当前复习间隔为：当天、1 天、2 天、4 天、7 天、15 天、30 天、60 天。
- 到期或逾期错题会进入“今日优先复习”。
- 错题列表支持标记已复习，并显示下一次复习计划。
- 导出时可选择 PDF 样式：
  - `详细复盘`：完整展示题目、错答、正解、错因。
  - `紧凑打印`：压缩间距，适合大量错题省纸打印。
  - `练习自测`：先输出题目和答题区，答案解析集中放在后面。

### 界面设置

- 支持用户名设置。
- 支持 MinerU Token 配置和清除。
- 支持夜间模式，切换后会记住当前主题。
- 支持数据备份和导入恢复，可在设置中下载或导入备份包。
- 支持检查 GitHub Releases 新版本，打包版可自动下载、替换并重启。

## 快速开始

### 方式一：运行 EXE

下载 Releases 中的 `Tracker.zip`，解压后运行 `Tracker.exe`。

程序启动后会自动打开：

```text
http://127.0.0.1:8000
```

首次运行会在 EXE 同级目录创建 `data/` 文件夹：

```text
data/
  errors.json
  subjects.json
  config.json
```

这些文件保存错题、科目和配置。可以在“设置”中点击“备份数据”下载 zip 备份包，迁移到新设备时再通过“导入备份”恢复。导入前程序会自动把当前数据保存到 `data/backups/`，避免误覆盖后无法找回。

### 方式二：源码运行

需要提前安装：

- Python 3.10+
- Node.js

在项目根目录运行：

```powershell
.\start.ps1
```

脚本会安装 Python 依赖，必要时构建前端，然后启动后端服务。

也可以手动运行：

```powershell
pip install -r requirements.txt
cd frontend
npm install
npm run build
cd ..
python run.py
```

访问：

```text
http://127.0.0.1:8000
```

## OCR 配置

1. 前往 [MinerU](https://mineru.net) 注册并获取 API Token。
2. 打开应用左下角“设置”。
3. 将 Token 粘贴到 `MinerU Token` 输入框并保存。
4. 添加或编辑错题时点击 `OCR 插入`，也可以直接粘贴截图。

如果未配置 Token，OCR 功能会提示先完成设置。

## 开发说明

项目采用前后端分离开发，生产运行时由 FastAPI 托管构建后的前端静态文件。

```text
.
├── backend/
│   ├── api.py          # FastAPI 接口和 SPA 静态文件托管
│   └── mineru.py       # MinerU OCR 调用与结果解析
├── frontend/
│   ├── src/
│   │   ├── components/ # Vue 页面和组件
│   │   ├── store/      # 前端轻量状态
│   │   ├── utils/      # Markdown 渲染、PDF 导出
│   │   └── api/        # 前端 API 封装
│   └── vite.config.js  # Vite 配置，开发时代理 /api
├── utils/
│   ├── data_store.py   # JSON 文件读写
│   ├── error_manager.py# 错题和科目逻辑
│   └── daily_push.py   # 每日推送知识点
├── run.py              # 应用入口
├── start.ps1           # 源码启动脚本
└── requirements.txt
```

### 后端

主要接口位于 `backend/api.py`：

- `GET /api/subjects`
- `POST /api/subjects`
- `DELETE /api/subjects/{name}`
- `GET /api/errors`
- `POST /api/errors`
- `PUT /api/errors/{id}`
- `PUT /api/errors/{id}/review`
- `DELETE /api/errors/{id}`
- `GET /api/daily-push`
- `POST /api/ocr`
- `GET /api/settings/token`
- `PUT /api/settings/token`
- `DELETE /api/settings/token`
- `PUT /api/settings/username`
- `GET /api/backup/export`
- `POST /api/backup/import`
- `GET /api/version`
- `GET /api/update/check`
- `POST /api/update/apply`

### 前端开发

进入前端目录：

```powershell
cd frontend
npm install
npm run dev
```

Vite 默认运行在：

```text
http://127.0.0.1:5173
```

开发模式下 `/api` 会代理到：

```text
http://127.0.0.1:8000
```

因此前端开发时需要同时启动后端：

```powershell
python run.py
```

### 打包发布

项目提供发布脚本：

```powershell
.\scripts\build-release.ps1
```

该脚本会：

1. 安装并构建前端。
2. 使用 PyInstaller 构建 `Tracker.exe` 和 `Updater.exe`。
3. 写入发布包版本信息 `version.json`。
4. 生成发布目录和 `Tracker.zip`。

示例：

```powershell
.\scripts\build-release.ps1 -Version 1.0.0 -Upload
```

发布到 GitHub Releases 时，资源文件名保持为 `Tracker.zip`。应用会从 `Zilvren/Learning-Assitant` 的 latest release 检查该资源并执行自动更新。

自动更新会在设置页显示当前版本和检查结果。打包版发现新版本后，点击“立即更新”会下载 `Tracker.zip` 到 `data/updates/`，生成 `data/backups/pre-update-*.zip`，启动 `Updater.exe` 替换程序文件并重启；`data/` 目录不会被更新包覆盖。

## 数据说明

默认数据文件位于 `data/`：

- `errors.json`：错题列表。
- `subjects.json`：科目列表。
- `config.json`：MinerU Token、用户名等配置。
- `knowledge.json`：可选，自定义每日知识点；不存在时使用内置默认知识点。

设置中的备份功能会把以上本地数据打包为 zip。导入备份会覆盖当前对应数据；覆盖前会自动生成一份 `data/backups/pre-import-*.zip` 恢复点。

错题 ID 使用现有最大 ID 加 1 生成，删除旧错题后不会复用旧编号。
每道错题会记录 `review_count`、`last_review`、`review_stage` 和 `next_review`，用于艾宾浩斯复习调度。

## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | Vue 3 + Vite |
| 后端 | FastAPI + Uvicorn |
| Markdown | markdown-it |
| 公式渲染 | KaTeX |
| OCR | MinerU API v4 |
| 数据存储 | 本地 JSON 文件 |
| 打包 | PyInstaller |

## License

MIT
