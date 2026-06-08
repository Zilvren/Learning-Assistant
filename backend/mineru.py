"""
MinerU API integration - PDF/Image to Markdown via precision parse API.
Requires token from https://mineru.net/apiManage
"""

import os, json, time, tempfile, zipfile, requests
import sys, os
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from utils.data_store import load_json, save_json

MINERU_BASE = "https://mineru.net/api/v4"
TEMP_DIR = os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "temp")
try:
    os.makedirs(TEMP_DIR, exist_ok=True)
except:
    TEMP_DIR = tempfile.gettempdir()


def get_token():
    config = load_json("config.json", default={"mineru_token": ""})
    token = config.get("mineru_token", "").strip()
    if not token:
        token = os.environ.get("MINERU_TOKEN", "").strip()
    if not token:
        raise ValueError("MinerU token not configured. Set mineru_token in data/config.json or MINERU_TOKEN env var.")
    return token


def submit_task(file_path: str, file_name: str) -> tuple[str, str]:
    """
    Submit an image/PDF to MinerU via batch file upload.
    Returns (batch_id, extract_task_id) after upload completes.
    """
    token = get_token()
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

    # Step 1: Get upload URL
    batch_url = f"{MINERU_BASE}/file-urls/batch"
    data = {
        "files": [{"name": file_name, "data_id": file_name.replace(".", "_")}],
        "model_version": "vlm",
        "enable_formula": True,
        "enable_table": False,
        "language": "ch",
    }
    resp = requests.post(batch_url, headers=headers, json=data, timeout=30)
    result = resp.json()

    if result.get("code") != 0:
        raise Exception(f"MinerU batch request failed: {result.get('msg')}")

    batch_id = result["data"]["batch_id"]
    file_urls = result["data"]["file_urls"]

    if not file_urls:
        raise Exception("No upload URL returned")

    # Step 2: Upload file to presigned URL
    with open(file_path, "rb") as f:
        upload_resp = requests.put(file_urls[0], data=f, timeout=120)
        if upload_resp.status_code not in (200, 201):
            raise Exception(f"Upload failed: HTTP {upload_resp.status_code}")

    # Step 3: Poll for individual task IDs from the batch
    # The batch ID can be used to query results
    # MinerU assigns task_ids automatically after upload
    # We poll the batch endpoint to get them
    return batch_id, ""


def poll_batch_result(batch_id: str, timeout: int = 300) -> str:
    """
    Poll MinerU for batch results. Returns the markdown content string.
    Polls every 5 seconds until all files are done or timeout.
    """
    token = get_token()
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {token}"
    }

    start_time = time.time()
    while time.time() - start_time < timeout:
        # Query batch results via the extract endpoint
        # Actually, we need to use the batch query endpoint
        resp = requests.get(
            f"{MINERU_BASE}/extract/task",
            headers=headers,
            timeout=10
        )
        # The batch results might be accessible via a different endpoint
        # Let''s try polling by downloading the zip when done
        
        # Actually, we need the individual task_id from the batch
        # Let''s use a different approach: submit via single file URL
        time.sleep(5)

    raise TimeoutError(f"MinerU task {batch_id} timed out after {timeout}s")


