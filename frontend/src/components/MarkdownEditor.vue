<script setup>
import { ref, computed } from "vue"

const props = defineProps({
  modelValue: { type: String, default: "" },
  placeholder: { type: String, default: "输入内容，支持 Markdown + LaTeX..." },
  rows: { type: Number, default: 6 },
  fill: { type: Boolean, default: false },
  label: { type: String, default: "" },
})

const emit = defineEmits(["update:modelValue"])
const textarea = ref(null)
const expandImgs = ref(false)

const base64Re = /!\[[^\]]*\]\(data:image\/[^;]+;base64,[^)]+\)|<img\s+[^>]*src="data:image\/[^;]+;base64,[^"]+"[^>]*>/g

const hasBase64 = computed(() => {
  base64Re.lastIndex = 0
  return base64Re.test(props.modelValue)
})

// Collapsed view: replace each base64 image with <<IMG_N>>
const editText = computed(() => {
  if (expandImgs.value || !hasBase64.value) return props.modelValue
  let n = 0
  base64Re.lastIndex = 0
  const html = props.modelValue.replace(base64Re, () => `<<IMG_${++n}>>`)
  base64Re.lastIndex = 0
  return html
})

function onInput(e) {
  if (!expandImgs.value && hasBase64.value) {
    let result = e.target.value
    let idx = 0
    base64Re.lastIndex = 0
    for (const m of props.modelValue.matchAll(base64Re)) {
      result = result.replace(`<<IMG_${++idx}>>`, m[0])
    }
    base64Re.lastIndex = 0
    emit("update:modelValue", result)
  } else {
    emit("update:modelValue", e.target.value)
  }
}

function insertText(before, after = "") {
  if (!expandImgs.value && hasBase64.value) return  // avoid cursor confusion when collapsed
  const ta = textarea.value
  if (!ta) return
  const start = ta.selectionStart
  const end = ta.selectionEnd
  const selected = props.modelValue.slice(start, end)
  const newVal = props.modelValue.slice(0, start) + before + selected + after + props.modelValue.slice(end)
  emit("update:modelValue", newVal)
  requestAnimationFrame(() => {
    ta.focus()
    ta.setSelectionRange(start + before.length, start + before.length + selected.length)
  })
}

const tools = [
  { icon: "B", title: "加粗", action: () => insertText("**", "**") },
  { icon: "I", title: "斜体", action: () => insertText("*", "*") },
  { icon: "S̶", title: "删除线", action: () => insertText("~~", "~~") },
  { icon: "H", title: "标题", action: () => insertText("## ") },
  { icon: "•", title: "列表", action: () => insertText("- ") },
  { icon: "🔗", title: "链接", action: () => insertText("[", "](url)") },
  { icon: "📷", title: "图片", action: () => insertText("![alt](", ")") },
  { icon: "```", title: "代码块", action: () => insertText("```\n", "\n```") },
  { icon: "$", title: "行内公式", action: () => insertText("$", "$") },
  { icon: "$$", title: "公式块", action: () => insertText("$$\n", "\n$$") },
]
defineExpose({ insertText })
</script>

<template>
  <div class="md-editor" :class="{ 'md-editor--fill': fill }">
    <div v-if="label || fill" class="md-editor-header">
      <span class="form-label" style="margin-bottom:0">{{ label }}</span>
    </div>

    <!-- Toolbar -->
    <div class="md-toolbar">
      <button v-for="t in tools" :key="t.icon" class="md-tool" :title="t.title" @click="t.action">
        {{ t.icon }}
      </button>
      <span style="flex:1"></span>
      <button v-if="hasBase64" class="md-tool" title="折叠/展开 base64 图片" @click="expandImgs = !expandImgs" style="width:auto;padding:0 8px;font-size:11px">
        {{ expandImgs ? '📋 收起图片' : '🖼️ 展开图片' }}
      </button>
    </div>

    <!-- Editor -->
    <textarea ref="textarea"
              :value="editText"
              @input="onInput"
              class="md-textarea" :class="{ 'md-textarea--fill': fill }"
              :placeholder="placeholder"
              :rows="expandImgs || !hasBase64 ? rows : 3"
              spellcheck="false"></textarea>
  </div>
</template>

<style scoped>
.md-editor { border: 1px solid var(--border); border-radius: 8px; overflow: hidden; }
.md-editor-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 8px 12px; border-bottom: 1px solid var(--border);
  background: var(--bg);
}
.md-editor-tabs { display: flex; gap: 4px; }
.md-editor-tabs button {
  padding: 4px 12px; border-radius: 6px; border: none;
  font-size: 13px; cursor: pointer; background: transparent;
  color: var(--text-muted); transition: all .15s;
}
.md-editor-tabs button.active { background: var(--accent); color: #fff; }
.md-editor-tabs button:not(.active):hover { background: rgba(0,0,0,.05); }

.md-toolbar {
  display: flex; gap: 2px; padding: 4px 6px;
  border-bottom: 1px solid var(--border); background: var(--bg);
  flex-wrap: wrap;
}
.md-tool {
  width: 30px; height: 28px; display: flex; align-items: center;
  justify-content: center; border-radius: 4px; border: none;
  background: transparent; cursor: pointer; font-size: 12px;
  color: var(--text-sec); transition: all .12s;
}
.md-tool:hover { background: rgba(0,0,0,.06); color: var(--text); }

.md-textarea {
  width: 100%; border: none; resize: vertical; padding: 12px;
  font-family: "Consolas", "Courier New", monospace;
  font-size: 14.5px; line-height: 1.6; color: var(--text);
  background: var(--surface); outline: none; min-height: 100px;
}
.md-textarea::placeholder { color: var(--text-muted); }
.md-editor--fill { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.md-editor--fill .md-textarea--fill { flex: 1; height: auto !important; min-height: 150px; resize: none; }
.md-editor--fill .md-preview { flex: 1; overflow-y: auto; min-height: 150px; }

.md-preview {
  padding: 12px; min-height: 80px;
  font-size: 15px; line-height: 1.7;
}
</style>
