<script setup>
import { computed, nextTick, ref, watch } from "vue"
import { AlignCenter, AlignLeft, AlignRight, Bold, Braces, Code2, Heading2, Image, Italic, Link, List, Sigma, Strikethrough, Trash2, X } from "lucide-vue-next"

const props = defineProps({
  modelValue: { type: String, default: "" },
  placeholder: { type: String, default: "输入内容，支持 Markdown + LaTeX…" },
  rows: { type: Number, default: 6 },
  fill: { type: Boolean, default: false },
  label: { type: String, default: "" },
})
const emit = defineEmits(["update:modelValue", "scroll"])
const textarea = ref(null)
const visualEditor = ref(null)
const imageInput = ref(null)
const imageBusy = ref(false)
const imageError = ref("")
const visualSource = ref(props.modelValue)
const imageNameDrafts = ref({})
const activeImageIndex = ref(null)
const visualTextSelection = ref(null)
const maxEmbeddedImageSize = 20 * 1024 * 1024
const supportedImageTypes = new Set(["image/png", "image/jpeg", "image/jpg", "image/gif", "image/webp"])
const imageMarkdownPattern = /!\[([^\]]*)\]\(\s*(data:image\/(?:png|jpe?g|gif|webp);base64,[A-Za-z0-9+/=\s]+)(?:\s+"([^"]*)")?\s*\)/gi

// imageAltText 协调当前组件的状态和交互。
function imageAltText(filename) {
  const name = String(filename || "图片").replace(/\.[^.]+$/, "")
  const cleaned = name.replace(/[()[\]\\\r\n]/g, " ").replace(/\s+/g, " ").trim()
  return cleaned && !cleaned.toLowerCase().includes("data:image/") ? cleaned : "图片"
}

// imageWidth 协调当前组件的状态和交互。
function imageWidth(value) {
  const width = Number.parseInt(value, 10)
  return Number.isInteger(width) && width >= 120 && width <= 1200 ? width : 400
}

// imageSetting 协调当前组件的状态和交互。
function imageSetting(title, name) {
  return new RegExp(`(?:^|;)${name}=(-?\\d{1,4})(?:;|$)`).exec(String(title || ""))?.[1]
}

// imageSettings 协调当前组件的状态和交互。
function imageSettings(title) {
  return {
    width: imageWidth(imageSetting(title, "width")),
    alignment: /(?:^|;)align=(left|center|right)(?:;|$)/.exec(String(title || ""))?.[1] || "left",
  }
}

// imageMarkdown 协调当前组件的状态和交互。
function imageMarkdown(image) {
  const alignment = ["left", "center", "right"].includes(image.alignment) ? image.alignment : "left"
  const source = String(image.src || "").replace(/\s/g, "")
  return `![${imageAltText(image.alt)}](${source} "width=${imageWidth(image.width)};align=${alignment}")`
}

// parseEmbeddedImages 协调当前组件的状态和交互。
function parseEmbeddedImages(source) {
  const images = []
  imageMarkdownPattern.lastIndex = 0
  let match
  while ((match = imageMarkdownPattern.exec(source))) {
    const settings = imageSettings(match[3])
    images.push({
      index: images.length,
      raw: match[0],
      start: match.index,
      end: match.index + match[0].length,
      alt: imageAltText(match[1]),
      src: match[2],
      ...settings,
    })
  }
  return images
}

const embeddedImages = computed(() => parseEmbeddedImages(visualSource.value))
const hasEmbeddedImages = computed(() => parseEmbeddedImages(props.modelValue).length > 0)
const visualSegments = computed(() => {
  const segments = []
  let offset = 0
  for (const image of embeddedImages.value) {
    if (image.start > offset) segments.push({ type: "text", value: visualSource.value.slice(offset, image.start), start: offset, end: image.start })
    segments.push({ type: "image", image })
    offset = image.end
  }
  if (offset < visualSource.value.length || !segments.length || segments.at(-1)?.type === "image") {
    segments.push({ type: "text", value: visualSource.value.slice(offset), start: offset, end: visualSource.value.length })
  }
  return segments
})

