# 11408 考研学习追踪器

专为 408 计算机考研设计的错题管理与学习辅助工具，支持 Markdown + LaTeX 编辑、OCR 识别、每日知识点推送、薄弱点分析及 PDF 错题导出。

## ✨ 功能

- **错题管理** — 多选题、简答题，Markdown + LaTeX 公式编辑，实时预览
- **OCR 识别** — 拍照/截图自动转 Markdown（需配置 [MinerU](https://mineru.net) Token）
- **每日推送** — 内置 80+ 条 408 知识点 + 考研英语高频词，每日随机推送
- **薄弱点分析** — 各科错题分布可视化，自动标记待复习题目
- **PDF 导出** — 一键导出排版精美的错题本 PDF
- **科目管理** — 支持自定义科目增删

## 🚀 快速开始

### 方式一：运行 EXE（推荐）

下载 [Releases](../../releases) 中的 `11408学习追踪器.exe`，双击运行，自动打开浏览器。

> EXE 会在同级目录创建 `data/` 文件夹存放数据，删除即重置。

### 方式二：源码运行

```bash
# 1. 安装 Python 依赖
pip install fastapi uvicorn pydantic requests

# 2. 构建前端
cd frontend
npm install
npm run build
cd ..

# 3. 启动
python backend/api.py
```

浏览器打开 http://127.0.0.1:8000

## 🔧 MinerU OCR 配置

1. 注册 [MinerU](https://mineru.net) 获取 API Token
2. 在应用左侧 ⚙️ 管理科目 → 粘贴 Token → 保存
3. 添加错题时点击 📷 OCR 导入，或 Ctrl+V 粘贴图片

## 🛠 技术栈

| 层 | 技术 |
|---|------|
| 前端 | Vue 3 + Vite |
| 后端 | FastAPI + Uvicorn |
| 公式 | KaTeX |
| OCR | MinerU API v4 |
| 打包 | PyInstaller |

## 📄 License

MIT
