#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
11408 考研学习追踪器 - GUI 版本
基于 Tkinter，无需额外安装
"""

import sys, io
if sys.platform == 'win32':
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8')

import tkinter as tk
from tkinter import ttk, messagebox, scrolledtext
import sys
sys.path.insert(0, __import__('os').path.dirname(__import__('os').path.abspath(__file__)))

from utils.error_manager import (
    SUBJECTS, add_error, list_errors, review_error,
    delete_error, format_error
)
from utils.stats import show_stats as cli_show_stats
from utils.daily_push import daily_push as cli_daily_push
from utils.data_store import load_json


class StudyTrackerGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("11408 考研学习追踪器")
        self.root.geometry("900x650")
        self.root.configure(bg='#f5f5f5')

        # 字体
        self.font_title = ('Microsoft YaHei', 16, 'bold')
        self.font_label = ('Microsoft YaHei', 11)
        self.font_text = ('Consolas', 10)

        self._build_sidebar()
        self._build_main_area()
        self.show_home()

    def _build_sidebar(self):
        sidebar = tk.Frame(self.root, width=180, bg='#2c3e50')
        sidebar.pack(side=tk.LEFT, fill=tk.Y)
        sidebar.pack_propagate(False)

        title = tk.Label(sidebar, text='11408', font=('Microsoft YaHei', 20, 'bold'),
                         bg='#2c3e50', fg='white')
        title.pack(pady=20)

        buttons = [
            ('每日推送', self.show_home),
            ('添加错题', self.show_add_error),
            ('查看错题', self.show_list_errors),
            ('薄弱点分析', self.show_stats),
        ]
        for text, cmd in buttons:
            btn = tk.Button(sidebar, text=text, font=self.font_label,
                           bg='#34495e', fg='white', activebackground='#1abc9c',
                           relief=tk.FLAT, cursor='hand2',
                           command=cmd)
            btn.pack(fill=tk.X, padx=15, pady=8)

    def _build_main_area(self):
        self.main_frame = tk.Frame(self.root, bg='#f5f5f5')
        self.main_frame.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, padx=20, pady=20)

    def _clear_main(self):
        for w in self.main_frame.winfo_children():
            w.destroy()

    def show_home(self):
        self._clear_main()
        tk.Label(self.main_frame, text='每日学习推送', font=self.font_title,
                bg='#f5f5f5', fg='#2c3e50').pack(anchor='w', pady=(0, 10))

        text = scrolledtext.ScrolledText(self.main_frame, font=self.font_text,
                                         wrap=tk.WORD, bg='white', fg='#333',
                                         padx=10, pady=10)
        text.pack(fill=tk.BOTH, expand=True)

        # 重定向输出到文本框
        class TextRedirector:
            def __init__(self, widget):
                self.widget = widget
            def write(self, s):
                self.widget.insert(tk.END, s)
                self.widget.see(tk.END)
            def flush(self):
                pass

        old_stdout = sys.stdout
        sys.stdout = TextRedirector(text)
        try:
            cli_daily_push()
        except Exception as e:
            text.insert(tk.END, f'\n加载推送内容时出错: {e}\n')
        sys.stdout = old_stdout

    def show_add_error(self):
        self._clear_main()
        tk.Label(self.main_frame, text='添加错题', font=self.font_title,
                bg='#f5f5f5', fg='#2c3e50').pack(anchor='w', pady=(0, 15))

        form = tk.Frame(self.main_frame, bg='#f5f5f5')
        form.pack(fill=tk.X)

        # 科目
        tk.Label(form, text='科目：', font=self.font_label, bg='#f5f5f5').grid(row=0, column=0, sticky='e', pady=8)
        subject_var = tk.StringVar(value=SUBJECTS[0])
        subject_cb = ttk.Combobox(form, textvariable=subject_var, values=SUBJECTS, state='readonly', width=18)
        subject_cb.grid(row=0, column=1, sticky='w', pady=8)

        # 题目
        tk.Label(form, text='题目：', font=self.font_label, bg='#f5f5f5').grid(row=1, column=0, sticky='ne', pady=8)
        question_txt = tk.Text(form, width=60, height=3, font=self.font_text)
        question_txt.grid(row=1, column=1, sticky='w', pady=8)

        # 错答
        tk.Label(form, text='错答：', font=self.font_label, bg='#f5f5f5').grid(row=2, column=0, sticky='e', pady=8)
        wrong_entry = tk.Entry(form, width=60, font=self.font_text)
        wrong_entry.grid(row=2, column=1, sticky='w', pady=8)

        # 正解
        tk.Label(form, text='正解：', font=self.font_label, bg='#f5f5f5').grid(row=3, column=0, sticky='e', pady=8)
        correct_entry = tk.Entry(form, width=60, font=self.font_text)
        correct_entry.grid(row=3, column=1, sticky='w', pady=8)

        # 错因
        tk.Label(form, text='错因：', font=self.font_label, bg='#f5f5f5').grid(row=4, column=0, sticky='ne', pady=8)
        reason_txt = tk.Text(form, width=60, height=2, font=self.font_text)
        reason_txt.grid(row=4, column=1, sticky='w', pady=8)

        # 标签
        tk.Label(form, text='标签：', font=self.font_label, bg='#f5f5f5').grid(row=5, column=0, sticky='e', pady=8)
        tags_entry = tk.Entry(form, width=60, font=self.font_text)
        tags_entry.grid(row=5, column=1, sticky='w', pady=8)
        tk.Label(form, text='（用空格分隔，如：极限 导数）', font=('Microsoft YaHei', 9), bg='#f5f5f5', fg='gray').grid(row=6, column=1, sticky='w')

        def submit():
            subject = subject_var.get()
            question = question_txt.get('1.0', tk.END).strip()
            wrong = wrong_entry.get().strip() or '未记录'
            correct = correct_entry.get().strip() or '未记录'
            reason = reason_txt.get('1.0', tk.END).strip() or '未记录'
            tags = tags_entry.get().strip().split()

            if not question:
                messagebox.showwarning('提示', '题目不能为空')
                return

            add_error(subject, question, wrong, correct, reason, tags)
            messagebox.showinfo('成功', f'已记录错题 [{subject}]')

            # 清空表单
            question_txt.delete('1.0', tk.END)
            wrong_entry.delete(0, tk.END)
            correct_entry.delete(0, tk.END)
            reason_txt.delete('1.0', tk.END)
            tags_entry.delete(0, tk.END)

        btn = tk.Button(self.main_frame, text='提交错题', font=self.font_label,
                       bg='#1abc9c', fg='white', activebackground='#16a085',
                       relief=tk.FLAT, cursor='hand2', command=submit)
        btn.pack(anchor='w', pady=20)

    def show_list_errors(self):
        self._clear_main()
        tk.Label(self.main_frame, text='错题列表', font=self.font_title,
                bg='#f5f5f5', fg='#2c3e50').pack(anchor='w', pady=(0, 10))

        # 筛选栏
        filter_frame = tk.Frame(self.main_frame, bg='#f5f5f5')
        filter_frame.pack(fill=tk.X, pady=(0, 10))

        tk.Label(filter_frame, text='科目：', font=self.font_label, bg='#f5f5f5').pack(side=tk.LEFT)
        filter_subject = ttk.Combobox(filter_frame, values=['全部'] + SUBJECTS, state='readonly', width=12)
        filter_subject.set('全部')
        filter_subject.pack(side=tk.LEFT, padx=(0, 10))

        tk.Label(filter_frame, text='关键词：', font=self.font_label, bg='#f5f5f5').pack(side=tk.LEFT)
        filter_keyword = tk.Entry(filter_frame, width=25, font=self.font_text)
        filter_keyword.pack(side=tk.LEFT, padx=(0, 10))

        # Treeview
        columns = ('id', 'subject', 'question', 'review')
        tree = ttk.Treeview(self.main_frame, columns=columns, show='headings', height=18)
        tree.heading('id', text='编号')
        tree.heading('subject', text='科目')
        tree.heading('question', text='题目摘要')
        tree.heading('review', text='复习次数')
        tree.column('id', width=50, anchor='center')
        tree.column('subject', width=100, anchor='center')
        tree.column('question', width=500)
        tree.column('review', width=80, anchor='center')
        tree.pack(fill=tk.BOTH, expand=True)

        # 详情框
        detail_text = scrolledtext.ScrolledText(self.main_frame, height=6, font=self.font_text,
                                                wrap=tk.WORD, bg='white', fg='#333')
        detail_text.pack(fill=tk.X, pady=(10, 0))

        def refresh():
            tree.delete(*tree.get_children())
            subject = filter_subject.get()
            keyword = filter_keyword.get().strip()
            errors = list_errors(
                subject=subject if subject != '全部' else None,
                keyword=keyword if keyword else None
            )
            for e in errors:
                q = e['question'].replace('\n', ' ')[:50]
                tree.insert('', tk.END, values=(e['id'], e['subject'], q, e.get('review_count', 0)))

        def on_select(event):
            sel = tree.selection()
            if not sel:
                return
            item = tree.item(sel[0])
            eid = item['values'][0]
            errors = load_json('errors.json', [])
            for e in errors:
                if e['id'] == eid:
                    detail_text.delete('1.0', tk.END)
                    detail_text.insert(tk.END, format_error(e))
                    return

        tree.bind('<<TreeviewSelect>>', on_select)

        def do_review():
            sel = tree.selection()
            if not sel:
                messagebox.showwarning('提示', '请先选择一道错题')
                return
            eid = tree.item(sel[0])['values'][0]
            review_error(eid)
            refresh()

        def do_delete():
            sel = tree.selection()
            if not sel:
                messagebox.showwarning('提示', '请先选择一道错题')
                return
            eid = tree.item(sel[0])['values'][0]
            if messagebox.askyesno('确认', f'确定删除错题 #{eid}？'):
                delete_error(eid)
                detail_text.delete('1.0', tk.END)
                refresh()

        btn_frame = tk.Frame(self.main_frame, bg='#f5f5f5')
        btn_frame.pack(fill=tk.X, pady=5)
        tk.Button(btn_frame, text='查询', font=self.font_label, bg='#3498db', fg='white',
                 relief=tk.FLAT, cursor='hand2', command=refresh).pack(side=tk.LEFT, padx=(0, 5))
        tk.Button(btn_frame, text='标记复习', font=self.font_label, bg='#1abc9c', fg='white',
                 relief=tk.FLAT, cursor='hand2', command=do_review).pack(side=tk.LEFT, padx=5)
        tk.Button(btn_frame, text='删除', font=self.font_label, bg='#e74c3c', fg='white',
                 relief=tk.FLAT, cursor='hand2', command=do_delete).pack(side=tk.LEFT, padx=5)

        refresh()

    def show_stats(self):
        self._clear_main()
        tk.Label(self.main_frame, text='薄弱点分析', font=self.font_title,
                bg='#f5f5f5', fg='#2c3e50').pack(anchor='w', pady=(0, 10))

        text = scrolledtext.ScrolledText(self.main_frame, font=self.font_text,
                                         wrap=tk.WORD, bg='white', fg='#333',
                                         padx=10, pady=10)
        text.pack(fill=tk.BOTH, expand=True)

        class TextRedirector:
            def __init__(self, widget):
                self.widget = widget
            def write(self, s):
                self.widget.insert(tk.END, s)
                self.widget.see(tk.END)
            def flush(self):
                pass

        old_stdout = sys.stdout
        sys.stdout = TextRedirector(text)
        try:
            cli_show_stats()
        except Exception as e:
            text.insert(tk.END, f'\n统计出错: {e}\n')
        sys.stdout = old_stdout


def main():
    root = tk.Tk()
    app = StudyTrackerGUI(root)
    root.mainloop()


if __name__ == '__main__':
    main()
