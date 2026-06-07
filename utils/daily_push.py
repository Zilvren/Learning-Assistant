import random
from .data_store import load_json, today_str
from .error_manager import SUBJECTS


def get_knowledge_base():
    return load_json('knowledge.json', {
        '数据结构': [
            '时间复杂度：O(1) < O(logn) < O(n) < O(nlogn) < O(n²) < O(n³) < O(2ⁿ) < O(n!)',
            '栈：后进先出(LIFO)；队列：先进先出(FIFO)',
            '二叉树性质：度为0的结点 = 度为2的结点 + 1',
            '哈夫曼树：带权路径长度最小的二叉树，不存在度为1的结点',
            '图的存储：邻接矩阵适合稠密图，邻接表适合稀疏图',
            '拓扑排序：AOV网，用入度为0的顶点入队，依次删除边',
            '排序稳定性：插泡归基稳，快选堆希不稳',
            '快速排序平均时间O(nlogn)，最坏O(n²)，空间O(logn)',
            '堆排序：建堆O(n)，调整O(logn)，总时间O(nlogn)，空间O(1)',
            'B树/B+树：B+树数据全在叶子，适合文件索引和数据库',
        ],
        '计算机组成原理': [
            'IEEE 754单精度：1位符号 + 8位阶码(移码) + 23位尾数(原码)',
            'Cache映射：直接映射、全相联映射、组相联映射',
            'Cache命中率 = 命中次数 / 总访问次数',
            '指令周期 = 取指周期 + 执行周期(+ 间址周期 + 中断周期)',
            '微程序控制器：控制存储器CM存放微指令，用微地址寻址',
            '数据通路：ALU、寄存器、总线之间的数据传送路径',
            'DMA：直接在内存和I/O设备间传数据，不经过CPU',
            '中断隐指令：关中断、保存断点、引出中断服务程序（由硬件自动完成）',
            '浮点数加减：对阶 → 尾数运算 → 规格化 → 舍入 → 判溢出',
            '流水线冒险：结构冒险、数据冒险、控制冒险',
        ],
        '操作系统': [
            '进程三态：就绪、运行、阻塞（等待）',
            '临界区访问准则：空闲让进、忙则等待、有限等待、让权等待',
            '死锁必要条件：互斥、不剥夺、请求保持、循环等待（缺一不可）',
            '银行家算法：避免死锁，试探分配后检查安全性',
            '页面置换算法：FIFO、LRU、CLOCK、OPT',
            '磁盘调度：FCFS、SSTF、SCAN、C-SCAN、LOOK',
            '文件分配：连续分配、链接分配、索引分配（FAT、inode）',
            '虚拟内存：请求分页/分段，局部性原理（时间+空间）',
            '进程通信：共享存储、消息传递、管道通信',
            '调度算法：FCFS、SJF、优先级、时间片轮转、多级反馈队列',
        ],
        '计算机网络': [
            'OSI七层：物数网传会表应；TCP/IP四层：网际网传应',
            '奈氏准则：码元速率 ≤ 2W（W为带宽），极限波特率',
            '香农定理：信道极限速率 = W·log₂(1+S/N)',
            'CSMA/CD：争用期 = 2τ，最小帧长 = 2τ × 传输速率',
            'IP地址分类：A(1-126)、B(128-191)、C(192-223)',
            '子网掩码：网络位全1，主机位全0，与IP按位与得网络地址',
            'TCP三次握手：SYN → SYN+ACK → ACK',
            'TCP四次挥手：FIN → ACK → FIN → ACK',
            '路由协议：RIP（距离向量）、OSPF（链路状态）、BGP（路径向量）',
            'NAT：私有IP转公有IP，缓解IPv4地址不足',
            'HTTP/1.1默认持久连接，HTTP/2多路复用',
            'DNS：递归查询 + 迭代查询，UDP传输（报文小）',
        ],
        '数学': [
            '等价无穷小：x→0时，sinx~x，tanx~x，1-cosx~x²/2，eˣ-1~x，ln(1+x)~x',
            '泰勒公式：sinx = x - x³/3! + x⁵/5! - ...',
            '洛必达法则：0/0 或 ∞/∞ 型，分子分母分别求导',
            '定积分几何意义：曲边梯形面积（注意正负）',
            '格林公式：∮Pdx+Qdy = ∬(∂Q/∂x - ∂P/∂y)dxdy',
            '高斯公式：∯Pdydz+Qdzdx+Rdxdy = ∭(∂P/∂x+∂Q/∂y+∂R/∂z)dV',
            '级数收敛：比较判别法、比值判别法、根值判别法',
            '矩阵秩：r(AB) ≤ min(r(A), r(B))；A可逆则r(AB)=r(B)',
        ],
        '英语': [
            'abandon v. 放弃（考研真题高频但非最高频，别只背这个😂）',
            'ambiguous a. 模棱两可的',
            'controversial a. 有争议的',
            'conventional a. 传统的，惯例的',
            'discipline n. 学科；纪律；训练',
            'distinguish v. 区分，辨别',
            'dominant a. 主导的，占优势的',
            'eliminate v. 排除，消除',
            'fundamental a. 根本的，基础的',
            'hypothesis n. 假设，假说',
            'implication n. 含义；暗示；影响',
            'inevitable a. 不可避免的',
            'infrastructure n. 基础设施',
            'manifest v. 表明 a. 明显的',
            'notion n. 概念，看法',
            'paradox n. 悖论；自相矛盾',
            'prerequisite n. 先决条件',
            'substantial a. 大量的；实质的',
            'underlying a. 潜在的，根本的',
            'vulnerable a. 脆弱的，易受影响的',
        ]
    })


def daily_push():
    kb = get_knowledge_base()
    errors = load_json('errors.json', [])

    print(f'\n🌟 ========== 今日学习推送 ({today_str()}) ==========')

    # 1. 推送 408 知识点（各科各一道）
    subjects_408 = ['数据结构', '计算机组成原理', '操作系统', '计算机网络']
    print('\n📚 【408 知识点】')
    for s in subjects_408:
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
