from .data_store import load_json


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
