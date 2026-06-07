<script setup>
import { ref, onMounted } from "vue"
import { api } from "../api/index.js"
import MarkdownEditor from "./MarkdownEditor.vue"
import MarkdownRenderer from "./MarkdownRenderer.vue"

const emit = defineEmits(["snack"])
const subjects = ref([])
const editTab = ref("question")
const formTabs = [
  { key: "question", label: "题目" },
  { key: "wrong", label: "错答" },
  { key: "correct", label: "正解" },
  { key: "reason", label: "错因" },
  { key: "tags", label: "标签" },
]
const form = ref({
  subject: "",
  question: "",
  wrong: "",
  correct: "",
  reason: "",
  tags: "",
})

onMounted(async () => {
  try { const r = await api.getSubjects(); subjects.value = r.subjects }
  catch (e) { emit("snack", e.message, "#ef4444") }
})

async function submit() {
  if (!form.value.question.trim()) { emit("snack", "题目不能为空", "#f59e0b"); return }
  try {
    await api.addError({
      subject: form.value.subject,
      question: form.value.question.trim(),
      wrong: form.value.wrong.trim() || "未记录",
      correct: form.value.correct.trim() || "未记录",
      reason: form.value.reason.trim() || "未记录",
      tags: form.value.tags.trim() ? form.value.tags.trim().split(/\s+/) : [],
    })
    emit("snack", `已记录错题 [${form.value.subject}]`)
    form.value = { subject: form.value.subject, question: "", wrong: "", correct: "", reason: "", tags: "" }
  } catch (e) { emit("snack", e.message, "#ef4444") }
}
</script>

<template>
  <div>
    <h2 style="font-size:22px;font-weight:600;margin-bottom:16px">添加错题</h2>
    <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px;align-items:stretch">
      <!-- Left: Form -->
      <div style="display:flex;flex-direction:column;min-height:0">
        <div class="form-group">
          <label class="form-label">科目</label>
          <select v-model="form.subject" class="form-select">
            <option v-for="s in subjects" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>

        <div class="form-group">
          <label class="form-label">题目 <span style="color:var(--text-muted);font-weight:400">（支持 Markdown + LaTeX）</span></label>
          <MarkdownEditor v-model="form.question" placeholder="输入题目内容，支持 **Markdown** 和 $LaTeX$ 公式..." :rows="6" />
        </div>

        <div class="form-group">
          <label class="form-label">错答 <span style="color:var(--text-muted);font-weight:400">（支持 Markdown）</span></label>
          <MarkdownEditor v-model="form.wrong" placeholder="输入错误答案..." :rows="2" />
        </div>

        <div class="form-group">
          <label class="form-label">正解 <span style="color:var(--text-muted);font-weight:400">（支持 Markdown）</span></label>
          <MarkdownEditor v-model="form.correct" placeholder="输入正确答案..." :rows="2" />
        </div>

        <div class="form-group">
          <label class="form-label">错因 <span style="color:var(--text-muted);font-weight:400">（支持 Markdown）</span></label>
          <MarkdownEditor v-model="form.reason" placeholder="输入错误原因..." :rows="2" />
        </div>

        <div class="form-group">
          <label class="form-label">标签</label>
          <input v-model="form.tags" class="form-input" placeholder="用空格分隔，如: 极限 导数" />
        </div>

        <button class="btn btn-primary" @click="submit" style="font-size:13px;padding:10px 24px">提交错题</button>
      </div>

      <!-- Right: Live Preview -->
      <div>
        <div class="form-label" style="margin-bottom:8px">实时预览</div>
        <div class="card" style="padding:16px;min-height:400px;position:sticky;top:28px">
          <template v-if="!form.question.trim() && !form.wrong.trim() && !form.correct.trim() && !form.reason.trim()">
            <p style="color:var(--text-muted);font-size:13px">在左侧输入后将在此处显示完整预览</p>
          </template>
          <template v-else>
            <div v-if="form.question.trim()" style="margin-bottom:12px;padding-bottom:12px;border-bottom:1px solid var(--border)">
              <div style="font-size:12.5px;font-weight:600;color:var(--text-sec);margin-bottom:6px">📝 题目</div>
              <MarkdownRenderer :content="form.question" />
            </div>
            <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-bottom:12px">
              <div v-if="form.wrong.trim()" style="padding:8px;background:rgba(239,68,68,.08);border-radius:8px;border:1px solid rgba(239,68,68,.25);border-left:4px solid #ef4444">
                <div style="font-size:12.5px;font-weight:600;color:#ef4444;margin-bottom:4px">❌ 错答</div>
                <MarkdownRenderer :content="form.wrong" />
              </div>
              <div v-if="form.correct.trim()" style="padding:8px;background:rgba(16,185,129,.06);border-radius:8px;border:1px solid rgba(16,185,129,.2);border-left:4px solid #10b981">
                <div style="font-size:12.5px;font-weight:600;color:#10b981;margin-bottom:4px">✅ 正解</div>
                <MarkdownRenderer :content="form.correct" />
              </div>
            </div>
            <div v-if="form.reason.trim()">
              <div style="font-size:12.5px;font-weight:600;color:var(--text-sec);margin-bottom:4px">🔍 错因</div>
              <MarkdownRenderer :content="form.reason" />
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
