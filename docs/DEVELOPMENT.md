# 开发说明

项目采用 FastAPI + Vue 3。开发时 Vite 代理 `/api` 到后端，生产运行时由 FastAPI 托管构建后的前端静态文件。

## 目录结构

```text
.
├── backend/
│   ├── api.py             # FastAPI 接口和 SPA 托管
│   ├── mineru.py          # MinerU OCR 调用
│   └── update_service.py  # 自动更新检查、下载和启动 updater
├── frontend/
│   └── src/
│       ├── api/           # 前端 API 封装
│       ├── components/    # Vue 组件
│       ├── store/         # 前端状态
│       └── utils/         # Markdown、PDF 等工具
├── utils/
│   ├── data_store.py      # JSON 数据读写
│   ├── error_manager.py   # 错题和科目逻辑
│   └── daily_push.py      # 每日推送知识点
├── run.py                 # 主程序入口
├── updater.py             # 自动更新替换器入口
├── version.json           # 当前版本和更新源信息
├── start.ps1              # 源码启动脚本
└── scripts/
    └── build-release.ps1  # 发布打包脚本
```

## 后端接口

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

## 前端开发

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

因此前端开发时需要同时启动后端：

```powershell
python run.py
```

## 后端检查

```powershell
python -c "from backend.api import app; print('backend import ok')"
python -m compileall backend updater.py run.py
```

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
| 更新 | GitHub Releases + Updater.exe |
