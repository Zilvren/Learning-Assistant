import sys, os, io
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
if sys.platform == 'win32':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

from fastapi import FastAPI, HTTPException, UploadFile, File, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, HTMLResponse
from fastapi.staticfiles import StaticFiles
from pydantic import BaseModel
from typing import Optional

from utils import error_manager
from utils.error_manager import (
    add_error, list_errors, review_error,
    delete_error, format_error, load_subjects, save_subjects, all_tags
)

from utils.stats import show_stats
from utils.daily_push import daily_push, get_knowledge_base
from utils.pdf_export import generate_pdf
from utils.data_store import load_json, save_json, today_str
from backend.mineru import ocr_image

app = FastAPI(title="错题追踪器")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)



# ============================================================
#  Models
# ============================================================

class AddErrorRequest(BaseModel):
    subject: str
    question: str
    title: str = ""
    wrong: str = "未记录"
    correct: str = "未记录"
    reason: str = "未记录"
    tags: list[str] = []
    reason_tags: list[str] = []
class UpdateErrorRequest(BaseModel):
    subject: Optional[str] = None
    title: Optional[str] = None
    question: Optional[str] = None
    wrong: Optional[str] = None
    correct: Optional[str] = None
    reason: Optional[str] = None
    tags: Optional[list[str]] = None
    reason_tags: Optional[list[str]] = None




# ============================================================
#  Endpoints
# ============================================================

@app.get("/api/subjects")
def api_get_subjects():
    return {"subjects": error_manager.SUBJECTS}


@app.post("/api/subjects")
def add_subject(data: dict):
    name = data.get("name", "").strip()
    if not name:
        raise HTTPException(400, "科目名称不能为空")
    subjects = error_manager.SUBJECTS[:]
    if name in subjects:
        raise HTTPException(400, "科目已存在")
    subjects.append(name)
    save_subjects(subjects)
    error_manager.SUBJECTS = subjects
    return {"subjects": subjects}


@app.delete("/api/subjects/{name}")
def remove_subject(name: str):
    subjects = error_manager.SUBJECTS[:]
    if name not in subjects:
        raise HTTPException(404, "科目不存在")
    subjects.remove(name)
    save_subjects(subjects)
    error_manager.SUBJECTS = subjects
    return {"subjects": subjects}


@app.get("/api/errors")
def get_errors(subject: Optional[str] = None, keyword: Optional[str] = None, tag: Optional[str] = None, reason_tag: Optional[str] = None):
    errors = list_errors(subject=subject, keyword=keyword, tag=tag, reason_tag=reason_tag)
    return {"errors": errors, "total": len(errors)}

@app.get("/api/tags")
def get_tags():
    return {"tags": all_tags()}


@app.post("/api/errors")
def create_error(req: AddErrorRequest):
    if req.subject not in error_manager.SUBJECTS:
        raise HTTPException(400, f"无效科目，可选：{', '.join(error_manager.SUBJECTS)}")
    if not req.question.strip():
        raise HTTPException(400, "题目不能为空")
    ok = add_error(req.subject, req.question, req.wrong, req.correct, req.reason, req.tags, req.title, req.reason_tags)
    if ok:
        errors = load_json("errors.json", [])
        return {"id": errors[-1]["id"], "message": "添加成功"}
    raise HTTPException(500, "添加失败")


@app.put("/api/errors/{error_id}/review")
def mark_review(error_id: int):
    errors = load_json("errors.json", [])
    for e in errors:
        if e["id"] == error_id:
            review_error(error_id)
            return {"message": f"错题 #{error_id} 已标记复习"}
    raise HTTPException(404, f"未找到错题 #{error_id}")


@app.put("/api/errors/{error_id}")
def update_error(error_id: int, req: UpdateErrorRequest):
    errors = load_json("errors.json", [])
    for e in errors:
        if e["id"] == error_id:
            if req.subject is not None:
                if req.subject not in error_manager.SUBJECTS:
                    raise HTTPException(400, f"无效科目")
                e["subject"] = req.subject
            if req.title is not None:
                e["title"] = req.title
            if req.question is not None:
                if not req.question.strip():
                    raise HTTPException(400, "题目不能为空")
                e["question"] = req.question
            if req.wrong is not None:
                e["wrong"] = req.wrong
            if req.correct is not None:
                e["correct"] = req.correct
            if req.reason is not None:
                e["reason"] = req.reason
            if req.tags is not None:
                e["tags"] = req.tags
            if req.reason_tags is not None:
                e["reason_tags"] = req.reason_tags
            save_json("errors.json", errors)
            return {"message": f"错题 #{error_id} 已更新"}
    raise HTTPException(404, f"未找到错题 #{error_id}")

@app.delete("/api/errors/{error_id}")
def remove_error(error_id: int):
    ok = delete_error(error_id)
    if ok:
        return {"message": f"错题 #{error_id} 已删除"}
    raise HTTPException(404, f"未找到错题 #{error_id}")


