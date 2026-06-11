#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
错题追踪器
功能：错题记录、薄弱点分析、每日知识点推送
"""

import sys, io
if sys.platform == 'win32':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

sys.path.insert(0, __import__('os').path.dirname(__import__('os').path.abspath(__file__)))

from utils.error_manager import (
    SUBJECTS, add_error, list_errors, review_error,
    delete_error, format_error
)
from utils.stats import show_stats
from utils.daily_push import daily_push


def print_menu():
    print("""
╔══════════════════════════════════════╗
║      📚 错题追踪器 📚       ║
╠══════════════════════════════════════╣
║  1. 🌟 每日学习推送                   ║
║  2. ➕ 添加错题                       ║
║  3. 📋 查看错题（支持筛选/搜索）       ║
║  4. 🔄 标记错题已复习                 ║
║  5. 📊 薄弱点分析                     ║
║  6. ❌ 删除错题                       ║
║  0. 🚪 退出                           ║
╚══════════════════════════════════════╝
""")


def input_subject():
    print('\n可选科目：')
    for i, s in enumerate(SUBJECTS, 1):
        print(f'  {i}. {s}')
    try:
        choice = int(input('选择科目编号：'))
        if 1 <= choice <= len(SUBJECTS):
            return SUBJECTS[choice - 1]
    except ValueError:
        pass
    print('❌ 无效选择')
    return None


def do_add_error():
    subject = input_subject()
    if not subject:
        return
    print('\n请填写错题信息（直接回车可跳过非必填项）：')
    question = input('题目：').strip()
    if not question:
        print('❌ 题目不能为空')
        return
    wrong = input('你的错答：').strip() or '未记录'
    correct = input('正确答案：').strip() or '未记录'
    reason = input('错误原因：').strip() or '未记录'
    tags_str = input('标签（用空格分隔，如 极限 导数）：').strip()
    tags = tags_str.split() if tags_str else []
    add_error(subject, question, wrong, correct, reason, tags)


def do_list_errors():
    print('\n【筛选选项】')
    print('  1. 查看全部')
    print('  2. 按科目筛选')
    print('  3. 关键词搜索')
    choice = input('选择：').strip()

    subject = None
    keyword = None

    if choice == '2':
        subject = input_subject()
        if not subject:
            return
    elif choice == '3':
        keyword = input('输入关键词：').strip()

    errors = list_errors(subject=subject, keyword=keyword)
    if not errors:
        print('\n📭 没有找到符合条件的错题')
        return

    print(f'\n共找到 {len(errors)} 道错题：')
    for e in errors:
        print(format_error(e))


def do_review():
    try:
        eid = int(input('输入要标记复习的错题编号：'))
        review_error(eid)
    except ValueError:
        print('❌ 请输入数字编号')


def do_delete():
    try:
        eid = int(input('输入要删除的错题编号：'))
        confirm = input(f'确认删除 #{eid}？(y/N)：').strip().lower()
        if confirm == 'y':
            delete_error(eid)
    except ValueError:
        print('❌ 请输入数字编号')


def main():
    print('\n🎓 欢迎来到 错题追踪器！')
    print('   数据保存在 study_tracker/data/ 目录下\n')

    while True:
        print_menu()
        choice = input('请选择功能：').strip()

        if choice == '1':
            daily_push()
        elif choice == '2':
            do_add_error()
        elif choice == '3':
            do_list_errors()
        elif choice == '4':
            do_review()
        elif choice == '5':
            show_stats()
        elif choice == '6':
            do_delete()
        elif choice == '0':
            print('\n👋 感谢使用！')
            break
        else:
            print('\n❌ 无效选择，请重新输入')

        input('\n按回车继续...')


if __name__ == '__main__':
    try:
        main()
    except KeyboardInterrupt:
        print('\n\n👋 已退出，加油！')
        sys.exit(0)
