<script setup>
import { computed, ref, onMounted } from "vue"
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
const searchMode = ref("全部")
const searchModes = ["全部", "题目", "题目标签", "错因标签"]
const selectedId = ref(null)
const detail = ref(null)
const showDeleteDlg = ref(false)
const showAddDlg = ref(false)
const showEditDlg = ref(false)
const showExportDlg = ref(false)
const exportStyle = ref("detailed")
const ocrInputAdd = ref(null)
const ocrInputEdit = ref(null)
const ocrLoading = ref(false)
const addActiveField = ref("question")
const editActiveField = ref("question")
const splitLayout = ref(null)
const leftPane = ref(Number(localStorage.getItem("errorListLeftPane") || 42))
const editorLayoutAdd = ref(null)
const editorLayoutEdit = ref(null)
const editorLeftPane = ref(Number(localStorage.getItem("editorLeftPane") || 56))
const addForm = ref(emptyForm())
const editForm = ref(emptyForm())
const { subjectRef } = useSubjects()
const filterSubjects = computed(() => ["全部", ...subjectRef.value])
const selectedError = computed(() => errors.value.find(e => e.id === selectedId.value) || detail.value)
const today = new Date().toISOString().slice(0, 10)
const pendingReviews = computed(() => errors.value.filter(e => isDue(e)).length)

const colorPool = ["#0EA5E9","#8B5CF6","#10B981","#F97316","#EC4899","#F59E0B","#6366F1","#14B8A6","#F43F5E","#EAB308"]
const editorFields = [
  { key: "question", label: "题目" },
  { key: "wrong", label: "错答" },
  { key: "correct", label: "正解" },
  { key: "reason", label: "错因" },
]
const exportStyles = [
  { key: "detailed", title: "详细复盘", desc: "题目、错答、正解、错因完整展开，适合整理归档。" },
  { key: "compact", title: "紧凑打印", desc: "缩小间距和字号，适合一次打印大量错题。" },
  { key: "practice", title: "练习自测", desc: "先出题并留答题区，答案解析集中放在后面。" },
]
function hashCode(s) { let h = 0; for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0; return h }
function subjectColor(name) { return colorPool[Math.abs(hashCode(name || "")) % colorPool.length] }
function emptyForm(subject = "") { return { subject, question:"", title:"", wrong:"", correct:"", reason:"", tags:"", reason_tags:"" } }
function showText(v) { const t = (v || "").trim(); return t && t !== "未记录" }
function fieldLabel(key) { return editorFields.find(f => f.key === key)?.label || "题目" }
function isDue(e) { return (e.next_review || e.created?.slice(0, 10) || today) <= today }
function reviewPlanText(e) {
  const next = e.next_review || e.created?.slice(0, 10)
  const round = (e.review_count || 0) + 1
  if (!next) return `第 ${round} 轮复习`
  if (next < today) return `逾期 ${next} · 第 ${round} 轮`
  if (next === today) return `今日到期 · 第 ${round} 轮`
  return `下次 ${next} · 第 ${round} 轮`
}

onMounted(async () => {
  try {
    const r = await api.getSubjects()
    subjectRef.value = r.subjects
    addForm.value.subject = r.subjects[0] || ""
  } catch (e) { emit("snack", e.message, "#ef4444") }
  await refresh()
})

