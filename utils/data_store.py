import json
import os, sys
from datetime import datetime

if getattr(sys, 'frozen', False):
    DATA_DIR = os.path.join(os.path.dirname(sys.executable), 'data')
else:
    DATA_DIR = os.path.join(os.path.dirname(__file__), '..', 'data')


def _path(filename):
    os.makedirs(DATA_DIR, exist_ok=True)
    return os.path.join(DATA_DIR, filename)


def load_json(filename, default=None):
    path = _path(filename)
    if not os.path.exists(path):
        return default if default is not None else {}
    with open(path, 'r', encoding='utf-8') as f:
        return json.load(f)


def save_json(filename, data):
    path = _path(filename)
    os.makedirs(DATA_DIR, exist_ok=True)
    with open(path, 'w', encoding='utf-8') as f:
        json.dump(data, f, ensure_ascii=False, indent=2)


def now_str():
    return datetime.now().strftime('%Y-%m-%d %H:%M:%S')


def today_str():
    return datetime.now().strftime('%Y-%m-%d')
