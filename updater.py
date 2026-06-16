#!/usr/bin/env python3
import argparse
import ctypes
import json
import os
import shutil
import subprocess
import sys
import time
import zipfile
from datetime import datetime


SKIP_NAMES = {"data", ".git", "__pycache__"}
RETRY_COUNT = 20
RETRY_DELAY = 0.5


def now():
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")


def ensure_dir(path):
    os.makedirs(path, exist_ok=True)
    return path


def write_log(log_path, message):
    ensure_dir(os.path.dirname(log_path))
    with open(log_path, "a", encoding="utf-8") as f:
        f.write(f"[{now()}] {message}\n")


def wait_for_pid(pid, log_path, timeout=60):
    if pid <= 0:
        return
    write_log(log_path, f"Waiting for process {pid} to exit")
    if os.name == "nt":
        handle = ctypes.windll.kernel32.OpenProcess(0x00100000, False, pid)
        if handle:
            ctypes.windll.kernel32.WaitForSingleObject(handle, int(timeout * 1000))
            ctypes.windll.kernel32.CloseHandle(handle)
        return

    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            os.kill(pid, 0)
        except OSError:
            return
        time.sleep(0.5)


def pick_payload_root(extract_dir, app_exe):
    direct = os.path.join(extract_dir, app_exe)
    if os.path.isfile(direct):
        return extract_dir
    children = [
        os.path.join(extract_dir, name)
        for name in os.listdir(extract_dir)
        if os.path.isdir(os.path.join(extract_dir, name))
    ]
    if len(children) == 1 and os.path.isfile(os.path.join(children[0], app_exe)):
        return children[0]
    return extract_dir


def backup_target(target, rollback_dir, app_dir):
    if not os.path.exists(target):
        return
    rel = os.path.relpath(target, app_dir)
    backup = os.path.join(rollback_dir, rel)
    ensure_dir(os.path.dirname(backup))
    if os.path.isdir(target):
        shutil.copytree(target, backup, dirs_exist_ok=True)
    else:
        shutil.copy2(target, backup)


def copy_file_with_retry(source, target, log_path):
    ensure_dir(os.path.dirname(target))
    last_error = None
    for attempt in range(1, RETRY_COUNT + 1):
        try:
            shutil.copy2(source, target)
            return
        except OSError as exc:
            last_error = exc
            write_log(log_path, f"Retry {attempt}/{RETRY_COUNT} copying {os.path.basename(target)}: {exc}")
            time.sleep(RETRY_DELAY)
    raise last_error


def remove_path_with_retry(path, log_path):
    if not os.path.exists(path):
        return
    last_error = None
    for attempt in range(1, RETRY_COUNT + 1):
        try:
            if os.path.isdir(path):
                shutil.rmtree(path)
            else:
                os.remove(path)
            return
        except OSError as exc:
            last_error = exc
            write_log(log_path, f"Retry {attempt}/{RETRY_COUNT} removing {os.path.basename(path)}: {exc}")
            time.sleep(RETRY_DELAY)
    raise last_error


def restore_rollback(rollback_dir, app_dir, log_path):
    if not os.path.isdir(rollback_dir):
        return
    write_log(log_path, "Restoring rollback files")
    for root, dirs, files in os.walk(rollback_dir):
        rel_root = os.path.relpath(root, rollback_dir)
        target_root = app_dir if rel_root == "." else os.path.join(app_dir, rel_root)
        ensure_dir(target_root)
        for name in files:
            shutil.copy2(os.path.join(root, name), os.path.join(target_root, name))
        for name in dirs:
            ensure_dir(os.path.join(target_root, name))