watch(() => props.modelValue, (value) => {
  const next = String(value ?? "")
  // The parent echoes editor updates back through v-model. Only ignore that
  // echo when the visual surface already represents the same source. A plain
  // `last emitted value` check breaks when Vue reuses this component while
  // navigating between notes: returning to an image note then leaves the
  // editor blank while the preview still shows the image.
  if (next === visualSource.value) return
  visualSource.value = next
  activeImageIndex.value = null
  imageNameDrafts.value = {}
  visualTextSelection.value = null
})

// setValue 协调当前组件的状态和交互。
function setValue(value, rebuildVisual = false) {
  const next = String(value ?? "")
  if (rebuildVisual) {
    visualSource.value = next
    visualTextSelection.value = null
  }
  emit("update:modelValue", next)
}

// editorSelection 协调当前组件的状态和交互。
function editorSelection() {
  const input = textarea.value
  if (input) return { mode: "textarea", value: input.value, start: input.selectionStart, end: input.selectionEnd }

  const selection = visualTextSelection.value
  if (selection) {
    return {
      mode: "visual",
      value: visualSource.value,
      start: selection.start + selection.selectionStart,
      end: selection.start + selection.selectionEnd,
      segmentIndex: selection.segmentIndex,
      segmentStart: selection.start,
    }
  }
  return { mode: "visual", value: props.modelValue, start: props.modelValue.length, end: props.modelValue.length }
}

// onInput 协调当前组件的状态和交互。
function onInput(event) { setValue(event.target.value) }

// notifyScroll 将编辑器滚动位置上报给页面，以便收起不需要的顶部界面。
function notifyScroll(event) { emit("scroll", event) }
// rememberVisualTextSelection 协调当前组件的状态和交互。
function rememberVisualTextSelection(segment, target, segmentIndex) {
  visualTextSelection.value = {
    segmentIndex,
    start: segment.start,
    selectionStart: Number.isInteger(target.selectionStart) ? target.selectionStart : String(target.value || "").length,
    selectionEnd: Number.isInteger(target.selectionEnd) ? target.selectionEnd : String(target.value || "").length,
  }
}

// onVisualTextInput 协调当前组件的状态和交互。
function onVisualTextInput(segment, segmentIndex, event) {
  const value = event.target.value
  const next = visualSource.value.slice(0, segment.start) + value + visualSource.value.slice(segment.end)
  visualSource.value = next
  rememberVisualTextSelection({ ...segment, end: segment.start + value.length }, event.target, segmentIndex)
  setValue(next)
}

// visualTextStyle 协调当前组件的状态和交互。
function visualTextStyle(value) {
  const lines = Math.max(1, String(value || "").split(/\r?\n/).length)
  return { height: `${Math.max(38, lines * 24 + 14)}px` }
}

// focusVisualTextSegment 协调当前组件的状态和交互。
async function focusVisualTextSegment(segmentIndex, start, end = start) {
  await nextTick()
  const target = visualEditor.value?.querySelector(`[data-visual-text-segment="${segmentIndex}"]`) || visualEditor.value?.querySelector(".md-text-segment")
  if (!target) return
  target.focus()
  const max = target.value.length
  target.setSelectionRange(Math.max(0, Math.min(start, max)), Math.max(0, Math.min(end, max)))
}

// replaceSelection 协调当前组件的状态和交互。
async function replaceSelection(replacement) {
  const selection = editorSelection()
  const next = selection.value.slice(0, selection.start) + replacement + selection.value.slice(selection.end)
  setValue(next, selection.mode === "visual")
  await nextTick()
  if (selection.mode === "textarea" && textarea.value) {
    textarea.value.focus()
    textarea.value.setSelectionRange(selection.start + replacement.length, selection.start + replacement.length)
  } else {
    const relative = selection.segmentStart == null ? 0 : selection.start - selection.segmentStart + replacement.length
    await focusVisualTextSegment(selection.segmentIndex ?? 0, relative)
  }
}

