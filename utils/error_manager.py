from .data_store import load_json, save_json, now_str

def load_subjects():
    return load_json("subjects.json", [])

def save_subjects(data):
    save_json("subjects.json", data)

SUBJECTS = load_subjects()


def add_error(subject, question, wrong, correct, reason, tags=None, title=None, reason_tags=None):
    if subject not in SUBJECTS:
        print(f'未知科目，可选：{", ".join(SUBJECTS)}')
        return False
    data = load_json('errors.json', [])
    entry = {
        'id': len(data) + 1,
        'subject': subject,
        'title': title or question[:40],
        'question': question,
        'wrong': wrong,
        'correct': correct,
        'reason': reason,
        'tags': tags or [],
        'reason_tags': reason_tags or [],
        'created': now_str(),
        'review_count': 0,
        'last_review': None
    }
    data.append(entry)
    save_json('errors.json', data)
    print(f'✅ 已记录错题 #{entry["id"]} [{subject}]')
    return True


def list_errors(subject=None, keyword=None, tag=None, reason_tag=None):
    data = load_json('errors.json', [])
    if subject:
        data = [e for e in data if e['subject'] == subject]
    if keyword:
        kw = keyword.lower()
        data = [e for e in data if kw in e['question'].lower() or kw in (e.get('title') or '').lower() or kw in (e.get('reason') or '').lower() or any(kw in t.lower() for t in e.get('tags', [])) or any(kw in t.lower() for t in e.get('reason_tags', []))]
    if tag:
        data = [e for e in data if any(tag.lower() in t.lower() for t in e.get('tags', []))]
    if reason_tag:
        data = [e for e in data if any(reason_tag.lower() in t.lower() for t in e.get('reason_tags', []))]
    return data

def all_tags():
    data = load_json('errors.json', [])
    tags = set()
    for e in data:
        for t in e.get('tags', []):
            tags.add(t)
        for t in e.get('reason_tags', []):
            tags.add(t)
    return sorted(tags)


def review_error(error_id):
    data = load_json('errors.json', [])
    for e in data:
        if e['id'] == error_id:
            e['review_count'] = e.get('review_count', 0) + 1
            e['last_review'] = now_str()
            save_json('errors.json', data)
            print(f'✅ 错题 #{error_id} 复习次数 +1（共 {e["review_count"]} 次）')
            return
    print(f'❌ 未找到错题 #{error_id}')


def delete_error(error_id):
    data = load_json('errors.json', [])
    new_data = [e for e in data if e['id'] != error_id]
    if len(new_data) == len(data):
        print(f'❌ 未找到错题 #{error_id}')
        return False
    save_json('errors.json', new_data)
    print(f'✅ 已删除错题 #{error_id}')
    return True
