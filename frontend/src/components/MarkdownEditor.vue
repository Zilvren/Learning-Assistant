<script setup>
import { computed, nextTick, ref } from "vue"
import { Bold, Braces, Code2, Heading2, Image, Italic, Link, List, Sigma, Strikethrough, X } from "lucide-vue-next"

const props = defineProps({
  modelValue: { type: String, default: "" },
  placeholder: { type: String, default: "输入内容，支持 Markdown + LaTeX…" },
  rows: { type: Number, default: 6 },
  fill: { type: Boolean, default: false },
  label: { type: String, default: "" },
})
const emit = defineEmits(["update:modelValue"])
const textarea = ref(null)
const expandImages = ref(false)
const base64Pattern = /!\[[^\]]*\]\(data:image\/[^;]+;base64,[^)]+\)|<img\s+[^>]*src="data:image\/[^;]+;base64,[^"]+"[^>]*>/g
const hasBase64 = computed(() => { base64Pattern.lastIndex = 0; return base64Pattern.test(props.modelValue) })
const editText = computed(() => {
  if (expandImages.value || !hasBase64.value) return props.modelValue
  let index = 0
  base64Pattern.lastIndex = 0
  return props.modelValue.replace(base64Pattern, () => `<<IMG_${++index}>>`)
})

function restoreImages(value) {
  if (expandImages.value || !hasBase64.value) return value
  let result = value
  let index = 0
  base64Pattern.lastIndex = 0
  for (const match of props.modelValue.matchAll(base64Pattern)) result = result.replace(`<<IMG_${++index}>>`, match[0])
  return result
}

function onInput(event) { emit("update:modelValue", restoreImages(event.target.value)) }
async function insertText(before, after = "") {
  const input = textarea.value
  if (!input) return
  const start = input.selectionStart
  const end = input.selectionEnd
  const value = editText.value
  const selected = value.slice(start, end)
  const next = value.slice(0, start) + before + selected + after + value.slice(end)
  emit("update:modelValue", restoreImages(next))
  await nextTick()
  input.focus()
  input.setSelectionRange(start + before.length, start + before.length + selected.length)
}

const tools = [
  { label: "加粗", icon: Bold, action: () => insertText("**", "**") },
  { label: "斜体", icon: Italic, action: () => insertText("*", "*") },
  { label: "删除线", icon: Strikethrough, action: () => insertText("~~", "~~") },
  { label: "二级标题", icon: Heading2, action: () => insertText("## ") },
  { label: "列表", icon: List, action: () => insertText("- ") },
  { label: "链接", icon: Link, action: () => insertText("[", "](url)") },
  { label: "图片", icon: Image, action: () => insertText("![说明](", ")") },
  { label: "代码块", icon: Code2, action: () => insertText("```\n", "\n```") },
  { label: "行内公式", icon: Sigma, action: () => insertText("$", "$") },
  { label: "公式块", icon: Braces, action: () => insertText("$$\n", "\n$$") },
  { label: "公式删除线", icon: X, action: () => insertText("\\xcancel{", "}") },
]
defineExpose({ insertText })
</script>

<template>
  <div class="md-editor" :class="{ 'md-editor--fill': fill }">
    <div class="md-toolbar" role="toolbar" :aria-label="`${label || 'Markdown'} 编辑工具`">
      <button v-for="tool in tools" :key="tool.label" type="button" :aria-label="tool.label" :title="tool.label" @click="tool.action"><component :is="tool.icon" :size="15" /></button>
      <span class="md-toolbar__spacer"></span>
      <button v-if="hasBase64" type="button" class="md-toolbar__text" @click="expandImages = !expandImages">{{ expandImages ? "折叠图片数据" : "展开图片数据" }}</button>
    </div>
    <textarea ref="textarea" :value="editText" class="md-textarea" :class="{ 'md-textarea--fill': fill }" :placeholder="placeholder" :rows="expandImages || !hasBase64 ? rows : 3" spellcheck="false" @input="onInput"></textarea>
  </div>
</template>