// insertText 协调当前组件的状态和交互。
async function insertText(before, after = "") {
  const selection = editorSelection()
  const selected = selection.value.slice(selection.start, selection.end)
  const next = selection.value.slice(0, selection.start) + before + selected + after + selection.value.slice(selection.end)
  setValue(next, selection.mode === "visual")
  await nextTick()
  if (selection.mode === "textarea" && textarea.value) {
    textarea.value.focus()
    textarea.value.setSelectionRange(selection.start + before.length, selection.start + before.length + selected.length)
  } else {
    const relativeStart = selection.segmentStart == null ? 0 : selection.start - selection.segmentStart + before.length
    await focusVisualTextSegment(selection.segmentIndex ?? 0, relativeStart, relativeStart + selected.length)
  }
}

// openImagePicker 协调当前组件的状态和交互。
function openImagePicker() {
  imageError.value = ""
  imageInput.value?.click()
}

// readAsDataUrl 协调当前组件的状态和交互。
function readAsDataUrl(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onerror = () => reject(new Error("文件读取失败"))
    reader.onload = () => resolve(reader.result)
    reader.readAsDataURL(file)
  })
}

// imageFilename 为剪贴板截图补充一个可读的 Markdown 图片名称。
function imageFilename(file) {
  const name = String(file?.name || "").trim()
  if (name) return name
  const extension = { "image/png": "png", "image/jpeg": "jpg", "image/jpg": "jpg", "image/gif": "gif", "image/webp": "webp" }[file?.type] || "png"
  return `剪贴板图片.${extension}`
}

// embedImage 校验并将本地文件或剪贴板图片嵌入为可随笔记保存的 Markdown 图片。
async function embedImage(file) {
  if (!file) return
  imageError.value = ""
  if (!supportedImageTypes.has(file.type)) {
    imageError.value = "仅支持 PNG、JPG、GIF 或 WebP 图片"
    return
  }
  if (file.size > maxEmbeddedImageSize) {
    imageError.value = "图片请控制在 20 MB 以内"
    return
  }

  imageBusy.value = true
  try {
    const dataUrl = await readAsDataUrl(file)
    if (typeof dataUrl !== "string" || !/^data:image\/(png|jpeg|gif|webp);base64,[A-Za-z0-9+/=]+$/i.test(dataUrl)) {
      throw new Error("图片编码无效")
    }
    await replaceSelection(imageMarkdown({ alt: imageFilename(file), src: dataUrl, width: 400 }))
  } catch (error) {
    imageError.value = error instanceof Error ? error.message : "图片读取失败"
  } finally {
    imageBusy.value = false
  }
}

// onImageSelected 协调当前组件的状态和交互。
async function onImageSelected(event) {
  const file = event.target.files?.[0]
  event.target.value = ""
  await embedImage(file)
}

// onPaste 在光标所在处粘贴截图或复制的图片；普通文字粘贴仍保持浏览器默认行为。
function onPaste(event) {
  if (event.target instanceof HTMLInputElement && event.target.type !== "file") return
  const item = Array.from(event.clipboardData?.items || []).find(candidate => candidate.kind === "file" && supportedImageTypes.has(candidate.type))
  const file = item?.getAsFile()
  if (!file) return
  event.preventDefault()
  void embedImage(file)
}

// updateEmbeddedImage 协调当前组件的状态和交互。
function updateEmbeddedImage(index, patch) {
  const currentImages = parseEmbeddedImages(visualSource.value)
  const current = currentImages[index]
  if (!current) return
  const replacement = imageMarkdown({ ...current, ...patch })
  const next = visualSource.value.slice(0, current.start) + replacement + visualSource.value.slice(current.end)
  setValue(next, true)
}

// displayedImageName 协调当前组件的状态和交互。
function displayedImageName(image) {
  return imageNameDrafts.value[image.index] ?? image.alt
}

