<script setup>
import { ref, onMounted } from "vue"
import { api } from "../api/index.js"
import MarkdownRenderer from "./MarkdownRenderer.vue"
import MarkdownEditor from "./MarkdownEditor.vue"
import { renderMd } from "../utils/markdown.js"
import { exportPdfReport } from "../utils/pdfExport.js"
import { useSubjects } from "../store/subjects.js"

const emit = defineEmits(["snack"])
const errors = ref([])
const currentSubject = ref("全部")
const keyword = ref("")
const selectedId = ref(null)
const editingError = ref(null)
const editTab = ref("question")
const formTabs = [
  { key: "question", label: "题目" },
  { key: "wrong", label: "错答" },
  { key: "correct", label: "正解" },
  { key: "reason", label: "错因" },
  { key: "tags", label: "标签" },
]
const detail = ref(null)
const showDeleteDlg = ref(false)
const showAddDlg = ref(false)
const showEditDlg = ref(false)
const ocrInputAdd = ref(null)
const ocrInputEdit = ref(null)
const ocrLoading = ref(false)
const addForm = ref({ subject:"", question:"", wrong:"", correct:"", reason:"", tags:"" })
const addTab = ref("question")
const editForm = ref({ subject:"", question:"", wrong:"", correct:"", reason:"", tags:"" })
const { subjectRef } = useSubjects()
const subjects = subjectRef

const colors = {}
const colorPool = ["#0EA5E9","#8B5CF6","#10B981","#F97316","#EC4899","#F59E0B","#6366F1","#14B8A6","#F43F5E","#EAB308"]
function subjectColor(name) { return colors[name] || colorPool[Math.abs(hashCode(name)) % colorPool.length] }
function hashCode(s) { let h = 0; for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0; return h }
function showText(v) { const t = (v || '').trim(); return t && t !== '未记录' }

onMounted(async () => {
  try {
    const r = await api.getSubjects()
    subjects.value = ["全部", ...r.subjects]
  } catch (e) { emit("snack", e.message, "#ef4444") }
  await refresh()
})

