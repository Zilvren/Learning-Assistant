import sys, os, io
import json
import zipfile
from datetime import datetime
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
if sys.platform == 'win32':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

from fastapi import FastAPI, HTTPException, Request
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, StreamingResponse
from pydantic import BaseModel
from typing import Optional

from utils import error_manager
from utils.error_manager import (
    add_error, list_errors, review_error,
    delete_error, save_subjects, all_tags
)

from utils.daily_push import get_knowledge_base
from utils.data_store import DATA_DIR, load_json, save_json, today_str
from backend.mineru import ocr_image
from backend import update_service

app = FastAPI(title="错题追踪器")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

BACKUP_FILES = {"errors.json", "subjects.json", "config.json", "knowledge.json"}
BACKUP_MAX_FILE_SIZE = 10 * 1024 * 1024


def validate_backup_data(name: str, data):
    if name == "errors.json" and not isinstance(data, list):
        raise HTTPException(400, "errors.json 数据结构不正确")
    if name == "errors.json" and any(not isinstance(item, dict) for item in data):
        raise HTTPException(400, "errors.json 数据结构不正确")
    if name == "subjects.json" and not isinstance(data, list):
        raise HTTPException(400, "subjects.json 数据结构不正确")
    if name == "subjects.json" and any(not isinstance(item, str) for item in data):
        raise HTTPException(400, "subjects.json 数据结构不正确")
    if name == "config.json" and not isinstance(data, dict):
        raise HTTPException(400, "config.json 数据结构不正确")
    if name == "knowledge.json" and not isinstance(data, dict):
        raise HTTPException(400, "knowledge.json 数据结构不正确")


def save_current_backup_snapshot(prefix: str = "pre-import"):
    backup_dir = os.path.join(DATA_DIR, "backups")
    os.makedirs(backup_dir, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    snapshot = os.path.join(backup_dir, f"{prefix}-{stamp}.zip")
    with zipfile.ZipFile(snapshot, "w", zipfile.ZIP_DEFLATED) as zf:
        for filename in sorted(BACKUP_FILES):
            path = os.path.join(DATA_DIR, filename)
            if os.path.isfile(path):
                zf.write(path, arcname=filename)
    return snapshot



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
            updated = review_error(error_id)
            return {
                "message": f"错题 #{error_id} 已标记复习",
                "next_review": updated.get("next_review") if updated else None,
                "review_count": updated.get("review_count") if updated else None,
            }
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


@app.get("/api/daily-push")
def get_daily_push():
    kb = get_knowledge_base()
    errors = list_errors()
    import random
    # Pick one tip per subject
    knowledge = {}
    for s in error_manager.SUBJECTS:
        if s in kb and kb[s]:
            knowledge[s] = random.choice(kb[s])
    today = today_str()
    due_errors = [e for e in errors if (e.get("next_review") or e.get("created", today)[:10]) <= today]
    due_errors.sort(key=lambda e: ((e.get("next_review") or e.get("created", today)[:10]), e.get("id", 0)))
    weak = [{
        "id": e["id"],
        "subject": e["subject"],
        "title": e.get("title") or f"未命名错题 #{e['id']}",
        "question": e.get("question", ""),
        "wrong": e.get("wrong", "未记录"),
        "correct": e.get("correct", "未记录"),
        "reason": e.get("reason", "未记录"),
        "tags": e.get("tags", []),
        "reason_tags": e.get("reason_tags", []),
        "created": e.get("created"),
        "review_count": e.get("review_count", 0),
        "last_review": e.get("last_review"),
        "review_stage": e.get("review_stage", e.get("review_count", 0)),
        "next_review": e.get("next_review"),
    } for e in due_errors]
    total = len(errors)
    overdue = sum(1 for e in due_errors if (e.get("next_review") or today) < today)
    if due_errors:
        advice = f"今天有 {len(due_errors)} 道错题到期复习，其中 {overdue} 道已逾期，建议先清空复习队列"
    else:
        advice = "今天没有到期错题，可以新增整理或轻量回看近期内容"
    return {
        "date": today_str(),
        "total_errors": total,
        "reviewed": sum(1 for e in errors if e.get("review_count", 0) > 0),
        "due_count": len(due_errors),
        "overdue_count": overdue,
        "knowledge": knowledge,
        "weak_errors": weak,
        "advice": advice,
    }


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
    """Set MinerU token. Empty values keep the current token unchanged."""
    token = data.get("token", "").strip()
    if not token:
        return {"message": "Token unchanged"}
    config = load_json("config.json", default={})
    config["mineru_token"] = token
    save_json("config.json", config)
    return {"message": "Token saved"}

@app.delete("/api/settings/token")
def clear_settings_token():
    config = load_json("config.json", default={})
    config["mineru_token"] = ""
    save_json("config.json", config)
    return {"message": "Token cleared"}

@app.put("/api/settings/username")
def set_username(data: dict):
    name = data.get("name", "").strip()
    config = load_json("config.json", default={})
    config["username"] = name
    save_json("config.json", config)
    return {"message": "Username saved"}


@app.get("/api/backup/export")
def export_backup():
    """Download local JSON data as a zip backup."""
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, "w", zipfile.ZIP_DEFLATED) as zf:
        for filename in sorted(BACKUP_FILES):
            path = os.path.join(DATA_DIR, filename)
            if os.path.isfile(path):
                zf.write(path, arcname=filename)
    buffer.seek(0)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    headers = {"Content-Disposition": f'attachment; filename="study-tracker-backup-{stamp}.zip"'}
    return StreamingResponse(buffer, media_type="application/zip", headers=headers)


@app.post("/api/backup/import")
async def import_backup(request: Request):
    """Restore JSON data files from a zip backup."""
    body = await request.body()
    if not body:
        raise HTTPException(400, "备份文件不能为空")

    try:
        with zipfile.ZipFile(io.BytesIO(body), "r") as zf:
            candidates = []
            for info in zf.infolist():
                if info.is_dir():
                    continue
                name = os.path.basename(info.filename)
                if name in BACKUP_FILES:
                    candidates.append((name, info))
                elif info.filename.endswith(".json"):
                    raise HTTPException(400, f"备份包包含不支持的数据文件：{info.filename}")

            if not candidates:
                raise HTTPException(400, "备份包中没有可恢复的数据文件")

            parsed = {}
            for name, info in candidates:
                if info.file_size > BACKUP_MAX_FILE_SIZE:
                    raise HTTPException(400, f"{name} 文件过大")
                try:
                    parsed[name] = json.loads(zf.read(info).decode("utf-8"))
                except json.JSONDecodeError:
                    raise HTTPException(400, f"{name} 不是有效 JSON 文件")
                validate_backup_data(name, parsed[name])

            os.makedirs(DATA_DIR, exist_ok=True)
            snapshot = save_current_backup_snapshot()
            for name, data in parsed.items():
                save_json(name, data)

        error_manager.SUBJECTS = error_manager.load_subjects()
        return {
            "message": "备份导入成功",
            "files": sorted(parsed.keys()),
            "snapshot": os.path.basename(snapshot),
        }
    except zipfile.BadZipFile:
        raise HTTPException(400, "请上传有效的 zip 备份文件")


@app.get("/api/version")
def get_version():
    return update_service.get_version_response()


@app.get("/api/update/check")
def check_update(force: bool = False):
    return update_service.check_update(force=force)


@app.post("/api/update/apply")
def apply_update():
    try:
        return update_service.apply_update()
    except update_service.UpdateError as e:
        raise HTTPException(400, str(e))

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