// draftImageName 协调当前组件的状态和交互。
function draftImageName(index, value) {
  imageNameDrafts.value = { ...imageNameDrafts.value, [index]: value }
}

// saveImageName 协调当前组件的状态和交互。
function saveImageName(index) {
  if (!(index in imageNameDrafts.value)) return
  const alt = imageNameDrafts.value[index]
  const { [index]: _, ...remaining } = imageNameDrafts.value
  imageNameDrafts.value = remaining
  updateEmbeddedImage(index, { alt })
}

// updateImageWidth 协调当前组件的状态和交互。
function updateImageWidth(index, value) {
  updateEmbeddedImage(index, { width: imageWidth(value) })
}

// updateImageAlignment 协调当前组件的状态和交互。
function updateImageAlignment(index, alignment) {
  updateEmbeddedImage(index, { alignment })
}

// removeEmbeddedImage 协调当前组件的状态和交互。
function removeEmbeddedImage(index) {
  const currentImages = parseEmbeddedImages(visualSource.value)
  const current = currentImages[index]
  if (!current) return
  activeImageIndex.value = null
  const next = visualSource.value.slice(0, current.start) + visualSource.value.slice(current.end)
  setValue(next, true)
}

// toggleImageControls 协调当前组件的状态和交互。
function toggleImageControls(index) {
  activeImageIndex.value = activeImageIndex.value === index ? null : index
}

// applyAlignment 协调当前组件的状态和交互。
async function applyAlignment(alignment) {
  const selection = editorSelection()
  const selected = selection.value.slice(selection.start, selection.end)
  if (!selected.trim() && Number.isInteger(activeImageIndex.value)) {
    updateImageAlignment(activeImageIndex.value, alignment)
    activeImageIndex.value = null
    return
  }
  if (!selected.trim()) return

  const blockStart = selection.value.lastIndexOf("\n", Math.max(0, selection.start - 1)) + 1
  const nextBreak = selection.value.indexOf("\n", selection.end)
  const blockEnd = nextBreak === -1 ? selection.value.length : nextBreak
  const block = selection.value.slice(blockStart, blockEnd)
  const wrapped = `[[align:${alignment}]]\n${block}\n[[/align]]`
  const next = selection.value.slice(0, blockStart) + wrapped + selection.value.slice(blockEnd)
  setValue(next, selection.mode === "visual")
  await nextTick()
  if (selection.mode === "textarea" && textarea.value) textarea.value.focus()
  else await focusVisualTextSegment(selection.segmentIndex ?? 0, 0)
}

const tools = [
  { label: "加粗", icon: Bold, action: () => insertText("**", "**") },
  { label: "斜体", icon: Italic, action: () => insertText("*", "*") },
  { label: "删除线", icon: Strikethrough, action: () => insertText("~~", "~~") },
  { label: "二级标题", icon: Heading2, action: () => insertText("## ") },
  { label: "列表", icon: List, action: () => insertText("- ") },
  { label: "链接", icon: Link, action: () => insertText("[", "](url)") },
  { label: "插入图片", icon: Image, action: openImagePicker },
  { label: "左对齐", icon: AlignLeft, action: () => applyAlignment("left") },
  { label: "居中", icon: AlignCenter, action: () => applyAlignment("center") },
  { label: "右对齐", icon: AlignRight, action: () => applyAlignment("right") },
  { label: "代码块", icon: Code2, action: () => insertText("```\n", "\n```") },
  { label: "行内公式", icon: Sigma, action: () => insertText("$", "$") },
  { label: "公式块", icon: Braces, action: () => insertText("$$\n", "\n$$") },
  { label: "公式删除线", icon: X, action: () => insertText("\\xcancel{", "}") },
]
defineExpose({ insertText })
</script>

