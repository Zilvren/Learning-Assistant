import json
import os
import re
import shutil
import subprocess
import sys
import threading
import time
import zipfile
from datetime import datetime

import requests

from utils.data_store import DATA_DIR


ROOT_DIR = os.getcwd()
VERSION_FILE = "version.json"
DEFAULT_VERSION = {
    "version": "0.0.0-dev",
    "repo": "Zilvren/Learning-Assitant",
    "asset_name": "Tracker.zip",
    "app_exe": "Tracker.exe",
}
DATA_BACKUP_FILES = {"errors.json", "subjects.json", "config.json", "knowledge.json"}


class UpdateError(Exception):
    pass


def _root_path(*parts):
    return os.path.join(ROOT_DIR, *parts)


def _exe_root():
    if getattr(sys, "frozen", False):
        return os.path.dirname(sys.executable)
    return ROOT_DIR


def _data_path(*parts):
    os.makedirs(DATA_DIR, exist_ok=True)
    return os.path.join(DATA_DIR, *parts)


def _normalize_version(value):
    return str(value or "").strip().lstrip("vV")


def _version_key(value):
    text = _normalize_version(value)
    if re.fullmatch(r"\d{4}\.\d{2}\.\d{2}-\d{4}", text):
        date_part, time_part = text.split("-", 1)
        year, month, day = (int(x) for x in date_part.split("."))
        hour = int(time_part[:2])
        minute = int(time_part[2:])
        return (2, year, month, day, hour, minute)

    if re.fullmatch(r"\d+(\.\d+)*", text):
        parts = [int(x) for x in text.split(".")]
        while len(parts) < 6:
            parts.append(0)
        return (1, *parts[:6])

    return (0, text)


def compare_versions(left, right):
    left_key = _version_key(left)
    right_key = _version_key(right)
    return (left_key > right_key) - (left_key < right_key)


def load_version_info():
    info = DEFAULT_VERSION.copy()
    candidates = []
    if getattr(sys, "frozen", False):
        candidates.append(os.path.join(_exe_root(), VERSION_FILE))
    candidates.append(_root_path(VERSION_FILE))
    candidates.append(os.path.join(DATA_DIR, VERSION_FILE))
    for path in candidates:
        if os.path.isfile(path):
            with open(path, "r", encoding="utf-8-sig") as f:
                loaded = json.load(f)
            if isinstance(loaded, dict):
                info.update({k: v for k, v in loaded.items() if v})
                break
    updater_path = _root_path("Updater.exe")
    info["can_auto_update"] = bool(getattr(sys, "frozen", False) and os.path.isfile(updater_path))
    info["updater_path"] = updater_path if info["can_auto_update"] else ""
    return info


def get_version_response():
    info = load_version_info()
    return {
        "version": info["version"],
        "repo": info["repo"],
        "asset_name": info["asset_name"],
        "app_exe": info["app_exe"],
        "can_auto_update": info["can_auto_update"],
    }


def fetch_latest_release():
    info = load_version_info()
    url = f"https://api.github.com/repos/{info['repo']}/releases/latest"
    try:
        resp = requests.get(url, timeout=12, headers={"Accept": "application/vnd.github+json"})
        resp.raise_for_status()
    except requests.RequestException as exc:
        raise UpdateError(f"检查更新失败：{exc}") from exc

    release = resp.json()
    assets = release.get("assets") or []
    asset = next((item for item in assets if item.get("name") == info["asset_name"]), None)
    latest_version = _normalize_version(release.get("tag_name"))
    current_version = _normalize_version(info["version"])
    result = {
        "current_version": info["version"],
        "latest_version": latest_version,
        "tag_name": release.get("tag_name", ""),
        "has_update": bool(latest_version and compare_versions(latest_version, current_version) > 0),
        "repo": info["repo"],
        "asset_name": info["asset_name"],
        "asset_found": bool(asset),
        "asset_size": asset.get("size", 0) if asset else 0,
        "published_at": release.get("published_at", ""),
        "html_url": release.get("html_url", ""),
        "notes": release.get("body", "") or "",
        "can_auto_update": info["can_auto_update"],
        "download_url": asset.get("browser_download_url", "") if asset else "",
    }
    return result


def check_update(force=False):
    try:
        release = fetch_latest_release()
        release["ok"] = True
        release["checked_at"] = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        release.pop("download_url", None)
        return release
    except UpdateError as exc:
        return {
            "ok": False,
            "message": str(exc),
            "checked_at": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            **get_version_response(),
        }


def save_pre_update_snapshot():
    backup_dir = _data_path("backups")
    os.makedirs(backup_dir, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    snapshot = os.path.join(backup_dir, f"pre-update-{stamp}.zip")
    with zipfile.ZipFile(snapshot, "w", zipfile.ZIP_DEFLATED) as zf:
        for filename in sorted(DATA_BACKUP_FILES):
            path = _data_path(filename)
            if os.path.isfile(path):
                zf.write(path, arcname=filename)
    return snapshot


def download_update_package(release):
    if not release.get("download_url"):
        raise UpdateError("最新 Release 中没有找到 Tracker.zip")
    updates_dir = _data_path("updates")
    os.makedirs(updates_dir, exist_ok=True)
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    target = os.path.join(updates_dir, f"Tracker-{release['latest_version']}-{stamp}.zip")
    tmp = target + ".download"

    try:
        with requests.get(release["download_url"], stream=True, timeout=30) as resp:
            resp.raise_for_status()
            with open(tmp, "wb") as f:
                for chunk in resp.iter_content(chunk_size=1024 * 256):
                    if chunk:
                        f.write(chunk)
        os.replace(tmp, target)
    except requests.RequestException as exc:
        if os.path.exists(tmp):
            os.remove(tmp)
        raise UpdateError(f"下载更新失败：{exc}") from exc

    if not zipfile.is_zipfile(target):
        os.remove(target)
        raise UpdateError("下载的更新包不是有效 zip 文件")
    return target


def _exit_later(delay=1.0):
    def _stop():
        time.sleep(delay)
        os._exit(0)

    threading.Thread(target=_stop, daemon=True).start()


def apply_update():
    info = load_version_info()
    if not info["can_auto_update"]:
        raise UpdateError("当前运行环境不支持自动替换，请使用打包后的 Tracker.exe")

    release = fetch_latest_release()
    if compare_versions(release["latest_version"], info["version"]) <= 0:
        return {"message": "当前已是最新版本", **check_update(force=True)}
    if not release["asset_found"]:
        raise UpdateError(f"最新 Release 中没有找到 {info['asset_name']}")

    snapshot = save_pre_update_snapshot()
    package = download_update_package(release)
    updater = info["updater_path"]
    app_dir = ROOT_DIR

    cmd = [
        updater,
        "--package",
        package,
        "--app-dir",
        app_dir,
        "--app-exe",
        info["app_exe"],
        "--pid",
        str(os.getpid()),
    ]
    creationflags = 0
    if os.name == "nt":
        creationflags = subprocess.CREATE_NEW_PROCESS_GROUP | subprocess.DETACHED_PROCESS
    subprocess.Popen(cmd, cwd=app_dir, close_fds=True, creationflags=creationflags)
    _exit_later()

    return {
        "message": "更新包已下载，程序即将重启并安装更新",
        "latest_version": release["latest_version"],
        "package": package,
        "snapshot": os.path.basename(snapshot),
    }
