<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue"
import { renderMd } from "../utils/markdown.js"

const props = defineProps({
  content: { type: String, default: "" },
  inline: { type: Boolean, default: false },
})

const html = computed(() => {
  if (!props.content) return ""
  return renderMd(props.content)
})

const contentElement = ref(null)
let renderSequence = 0
let mermaidLoader = null

// loadMermaid 延迟加载较大的图表依赖，让不含流程图的普通笔记保持原有加载速度。
function loadMermaid() {
  mermaidLoader ||= import("mermaid").then(module => module.default)
  return mermaidLoader
}

// renderMermaidDiagrams 将受控的 Mermaid 代码块渲染为 SVG；单个图表报错时保留原始代码。
async function renderMermaidDiagrams() {
  const sequence = ++renderSequence
  await nextTick()
  const diagrams = Array.from(contentElement.value?.querySelectorAll(".mermaid") || [])
  if (!diagrams.length) return

  const mermaid = await loadMermaid()
  if (sequence !== renderSequence) return
  mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme: "base", fontFamily: "inherit" })

  await Promise.all(diagrams.map(async (diagram, index) => {
    const definition = diagram.textContent?.trim() || ""
    if (!definition) return
    try {
      const { svg } = await mermaid.render(`study-tracker-mermaid-${sequence}-${index}`, definition)
      if (sequence !== renderSequence || !diagram.isConnected) return
      diagram.innerHTML = svg
      diagram.classList.add("mermaid--rendered")
    } catch {
      if (sequence !== renderSequence || !diagram.isConnected) return
      diagram.classList.add("mermaid--error")
      diagram.setAttribute("data-mermaid-error", "图表语法错误，已显示原始代码")
    }
  }))
}

watch(html, () => { void renderMermaidDiagrams() }, { flush: "post", immediate: true })
onBeforeUnmount(() => { renderSequence++ })
</script>

<template>
  <div v-if="content" ref="contentElement" class="markdown-body" :class="{ inline }" v-html="html"></div>
  <span v-else style="color:var(--text-muted);font-size:12px">未记录</span>
</template>

<style scoped>
.inline { display: inline; }
.inline :deep(p) { display: inline; margin: 0; }
:deep(.mermaid) { display: grid; place-items: center; min-width: 0; margin: 1.2em 0; padding: 18px; overflow: auto; border: 1px solid var(--line); border-radius: 14px; color: var(--ink); background: color-mix(in srgb, var(--accent-soft) 34%, var(--sheet)); }
:deep(.mermaid svg) { display: block; max-width: 100%; height: auto; }
:deep(.mermaid--error) { display: block; color: var(--ink-secondary); background: var(--sheet-soft); font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: .86em; line-height: 1.6; white-space: pre-wrap; }
</style>