<template>
  <div class="md-editor" :class="{ 'md-editor--fill': fill }" @paste="onPaste">
    <div class="md-toolbar" role="toolbar" :aria-label="`${label || 'Markdown'} 编辑工具`">
      <button v-for="tool in tools" :key="tool.label" type="button" :aria-label="tool.label" :title="tool.label" :disabled="tool.label === '插入图片' && imageBusy" @mousedown.prevent @click="tool.action"><component :is="tool.icon" :size="15" /></button>
      <span class="md-toolbar__spacer"></span>
      <span v-if="imageError" class="md-toolbar__status" role="status">{{ imageError }}</span>
    </div>
    <input ref="imageInput" class="md-image-input" type="file" accept="image/png,image/jpeg,image/gif,image/webp" tabindex="-1" @change="onImageSelected" />
    <div v-if="hasEmbeddedImages" ref="visualEditor" class="md-textarea md-textarea--visual" :class="{ 'md-textarea--fill': fill }" role="group" aria-label="含图片的笔记正文" @scroll="notifyScroll">
      <template v-for="(segment, segmentIndex) in visualSegments" :key="`${segment.type}-${segmentIndex}`">
        <textarea v-if="segment.type === 'text'" class="md-text-segment" :data-visual-text-segment="segmentIndex" :value="segment.value" :style="visualTextStyle(segment.value)" :placeholder="segmentIndex === 0 ? placeholder : '继续输入…'" spellcheck="false" @focus="rememberVisualTextSelection(segment, $event.target, segmentIndex)" @click="rememberVisualTextSelection(segment, $event.target, segmentIndex)" @select="rememberVisualTextSelection(segment, $event.target, segmentIndex)" @input="onVisualTextInput(segment, segmentIndex, $event)" />
        <div v-else class="md-image-row" :class="'md-image-row--align-' + segment.image.alignment">
          <span class="md-image-control" :class="{ 'is-active': activeImageIndex === segment.image.index }" :data-image-index="segment.image.index">
            <button class="md-image-control__button" type="button" :aria-label="'编辑图片 ' + (segment.image.index + 1)" title="点击编辑名称和尺寸；选中后可用上方对齐按钮。按 Delete 可删除" @click.stop="toggleImageControls(segment.image.index)" @keydown.delete.stop.prevent="removeEmbeddedImage(segment.image.index)" @keydown.backspace.stop.prevent="removeEmbeddedImage(segment.image.index)"><Image :size="17" /><span>{{ segment.image.index + 1 }}</span></button>
            <span v-if="activeImageIndex === segment.image.index" class="md-image-control__details">
              <input class="md-image-control__name" :value="displayedImageName(segment.image)" aria-label="图片名称" @input.stop="draftImageName(segment.image.index, $event.target.value)" @blur.stop="saveImageName(segment.image.index)" @keydown.enter.prevent.stop="$event.target.blur()" />
              <span class="md-image-control__sizes" aria-label="图片显示宽度">
                <button v-for="size in [{ label: '小', width: 240 }, { label: '中', width: 400 }, { label: '大', width: 640 }]" :key="size.width" type="button" :class="{ active: segment.image.width === size.width }" :aria-label="'图片 ' + (segment.image.index + 1) + '：' + size.label" @mousedown.prevent @click.stop="updateImageWidth(segment.image.index, size.width)">{{ size.label }}</button>
                <label>宽 <input type="number" min="120" max="1200" step="10" :value="segment.image.width" aria-label="自定义图片宽度" @input.stop @change.stop="updateImageWidth(segment.image.index, $event.target.value)" /><span>px</span></label>
              </span>
              <button class="md-image-control__delete" type="button" :aria-label="'删除图片 ' + (segment.image.index + 1)" title="删除图片" @mousedown.prevent @click.stop="removeEmbeddedImage(segment.image.index)"><Trash2 :size="13" /></button>
            </span>
          </span>
        </div>
      </template>
    </div>
    <textarea v-else ref="textarea" :value="modelValue" class="md-textarea" :class="{ 'md-textarea--fill': fill }" :placeholder="placeholder" :rows="rows" spellcheck="false" @input="onInput" @scroll="notifyScroll"></textarea>
  </div>
</template>