async function refresh() {
  try {
    const s = currentSubject.value === "全部" ? null : currentSubject.value
    const k = keyword.value.trim() || null
    const q = searchMode.value === "题目" ? k : null
    const t = searchMode.value === "题目标签" ? k : null
    const rt = searchMode.value === "错因标签" ? k : null
    const kw = searchMode.value === "全部" ? k : null
    const r = await api.getErrors(s, q || kw, t, rt)
    errors.value = r.errors
    if (selectedId.value) {
      detail.value = errors.value.find(e => e.id === selectedId.value) || null
      if (!detail.value) selectedId.value = null
    }
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

function selectError(e) {
  selectedId.value = e.id
  detail.value = e
}

function openAddDialog() {
  addForm.value = emptyForm(subjectRef.value[0] || "")
  addActiveField.value = "question"
  showAddDlg.value = true
}

function openEditDialog() {
  const e = selectedError.value
  if (!e) { emit("snack", "请先选择一道错题", "#f59e0b"); return }
  editForm.value = {
    subject: e.subject,
    title: e.title || e.question?.slice(0, 40) || "",
    question: e.question || "",
    wrong: e.wrong || "",
    correct: e.correct || "",
    reason: e.reason || "",
    tags: (e.tags || []).join(" "),
    reason_tags: (e.reason_tags || []).join(" "),
  }
  editActiveField.value = "question"
  showEditDlg.value = true
}

function payloadFromForm(form) {
  return {
    subject: form.subject,
    title: form.title.trim(),
    question: form.question.trim(),
    wrong: form.wrong.trim() || "未记录",
    correct: form.correct.trim() || "未记录",
    reason: form.reason.trim() || "未记录",
    tags: form.tags.trim() ? form.tags.trim().split(/\s+/) : [],
    reason_tags: form.reason_tags.trim() ? form.reason_tags.trim().split(/\s+/) : [],
  }
}

async function saveAdd(){
  if(!addForm.value.subject){emit("snack","请先添加并选择科目","#f59e0b");return}
  if(!addForm.value.question.trim()){emit("snack","题目不能为空","#f59e0b");return}
  try{
    const r = await api.addError(payloadFromForm(addForm.value))
    showAddDlg.value = false
    emit("snack","错题已添加")
    await refresh()
    const created = errors.value.find(e => e.id === r.id)
    if (created) selectError(created)
  }catch(e){emit("snack",e.message,"#ef4444")}
}

async function saveEdit(){
  if(!editForm.value.question.trim()){emit("snack","题目不能为空","#f59e0b");return}
  try{
    await api.updateError(selectedId.value, payloadFromForm(editForm.value))
    showEditDlg.value = false
    emit("snack",`错题 #${selectedId.value} 已更新`)
    await refresh()
  }catch(e){emit("snack",e.message,"#ef4444")}
}

async function doReview() {
  if (!selectedId.value) { emit("snack", "请先选择一道错题", "#f59e0b"); return }
  try {
    const result = await api.reviewError(selectedId.value)
    emit("snack", result.next_review ? `已复习 #${selectedId.value}，下次 ${result.next_review}` : `已标记复习 #${selectedId.value}`)
    await refresh()
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

function confirmDelete() {
  if (!selectedId.value) { emit("snack", "请先选择一道错题", "#f59e0b"); return }
  showDeleteDlg.value = true
}

async function doDelete() {
  try {
    const id = selectedId.value
    await api.deleteError(id)
    emit("snack", `已删除 #${id}`, "#ef4444")
    showDeleteDlg.value = false
    selectedId.value = null
    detail.value = null
    await refresh()
  } catch (e) { emit("snack", e.message, "#ef4444") }
}

function openExportDialog() {
  exportStyle.value = localStorage.getItem("pdfExportStyle") || "detailed"
  showExportDlg.value = true
}

async function exportPdf(style = exportStyle.value) {
  try {
    localStorage.setItem("pdfExportStyle", style)
    const all = await api.getErrors()
    showExportDlg.value = false
    await exportPdfReport(all.errors, { style })
  } catch (e) {
    emit("snack", e.message || "导出失败", "#ef4444")
  }
}

async function doOcr(blob, formRef, targetRef) {
  let tokenOk = false
  try { const t = await api.getToken(); tokenOk = t.configured } catch(e) {}
  if (!tokenOk) { emit("snack","请先在设置中配置 MinerU Token 再使用 OCR","#f59e0b"); return }
  ocrLoading.value = true
  try {
    const result = await api.ocrImage(new File([blob], "clipboard.png", {type: blob.type || "image/png"}))
    const text = (result.markdown || "").replace(/\$\$([^$]+)\$\$/g, (_, m) => "$" + m.replace(/\n\s*/g, " ") + "$")
    const key = targetRef.value
    formRef.value[key] = (formRef.value[key] || "") + (formRef.value[key] ? "\n\n" : "") + text
    emit("snack", "OCR 识别完成")
  } catch (err) { emit("snack", err.message, "#ef4444") }
  finally { ocrLoading.value = false }
}

function onPasteAdd(e) {
  const item = [...(e.clipboardData?.items || [])].find(i => i.type.startsWith("image/"))
  if (item) { e.preventDefault(); doOcr(item.getAsFile(), addForm, addActiveField) }
}
function onPasteEdit(e) {
  const item = [...(e.clipboardData?.items || [])].find(i => i.type.startsWith("image/"))
  if (item) { e.preventDefault(); doOcr(item.getAsFile(), editForm, editActiveField) }
}
function onOcrFileAdd(e) {
  const file = e.target.files[0]
  if (file) doOcr(file, addForm, addActiveField)
  e.target.value = ""
}
function onOcrFileEdit(e) {
  const file = e.target.files[0]
  if (file) doOcr(file, editForm, editActiveField)
  e.target.value = ""
}
function setTagSearch(mode, tag) {
  searchMode.value = mode
  keyword.value = tag
  refresh()
}

function startResize(e) {
  if (window.innerWidth <= 1060) return
  e.preventDefault()
  const el = splitLayout.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  const onMove = (ev) => {
    const x = ev.clientX ?? ev.touches?.[0]?.clientX
    if (x == null) return
    const percent = Math.min(65, Math.max(28, ((x - rect.left) / rect.width) * 100))
    leftPane.value = Math.round(percent * 10) / 10
  }
  const onUp = () => {
    localStorage.setItem("errorListLeftPane", String(leftPane.value))
    window.removeEventListener("mousemove", onMove)
    window.removeEventListener("mouseup", onUp)
    window.removeEventListener("touchmove", onMove)
    window.removeEventListener("touchend", onUp)
    document.body.classList.remove("resizing-pane")
  }
  document.body.classList.add("resizing-pane")
  window.addEventListener("mousemove", onMove)
  window.addEventListener("mouseup", onUp)
  window.addEventListener("touchmove", onMove, { passive: false })
  window.addEventListener("touchend", onUp)
}

function startEditorResize(e, layoutEl) {
  if (window.innerWidth <= 1060) return
  e.preventDefault()
  const el = layoutEl?.value || layoutEl
  if (!el) return
  const rect = el.getBoundingClientRect()
  const onMove = (ev) => {
    const x = ev.clientX ?? ev.touches?.[0]?.clientX
    if (x == null) return
    const percent = Math.min(72, Math.max(42, ((x - rect.left) / rect.width) * 100))
    editorLeftPane.value = Math.round(percent * 10) / 10
  }
  const onUp = () => {
    localStorage.setItem("editorLeftPane", String(editorLeftPane.value))
    window.removeEventListener("mousemove", onMove)
    window.removeEventListener("mouseup", onUp)
    window.removeEventListener("touchmove", onMove)
    window.removeEventListener("touchend", onUp)
    document.body.classList.remove("resizing-pane")
  }
  document.body.classList.add("resizing-pane")
  window.addEventListener("mousemove", onMove)
  window.addEventListener("mouseup", onUp)
  window.addEventListener("touchmove", onMove, { passive: false })
  window.addEventListener("touchend", onUp)
}

</script>

<template>
  <div class="errors-workbench">
    <header class="page-head">
      <div>
        <h2>错题复习</h2>
        <p>{{ errors.length }} 道当前结果 · {{ pendingReviews }} 道待复习</p>
      </div>
      <div class="page-actions">
        <button class="btn btn-ghost" @click="openExportDialog">导出</button>
        <button class="btn btn-primary" @click="openAddDialog">添加错题</button>
      </div>
    </header>

    <section class="filters-panel">
      <div class="subject-filter">
        <button v-for="s in filterSubjects" :key="s" class="chip" :class="{ active: currentSubject === s }" @click="currentSubject = s; refresh()">{{ s }}</button>
      </div>
      <div class="search-filter">
        <select v-model="searchMode" class="form-select">
          <option v-for="m in searchModes" :key="m" :value="m">{{ m }}</option>
        </select>
        <input v-model="keyword" class="form-input" placeholder="搜索标题、题目、标签或错因" @keyup.enter="refresh" />
        <button class="btn btn-ghost" @click="refresh">查询</button>
      </div>
    </section>

    <section ref="splitLayout" class="split-layout" :style="{ '--left-pane': leftPane + '%' }">
      <div class="error-list-panel">
        <article v-for="e in errors" :key="e.id" class="error-row" :class="{ active: selectedId === e.id }" @click="selectError(e)">
          <div class="row-top">
            <span class="row-id">#{{ e.id }}</span>
            <span class="badge" :style="{ background: subjectColor(e.subject) }">{{ e.subject }}</span>
            <span class="review-count" :class="{ due: isDue(e) }">{{ reviewPlanText(e) }}</span>
          </div>
          <div class="row-title" v-html="renderMd(e.title || e.question?.slice(0, 50) || '')"></div>
          <div class="row-tags">
            <button v-for="t in e.tags || []" :key="t" class="mini-chip" @click.stop="setTagSearch('题目标签', t)">{{ t }}</button>
            <button v-for="t in e.reason_tags || []" :key="t" class="mini-chip reason" @click.stop="setTagSearch('错因标签', t)">{{ t }}</button>
          </div>
        </article>
        <div v-if="!errors.length" class="empty-state">暂无错题数据</div>
      </div>

      <div class="pane-resizer" title="拖动调整列表和详情宽度" @mousedown="startResize" @touchstart="startResize">
        <span></span>
      </div>

      <aside class="detail-panel">
        <template v-if="selectedError">
          <div class="detail-head">
            <div>
              <span class="row-id">#{{ selectedError.id }}</span>
              <h3>{{ selectedError.title || selectedError.question?.slice(0, 40) }}</h3>
              <p>{{ selectedError.subject }} · {{ selectedError.created?.slice(0,10) }} · {{ reviewPlanText(selectedError) }}</p>
            </div>
            <span class="badge" :style="{ background: subjectColor(selectedError.subject) }">{{ selectedError.subject }}</span>
          </div>
          <div class="detail-actions">
            <button class="btn btn-primary" @click="doReview">标记已复习</button>
            <button class="btn btn-ghost" @click="openEditDialog">编辑</button>
            <button class="btn btn-ghost danger-action" @click="confirmDelete">删除</button>
          </div>

          <div class="detail-scroll">
            <section class="detail-section">
              <h4>题目</h4>
              <MarkdownRenderer :content="selectedError.question" />
            </section>
            <section v-if="showText(selectedError.wrong)" class="detail-section wrong-block">
              <h4>错答</h4>
              <MarkdownRenderer :content="selectedError.wrong" />
            </section>
            <section v-if="showText(selectedError.correct)" class="detail-section correct-block">
              <h4>正解</h4>
              <MarkdownRenderer :content="selectedError.correct" />
            </section>
            <section v-if="showText(selectedError.reason)" class="detail-section">
              <h4>错因</h4>
              <MarkdownRenderer :content="selectedError.reason" />
            </section>
            <div class="detail-tags">
              <span v-for="t in selectedError.tags || []" :key="t" class="mini-chip">{{ t }}</span>
              <span v-for="t in selectedError.reason_tags || []" :key="t" class="mini-chip reason">{{ t }}</span>
            </div>
          </div>
        </template>
        <div v-else class="empty-detail">
          <h3>选择一道错题</h3>
          <p>详情、复习、编辑和删除操作会固定显示在这里。</p>
        </div>
      </aside>
    </section>

    <Teleport to="body">
      <div v-if="showAddDlg" class="dialog-overlay editor-overlay" @paste="onPasteAdd">
        <div class="editor-dialog">
          <div class="editor-head">
            <h3>添加错题</h3>
            <button class="btn icon-btn" @click="showAddDlg = false">×</button>
          </div>
          <div ref="editorLayoutAdd" class="editor-body" :style="{ '--editor-left-pane': editorLeftPane + '%' }">
            <div class="form-stack">
              <div class="meta-grid">
                <label><span>科目</span><select v-model="addForm.subject" class="form-select"><option v-for="s in subjectRef" :key="s" :value="s">{{ s }}</option></select></label>
                <label><span>标题</span><input v-model="addForm.title" class="form-input" placeholder="列表中显示的标题" /></label>
                <label><span>题目标签</span><input v-model="addForm.tags" class="form-input" placeholder="空格分隔" /></label>
                <label><span>错因标签</span><input v-model="addForm.reason_tags" class="form-input" placeholder="空格分隔" /></label>
              </div>
              <div class="ocr-strip">
                <div class="field-tabs">
                  <button v-for="f in editorFields" :key="f.key" class="field-tab" :class="{ active: addActiveField === f.key }" @click="addActiveField = f.key">{{ f.label }}</button>
                </div>
                <button class="btn btn-ghost" :disabled="ocrLoading" @click="ocrInputAdd?.click()">OCR 插入</button>
              </div>
              <section class="single-editor">
                <h4>{{ fieldLabel(addActiveField) }}</h4>
                <MarkdownEditor v-if="addActiveField === 'question'" v-model="addForm.question" :fill="true" />
                <MarkdownEditor v-else-if="addActiveField === 'wrong'" v-model="addForm.wrong" :fill="true" />
                <MarkdownEditor v-else-if="addActiveField === 'correct'" v-model="addForm.correct" :fill="true" />
                <MarkdownEditor v-else v-model="addForm.reason" :fill="true" />
              </section>
            </div>
            <div class="editor-resizer" title="拖动调整编辑和预览宽度" @mousedown="startEditorResize($event, $refs.editorLayoutAdd)" @touchstart="startEditorResize($event, $refs.editorLayoutAdd)">
              <span></span>
            </div>
            <aside class="preview-panel">
              <div class="preview-head"><h4>实时预览</h4><span>{{ addForm.subject }}</span></div>
              <div v-if="!showText(addForm.question) && !showText(addForm.wrong) && !showText(addForm.correct) && !showText(addForm.reason)" class="empty-state">输入后显示预览</div>
              <template v-else>
                <section v-if="showText(addForm.question)"><h5>题目</h5><MarkdownRenderer :content="addForm.question" /></section>
                <section v-if="showText(addForm.wrong)" class="wrong-block"><h5>错答</h5><MarkdownRenderer :content="addForm.wrong" /></section>
                <section v-if="showText(addForm.correct)" class="correct-block"><h5>正解</h5><MarkdownRenderer :content="addForm.correct" /></section>
                <section v-if="showText(addForm.reason)"><h5>错因</h5><MarkdownRenderer :content="addForm.reason" /></section>
              </template>
            </aside>
          </div>
          <div class="editor-footer">
            <input type="file" accept="image/*" hidden ref="ocrInputAdd" @change="onOcrFileAdd" />
            <span>{{ ocrLoading ? "识别中..." : `OCR 和粘贴图片会插入到「${fieldLabel(addActiveField)}」` }}</span>
            <button class="btn" @click="showAddDlg = false">取消</button>
            <button class="btn btn-primary" @click="saveAdd">提交</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showEditDlg" class="dialog-overlay editor-overlay" @paste="onPasteEdit">
        <div class="editor-dialog">
          <div class="editor-head">
            <h3>编辑错题 #{{ selectedId }}</h3>
            <button class="btn icon-btn" @click="showEditDlg = false">×</button>
          </div>
          <div ref="editorLayoutEdit" class="editor-body" :style="{ '--editor-left-pane': editorLeftPane + '%' }">
            <div class="form-stack">
              <div class="meta-grid">
                <label><span>科目</span><select v-model="editForm.subject" class="form-select"><option v-for="s in subjectRef" :key="s" :value="s">{{ s }}</option></select></label>
                <label><span>标题</span><input v-model="editForm.title" class="form-input" placeholder="列表中显示的标题" /></label>
                <label><span>题目标签</span><input v-model="editForm.tags" class="form-input" placeholder="空格分隔" /></label>
                <label><span>错因标签</span><input v-model="editForm.reason_tags" class="form-input" placeholder="空格分隔" /></label>
              </div>
              <div class="ocr-strip">
                <div class="field-tabs">
                  <button v-for="f in editorFields" :key="f.key" class="field-tab" :class="{ active: editActiveField === f.key }" @click="editActiveField = f.key">{{ f.label }}</button>
                </div>
                <button class="btn btn-ghost" :disabled="ocrLoading" @click="ocrInputEdit?.click()">OCR 插入</button>
              </div>
              <section class="single-editor">
                <h4>{{ fieldLabel(editActiveField) }}</h4>
                <MarkdownEditor v-if="editActiveField === 'question'" v-model="editForm.question" :fill="true" />
                <MarkdownEditor v-else-if="editActiveField === 'wrong'" v-model="editForm.wrong" :fill="true" />
                <MarkdownEditor v-else-if="editActiveField === 'correct'" v-model="editForm.correct" :fill="true" />
                <MarkdownEditor v-else v-model="editForm.reason" :fill="true" />
              </section>
            </div>
            <div class="editor-resizer" title="拖动调整编辑和预览宽度" @mousedown="startEditorResize($event, $refs.editorLayoutEdit)" @touchstart="startEditorResize($event, $refs.editorLayoutEdit)">
              <span></span>
            </div>
            <aside class="preview-panel">
              <div class="preview-head"><h4>实时预览</h4><span>{{ editForm.subject }}</span></div>
              <div v-if="!showText(editForm.question) && !showText(editForm.wrong) && !showText(editForm.correct) && !showText(editForm.reason)" class="empty-state">输入后显示预览</div>
              <template v-else>
                <section v-if="showText(editForm.question)"><h5>题目</h5><MarkdownRenderer :content="editForm.question" /></section>
                <section v-if="showText(editForm.wrong)" class="wrong-block"><h5>错答</h5><MarkdownRenderer :content="editForm.wrong" /></section>
                <section v-if="showText(editForm.correct)" class="correct-block"><h5>正解</h5><MarkdownRenderer :content="editForm.correct" /></section>
                <section v-if="showText(editForm.reason)"><h5>错因</h5><MarkdownRenderer :content="editForm.reason" /></section>
              </template>
            </aside>
          </div>
          <div class="editor-footer">
            <input type="file" accept="image/*" hidden ref="ocrInputEdit" @change="onOcrFileEdit" />
            <span>{{ ocrLoading ? "识别中..." : `OCR 和粘贴图片会插入到「${fieldLabel(editActiveField)}」` }}</span>
            <button class="btn" @click="showEditDlg = false">取消</button>
            <button class="btn btn-primary" @click="saveEdit">保存</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showExportDlg" class="dialog-overlay" @click.self="showExportDlg = false">
        <div class="dialog export-dialog">
          <div class="export-head">
            <div>
              <h3>选择导出样式</h3>
              <p>导出会包含全部错题；当前列表显示 {{ errors.length }} 道。</p>
            </div>
            <button class="btn icon-btn" @click="showExportDlg = false">×</button>
          </div>
          <div class="export-options">
            <button
              v-for="style in exportStyles"
              :key="style.key"
              type="button"
              class="export-option"
              :class="{ active: exportStyle === style.key }"
              @click="exportStyle = style.key"
            >
              <span class="export-check"></span>
              <strong>{{ style.title }}</strong>
              <small>{{ style.desc }}</small>
            </button>
          </div>
          <div class="export-footer">
            <button class="btn" @click="showExportDlg = false">取消</button>
            <button class="btn btn-primary" @click="exportPdf()">导出 PDF</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showDeleteDlg" class="dialog-overlay" @click.self="showDeleteDlg = false">
        <div class="dialog confirm-dialog">
          <h3>确认删除</h3>
          <p>确定要删除错题 #{{ selectedId }} 吗？此操作不可撤销。</p>
          <div>
            <button class="btn" @click="showDeleteDlg = false">取消</button>
            <button class="btn btn-danger" @click="doDelete">删除</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style>
.errors-workbench { display: flex; flex-direction: column; gap: 14px; height: calc(100vh - 56px); min-height: 0; }
.page-head, .filters-panel { display: flex; justify-content: space-between; align-items: center; gap: 14px; }
.page-head h2 { font-size: 22px; margin-bottom: 3px; }
.page-head p { color: var(--text-muted); font-size: 13px; }
.page-actions, .search-filter, .subject-filter { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.search-filter .form-select { width: 96px; }
.search-filter .form-input { width: 260px; }
.filters-panel { padding: 12px; border: 1px solid var(--border); border-radius: 12px; background: var(--surface); }
.split-layout { display: grid; grid-template-columns: minmax(300px, var(--left-pane, 42%)) 10px minmax(360px, 1fr); gap: 8px; min-height: 0; flex: 1; }
.error-list-panel, .detail-panel { min-height: 0; overflow: auto; border: 1px solid var(--border); border-radius: 12px; background: var(--surface); }
.error-list-panel { padding: 8px; }
.pane-resizer { display: flex; align-items: center; justify-content: center; cursor: col-resize; user-select: none; border-radius: 8px; }
.pane-resizer:hover { background: var(--surface-hover); }
.pane-resizer span { width: 3px; height: 42px; border-radius: 2px; background: var(--border); transition: background .15s, height .15s; }
.pane-resizer:hover span, .resizing-pane .pane-resizer span { height: 72px; background: var(--accent); }
.resizing-pane { cursor: col-resize; user-select: none; }
.error-row { padding: 12px; border-radius: 10px; border: 1px solid transparent; cursor: pointer; transition: background .15s, border-color .15s; }
.error-row + .error-row { margin-top: 6px; }
.error-row:hover { background: var(--surface-muted); }
.error-row.active { background: rgba(99,102,241,.08); border-color: rgba(99,102,241,.24); }
.row-top { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.row-id { font-weight: 700; color: var(--accent); font-size: 13px; }
.review-count { margin-left: auto; font-size: 12px; color: var(--text-muted); }
.review-count.due { color: var(--warning); font-weight: 700; }
.row-title { font-size: 13.5px; line-height: 1.5; max-height: 3em; overflow: hidden; color: var(--text); }
.row-title :deep(p) { margin: 0; display: inline; }
.row-tags, .detail-tags { display: flex; gap: 5px; flex-wrap: wrap; margin-top: 9px; }
.mini-chip { border: 1px solid var(--border); background: var(--surface-muted); color: var(--text-sec); border-radius: 6px; padding: 2px 6px; font-size: 11px; cursor: pointer; }
.mini-chip.reason { color: #b45309; background: #fffbeb; border-color: #fde68a; }
.detail-panel { display: flex; flex-direction: column; padding: 16px; }
.detail-head { display: flex; justify-content: space-between; gap: 14px; align-items: flex-start; padding-bottom: 12px; border-bottom: 1px solid var(--border); }
.detail-head h3 { font-size: 18px; line-height: 1.4; margin: 4px 0; }
.detail-head p { color: var(--text-muted); font-size: 12px; }
.detail-actions { display: flex; gap: 8px; padding: 12px 0; border-bottom: 1px solid var(--border); }
.danger-action { color: var(--danger); background: rgba(239,68,68,.08); }
.detail-scroll { overflow: auto; padding-top: 12px; }
.detail-section { padding: 12px 0; border-bottom: 1px solid var(--border); }
.detail-section h4 { font-size: 13px; color: var(--text-sec); margin-bottom: 8px; }
.wrong-block, .correct-block { border-radius: 10px; padding: 12px; margin: 10px 0; border: 1px solid; }
.wrong-block { background: rgba(239,68,68,.04); border-color: rgba(239,68,68,.18); }
.correct-block { background: rgba(16,185,129,.04); border-color: rgba(16,185,129,.16); }
.empty-detail, .empty-state { margin: auto; text-align: center; color: var(--text-muted); font-size: 13px; padding: 28px; }
.empty-detail h3 { color: var(--text); margin-bottom: 6px; }
.editor-overlay { align-items: center; }
.editor-dialog { width: 94vw; height: 92vh; display: flex; flex-direction: column; background: var(--surface); border-radius: 12px; box-shadow: var(--shadow-lg); overflow: hidden; }
.editor-head { display: flex; justify-content: space-between; align-items: center; padding: 16px 18px; border-bottom: 1px solid var(--border); }
.editor-head h3 { font-size: 17px; }
.icon-btn {
  width: 40px;
  height: 40px;
  justify-content: center;
  padding: 0;
  border-radius: 9px;
  border-color: var(--border);
  background: var(--surface-muted);
  color: var(--text-sec);
  font-size: 26px;
  line-height: 1;
}
.icon-btn:hover {
  background: var(--surface-hover);
  color: var(--accent);
  filter: none;
}
.editor-body { display: grid; grid-template-columns: minmax(420px, var(--editor-left-pane, 58%)) 10px minmax(360px, 1fr); gap: 0; min-height: 0; flex: 1; }
.form-stack { overflow: hidden; padding: 18px; border-right: 1px solid var(--border); display: flex; flex-direction: column; min-height: 0; }
.form-stack section { margin-top: 14px; }
.form-stack h4 { font-size: 13px; margin-bottom: 6px; color: var(--text-sec); }
.meta-grid { display: grid; grid-template-columns: 140px 1fr 1fr 1fr; gap: 10px; margin-bottom: 10px; flex-shrink: 0; }
.meta-grid span { display: block; margin-bottom: 5px; font-size: 12px; color: var(--text-sec); font-weight: 600; }
.ocr-strip { display: flex; gap: 8px; align-items: center; padding: 10px; border: 1px solid var(--border); border-radius: 10px; background: var(--surface-muted); }
.field-tabs { display: flex; gap: 4px; flex: 1; min-width: 0; }
.field-tab { border: 1px solid transparent; background: transparent; color: var(--text-sec); border-radius: 7px; padding: 6px 14px; cursor: pointer; font-weight: 600; font-size: 12px; }
.field-tab:hover { background: var(--surface-hover); color: var(--accent); }
.field-tab.active { background: var(--accent); border-color: var(--accent); color: #fff; }
.single-editor { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.single-editor .md-editor { flex: 1; min-height: 0; }
.editor-resizer { display: flex; align-items: center; justify-content: center; cursor: col-resize; user-select: none; background: var(--surface-muted); border-right: 1px solid var(--border); }
.editor-resizer:hover { background: var(--surface-hover); }
.editor-resizer span { width: 3px; height: 54px; border-radius: 2px; background: var(--border); transition: background .15s, height .15s; }
.editor-resizer:hover span, .resizing-pane .editor-resizer span { height: 84px; background: var(--accent); }
.preview-panel { overflow: auto; padding: 18px; background: var(--surface-soft); }
.preview-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.preview-head h4 { font-size: 14px; }
.preview-head span { color: var(--text-muted); font-size: 12px; }
.preview-panel section { padding: 12px 0; border-bottom: 1px solid var(--border); }
.preview-panel h5 { font-size: 12px; color: var(--text-sec); margin-bottom: 8px; }
.editor-footer { display: flex; align-items: center; gap: 12px; padding: 14px 18px; border-top: 1px solid var(--border); }
.editor-footer span { margin-right: auto; color: var(--text-muted); font-size: 12px; }
.editor-footer .btn {
  min-width: 96px;
  height: 42px;
  justify-content: center;
  padding: 0 20px;
  border-radius: 9px;
  font-size: 14.5px;
}
.editor-footer .btn-primary {
  min-width: 108px;
  font-weight: 700;
}
.export-dialog {
  width: min(620px, 92vw);
  padding: 0;
  overflow: hidden;
}
.export-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
}
.export-head h3 {
  font-size: 17px;
  margin-bottom: 4px;
}
.export-head p {
  color: var(--text-muted);
  font-size: 12.5px;
}
.export-options {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 18px 20px;
}
.export-option {
  min-height: 138px;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: 10px;
  background: var(--surface-soft);
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color .15s, background .15s, box-shadow .15s, transform .15s;
}
.export-option:hover {
  border-color: color-mix(in srgb, var(--accent) 35%, var(--border));
  background: var(--surface-muted);
}
.export-option.active {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--accent) 16%, transparent);
}
.export-option.active .export-check {
  background: var(--accent);
  border-color: var(--accent);
}
.export-option.active .export-check::after {
  opacity: 1;
}
.export-check {
  width: 18px;
  height: 18px;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: var(--surface);
  position: relative;
}
.export-check::after {
  content: "";
  position: absolute;
  inset: 5px;
  border-radius: 50%;
  background: #fff;
  opacity: 0;
}
.export-option strong {
  font-size: 14px;
}
.export-option small {
  color: var(--text-sec);
  font-size: 12px;
  line-height: 1.55;
}
.export-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--border);
  background: var(--surface-muted);
}
.confirm-dialog h3 { margin-bottom: 10px; }
.confirm-dialog p { color: var(--text-sec); margin-bottom: 18px; }
.confirm-dialog div { display: flex; justify-content: flex-end; gap: 8px; }
@media (max-width: 1060px) {
  .errors-workbench { height: auto; }
  .split-layout, .editor-body { grid-template-columns: 1fr; }
  .pane-resizer { display: none; }
  .editor-resizer { display: none; }
  .detail-panel { min-height: 520px; }
  .form-stack { border-right: none; border-bottom: 1px solid var(--border); }
  .meta-grid { grid-template-columns: 1fr 1fr; }
  .export-options { grid-template-columns: 1fr; }
  .export-option { min-height: auto; }
}
</style>
