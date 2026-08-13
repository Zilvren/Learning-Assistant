<script setup>
import { computed, nextTick, ref, watch } from "vue"
import { FileImage, GripVertical, ScanText } from "lucide-vue-next"
import { api } from "../../api/index.js"
import { hasContent } from "../../composables/useErrorLibrary.js"
import { useToast } from "../../store/toast.js"
import MarkdownEditor from "../MarkdownEditor.vue"
import MarkdownRenderer from "../MarkdownRenderer.vue"
import BaseButton from "../ui/BaseButton.vue"
import BaseDialog from "../ui/BaseDialog.vue"

const props = defineProps({
  open: Boolean,
  mode: { type: String, default: "add" },
  record: Object,
  subjects: { type: Array, default: () => [] },
  busy: Boolean,
})
const emit = defineEmits(["request-close", "save", "dirty-change"])
const toast = useToast()
const activeField = ref("question")
const form = ref(emptyForm())
const initialState = ref("")
const ocrInput = ref(null)
const ocrLoading = ref(false)
const editorLayout = ref(null)
const editorLeftPane = ref(Number(localStorage.getItem("editorLeftPane") || 56))

const fields = [
  { key: "question", label: "题目" },
  { key: "wrong", label: "错答" },
  { key: "correct", label: "正解" },
  { key: "reason", label: "错因" },
]

const dirty = computed(() => props.open && JSON.stringify(form.value) !== initialState.value)
const heading = computed(() => props.mode === "edit" ? `修订错题 #${props.record?.id}` : "录入新错题")
const subheading = computed(() => props.mode === "edit" ? "修改会保留原有复习进度。" : "用题目、订正与错因组成一则完整的学习批注。")

// emptyForm 协调当前组件的状态和交互。
function emptyForm() {
  return { subject: "", title: "", question: "", wrong: "", correct: "", reason: "", tags: "", reason_tags: "" }
}

// resetForm 协调当前组件的状态和交互。
function resetForm() {
  const item = props.record
  form.value = item ? {
    subject: item.subject || props.subjects[0] || "",
    title: item.title || item.question?.slice(0, 40) || "",
    question: item.question || "",
    wrong: item.wrong === "未记录" ? "" : item.wrong || "",
    correct: item.correct === "未记录" ? "" : item.correct || "",
    reason: item.reason === "未记录" ? "" : item.reason || "",
    tags: (item.tags || []).join(" "),
    reason_tags: (item.reason_tags || []).join(" "),
  } : { ...emptyForm(), subject: props.subjects[0] || "" }
  activeField.value = "question"
  nextTick(() => { initialState.value = JSON.stringify(form.value) })
}

// payload 协调当前组件的状态和交互。
function payload() {
  return {
    subject: form.value.subject,
    title: form.value.title.trim(),
    question: form.value.question.trim(),
    wrong: form.value.wrong.trim() || "未记录",
    correct: form.value.correct.trim() || "未记录",
    reason: form.value.reason.trim() || "未记录",
    tags: form.value.tags.trim() ? form.value.tags.trim().split(/\s+/) : [],
    reason_tags: form.value.reason_tags.trim() ? form.value.reason_tags.trim().split(/\s+/) : [],
  }
}

// submit 协调当前组件的状态和交互。
function submit() {
  if (!form.value.subject) return toast.warning("请先选择科目")
  if (!form.value.question.trim()) return toast.warning("题目不能为空")
  emit("save", payload())
}

// runOcr 协调当前组件的状态和交互。
async function runOcr(blob) {
  if (!blob || ocrLoading.value) return
  let configured = false
  try { configured = (await api.getToken()).configured } catch { /* handled by OCR request */ }
  if (!configured) return toast.warning("请先在设置中心配置 MinerU Token")
  ocrLoading.value = true
  try {
    const file = blob instanceof File ? blob : new File([blob], "clipboard.png", { type: blob.type || "image/png" })
    const queued = await api.ocrImage(file)
    toast.info("OCR 已加入任务队列，识别完成后会自动插入")
    const result = await api.waitForOCRTask(queued.task?.id || queued.id)
    const text = (result.markdown || "").replace(/\$\$([^$]+)\$\$/g, (_, content) => `$${content.replace(/\n\s*/g, " ")}$`)
    const key = activeField.value
    form.value[key] += (form.value[key] ? "\n\n" : "") + text
    toast.success(`OCR 已插入“${fields.find((field) => field.key === key)?.label}”`)
  } catch (error) {
    toast.error(error.message || "OCR 识别失败")
  } finally {
    ocrLoading.value = false
  }
}

// onPaste 协调当前组件的状态和交互。
function onPaste(event) {
  const item = [...(event.clipboardData?.items || [])].find((entry) => entry.type.startsWith("image/"))
  if (!item) return
  event.preventDefault()
  runOcr(item.getAsFile())
}

