#!/usr/bin/env python3
"""11408 考研学习追踪器 - EXE 入口"""
import sys, os

# PyInstaller 打包后需要找到数据目录
if getattr(sys, 'frozen', False):
    os.chdir(os.path.dirname(sys.executable))
else:
    os.chdir(os.path.dirname(os.path.abspath(__file__)))

sys.path.insert(0, os.getcwd())

# 首次运行自动创建空数据文件
import json
DATA_DIR = os.path.join(os.getcwd(), "data")
os.makedirs(DATA_DIR, exist_ok=True)
_defaults = {
    "errors.json": "[]",
    "subjects.json": '["数据结构","计算机组成原理","操作系统","计算机网络","数学","英语"]',
    "config.json": '{"mineru_token":""}',
}
for fname, content in _defaults.items():
    path = os.path.join(DATA_DIR, fname)
    if not os.path.exists(path):
        with open(path, "w", encoding="utf-8") as f:
            f.write(content)

from backend.api import app, open_browser
import uvicorn

if __name__ == "__main__":
    open_browser()
    uvicorn.run(app, host="127.0.0.1", port=8000)