async function refresh() {
  try {
    const s = currentSubject.value === "全部" ? null : currentSubject.value
    const k = keyword.value.trim() || null
    const r = await api.getErrors(s, k)
    errors.value = r.errors
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

function selectError(e) {
  if (selectedId.value === e.id && detail.value) {
    detail.value = null
    selectedId.value = null
    editingError.value = null
  } else {
    selectedId.value = e.id
    editingError.value = e
    detail.value = e
  }
}

async function saveAdd(){
  if(!addForm.value.question.trim()){emit("snack","题目不能为空","#f59e0b");return}
  try{
    await api.addError({
      subject: addForm.value.subject,
      question: addForm.value.question.trim(),
      wrong: addForm.value.wrong.trim()||"未记录",
      correct: addForm.value.correct.trim()||"未记录",
      reason: addForm.value.reason.trim()||"未记录",
      tags: addForm.value.tags.trim()?addForm.value.tags.trim().split(/\s+/):[],
    })
    showAddDlg.value = false
    addForm.value = { subject:subjects.value[1]||"", question:"", wrong:"", correct:"", reason:"", tags:"" }
    addTab.value = "question"
    emit("snack","错题已添加")
    await refresh()
  }catch(e){emit("snack",e.message,"#ef4444")}
}

function doEdit(){
  const e = editingError.value
  if(!e){emit("snack","请先选择一道错题","#f59e0b");return}
  showEditDlg.value = true
  editForm.value = {
    subject: e.subject,
    question: e.question,
    wrong: e.wrong || "",
    correct: e.correct || "",
    reason: e.reason || "",
    tags: (e.tags||[]).join(" "),
  }
}

async function doReview() {
  if (!selectedId.value) { emit("snack", "请先选择一道错题", "#f59e0b"); return }
  try {
    await api.reviewError(selectedId.value)
    emit("snack", `已标记复习 #${selectedId.value}`)
    await refresh()
    detail.value = null; selectedId.value = null
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

function triggerOcrAdd() { ocrInputAdd.value?.click() }
function triggerOcrEdit() { ocrInputEdit.value?.click() }

async function doOcr(blob, targetForm, targetTab) {
  ocrLoading.value = true
  try {
    const result = await api.ocrImage(new File([blob], "clipboard.png", {type: blob.type || "image/png"}))
    targetForm.value[targetTab.value] = result.markdown || ""
    emit("snack", "OCR 识别完成")
  } catch (err) { emit("snack", err.message, "#ef4444") }
  finally { ocrLoading.value = false }
}

function onPasteAdd(e) {
  const items = e.clipboardData?.items || []
  for (const item of items) {
    if (item.type.startsWith("image/")) {
      e.preventDefault()
      doOcr(item.getAsFile(), addForm, addTab)
      return
    }
  }
}

function onPasteEdit(e) {
  const items = e.clipboardData?.items || []
  for (const item of items) {
    if (item.type.startsWith("image/")) {
      e.preventDefault()
      doOcr(item.getAsFile(), editForm, editTab)
      return
    }
  }
}

async function onOcrFileAdd(e) {
  const file = e.target.files[0]
  if (!file) return
  e.target.value = ""
  doOcr(file, addForm, addTab)
}

async function onOcrFileEdit(e) {
  const file = e.target.files[0]
  if (!file) return
  e.target.value = ""
  doOcr(file, editForm, editTab)
}

async function saveEdit(){
  if(!editForm.value.question.trim()){emit("snack","题目不能为空","#f59e0b");return}
  try{
    await api.updateError(selectedId.value,{
      subject: editForm.value.subject,
      question: editForm.value.question.trim(),
      wrong: editForm.value.wrong.trim()||"未记录",
      correct: editForm.value.correct.trim()||"未记录",
      reason: editForm.value.reason.trim()||"未记录",
      tags: editForm.value.tags.trim()?editForm.value.tags.trim().split(/\s+/):[],
    })
    // Update local array directly
    const idx = errors.value.findIndex(e=>e.id===selectedId.value)
    if(idx>=0){
      errors.value[idx] = {
        ...errors.value[idx],
        subject: editForm.value.subject,
        question: editForm.value.question.trim(),
        wrong: editForm.value.wrong.trim()||"未记录",
        correct: editForm.value.correct.trim()||"未记录",
        reason: editForm.value.reason.trim()||"未记录",
        tags: editForm.value.tags.trim()?editForm.value.tags.trim().split(/\s+/):[],
      }
      errors.value = [...errors.value]
    }
    showEditDlg.value = false
    detail.value = null
    emit("snack",`错题 #${selectedId.value} 已更新`)
    selectedId.value = null
    editingError.value = null
  }catch(e){emit("snack",e.message,"#ef4444")}
}

function confirmDelete() {
  if (!selectedId.value) { emit("snack", "请先选择一道错题", "#f59e0b"); return }
  showDeleteDlg.value = true
}

async function doDelete() {
  try {
    await api.deleteError(selectedId.value)
    emit("snack", `已删除 #${selectedId.value}`, "#ef4444")
    showDeleteDlg.value = false; selectedId.value = null; detail.value = null
    await refresh()
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

async function exportPdf() {
  try {
    const all = await api.getErrors()
    await exportPdfReport(all.errors)
  } catch (e) {
    emit("snack", e.message || "导出失败", "#ef4444")
  }
}

// Strip markdown for table summary display
function stripMd(s) {
  if (!s) return ""
  return s.replace(/\*\*(.+?)\*\*/g, "$1")
          .replace(/\*(.+?)\*/g, "$1")
          .replace(/`(.+?)`/g, "$1")
          .replace(/~~(.+?)~~/g, "$1")
          .replace(/\[(.+?)\]\(.+?\)/g, "$1")
          .replace(/#{1,6}\s/g, "")
          .replace(/\$\$(.+?)\$\$/gs, "$1")
          .replace(/\$(.+?)\$/g, "$1")
          .replace(/\n/g, " ")
          .trim()
}
function truncate(s, n = 50) { return stripMd(s).slice(0, n) }
</script>

<template>
  <div>
    <h2 style="font-size:22px;font-weight:600;margin-bottom:16px">错题列表</h2>

    <!-- Filters + Actions -->
    <div style="display:flex;flex-wrap:wrap;gap:12px;align-items:center;margin-bottom:14px;justify-content:space-between">
      <div style="display:flex;gap:6px;flex-wrap:wrap">
        <span v-for="s in subjects" :key="s" class="chip" :class="{ active: currentSubject === s }"
              @click="currentSubject = s; refresh()">{{ s }}</span>
      </div>
      <div style="display:flex;gap:8px;align-items:center;flex-wrap:wrap">
        <input v-model="keyword" class="form-input" style="width:180px" placeholder="搜索关键词..."
               @keyup.enter="refresh" />
        <button class="btn btn-ghost" @click="refresh">🔍 查询</button>
        <button class="btn btn-ghost" style="color:var(--danger);background:rgba(239,68,68,.1)" @click="confirmDelete">
          🗑️ 删除
        </button>
        <button class="btn btn-ghost" @click="showAddDlg = true" style="color:var(--accent);background:var(--accent)12">
          ➕ 添加
        </button>
        <button class="btn btn-ghost" @click="doEdit" :style="{ color: selectedId ? `var(--accent)` : `var(--text-muted)`, background: selectedId ? `var(--accent)12` : `transparent` }">
          ✏️ 编辑
        </button>
        <button class="btn btn-ghost" style="color:var(--accent);background:var(--accent)12" @click="exportPdf">
          📄 导出 PDF
        </button>
      </div>
    </div>

    <!-- Table -->
    <div class="card" style="padding:8px">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width:50px">编号</th>
            <th style="width:100px">科目</th>
            <th>题目摘要</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in errors" :key="e.id"
              :style="{ cursor:'pointer', background: selectedId === e.id ? 'rgba(99,102,241,.08)' : '' }"
              @click="selectError(e)">
            <td style="color:var(--text-sec)">{{ e.id }}</td>
            <td>
              <span class="badge" :style="{ background: subjectColor(e.subject) }">{{ e.subject }}</span>
            </td>
            <td><div class="md-inline" v-html="renderMd(e.question || '')"></div></td>
          </tr>
          <tr v-if="!errors.length">
            <td colspan="4" style="text-align:center;padding:24px;color:var(--text-muted)">暂无错题数据</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Detail with Markdown rendering -->
    <Transition name="detail">
    <div v-if="detail" class="card" style="margin-top:12px;padding:20px">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:12px">
        <div style="display:flex;gap:6px;align-items:center">
          <span style="font-size:14px">ℹ️</span>
          <span style="font-size:14px;font-weight:600">详情 · #{{ detail.id }}</span>
        </div>
        <span class="badge" :style="{ background: colors[detail.subject] || '#6366f1' }">{{ detail.subject }}</span>
      </div>

      <!-- Question -->
      <div style="margin-bottom:12px">
        <div style="font-size:13.5px;font-weight:600;color:var(--text-sec);margin-bottom:4px">📝 题目</div>
        <MarkdownRenderer :content="detail.question" />
      </div>

      <!-- Wrong / Correct side by side -->
      <div v-if="detail.wrong && detail.wrong !== '未记录' || detail.correct && detail.correct !== '未记录'" style="margin-bottom:12px">
        <div v-if="detail.wrong && detail.wrong !== '未记录'" class="wrong-card" style="padding:12px;background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.25);border-left:4px solid #ef4444;border-radius:8px;margin-bottom:8px">
          <div style="font-size:13.5px;font-weight:600;color:#ef4444;margin-bottom:4px">❌ 错答</div>
          <MarkdownRenderer :content="detail.wrong" />
        </div>
        <div v-if="detail.correct && detail.correct !== '未记录'" class="correct-card" style="padding:12px;background:rgba(16,185,129,.06);border:1px solid rgba(16,185,129,.2);border-left:4px solid #10b981;border-radius:8px">
          <div style="font-size:13.5px;font-weight:600;color:#10b981;margin-bottom:4px">✅ 正解</div>
          <MarkdownRenderer :content="detail.correct" />
      </div>
      </div>

      <!-- Reason -->
      <div v-if="detail.reason && detail.reason !== '未记录'" style="margin-bottom:8px">
        <div style="font-size:13.5px;font-weight:600;color:var(--text-sec);margin-bottom:4px">🔍 错因</div>
        <MarkdownRenderer :content="detail.reason" />
      </div>

      <!-- Meta -->
      <div style="display:flex;gap:16px;font-size:12.5px;color:var(--text-muted);margin-top:8px;padding-top:8px;border-top:1px solid var(--border)">
        <span v-if="detail.tags?.length">🏷️ {{ detail.tags.join(", ") }}</span>
        <span>💡 复习：{{ detail.review_count || 0 }} 次</span>
        <span>📅 {{ detail.created?.slice(0,10) }}</span>
      </div>
    </div>
    </Transition>

    <!-- Add Dialog -->
    <Teleport to="body">
      <div v-if="showAddDlg" class="dialog-overlay" @paste="onPasteAdd">
        <div class="dialog" style="width:94vw;height:92vh;display:flex;flex-direction:column;animation:fadeInUp .2s;border-radius:14px">
          <div style="display:flex;justify-content:space-between;align-items:center;padding:20px 20px 0">
            <h3 style="font-weight:600;font-size:16px">➕ 添加错题</h3>
            <button class="btn" style="font-size:18px;padding:4px 8px" @click="showAddDlg = false">✕</button>
          </div>
          <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px;flex:1;overflow:hidden;padding:16px 24px 24px">
            <div style="overflow-y:auto;padding-right:12px;display:flex;flex-direction:column;min-height:0">
              <div class="form-group">
                <label class="form-label">科目</label>
                <select v-model="addForm.subject" class="form-select">
                  <option v-for="s in subjectRef" :key="s" :value="s">{{ s }}</option>
                </select>
              </div>
              <div style="display:flex;gap:4px;margin-bottom:12px;flex-wrap:wrap">
                <button v-for="t in formTabs" :key="t.key" class="chip" :class="{ active: addTab === t.key }" @click="addTab = t.key" style="font-size:12px">{{ t.label }}</button>
              </div>
              <div v-if="addTab === 'question'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="addForm.question" :fill="true" :editOnly="true" />
              </div>
              <div v-if="addTab === 'wrong'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="addForm.wrong" :fill="true" :editOnly="true" />
              </div>
              <div v-if="addTab === 'correct'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="addForm.correct" :fill="true" :editOnly="true" />
              </div>
              <div v-if="addTab === 'reason'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="addForm.reason" :fill="true" :editOnly="true" />
              </div>
              <div v-if="addTab === 'tags'" class="form-group">
                <input v-model="addForm.tags" class="form-input" placeholder="用空格分隔" />
              </div>
              <div style="display:flex;gap:8px;align-items:center;margin-top:12px">
                <input type="file" accept="image/*" style="display:none" ref="ocrInputAdd" @change="onOcrFileAdd" />
                <button class="btn btn-ghost" style="font-size:11px;color:var(--accent)" @click="triggerOcrAdd" :disabled="ocrLoading">📷 OCR 导入</button>
                <span v-if="ocrLoading" style="font-size:11px;color:var(--text-muted)">识别中...</span>
                <span v-else style="font-size:10px;color:var(--text-muted)">或 Ctrl+V 粘贴图片</span>
                <div style="flex:1"></div>
                <button class="btn" @click="showAddDlg = false">取消</button>
                <button class="btn btn-primary" style="padding:8px 16px" @click="saveAdd">提交</button>
              </div>
            </div>
            <div style="overflow-y:auto;padding-left:12px;border-left:1px solid var(--border)">
              <div class="form-label" style="margin-bottom:8px">实时预览</div>
              <template v-if="!showText(addForm.question) && !showText(addForm.wrong) && !showText(addForm.correct) && !showText(addForm.reason)">
                <p style="color:var(--text-muted);font-size:13px">在左侧输入后将在此处显示预览</p>
              </template>
              <template v-else>
                <div v-if="showText(addForm.question)" style="margin-bottom:12px;padding-bottom:12px;border-bottom:1px solid var(--border)">
                  <div style="font-size:11px;font-weight:600;color:var(--text-sec);margin-bottom:6px">题目</div>
                  <MarkdownRenderer :content="addForm.question" />
                </div>
                <div v-if="showText(addForm.wrong) || showText(addForm.correct)" style="display:flex;flex-direction:column;gap:8px;margin-bottom:12px">
                  <div v-if="showText(addForm.wrong)" style="padding:8px;background:rgba(239,68,68,.04);border-radius:8px;border:1px solid rgba(239,68,68,.1)">
                    <div style="font-size:11px;font-weight:600;color:var(--danger);margin-bottom:4px">错答</div>
                    <MarkdownRenderer :content="addForm.wrong" />
                  </div>
                  <div v-if="showText(addForm.correct)" style="padding:8px;background:rgba(16,185,129,.04);border-radius:8px;border:1px solid rgba(16,185,129,.1)">
                    <div style="font-size:11px;font-weight:600;color:var(--success);margin-bottom:4px">正解</div>
                    <MarkdownRenderer :content="addForm.correct" />
                  </div>
                </div>
                <div v-if="showText(addForm.reason)">
                  <div style="font-size:11px;font-weight:600;color:var(--text-sec);margin-bottom:4px">错因</div>
                  <MarkdownRenderer :content="addForm.reason" />
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="showEditDlg" class="dialog-overlay" @paste="onPasteEdit">
        <div class="dialog" style="width:94vw;height:92vh;display:flex;flex-direction:column;animation:fadeInUp .2s;border-radius:14px">
          <div style="display:flex;justify-content:space-between;align-items:center;padding:20px 20px 0">
            <h3 style="font-weight:600;font-size:16px">编辑错题 #{{ selectedId }}</h3>
            <button class="btn" style="font-size:18px;padding:4px 8px" @click="showEditDlg = false">✕</button>
          </div>

          <div style="display:grid;grid-template-columns:1fr 1fr;gap:20px;flex:1;overflow:hidden;padding:16px 24px 24px">
            <!-- Left: Form -->
            <div style="overflow-y:auto;padding-right:12px;display:flex;flex-direction:column;min-height:0">
              <div class="form-group">
                <label class="form-label">科目</label>
                <select v-model="editForm.subject" class="form-select">
                  <option v-for="s in subjectRef" :key="s" :value="s">{{ s }}</option>
                </select>
              </div>

              <div style="display:flex;gap:4px;margin-bottom:12px;flex-wrap:wrap">
                <button v-for="t in formTabs" :key="t.key"
                        class="chip" :class="{ active: editTab === t.key }"
                        @click="editTab = t.key" style="font-size:12px">{{ t.label }}</button>
              </div>

              <div v-if="editTab === 'question'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="editForm.question" :fill="true" :editOnly="true" />
              </div>
              <div v-if="editTab === 'wrong'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="editForm.wrong" :fill="true" :editOnly="true" />
              </div>
              <div v-if="editTab === 'correct'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="editForm.correct" :fill="true" :editOnly="true" />
              </div>
              <div v-if="editTab === 'reason'" style="flex:1;display:flex;flex-direction:column;min-height:0">
                <MarkdownEditor v-model="editForm.reason" :fill="true" :editOnly="true" />
              </div>
              <div v-if="editTab === 'tags'" class="form-group">
                <input v-model="editForm.tags" class="form-input" placeholder="用空格分隔" />
              </div>

              <div style="display:flex;gap:8px;align-items:center;margin-top:12px">
                <input type="file" accept="image/*" style="display:none" ref="ocrInputEdit" @change="onOcrFileEdit" />
                <button class="btn btn-ghost" style="font-size:11px;color:var(--accent)" @click="triggerOcrEdit" :disabled="ocrLoading">📷 OCR 导入</button>
                <span v-if="ocrLoading" style="font-size:11px;color:var(--text-muted)">识别中...</span>
                <span v-else style="font-size:10px;color:var(--text-muted)">或 Ctrl+V 粘贴图片</span>
                <div style="flex:1"></div>
                <button class="btn" @click="showEditDlg = false">取消</button>
                <button class="btn btn-primary" style="padding:8px 16px" @click="saveEdit">保存</button>
              </div>
            </div>

            <!-- Right: Live Preview -->
            <div style="overflow-y:auto;padding-left:12px;border-left:1px solid var(--border)">
              <div class="form-label" style="margin-bottom:8px">实时预览</div>
              <template v-if="!showText(editForm.question) && !showText(editForm.wrong) && !showText(editForm.correct) && !showText(editForm.reason)">
                <p style="color:var(--text-muted);font-size:13px">在左侧输入后将在此处显示预览</p>
              </template>
              <template v-else>
                <div v-if="showText(editForm.question)" style="margin-bottom:12px;padding-bottom:12px;border-bottom:1px solid var(--border)">
                  <div style="font-size:11px;font-weight:600;color:var(--text-sec);margin-bottom:6px">题目</div>
                  <MarkdownRenderer :content="editForm.question" />
                </div>
                <div v-if="showText(editForm.wrong) || showText(editForm.correct)" style="display:flex;flex-direction:column;gap:8px;margin-bottom:12px">
                  <div v-if="showText(editForm.wrong)" style="padding:8px;background:rgba(239,68,68,.04);border-radius:8px;border:1px solid rgba(239,68,68,.1)">
                    <div style="font-size:11px;font-weight:600;color:var(--danger);margin-bottom:4px">错答</div>
                    <MarkdownRenderer :content="editForm.wrong" />
                  </div>
                  <div v-if="showText(editForm.correct)" style="padding:8px;background:rgba(16,185,129,.04);border-radius:8px;border:1px solid rgba(16,185,129,.1)">
                    <div style="font-size:11px;font-weight:600;color:var(--success);margin-bottom:4px">正解</div>
                    <MarkdownRenderer :content="editForm.correct" />
                  </div>
                </div>
                <div v-if="showText(editForm.reason)">
                  <div style="font-size:11px;font-weight:600;color:var(--text-sec);margin-bottom:4px">错因</div>
                  <MarkdownRenderer :content="editForm.reason" />
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Dialog -->
    <Teleport to="body">
      <div v-if="showDeleteDlg" class="dialog-overlay" @click.self="showDeleteDlg = false">
        <div class="dialog" style="animation:fadeInUp .2s">
          <h3 style="margin-bottom:12px;font-weight:600;font-size:16px">确认删除</h3>
          <p style="font-size:14.5px;color:var(--text-sec);margin-bottom:20px">确定要删除错题 #{{ selectedId }} 吗？此操作不可撤销。</p>
          <div style="display:flex;gap:8px;justify-content:flex-end">
            <button class="btn" @click="showDeleteDlg = false">取消</button>
            <button class="btn btn-danger" style="padding:8px 16px" @click="doDelete">删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.detail-enter-active { transition: all .25s ease-out; }
.detail-leave-active { transition: all .2s ease-in; }
.detail-enter-from { opacity: 0; transform: translateY(-12px); max-height: 0; }
.detail-leave-to { opacity: 0; transform: translateY(-8px); max-height: 0; overflow: hidden; }

.dialog-overlay {
  position: fixed; inset:0; background: rgba(0,0,0,.3);
  display: flex; align-items: center; justify-content: center;
  z-index: 9998; backdrop-filter: blur(2px);
}
.md-inline { font-size: 13px; line-height: 1.4; max-height: 2.8em; overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; }
.md-inline :deep(p) { display: inline; margin: 0; }
.md-inline :deep(hl), .md-inline :deep(h2), .md-inline :deep(h3), .md-inline :deep(blockquote), .md-inline :deep(pre), .md-inline :deep(ul), .md-inline :deep(ol), .md-inline :deep(table) { display: none; }
.md-inline :deep(code) { font-size: .9em; padding: 0 3px; }

.dialog {
  background: var(--surface); border-radius: var(--radius);
  padding: 24px; min-width: 320px; box-shadow: var(--shadow-lg);
}
</style>