@app.get("/api/stats")
def get_stats():
    errors = load_json("errors.json", [])
    total = len(errors)
    reviewed = sum(1 for e in errors if e.get("review_count", 0) >= 2)
    review_rate = int(reviewed / max(total, 1) * 100)
    subject_counts = {s: 0 for s in error_manager.SUBJECTS}
    for e in errors:
        subject_counts[e["subject"]] = subject_counts.get(e["subject"], 0) + 1
    weak = [{"id": e["id"], "subject": e["subject"],
             "question": e["question"][:60],
             "review_count": e.get("review_count", 0)}
            for e in errors if e.get("review_count", 0) < 2]
    return {
        "total": total,
        "reviewed": reviewed,
        "review_rate": review_rate,
        "subject_counts": subject_counts,
        "weak_errors": weak,
    }


@app.get("/api/daily-push")
def get_daily_push():
    kb = get_knowledge_base()
    errors = load_json("errors.json", [])
    import random
    # Pick one tip per subject
    knowledge = {}
    for s in error_manager.SUBJECTS:
        if s in kb and kb[s]:
            knowledge[s] = random.choice(kb[s])
    weak = [{"id": e["id"], "subject": e["subject"],
             "question": e["question"][:60],
             "review_count": e.get("review_count", 0)}
            for e in errors if e.get("review_count", 0) < 2]
    total = len(errors)
    advice = ("当前错题量较少，保持刷题节奏，注意归纳总结" if total < 20 else
              "错题量适中，重点复习标记为概念不清的题目" if total < 50 else
              "错题量较大，建议暂停刷新题，集中复习旧错题")
    return {
        "date": today_str(),
        "total_errors": total,
        "reviewed": sum(1 for e in errors if e.get("review_count", 0) >= 2),
        "knowledge": knowledge,
        "weak_errors": weak,
        "advice": advice,
    }


@app.post("/api/export-pdf")
def export_pdf():
    errors = load_json("errors.json", [])
    if not errors:
        raise HTTPException(400, "暂无错题数据")
    pdf_path = os.path.join(os.path.dirname(__file__), "temp_report.pdf")
    ok = generate_pdf(pdf_path)
    if not ok:
        raise HTTPException(500, "PDF 生成失败")
    return FileResponse(pdf_path, media_type="application/pdf",
                        filename=f"错题报告_{today_str()}.pdf")


@app.post("/api/ocr")
async def ocr_endpoint(request: Request):
    """Convert uploaded image to Markdown using MinerU."""
    import asyncio, concurrent.futures
    try:
        body = await request.body()
        if not body:
            raise HTTPException(400, "No file uploaded")
        loop = asyncio.get_running_loop()
        with concurrent.futures.ThreadPoolExecutor() as pool:
            md_content = await loop.run_in_executor(pool, ocr_image, body, "ocr_upload.png")
        return {"markdown": md_content}
    except ValueError as e:
        raise HTTPException(400, str(e))
    except Exception as e:
        raise HTTPException(500, f"OCR failed: {str(e)}")


@app.get("/api/settings/token")
def get_settings_token():
    """Get current MinerU token (masked) and username."""
    try:
        config = load_json("config.json", default={})
        token = config.get("mineru_token", "").strip()
        masked = token[:8] + "***" + token[-4:] if len(token) > 12 else "***"
        return {"token": masked, "configured": bool(token), "username": config.get("username", "")}
    except:
        return {"token": "", "configured": False, "username": ""}


@app.put("/api/settings/token")
def set_settings_token(data: dict):
    """Set MinerU token."""
    token = data.get("token", "").strip()
    config = load_json("config.json", default={})
    config["mineru_token"] = token
    save_json("config.json", config)
    return {"message": "Token saved"}

@app.put("/api/settings/username")
def set_username(data: dict):
    name = data.get("name", "").strip()
    config = load_json("config.json", default={})
    config["username"] = name
    save_json("config.json", config)
    return {"message": "Username saved"}

# ── SPA 静态文件（放在 API 路由之后，避免拦截 /api/*）──
FRONTEND_DIR = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "frontend", "dist")

@app.get("/{full_path:path}", include_in_schema=False)
async def serve_spa(full_path: str, request: Request):
    """Serve frontend or fallback to index.html for SPA."""
    safe = full_path if not full_path.startswith("/") else full_path.lstrip("/")
    file_path = os.path.join(FRONTEND_DIR, safe)
    if os.path.isfile(file_path):
        return FileResponse(file_path)
    return FileResponse(os.path.join(FRONTEND_DIR, "index.html"))


def open_browser():
    import webbrowser, threading, time
    def _open():
        time.sleep(1.5)
        webbrowser.open("http://127.0.0.1:8000")
    threading.Thread(target=_open, daemon=True).start()


if __name__ == "__main__":
    import uvicorn
    open_browser()
    uvicorn.run(app, host="127.0.0.1", port=8000)
