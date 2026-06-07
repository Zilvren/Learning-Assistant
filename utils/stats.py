from .data_store import load_json
from .error_manager import SUBJECTS


def show_stats():
    data = load_json('errors.json', [])
    if not data:
        print('\n📊 暂无错题数据，快去学习吧！')
        return

    print('\n📊 ========== 薄弱点分析 ==========')
    total = len(data)
    print(f'错题总数：{total} 道\n')

    subject_counts = {s: 0 for s in SUBJECTS}
    for e in data:
        subject_counts[e['subject']] = subject_counts.get(e['subject'], 0) + 1

    print('【各科错题分布】')
    max_count = max(subject_counts.values()) if subject_counts else 0
    for s in SUBJECTS:
        cnt = subject_counts.get(s, 0)
        bar = '█' * cnt + '░' * (max(max_count, 5) - cnt)
        flag = ' 🔥最薄弱' if cnt == max_count and cnt > 0 else ''
        print(f'  {s:10s} {bar} {cnt} 道{flag}')

    print('\n【需重点复习的错题】（复习次数 < 2）')
    weak = [e for e in data if e.get('review_count', 0) < 2]
    if not weak:
        print('  🎉 所有错题都已复习 2 次以上，继续保持！')
    else:
        for e in weak[:10]:
            print(f'  - #{e["id"]} [{e["subject"]}] {e["question"][:30]}...')
        if len(weak) > 10:
            print(f'  ... 还有 {len(weak)-10} 道')

    print('\n【最近一周新增】')
    from datetime import datetime, timedelta
    week_ago = datetime.now() - timedelta(days=7)
    recent = [e for e in data if datetime.strptime(e['created'][:10], '%Y-%m-%d') >= week_ago]
    print(f'  共 {len(recent)} 道')

    print('===================================\n')