def ocr_image(image_bytes: bytes, file_name: str = "question.png") -> str:
    """Convert an image to Markdown using MinerU precision API."""
    import time, requests, json, os, tempfile, zipfile

    # Step 1: Get batch upload URL (precision API)
    resp = requests.post(
        "https://mineru.net/api/v4/file-urls/batch",
        json={
            "files": [{"name": file_name}],
            "model_version": "vlm",
            "enable_formula": True,
            "enable_table": False,
            "language": "ch",
        },
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {get_token()}",
        },
        timeout=30
    )
    result = resp.json()
    if result.get("code") != 0:
        raise Exception(f"MinerU error: {result.get('msg')}")

    batch_id = result["data"]["batch_id"]
    upload_url = result["data"]["file_urls"][0]

    # Step 2: Upload image
    up = requests.put(upload_url, data=image_bytes, timeout=120)
    if up.status_code not in (200, 201):
        raise Exception(f"Upload failed: HTTP {up.status_code}")

    # Step 3: Poll for results
    start = time.time()
    timeout_val = 300
    task_id = None

    while time.time() - start < timeout_val:
        time.sleep(3)

        # Primary: query extract-results endpoint
        try:
            qr = requests.get(
                f"https://mineru.net/api/v4/extract-results/batch/{batch_id}",
                headers={"Authorization": f"Bearer {get_token()}"},
                timeout=10
            )
            if qr.status_code == 200:
                data = qr.json().get("data", {})
                extract_result = data.get("extract_result", [])
                if extract_result:
                    er = extract_result[0]
                    state = er.get("state", "")
                    if state == "done":
                        zip_url = er.get("full_zip_url", "")
                        if zip_url:
                            return download_and_extract_md(zip_url)
                    elif state == "failed":
                        raise Exception(f"MinerU failed: {er.get('err_msg')}")
                    tid = er.get("task_id")
                    if tid and not task_id:
                        task_id = tid
        except requests.exceptions.RequestException:
            pass

        # If we have a task ID, poll it directly (most reliable)
        if task_id:
            try:
                tr = requests.get(
                    f"https://mineru.net/api/v4/extract/task/{task_id}",
                    headers={"Authorization": f"Bearer {get_token()}"},
                    timeout=10
                )
                if tr.status_code == 200:
                    td = tr.json().get("data", {})
                    if td.get("state") == "done":
                        zip_url = td.get("full_zip_url", "")
                        if zip_url:
                            md = download_and_extract_md(zip_url)
                            print(f"[OCR] Returned {len(md)} chars of markdown")
                            return md
                    elif td.get("state") == "failed":
                        raise Exception(f"MinerU failed: {td.get('err_msg')}")
            except requests.exceptions.RequestException:
                pass

    raise TimeoutError(f"MinerU OCR timed out after {timeout_val}s")

class MinerUDownloadError(Exception):
    pass

def download_and_extract_md(zip_url: str) -> str:
    """Download MinerU result zip, extract full.md with images as base64 data URIs."""
    import base64, re

    import urllib3
    try:
        http = urllib3.PoolManager(cert_reqs='CERT_NONE')
        resp = http.request('GET', zip_url, timeout=60, retries=False)
        if resp.status != 200:
            raise MinerUDownloadError(f"Failed to download result zip: HTTP {resp.status}")
        content = resp.data
    except Exception as e:
        raise MinerUDownloadError(f"Failed to download result zip: {e}") from e

    zip_path = os.path.join(TEMP_DIR, "result.zip")
    with open(zip_path, "wb") as f:
        f.write(content)

    try:
        with zipfile.ZipFile(zip_path, "r") as zf:
            all_names = zf.namelist()
            # Find full.md
            md_name = None
            for name in all_names:
                if name.endswith("full.md"):
                    md_name = name
                    break
            if not md_name:
                raise Exception("full.md not found in zip")

            md_content = zf.read(md_name).decode("utf-8", errors="replace")

            # Build image map: filename → base64 data URI
            img_map = {}
            for name in all_names:
                if name.startswith("images/") and not name.endswith("/"):
                    ext = os.path.splitext(name)[1].lower().lstrip(".")
                    mime = {"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg",
                            "gif": "image/gif", "webp": "image/webp", "bmp": "image/bmp"}.get(ext, "image/png")
                    img_bytes = zf.read(name)
                    b64 = base64.b64encode(img_bytes).decode("ascii")
                    img_map[name] = f"data:{mime};base64,{b64}"

            # Replace image references with HTML img tags (default width 400)
            def replace_img(m):
                src = m.group(1)
                alt = m.group(0)  # fallback
                if src in img_map:
                    return f'<img src="{img_map[src]}" width="400">'
                basename = os.path.basename(src)
                for k, v in img_map.items():
                    if k.endswith(basename):
                        return f'<img src="{v}" width="400">'
                return m.group(0)

            md_content = re.sub(r'!\[([^\]]*)\]\(([^)]+)\)', replace_img, md_content)

            return md_content
    finally:
        if os.path.exists(zip_path):
            os.remove(zip_path)
