#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
PDF export module for 11408 study tracker.
Uses fpdf2 + Microsoft YaHei TTF font for Chinese PDF generation.
"""

import os
from datetime import datetime
from fpdf import FPDF

from .data_store import load_json
from .error_manager import SUBJECTS

FONT_PATH = "C:/Windows/Fonts/msyh.ttc"
FONT_BOLD_PATH = "C:/Windows/Fonts/msyhbd.ttc"


class StudyReport(FPDF):
    """Custom PDF class with Chinese font support."""

    def __init__(self):
        super().__init__(orientation='P', unit='mm', format='A4')
        self.set_auto_page_break(auto=True, margin=18)
        if os.path.exists(FONT_PATH):
            self.add_font("YaHei", "", FONT_PATH, uni=True)
        else:
            raise FileNotFoundError(f"Font not found: {FONT_PATH}")
        if os.path.exists(FONT_BOLD_PATH):
            self.add_font("YaHei", "B", FONT_BOLD_PATH, uni=True)
        else:
            self.add_font("YaHei", "B", FONT_PATH, uni=True)

    def header(self):
        if self.page_no() > 1:
            self.set_font("YaHei", "", 8)
            self.set_text_color(150, 150, 150)
            self.cell(0, 6, "11408 考研学习追踪器", align='R')
            self.ln(8)

    def footer(self):
        self.set_y(-15)
        self.set_font("YaHei", "", 8)
        self.set_text_color(150, 150, 150)
        self.cell(0, 10, f"{self.page_no()}", align='C')

    def section_title(self, title):
        self.set_font("YaHei", "B", 14)
        self.set_text_color(30, 41, 59)
        self.cell(0, 10, title)
        self.ln(6)
        self.set_draw_color(99, 102, 241)
        self.set_line_width(0.5)
        self.line(self.get_x(), self.get_y(), self.get_x() + 190, self.get_y())
        self.ln(6)

    def body_text(self, text, size=10):
        self.set_font("YaHei", "", size)
        self.set_text_color(71, 85, 105)
        self.multi_cell(0, 6, text)

    @staticmethod
    def subject_color(subject):
        colors = {
            '数据结构': (14, 165, 233),
            '计算机组成原理': (139, 92, 246),
            '操作系统': (16, 185, 129),
            '计算机网络': (249, 115, 22),
            '数学': (236, 72, 153),
            '英语': (245, 158, 11),
        }
        return colors.get(subject, (99, 102, 241))


def generate_pdf(save_path):
    """Generate a full analysis report PDF and save to save_path."""
    errors = load_json('errors.json', [])
    if not errors:
        return False

    total = len(errors)
    reviewed = sum(1 for e in errors if e.get('review_count', 0) >= 2)
    review_rate = int(reviewed / total * 100) if total > 0 else 0

    subject_counts = {s: 0 for s in SUBJECTS}
    for e in errors:
        subject_counts[e['subject']] = subject_counts.get(e['subject'], 0) + 1

    max_count = max(subject_counts.values()) if subject_counts else 1
    weakest = max(subject_counts, key=subject_counts.get) if total > 0 else "暂无"

    pdf = StudyReport()

    # ============================================================
    #  COVER PAGE
    # ============================================================
    pdf.add_page()
    pdf.ln(30)

    pdf.set_fill_color(99, 102, 241)
    pdf.rect(20, 25, 170, 3, 'F')

    pdf.set_font("YaHei", "B", 28)
    pdf.set_text_color(30, 41, 59)
    pdf.cell(0, 16, "11408 考研学习追踪器", align='C')
    pdf.ln(22)

    pdf.set_font("YaHei", "B", 16)
    pdf.set_text_color(99, 102, 241)
    pdf.cell(0, 10, "考研错题分析报告", align='C')
    pdf.ln(14)

    pdf.set_font("YaHei", "", 11)
    pdf.set_text_color(100, 116, 139)
    now_str = datetime.now().strftime('%Y-%m-%d %H:%M')
    pdf.cell(0, 8, f"生成时间：{now_str}", align='C')
    pdf.ln(16)

    box_width = 52
    box_height = 28
    x_start = 25
    y_start = pdf.get_y()

    def draw_stat_box(x, y, label, value, sub, r, g, b):
        pdf.set_fill_color(r, g, b)
        pdf.set_draw_color(r, g, b)
        pdf.rect(x, y, box_width, box_height, 'DF')
        pdf.set_xy(x, y + 5)
        pdf.set_font("YaHei", "B", 18)
        pdf.set_text_color(255, 255, 255)
        pdf.cell(box_width, 8, value, align='C')
        pdf.set_xy(x, y + 15)
        pdf.set_font("YaHei", "", 9)
        pdf.cell(box_width, 6, label, align='C')
        pdf.set_xy(x, y + 22)
        pdf.set_font("YaHei", "", 7)
        pdf.set_text_color(255, 255, 255)
        pdf.cell(box_width, 4, sub, align='C')

    draw_stat_box(x_start, y_start, "错题总数", str(total),
                  f"已复习 {reviewed} 道", 99, 102, 241)
    draw_stat_box(x_start + box_width + 7, y_start, "复习率", f"{review_rate}%",
                  f"{reviewed}/{total} 已完成", 16, 185, 129)
    draw_stat_box(x_start + 2 * (box_width + 7), y_start, "最薄弱",
                  weakest, f"{subject_counts.get(weakest, 0)} 道错题",
                  245, 158, 11)

    # ============================================================
    #  SUBJECT DISTRIBUTION
    # ============================================================
    pdf.add_page()
    pdf.section_title("科目分布")

    col_w = [50, 30, 90]
    pdf.set_font("YaHei", "B", 10)
    pdf.set_fill_color(241, 245, 249)
    pdf.set_text_color(30, 41, 59)
    pdf.cell(col_w[0], 8, " 科目", 'B', 0, 'C', fill=True)
    pdf.cell(col_w[1], 8, " 数量", 'B', 0, 'C', fill=True)
    pdf.cell(col_w[2], 8, " 分布条", 'B', 0, 'C', fill=True)
    pdf.ln(9)

    max_w = 80
    for s in SUBJECTS:
        cnt = subject_counts.get(s, 0)
        r, g, b = pdf.subject_color(s)
        bar_w = int(cnt / max(max_count, 1) * max_w) if max_count > 0 else 0

        pdf.set_font("YaHei", "B", 10)
        pdf.set_text_color(30, 41, 59)
        pdf.cell(col_w[0], 7, f"  {s}", 0, 0)

        pdf.set_font("YaHei", "B", 10)
        pdf.set_text_color(r, g, b)
        pdf.cell(col_w[1], 7, str(cnt), 0, 0, 'C')

        pdf.set_fill_color(r, g, b)
        pdf.rect(pdf.get_x(), pdf.get_y() + 1, bar_w, 5, 'F')
        if cnt == max_count and cnt > 0:
            pdf.set_font("YaHei", "", 8)
            pdf.set_text_color(245, 158, 11)
            pdf.cell(bar_w + 4, 7, " 最薄弱", 0, 0, 'C')

        pdf.ln(8)

    pdf.ln(6)

    # ============================================================
    #  ERROR DETAILS BY SUBJECT
    # ============================================================
    for s in SUBJECTS:
        subject_errors = [e for e in errors if e['subject'] == s]
        if not subject_errors:
            continue

        pdf.add_page()
        r, g, b = pdf.subject_color(s)
        pdf.set_fill_color(r, g, b)
        pdf.rect(20, pdf.get_y(), 170, 8, 'F')
        pdf.set_font("YaHei", "B", 13)
        pdf.set_text_color(255, 255, 255)
        pdf.set_y(pdf.get_y() + 1)
        pdf.cell(0, 8, f"  {s}  ({len(subject_errors)} 道)", align='L')
        pdf.ln(14)

        for e in subject_errors:
            if pdf.get_y() > 255:
                pdf.add_page()

            rc = e.get('review_count', 0)

            pdf.set_font("YaHei", "B", 10)
            pdf.set_text_color(30, 41, 59)
            pdf.cell(0, 7, f"#{e['id']}  |  复习: {rc} 次", align='L')
            pdf.ln(8)

            pdf.set_font("YaHei", "B", 9)
            pdf.set_text_color(71, 85, 105)
            pdf.cell(14, 6, "题目:")
            pdf.set_font("YaHei", "", 9)
            pdf.multi_cell(0, 6, e['question'])

            col_w2 = (pdf.w - pdf.l_margin - pdf.r_margin - 10) / 2
            pdf.set_font("YaHei", "B", 9)
            pdf.set_text_color(220, 38, 38)
            pdf.cell(14, 6, "错答:")
            pdf.set_font("YaHei", "", 9)
            pdf.set_text_color(71, 85, 105)
            pdf.cell(col_w2 - 14, 6, e.get('wrong', '未记录')[:40])
            pdf.set_font("YaHei", "B", 9)
            pdf.set_text_color(16, 185, 129)
            pdf.cell(14, 6, "正解:")
            pdf.set_font("YaHei", "", 9)
            pdf.set_text_color(71, 85, 105)
            pdf.cell(0, 6, e.get('correct', '未记录')[:40])
            pdf.ln(8)

            pdf.set_font("YaHei", "B", 9)
            pdf.set_text_color(71, 85, 105)
            pdf.cell(14, 6, "错因:")
            pdf.set_font("YaHei", "", 9)
            pdf.multi_cell(0, 6, e.get('reason', '未记录'))

            tags = e.get('tags', [])
            if tags:
                pdf.set_font("YaHei", "B", 9)
                pdf.set_text_color(71, 85, 105)
                pdf.cell(14, 6, "标签:")
                pdf.set_font("YaHei", "", 9)
                pdf.set_text_color(100, 116, 139)
                pdf.cell(0, 6, ", ".join(tags))
                pdf.ln(6)

            pdf.set_draw_color(226, 232, 240)
            pdf.set_line_width(0.3)
            pdf.line(pdf.get_x(), pdf.get_y() + 2, pdf.get_x() + 190, pdf.get_y() + 2)
            pdf.ln(6)

    # ============================================================
    #  WEAK POINTS
    # ============================================================
    pdf.add_page()
    pdf.section_title("重点复习区 (复习 < 2 次)")

    weak_errors = [e for e in errors if e.get('review_count', 0) < 2]
    if weak_errors:
        pdf.set_font("YaHei", "", 10)
        pdf.set_text_color(71, 85, 105)
        pdf.cell(0, 8, f"共 {len(weak_errors)} 道错题需重点复习", align='L')
        pdf.ln(12)

        col_w3 = [12, 80, 30, 68]
        pdf.set_font("YaHei", "B", 9)
        pdf.set_fill_color(241, 245, 249)
        pdf.set_text_color(30, 41, 59)
        pdf.cell(col_w3[0], 7, " 次数", 'B', 0, 'C', fill=True)
        pdf.cell(col_w3[1], 7, " 题目", 'B', 0, 'C', fill=True)
        pdf.cell(col_w3[2], 7, " 科目", 'B', 0, 'C', fill=True)
        pdf.cell(col_w3[3], 7, " 编号", 'B', 0, 'C', fill=True)
        pdf.ln(8)

        for e in weak_errors:
            rc = e.get('review_count', 0)
            if pdf.get_y() > 265:
                pdf.add_page()

            pdf.set_font("YaHei", "B", 9)
            if rc == 0:
                pdf.set_text_color(220, 38, 38)
            else:
                pdf.set_text_color(245, 158, 11)
            pdf.cell(col_w3[0], 6, str(rc), align='C')

            pdf.set_font("YaHei", "", 9)
            pdf.set_text_color(71, 85, 105)
            q = e['question'].replace('\n', ' ')[:38]
            pdf.cell(col_w3[1], 6, q)

            pdf.set_font("YaHei", "", 8)
            sc = pdf.subject_color(e['subject'])
            pdf.set_text_color(*sc)
            pdf.cell(col_w3[2], 6, e['subject'], align='C')

            pdf.set_font("YaHei", "", 8)
            pdf.set_text_color(148, 163, 184)
            pdf.cell(col_w3[3], 6, f"#{e['id']}", align='C')
            pdf.ln(7)
    else:
        pdf.set_font("YaHei", "", 11)
        pdf.set_text_color(16, 185, 129)
        pdf.cell(0, 8, "所有错题都已复习完成", align='C')

    # ============================================================
    #  STUDY ADVICE
    # ============================================================
    pdf.ln(16)
    pdf.section_title("学习建议")

    if total < 20:
        advice = "当前错题量较少，保持刷题节奏，注意归纳总结"
    elif total < 50:
        advice = "错题量适中，重点复习标记为概念不清的题目"
    else:
        advice = "错题量较大，建议暂停刷新题，集中复习旧错题"

    pdf.set_font("YaHei", "", 10)
    pdf.set_text_color(71, 85, 105)
    pdf.multi_cell(0, 7, advice)

    # Save
    try:
        pdf.output(save_path)
        return True
    except Exception as ex:
        print(f"PDF export error: {ex}")
        return False