// chooseOcrFile 协调当前组件的状态和交互。
function chooseOcrFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ""
  if (file) runOcr(file)
}

// pointerX 协调当前组件的状态和交互。
function pointerX(event) { return event.clientX ?? event.touches?.[0]?.clientX }
// startResize 协调当前组件的状态和交互。
function startResize(event) {
  if (window.innerWidth <= 1100 || !editorLayout.value) return
  event.preventDefault()
  const rect = editorLayout.value.getBoundingClientRect()
  // onMove 协调当前组件的状态和交互。
  const onMove = (moveEvent) => {
    const percent = ((pointerX(moveEvent) - rect.left) / rect.width) * 100
    editorLeftPane.value = Math.min(72, Math.max(42, percent))
  }
  // onUp 协调当前组件的状态和交互。
  const onUp = () => {
    localStorage.setItem("editorLeftPane", String(Math.round(editorLeftPane.value * 10) / 10))
    window.removeEventListener("pointermove", onMove)
    document.body.classList.remove("is-resizing")
  }
  document.body.classList.add("is-resizing")
  window.addEventListener("pointermove", onMove)
  window.addEventListener("pointerup", onUp, { once: true })
}

watch(() => props.open, (open) => { if (open) resetForm() }, { immediate: true })
watch(dirty, (value) => emit("dirty-change", value))
</script>

<template>
  <BaseDialog :open="open" :title="heading" :description="subheading" size="full" :close-on-backdrop="false" @close="emit('request-close')">
    <div ref="editorLayout" class="error-editor-layout" :style="{ '--editor-left-pane': `${editorLeftPane}%` }" @paste="onPaste">
      <div class="error-editor-form">
        <div class="editor-meta-grid">
          <label><span class="field-label">科目</span><select v-model="form.subject" class="field-control"><option v-for="subject in subjects" :key="subject">{{ subject }}</option></select></label>
          <label><span class="field-label">列表标题</span><input v-model="form.title" class="field-control" placeholder="用一句话概括题目" /></label>
          <label><span class="field-label">题目标签</span><input v-model="form.tags" class="field-control" placeholder="空格分隔" /></label>
          <label><span class="field-label">错因标签</span><input v-model="form.reason_tags" class="field-control" placeholder="空格分隔" /></label>
        </div>

        <div class="editor-field-tabs">
          <div role="tablist" aria-label="编辑字段">
            <button v-for="field in fields" :key="field.key" type="button" role="tab" :aria-selected="activeField === field.key" :class="{ active: activeField === field.key }" @click="activeField = field.key">{{ field.label }}</button>
          </div>
          <BaseButton size="sm" :busy="ocrLoading" @click="ocrInput?.click()"><template #icon><ScanText :size="16" /></template>OCR 插入</BaseButton>
          <input ref="ocrInput" type="file" accept="image/*" hidden @change="chooseOcrFile" />
        </div>

        <div class="active-editor">
          <div class="active-editor__caption"><FileImage :size="15" /><span>可直接粘贴截图进行 OCR；支持 Markdown 与 LaTeX。</span></div>
          <MarkdownEditor v-for="field in fields" v-show="activeField === field.key" :key="field.key" v-model="form[field.key]" :fill="true" :label="field.label" />
        </div>
      </div>

      <button type="button" class="editor-grip" aria-label="调整编辑与预览宽度" @pointerdown="startResize"><GripVertical :size="18" /></button>

      <aside class="editor-preview">
        <header><span class="page-eyebrow">实时预览</span><strong>{{ form.subject || "未选择科目" }}</strong></header>
        <div v-if="!fields.some((field) => hasContent(form[field.key]))" class="preview-blank">输入内容后，这里会像最终错题页一样排版。</div>
        <div v-else class="preview-manuscript">
          <section v-if="hasContent(form.question)"><h3>题目</h3><MarkdownRenderer :content="form.question" /></section>
          <section v-if="hasContent(form.wrong)" class="answer-note answer-note--wrong"><h3>错解批注</h3><MarkdownRenderer :content="form.wrong" /></section>
          <section v-if="hasContent(form.correct)" class="answer-note answer-note--correct"><h3>正解订正</h3><MarkdownRenderer :content="form.correct" /></section>
          <section v-if="hasContent(form.reason)"><h3>错因归纳</h3><MarkdownRenderer :content="form.reason" /></section>
        </div>
      </aside>
    </div>
    <template #footer>
      <span class="editor-save-note">{{ dirty ? "有尚未保存的修改" : "内容已保持同步" }}</span>
      <BaseButton @click="emit('request-close')">取消</BaseButton>
      <BaseButton variant="primary" :busy="busy" @click="submit">{{ mode === 'edit' ? '保存修订' : '收入错题本' }}</BaseButton>
    </template>
  </BaseDialog>
</template>