def replace_from_payload(payload_root, app_dir, rollback_dir, current_exe, log_path):
    names = sorted(os.listdir(payload_root), key=lambda item: item.lower().endswith(".exe"))
    for name in names:
        if name in SKIP_NAMES:
            write_log(log_path, f"Skipping {name}")
            continue

        source = os.path.join(payload_root, name)
        target = os.path.join(app_dir, name)
        if os.path.abspath(target).lower() == os.path.abspath(current_exe).lower():
            write_log(log_path, f"Skipping running updater {name}")
            continue

        backup_target(target, rollback_dir, app_dir)
        if os.path.isdir(source):
            if os.path.exists(target):
                remove_path_with_retry(target, log_path)
            shutil.copytree(source, target)
        else:
            copy_file_with_retry(source, target, log_path)
        write_log(log_path, f"Replaced {name}")


def ensure_version_file(payload_root, app_dir, rollback_dir, log_path):
    source = os.path.join(payload_root, "version.json")
    if not os.path.isfile(source):
        return
    target = os.path.join(app_dir, "version.json")
    backup_target(target, rollback_dir, app_dir)
    copy_file_with_retry(source, target, log_path)
    write_log(log_path, "Replaced version.json")


def read_payload_version(payload_root):
    path = os.path.join(payload_root, "version.json")
    if not os.path.isfile(path):
        return ""
    with open(path, "r", encoding="utf-8-sig") as f:
        data = json.load(f)
    return str(data.get("version", ""))


def launch_app(app_dir, app_exe, log_path):
    app_path = os.path.join(app_dir, app_exe)
    if not os.path.isfile(app_path):
        raise RuntimeError(f"App exe not found: {app_path}")
    write_log(log_path, f"Launching {app_path} --no-browser")
    creationflags = 0
    if os.name == "nt":
        creationflags = subprocess.CREATE_NEW_PROCESS_GROUP | subprocess.DETACHED_PROCESS
    subprocess.Popen([app_path, "--no-browser"], cwd=app_dir, close_fds=True, creationflags=creationflags)


def main():
    parser = argparse.ArgumentParser(description="Tracker updater")
    parser.add_argument("--package", required=True)
    parser.add_argument("--app-dir", required=True)
    parser.add_argument("--app-exe", required=True)
    parser.add_argument("--pid", type=int, default=0)
    args = parser.parse_args()

    app_dir = os.path.abspath(args.app_dir)
    updates_dir = ensure_dir(os.path.join(app_dir, "data", "updates"))
    log_path = os.path.join(updates_dir, "update.log")
    stamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    extract_dir = os.path.join(updates_dir, f"extract-{stamp}")
    rollback_dir = os.path.join(updates_dir, f"rollback-{stamp}")

    try:
        write_log(log_path, "Updater started")
        wait_for_pid(args.pid, log_path)
        ensure_dir(extract_dir)
        ensure_dir(rollback_dir)
        with zipfile.ZipFile(args.package, "r") as zf:
            zf.extractall(extract_dir)
        payload_root = pick_payload_root(extract_dir, args.app_exe)
        payload_version = read_payload_version(payload_root)
        if payload_version:
            write_log(log_path, f"Payload version {payload_version}")
        replace_from_payload(payload_root, app_dir, rollback_dir, sys.executable, log_path)
        ensure_version_file(payload_root, app_dir, rollback_dir, log_path)
        if payload_version:
            installed_version = read_payload_version(app_dir)
            write_log(log_path, f"Installed version {installed_version}")
            if installed_version != payload_version:
                raise RuntimeError(f"version.json was not updated: expected {payload_version}, got {installed_version}")
        write_log(log_path, "Update installed successfully")
        launch_app(app_dir, args.app_exe, log_path)
        return 0
    except Exception as exc:
        write_log(log_path, f"Update failed: {exc}")
        try:
            restore_rollback(rollback_dir, app_dir, log_path)
        except Exception as rollback_exc:
            write_log(log_path, f"Rollback failed: {rollback_exc}")
        try:
            launch_app(app_dir, args.app_exe, log_path)
        except Exception as launch_exc:
            write_log(log_path, f"Relaunch failed: {launch_exc}")
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
