#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
11408 考研学习追踪器 - Flet 版 (enhanced visual design)
"""

import sys, io, os, random
from datetime import datetime

if sys.platform == 'win32':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

import flet as ft

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from utils.error_manager import (
    SUBJECTS, add_error, list_errors, review_error,
    delete_error, format_error
)
from utils.daily_push import get_knowledge_base
from utils.pdf_export import generate_pdf
from utils.data_store import load_json, today_str
SUBJECT_COLORS = {
    '数据结构': '#0EA5E9',
    '计算机组成原理': '#8B5CF6',
    '操作系统': '#10B981',
    '计算机网络': '#F97316',
    '数学': '#EC4899',
    '英语': '#F59E0B',
}

SUBJECT_ICONS = {
    '数据结构': ft.Icons.ACCOUNT_TREE_OUTLINED,
    '计算机组成原理': ft.Icons.MEMORY_OUTLINED,
    '操作系统': ft.Icons.TERMINAL_OUTLINED,
    '计算机网络': ft.Icons.LANGUAGE_OUTLINED,
    '数学': ft.Icons.FUNCTIONS_OUTLINED,
    '英语': ft.Icons.TRANSLATE_OUTLINED,
}

CARD_RADIUS = 14
BTN_RADIUS = 10
PAGE_PAD = 30


def palette(is_dark):
    if is_dark:
        return {
            'bg': '#0F172A', 'surface': '#1E293B',
            'sidebar_start': '#0B1120', 'sidebar_end': '#0F172A',
            'accent': '#818CF8', 'accent_light': '#A5B4FC',
            'text': '#F1F5F9', 'text_sec': '#94A3B8', 'text_muted': '#64748B',
            'border': '#334155',
            'success': '#34D399', 'warning': '#FBBF24', 'danger': '#F87171',
            'hero_start': '#4F46E5', 'hero_end': '#7C3AED',
            'sidebar_text': '#FFFFFF', 'sidebar_sub': '#64748B',
            'shadow_light': '30', 'shadow_heavy': '50',
            'nav_hover': '#ffffff08', 'nav_active': '#818CF820',
            'input_bg': '#1E293B', 'chip_bg': '#334155', 'chip_active': '#818CF830',
            'table_header': '#1E293B', 'row_alt': '#1E293B80',
        }
    else:
        return {
            'bg': '#F8F9FA', 'surface': '#FFFFFF',
            'sidebar_start': '#1A1A2E', 'sidebar_end': '#16213E',
            'accent': '#6366F1', 'accent_light': '#818CF8',
            'text': '#1E293B', 'text_sec': '#64748B', 'text_muted': '#94A3B8',
            'border': '#E2E8F0',
            'success': '#10B981', 'warning': '#F59E0B', 'danger': '#EF4444',
            'hero_start': '#6366F1', 'hero_end': '#8B5CF6',
            'sidebar_text': '#FFFFFF', 'sidebar_sub': '#94A3B8',
            'shadow_light': '10', 'shadow_heavy': '18',
            'nav_hover': '#ffffff10', 'nav_active': '#6366F118',
            'input_bg': '#FFFFFF', 'chip_bg': '#F1F5F9', 'chip_active': '#6366F118',
            'table_header': '#F8FAFC', 'row_alt': '#F8FAFC',
        }


def card_shadow(p, heavy=False):
    op = p['shadow_heavy'] if heavy else p['shadow_light']
    return ft.BoxShadow(blur_radius=16 if heavy else 8, spread_radius=0, color='#000000' + op)


def hero_gradient(p):
    return ft.LinearGradient(
        begin=ft.Alignment(-1, -1), end=ft.Alignment(1, 1),
        colors=[p['hero_start'], p['hero_end']],
    )


def sidebar_gradient(p):
    return ft.LinearGradient(
        begin=ft.Alignment(0, -1), end=ft.Alignment(0, 1),
        colors=[p['sidebar_start'], p['sidebar_end']],
    )
def main(page: ft.Page):
    page.title = '11408 考研学习追踪器'
    page.window.width = 1060
    page.window.height = 700
    page.window.min_width = 900
    page.window.min_height = 600
    page.theme_mode = ft.ThemeMode.SYSTEM
    page.theme = ft.Theme(font_family='Microsoft YaHei')
    page.padding = 0
    page.fonts = {'mono': 'Consolas'}

    def is_dark():
        return page.theme_mode == ft.ThemeMode.DARK or (
            page.theme_mode == ft.ThemeMode.SYSTEM
            and page.platform_brightness == ft.Brightness.DARK
        )

    def get_p():
        return palette(is_dark())

    page.bgcolor = get_p()['bg']

    # -- PDF export (save with path dialog) --
    def do_export_pdf(e):
        import tkinter as tk
        from tkinter import filedialog
        errors = load_json("errors.json", [])
        if not errors:
            show_snack("暂无错题数据", get_p()["warning"])
            return
        root = tk.Tk()
        root.withdraw()
        path = filedialog.asksaveasfilename(
            defaultextension=".pdf",
            filetypes=[("PDF 文件", "*.pdf")],
            initialfile=f"错题报告_{today_str()}.pdf",
            title="导出错题报告 - PDF",
        )
        root.destroy()
        if path:
            ok = generate_pdf(path)
            if ok:
                show_snack(f"PDF 已导出: {path}")
            else:
                show_snack("导出失败", get_p()["danger"])

    def show_snack(text, color=None):
        if color is None:
            color = get_p()['success']
        page.snack_bar = ft.SnackBar(
            ft.Text(text, color=ft.Colors.WHITE),
            bgcolor=color,
            behavior=ft.SnackBarBehavior.FLOATING,
            shape=ft.RoundedRectangleBorder(radius=8),
        )
        page.snack_bar.open = True
        page.update()

    def action_btn(icon, text, on_click, color):
        clr = color or get_p()["accent"]
        return ft.TextButton(
            content=ft.Row([
                ft.Icon(icon, size=16, color=clr),
                ft.Text(text, size=12, weight=ft.FontWeight.W_500, color=clr),
            ], spacing=6, alignment=ft.MainAxisAlignment.CENTER),
            on_click=on_click,
            style=ft.ButtonStyle(
                bgcolor=clr + "12",
                overlay_color=clr + "35",
                padding=ft.padding.Padding(left=14, top=7, right=14, bottom=7),
                shape=ft.RoundedRectangleBorder(radius=BTN_RADIUS),
                side=ft.BorderSide(color=clr + "20", width=1),
            ),
        )

    active_nav = [0]
    sidebar_col = ft.Column(spacing=4, expand=True)

    def nav_item(icon_outlined, icon_filled, label, index):
        p = get_p()
        is_active = active_nav[0] == index

        def on_hover_nav(e):
            if not is_active:
                e.control.bgcolor = p['nav_hover'] if e.data == 'true' else None
                e.control.update()

        return ft.Container(
            content=ft.Row([
                ft.Container(
                    width=3, height=28, border_radius=2,
                    bgcolor=p['accent'] if is_active else None,
                    animate=ft.Animation(200, ft.AnimationCurve.EASE),
                ),
                ft.Icon(
                    icon_filled if is_active else icon_outlined,
                    color=p['sidebar_text'] if is_active else p['sidebar_sub'],
                    size=20,
                ),
                ft.Text(
                    label, size=13,
                    color=p['sidebar_text'] if is_active else p['sidebar_sub'],
                    weight=ft.FontWeight.W_600 if is_active else ft.FontWeight.NORMAL,
                ),
            ], spacing=12, vertical_alignment=ft.CrossAxisAlignment.CENTER),
            padding=ft.padding.Padding(left=12, top=10, right=16, bottom=10),
            border_radius=10,
            bgcolor=p['nav_active'] if is_active else None,
            on_hover=on_hover_nav,
            on_click=lambda e, i=index: navigate(i),
            animate=ft.Animation(200, ft.AnimationCurve.EASE),
        )

    def build_sidebar():
        p = get_p()
        sidebar_col.controls.clear()
        sidebar_col.controls.extend([
            nav_item(ft.Icons.LIGHTBULB_OUTLINED, ft.Icons.LIGHTBULB, '每日推送', 0),
            nav_item(ft.Icons.ADD_CIRCLE_OUTLINED, ft.Icons.ADD_CIRCLE, '添加错题', 1),
            nav_item(ft.Icons.LIST_ALT_OUTLINED, ft.Icons.LIST_ALT, '错题列表', 2),
            nav_item(ft.Icons.INSERT_CHART_OUTLINED, ft.Icons.INSERT_CHART, '薄弱分析', 3),
        ])

    build_sidebar()

    def toggle_theme(e):
        page.theme_mode = (
            ft.ThemeMode.DARK if not is_dark() else ft.ThemeMode.LIGHT
        )
        p = get_p()
        page.bgcolor = p['bg']
        pages = [build_home, build_add_error, build_view_errors, build_stats]
        page_switcher.content = pages[active_nav[0]]()
        build_sidebar()
        page.update()

    def sidebar():
        p = get_p()
        return ft.Container(
            content=ft.Column([
                ft.Container(
                    content=ft.Column([
                        ft.Row([
                            ft.Container(
                                content=ft.Icon(ft.Icons.SCHOOL,
                                                color=ft.Colors.WHITE, size=22),
                                width=40, height=40, border_radius=12,
                                bgcolor='#ffffff15',
                                alignment=ft.Alignment(0, 0),
                            ),
                        ], alignment=ft.MainAxisAlignment.CENTER),
                        ft.Text('11408', size=22, weight=ft.FontWeight.BOLD,
                                color=p['sidebar_text'],
                                text_align=ft.TextAlign.CENTER),
                        ft.Text('考研追踪器', size=11, color=p['sidebar_sub'],
                                text_align=ft.TextAlign.CENTER),
                    ], horizontal_alignment=ft.CrossAxisAlignment.CENTER, spacing=8),
                    padding=ft.padding.Padding(left=0, top=28, right=0, bottom=20),
                    gradient=sidebar_gradient(p),
                ),
                ft.Container(content=sidebar_col, padding=12, expand=True),
                ft.Container(
                    content=ft.Row([
                        ft.IconButton(
                            icon=ft.Icons.DARK_MODE if not is_dark()
                            else ft.Icons.LIGHT_MODE,
                            icon_color=p['sidebar_sub'],
                            icon_size=18,
                            on_click=toggle_theme,
                            tooltip='切换亮色/暗色主题',
                        ),
                    ], alignment=ft.MainAxisAlignment.CENTER),
                    padding=12,
                ),
            ]),
            width=200,
            gradient=sidebar_gradient(p),
        )
    def build_home():
        p = get_p()
        kb = get_knowledge_base()
        errors = load_json('errors.json', [])
        total_errors = len(errors)
        reviewed = sum(1 for e in errors if e.get('review_count', 0) >= 2)
        weak_errors = [e for e in errors if e.get('review_count', 0) < 2]
        random.shuffle(weak_errors)

        hour = datetime.now().hour
        greeting = '早上好' if 6 <= hour < 12 else (
            '下午好' if 12 <= hour < 18 else '晚上好'
        )

        # Hero banner
        hero = ft.Container(
            content=ft.Row([
                ft.Column([
                    ft.Text(f'{greeting}，考研人', size=22,
                            weight=ft.FontWeight.BOLD, color=ft.Colors.WHITE),
                    ft.Text(f'{today_str()}  ·  今日学习推送', size=13,
                            color='#ffffff90'),
                ], spacing=4),
                ft.Container(
                    content=ft.Row([
                        ft.Container(
                            content=ft.Column([
                                ft.Text(str(total_errors), size=20,
                                        weight=ft.FontWeight.BOLD,
                                        color=ft.Colors.WHITE,
                                        text_align=ft.TextAlign.CENTER),
                                ft.Text('错题总数', size=10, color='#ffffff80',
                                        text_align=ft.TextAlign.CENTER),
                            ], spacing=2,
                               horizontal_alignment=ft.CrossAxisAlignment.CENTER),
                            padding=ft.padding.Padding(
                                left=18, top=10, right=18, bottom=10),
                            border_radius=10, bgcolor='#ffffff15',
                        ),
                        ft.Container(
                            content=ft.Column([
                                ft.Text(
                                    f'{int(reviewed / max(total_errors, 1) * 100)}%',
                                    size=20, weight=ft.FontWeight.BOLD,
                                    color=ft.Colors.WHITE,
                                    text_align=ft.TextAlign.CENTER),
                                ft.Text('已复习', size=10, color='#ffffff80',
                                        text_align=ft.TextAlign.CENTER),
                            ], spacing=2,
                               horizontal_alignment=ft.CrossAxisAlignment.CENTER),
                            padding=ft.padding.Padding(
                                left=18, top=10, right=18, bottom=10),
                            border_radius=10, bgcolor='#ffffff15',
                        ),
                    ], spacing=10),
                ),
            ], alignment=ft.MainAxisAlignment.SPACE_BETWEEN,
               vertical_alignment=ft.CrossAxisAlignment.CENTER),
            gradient=hero_gradient(p),
            border_radius=CARD_RADIUS,
            padding=ft.padding.Padding(left=28, top=24, right=28, bottom=24),
            margin=ft.margin.Margin(bottom=24),
        )
        # Knowledge cards
        subjects_ordered = [
            '数据结构', '计算机组成原理', '操作系统',
            '计算机网络', '数学', '英语'
        ]

        def make_knowledge_card(subject):
            if subject not in kb or not kb[subject]:
                return None
            tip = random.choice(kb[subject])
            sc = SUBJECT_COLORS.get(subject, p['accent'])

            def on_hover_card(e):
                e.control.shadow = (
                    card_shadow(p, heavy=True)
                    if e.data == 'true' else card_shadow(p)
                )
                e.control.update()

            return ft.Container(
                content=ft.Row([
                    ft.Container(width=4, bgcolor=sc, border_radius=2),
                    ft.Column([
                        ft.Row([
                            ft.Icon(SUBJECT_ICONS.get(subject,
                                    ft.Icons.BOOK_OUTLINED),
                                    color=sc, size=18),
                            ft.Text(subject, size=14,
                                    weight=ft.FontWeight.W_600,
                                    color=p['text']),
                        ], spacing=8),
                        ft.Text(tip, size=12, color=p['text_sec']),
                    ], spacing=4, expand=True),
                ], spacing=12,
                   vertical_alignment=ft.CrossAxisAlignment.START),
                bgcolor=p['surface'], border_radius=CARD_RADIUS,
                padding=16, shadow=card_shadow(p),
                on_hover=on_hover_card,
                animate=ft.Animation(200, ft.AnimationCurve.EASE),
                margin=ft.margin.Margin(bottom=2),
            )

        knowledge_cards = []
        for s in subjects_ordered:
            card = make_knowledge_card(s)
            if card:
                knowledge_cards.append(card)

        # Review section
        review_cards = []
        for e in weak_errors[:5]:
            review_cards.append(
                ft.Container(
                    content=ft.Row([
                        ft.Text(f'#{e["id"]}', size=12,
                                weight=ft.FontWeight.BOLD,
                                color=SUBJECT_COLORS.get(e['subject'],
                                                         p['accent']),
                                width=36),
                        ft.Text(f'[{e["subject"]}]', size=11,
                                color=p['text_muted'], width=100),
                        ft.Text(e['question'][:35], size=12,
                                color=p['text_sec']),
                    ]),
                    padding=ft.padding.Padding(
                        left=12, top=8, right=12, bottom=8),
                    border_radius=8, bgcolor=p['bg'],
                )
            )

        if weak_errors:
            review_header = ft.Row([
                ft.Icon(ft.Icons.REPLAY_OUTLINED, color=p['warning'], size=18),
                ft.Text(f'今日复习推荐  ·  {len(weak_errors)} 道待复习',
                        size=14, weight=ft.FontWeight.W_600, color=p['text']),
            ], spacing=8)
        else:
            review_header = ft.Row([
                ft.Icon(ft.Icons.CHECK_CIRCLE, color=p['success'], size=18),
                ft.Text('所有错题已完成复习', size=14,
                        weight=ft.FontWeight.W_600, color=p['text']),
            ], spacing=8)
        review_section = ft.Container(
            content=ft.Column([
                review_header,
                ft.Divider(height=8, color=ft.Colors.TRANSPARENT),
                *review_cards,
            ] if review_cards else [review_header]),
            bgcolor=p['surface'], border_radius=CARD_RADIUS,
            padding=20, shadow=card_shadow(p),
            margin=ft.margin.Margin(top=20),
        )

        # Advice
        if total_errors < 20:
            advice = '当前错题量较少，保持刷题节奏，注意归纳总结'
        elif total_errors < 50:
            advice = '错题量适中，重点复习标记为概念不清的题目'
        else:
            advice = '错题量较大，建议暂停刷新题，集中复习旧错题'

        advice_card = ft.Container(
            content=ft.Row([
                ft.Container(
                    content=ft.Icon(ft.Icons.TIPS_AND_UPDATES,
                                    color=p['accent'], size=18),
                    width=36, height=36, border_radius=10,
                    bgcolor=p['accent'] + '15',
                    alignment=ft.Alignment(0, 0),
                ),
                ft.Text(advice, size=13, color=p['text_sec']),
            ], spacing=14,
               vertical_alignment=ft.CrossAxisAlignment.CENTER),
            bgcolor=p['surface'], border_radius=CARD_RADIUS,
            padding=ft.padding.Padding(left=20, top=14, right=20, bottom=14),
            shadow=card_shadow(p),
            margin=ft.margin.Margin(top=16),
        )

        scroll_content = ft.Column([
            hero,
            ft.Text('今日知识点', size=16, weight=ft.FontWeight.W_600, color=p['text']),
            ft.Divider(height=4, color=ft.Colors.TRANSPARENT),
            *knowledge_cards,
            review_section,
            advice_card,
        ], spacing=0, scroll=ft.ScrollMode.AUTO)

        return ft.Container(
            content=scroll_content, expand=True, padding=PAGE_PAD,
        )
    def build_add_error():
        p = get_p()

        subject_dd = ft.Dropdown(
            label='科目',
            options=[ft.dropdown.Option(s) for s in SUBJECTS],
            value=SUBJECTS[0],
            width=200,
        )

        question_tf = ft.TextField(
            label='题目', multiline=True, min_lines=2, max_lines=4,
            hint_text='输入题目内容...',
        )

        wrong_tf = ft.TextField(label='你的错答', hint_text='输入错误答案')
        correct_tf = ft.TextField(label='正确答案', hint_text='输入正确答案')
        reason_tf = ft.TextField(label='错误原因', hint_text='输入错误原因')
        tags_tf = ft.TextField(label='标签', hint_text='用空格分隔，如: 极限 导数')

        def clear_form():
            question_tf.value = ''
            wrong_tf.value = ''
            correct_tf.value = ''
            reason_tf.value = ''
            tags_tf.value = ''
            page.update()

        def on_submit(e):
            subject = subject_dd.value
            question = question_tf.value.strip()
            if not question:
                show_snack('题目不能为空', p['warning'])
                return
            wrong = wrong_tf.value.strip() or '未记录'
            correct = correct_tf.value.strip() or '未记录'
            reason = reason_tf.value.strip() or '未记录'
            tags = tags_tf.value.strip().split() if tags_tf.value else []
            add_error(subject, question, wrong, correct, reason, tags)
            show_snack(f'已记录错题 [{subject}]', p['success'])
            clear_form()

        form_group = ft.Container(
            content=ft.Column([
                subject_dd, question_tf,
                ft.Row([wrong_tf, correct_tf], spacing=12),
                reason_tf, tags_tf,
                ft.Divider(height=8, color=ft.Colors.TRANSPARENT),
                ft.Container(
                    content=ft.Text('提交错题', size=14,
                                    weight=ft.FontWeight.W_500,
                                    color=ft.Colors.WHITE),
                    padding=ft.padding.Padding(
                        left=24, top=10, right=24, bottom=10),
                    border_radius=BTN_RADIUS,
                    bgcolor=p['accent'],
                    on_click=on_submit,
                ),
            ], spacing=10),
            bgcolor=p['surface'],
            border_radius=CARD_RADIUS,
            padding=24,
            shadow=card_shadow(p),
        )

        return ft.Container(
            content=ft.Column([
                ft.Text('添加错题', size=20, weight=ft.FontWeight.W_600,
                        color=p['text']),
                ft.Divider(height=16, color=ft.Colors.TRANSPARENT),
                form_group,
            ]),
            expand=True, padding=PAGE_PAD,
        )
    def build_view_errors():
        p = get_p()
        current_subject = ['全部']

        def make_chip(label):
            is_active = current_subject[0] == label
            def on_chip_click(e):
                current_subject[0] = label
                refresh(None)
            return ft.Container(
                content=ft.Text(label, size=11,
                                weight=ft.FontWeight.W_500,
                                color=p['text'] if is_active else p['text_sec']),
                padding=ft.padding.Padding(left=12, top=5, right=12, bottom=5),
                border_radius=12,
                bgcolor=p['chip_active'] if is_active else p['chip_bg'],
                on_click=on_chip_click,
            )

        filter_keyword = ft.TextField(
            hint_text='搜索题目关键词...',
            border_radius=BTN_RADIUS,
            bgcolor=p['input_bg'],
            border_color=p['border'],
            prefix_icon=ft.Icons.SEARCH,
            text_style=ft.TextStyle(size=13, color=p['text']),
            hint_style=ft.TextStyle(color=p['text_muted']),
            width=220,
        )

        data_table = ft.DataTable(
            columns=[
                ft.DataColumn(ft.Text('编号', size=12,
                                      weight=ft.FontWeight.W_600,
                                      color=p['text_sec'])),
                ft.DataColumn(ft.Text('科目', size=12,
                                      weight=ft.FontWeight.W_600,
                                      color=p['text_sec'])),
                ft.DataColumn(ft.Text('题目摘要', size=12,
                                      weight=ft.FontWeight.W_600,
                                      color=p['text_sec'])),
                ft.DataColumn(ft.Text('复习', size=12,
                                      weight=ft.FontWeight.W_600,
                                      color=p['text_sec'])),
            ],
            column_spacing=8,
            border_radius=8,
            divider_thickness=1,
            data_row_min_height=38,
            data_row_max_height=48,
            heading_row_color=p['table_header'],
        )

        detail_text = ft.Text('选择一道错题查看详情', size=13,
                              color=p['text_muted'], font_family='mono')
        detail_container = ft.Container(
            content=ft.Column([
                ft.Row([
                    ft.Icon(ft.Icons.INFO_OUTLINED, size=16, color=p['text_muted']),
                    ft.Text('详情', size=13, weight=ft.FontWeight.W_600, color=p['text']),
                ], spacing=6),
                ft.Divider(height=8, color=ft.Colors.TRANSPARENT),
                detail_text,
            ]),
            bgcolor=p['surface'], border_radius=CARD_RADIUS,
            padding=20, shadow=card_shadow(p),
            animate=ft.Animation(250, ft.AnimationCurve.EASE),
        )

        selected_id = []

        def refresh(e):
            data_table.rows.clear()
            subject = current_subject[0] if current_subject[0] != '全部' else None
            kw = filter_keyword.value.strip() if filter_keyword.value else None
            errors = list_errors(subject=subject, keyword=kw if kw else None)

            for i, err in enumerate(errors):
                q = err['question'].replace('\n', ' ')[:50]
                sc = SUBJECT_COLORS.get(err['subject'], p['accent'])
                rc = err.get('review_count', 0)
                review_color = p['success'] if rc >= 2 else p['warning']
                row_color = p['row_alt'] if i % 2 == 0 else None

                data_table.rows.append(
                    ft.DataRow(
                        cells=[
                            ft.DataCell(ft.Text(str(err['id']), size=12, color=p['text_sec'])),
                            ft.DataCell(ft.Container(
                                content=ft.Text(err['subject'], size=11,
                                                color=ft.Colors.WHITE,
                                                weight=ft.FontWeight.W_500),
                                padding=ft.padding.Padding(left=8, top=3, right=8, bottom=3),
                                border_radius=6, bgcolor=sc,
                            )),
                            ft.DataCell(ft.Text(q, size=12, color=p['text'])),
                            ft.DataCell(ft.Container(
                                content=ft.Text(str(rc), size=12,
                                                weight=ft.FontWeight.BOLD,
                                                color=review_color,
                                                text_align=ft.TextAlign.CENTER),
                                padding=ft.padding.Padding(left=8, top=3, right=8, bottom=3),
                                border_radius=6, bgcolor=review_color + '15',
                            )),
                        ],
                        data=err['id'],
                        color=row_color,
                        on_select_change=on_row_select,
                    )
                )

            chips_row.controls = [make_chip('全部')] + [make_chip(s) for s in SUBJECTS]
            page.update()
        def on_row_select(e):
            if e.control.cells:
                eid = e.control.cells[0].content.value
                selected_id.clear()
                selected_id.append(int(eid))
                all_errors = load_json('errors.json', [])
                for err in all_errors:
                    if err['id'] == int(eid):
                        detail_text.value = format_error(err)
                        detail_text.color = p['text']
                        detail_text.update()
                        return

        def do_review(e):
            if not selected_id:
                show_snack('请先选择一道错题', p['warning'])
                return
            review_error(selected_id[0])
            show_snack(f'已标记复习 #{selected_id[0]}')
            refresh(None)

        def do_delete(e):
            if not selected_id:
                show_snack('请先选择一道错题', p['warning'])
                return

            def confirm_delete(ce):
                delete_error(selected_id[0])
                detail_text.value = ''
                detail_text.color = p['text_muted']
                show_snack(f'已删除 #{selected_id[0]}', p['danger'])
                page.dialog.open = False
                selected_id.clear()
                refresh(None)
                page.update()

            def close_dlg(ce):
                page.dialog.open = False
                page.update()

            page.dialog = ft.AlertDialog(
                title=ft.Text('确认删除', weight=ft.FontWeight.W_600),
                content=ft.Text(f'确定要删除错题 #{selected_id[0]} 吗？此操作不可撤销。'),
                actions=[
                    ft.TextButton('取消', on_click=close_dlg),
                    ft.Container(
                        content=ft.Text('删除', color=ft.Colors.WHITE, size=13),
                        padding=ft.padding.Padding(left=16, top=8, right=16, bottom=8),
                        border_radius=BTN_RADIUS, bgcolor=p['danger'],
                        on_click=confirm_delete,
                    ),
                ],
            )
            page.dialog.open = True
            page.update()
        chips_row = ft.Row(
            [make_chip('全部')] + [make_chip(s) for s in SUBJECTS],
            wrap=True, spacing=6, run_spacing=6,
        )

        action_row = ft.Row([
            chips_row,
            ft.Container(
                content=ft.Row([
                    filter_keyword,
                    action_btn(ft.Icons.CHECK_CIRCLE_OUTLINED, '标记复习', do_review, p['success']),
                    action_btn(ft.Icons.DELETE_OUTLINED, '删除', do_delete, p['danger']),
                    action_btn(ft.Icons.PICTURE_AS_PDF_OUTLINED, '导出 PDF', do_export_pdf, p['accent']),
                ], spacing=8),
            ),
        ], alignment=ft.MainAxisAlignment.SPACE_BETWEEN,
           vertical_alignment=ft.CrossAxisAlignment.CENTER, wrap=True)

        refresh(None)

        return ft.Container(
            content=ft.Column([
                ft.Text('错题列表', size=20, weight=ft.FontWeight.W_600, color=p['text']),
                ft.Divider(height=16, color=ft.Colors.TRANSPARENT),
                action_row,
                ft.Divider(height=14, color=ft.Colors.TRANSPARENT),
                ft.Container(
                    content=ft.Column([data_table], scroll=ft.ScrollMode.AUTO),
                    bgcolor=p['surface'], border_radius=CARD_RADIUS,
                    padding=8, expand=True, shadow=card_shadow(p),
                ),
                ft.Divider(height=12, color=ft.Colors.TRANSPARENT),
                detail_container,
            ], expand=True),
            expand=True, padding=PAGE_PAD,
        )
    def build_stats():
        p = get_p()
        errors = load_json('errors.json', [])
        total = len(errors)
        reviewed = sum(1 for e in errors if e.get('review_count', 0) >= 2)
        review_rate = int(reviewed / max(total, 1) * 100)

        subject_counts = {s: 0 for s in SUBJECTS}
        for e in errors:
            subject_counts[e['subject']] = subject_counts.get(e['subject'], 0) + 1

        max_count = max(subject_counts.values()) if subject_counts else 1
        weakest = max(subject_counts, key=subject_counts.get) if total > 0 else '暂无'
        weakest_color = SUBJECT_COLORS.get(weakest, p['accent'])

        def overview_card(icon, label, value, sub, color):
            return ft.Container(
                content=ft.Column([
                    ft.Row([
                        ft.Container(
                            content=ft.Icon(icon, size=18, color=color),
                            width=40, height=40, border_radius=12,
                            bgcolor=color + '18', alignment=ft.Alignment(0, 0),
                        ),
                    ]),
                    ft.Text(value, size=24, weight=ft.FontWeight.BOLD, color=p['text']),
                    ft.Text(label, size=12, color=p['text_sec']),
                    ft.Text(sub, size=11, color=p['text_muted']),
                ], spacing=4),
                bgcolor=p['surface'], border_radius=CARD_RADIUS,
                padding=20, shadow=card_shadow(p), expand=True,
            )

        overview_row = ft.Row([
            overview_card(ft.Icons.ERROR_OUTLINE, '错题总数', str(total),
                          f'已复习 {reviewed} 道', p['accent']),
            overview_card(ft.Icons.CHECK_CIRCLE_OUTLINE, '复习率', f'{review_rate}%',
                          f'{reviewed}/{total} 已完成', p['success']),
            overview_card(ft.Icons.WARNING_AMBER_OUTLINED, '最薄弱', weakest,
                          f'{subject_counts.get(weakest, 0)} 道错题', weakest_color),
        ], spacing=16)
        def progress_bar(subject, count):
            sc = SUBJECT_COLORS.get(subject, p['accent'])
            ratio = count / max(max_count, 1)
            target_width = int(ratio * 360)
            is_weakest = count == max_count and count > 0
            return ft.Container(
                content=ft.Column([
                    ft.Row([
                        ft.Text(subject, size=13, weight=ft.FontWeight.W_500,
                                color=p['text'], width=120),
                        ft.Text(str(count), size=13, weight=ft.FontWeight.BOLD,
                                color=sc, width=40),
                        ft.Container(
                            content=ft.Container(
                                width=target_width, height=22, border_radius=7,
                                gradient=ft.LinearGradient(
                                    begin=ft.Alignment(-1, 0), end=ft.Alignment(1, 0),
                                    colors=[sc, sc + 'CC'],
                                ),
                                animate=ft.Animation(600, ft.AnimationCurve.EASE_OUT),
                            ),
                            border_radius=7, bgcolor=p['bg'],
                            height=22, width=360,
                        ),
                        (ft.Icon(ft.Icons.LOCAL_FIRE_DEPARTMENT, color=p['warning'], size=16)
                         if is_weakest else ft.Container(width=16)),
                    ], vertical_alignment=ft.CrossAxisAlignment.CENTER, spacing=8),
                ]),
                padding=ft.padding.Padding(left=0, top=0, right=0, bottom=8),
            )

        progress_bars = [progress_bar(s, subject_counts.get(s, 0)) for s in SUBJECTS]

        weak_errors = [e for e in errors if e.get('review_count', 0) < 2]

        def weak_item(e):
            rc = e.get('review_count', 0)
            sc = SUBJECT_COLORS.get(e['subject'], p['accent'])
            dot_color = p['danger'] if rc == 0 else p['warning']
            return ft.Container(
                content=ft.Row([
                    ft.Container(width=8, height=8, border_radius=4, bgcolor=dot_color),
                    ft.Text(f'#{e["id"]}', size=12, weight=ft.FontWeight.BOLD, color=sc, width=36),
                    ft.Text(f'[{e["subject"]}]', size=11, color=p['text_muted'], width=110),
                    ft.Text(e['question'][:40], size=12, color=p['text_sec']),
                ], spacing=8, vertical_alignment=ft.CrossAxisAlignment.CENTER),
                padding=ft.padding.Padding(left=12, top=8, right=12, bottom=8),
                border_radius=8, bgcolor=p['bg'],
            )

        weak_list = [weak_item(e) for e in weak_errors[:10]]
        if len(weak_errors) > 10:
            weak_list.append(ft.Text(f'... 还有 {len(weak_errors) - 10} 道', size=12, color=p['text_muted']))
        weak_section = ft.Container(
            content=ft.Column([
                ft.Row([
                    ft.Icon(ft.Icons.WARNING_AMBER_OUTLINED, color=p['warning'], size=18),
                    ft.Text('需重点复习 （复习 < 2 次）', size=14,
                            weight=ft.FontWeight.W_600, color=p['text']),
                ]),
                ft.Divider(height=8, color=ft.Colors.TRANSPARENT),
                *(weak_list if weak_list else [
                    ft.Text('暂无需要重点复习的错题', size=13, color=p['success'])
                ]),
            ]),
            bgcolor=p['surface'], border_radius=CARD_RADIUS,
            padding=20, shadow=card_shadow(p),
            margin=ft.margin.Margin(top=20),
        )

        dist_section = ft.Container(
            content=ft.Column([
                ft.Text('各科错题分布', size=14, weight=ft.FontWeight.W_600, color=p['text']),
                ft.Divider(height=12, color=ft.Colors.TRANSPARENT),
                *progress_bars,
            ]),
            bgcolor=p['surface'], border_radius=CARD_RADIUS,
            padding=20, shadow=card_shadow(p),
            margin=ft.margin.Margin(top=20),
        )

        return ft.Container(
            content=ft.Column([
                ft.Row([
                    ft.Text('薄弱点分析', size=20, weight=ft.FontWeight.W_600, color=p['text']),
                    ft.Container(
                        content=ft.Row([
                            ft.Icon(ft.Icons.PICTURE_AS_PDF_OUTLINED, size=16, color=p['accent']),
                            ft.Text('导出 PDF', size=12, weight=ft.FontWeight.W_500, color=p['accent']),
                        ], spacing=6, alignment=ft.MainAxisAlignment.CENTER),
                        padding=ft.padding.Padding(left=14, top=7, right=14, bottom=7),
                        border_radius=BTN_RADIUS, bgcolor=p['accent'] + '12',
                        on_click=do_export_pdf,
                    ),
                ], alignment=ft.MainAxisAlignment.SPACE_BETWEEN,
                   vertical_alignment=ft.CrossAxisAlignment.CENTER),
                ft.Divider(height=16, color=ft.Colors.TRANSPARENT),
                overview_row, dist_section, weak_section,
            ], scroll=ft.ScrollMode.AUTO),
            expand=True, padding=PAGE_PAD,
        )
    def navigate(index):
        active_nav[0] = index
        pages = [build_home, build_add_error, build_view_errors, build_stats]
        page_switcher.content = pages[index]()
        build_sidebar()
        page.update()

    # Layout Assembly
    page_switcher = ft.AnimatedSwitcher(
        content=build_home(),
        transition=ft.AnimatedSwitcherTransition.FADE,
        duration=250,
        switch_in_curve=ft.AnimationCurve.EASE_OUT,
        switch_out_curve=ft.AnimationCurve.EASE_IN,
    )

    page.add(
        ft.Row([
            sidebar(),
            ft.VerticalDivider(width=1, color=get_p()['border']),
            ft.Container(
                content=page_switcher,
                expand=True,
                bgcolor=get_p()['bg'],
            ),
        ], expand=True, spacing=0)
    )

    page.update()


if __name__ == '__main__':
    ft.run(main)
