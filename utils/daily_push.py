import random
from .data_store import load_json, today_str
from .error_manager import SUBJECTS


def get_knowledge_base():
    return load_json('knowledge.json', {
        '数学': [
            '等价无穷小：x→0时，sinx~x，tanx~x，1-cosx~x²/2，eˣ-1~x，ln(1+x)~x',
            '泰勒公式：sinx = x - x³/3! + x⁵/5! - ...',
            '洛必达法则：0/0 或 ∞/∞ 型，分子分母分别求导',
            '定积分几何意义：曲边梯形面积',
            '矩阵秩：r(AB) ≤ min(r(A), r(B))',
        ],
        '英语': [
            'ambiguous a. 模棱两可的',
            'controversial a. 有争议的',
            'distinguish v. 区分，辨别',
            'fundamental a. 根本的，基础的',
            'hypothesis n. 假设，假说',
        ],
        '物理': [
            '牛顿第二定律：F = ma',
            '动能定理：W = ΔEₖ = ½mv² - ½mv₀²',
            '欧姆定律：I = U / R',
        ],
        '化学': [
            '阿伏伽德罗常数：Nₐ = 6.02 × 10²³ mol⁻¹',
            '勒夏特列原理：平衡向减弱改变的方向移动',
        ],
        '生物': [
            '细胞学说：一切生物都由细胞构成',
            'DNA双螺旋结构：A-T，C-G 碱基配对',
        ],
        '语文': [
            '修辞手法：比喻、拟人、夸张、排比、对偶、反复',
            '论证方法：举例论证、道理论证、对比论证、比喻论证',
        ],
    })


def daily_push():
    kb = get_knowledge_base()
    errors = load_json('errors.json', [])

    print(f'\n🌟 ========== 今日学习推送 ({today_str()}) ==========')

    # 1. 推送 408 知识点（各科各一道）
    print('\n📚 【知识点】')
    for s in SUBJECTS:
        if s in kb and kb[s]:
            tip = random.choice(kb[s])
            print(f'  [{s}] {tip}')

    # 2. 推送数学/英语各一
    print('\n📐 【数学技巧】')
    if '数学' in kb:
        print(f'  {random.choice(kb["数学"])}')

    print('\n🔤 【英语单词】')
    if '英语' in kb:
        print(f'  {random.choice(kb["英语"])}')

    # 3. 推荐复习错题
    print('\n🔄 【今日错题复习推荐】')
    weak_errors = [e for e in errors if e.get('review_count', 0) < 2]
    if weak_errors:
        random.shuffle(weak_errors)
        for e in weak_errors[:3]:
            print(f'  #{e["id"]} [{e["subject"]}] {e["question"][:40]}...')
        print(f'  共 {len(weak_errors)} 道待复习错题，建议每天复习 3-5 道')
    else:
        print('  🎉 错题都复习得不错！今天可以刷新题了')

    # 4. 学习建议
    total = len(errors)
    if total < 20:
        print('\n💡 建议：当前错题量较少，保持当前刷题量，注意归纳总结')
    elif total < 50:
        print('\n💡 建议：错题量适中，重点复习标记为"概念不清"的题目')
    else:
        print('\n💡 建议：错题量较大，建议暂停刷新题，集中复习旧错题')

    print('=========================================\n')
