<script setup>
import { computed } from "vue"
import { renderMd } from "../utils/markdown.js"

const props = defineProps({
  content: { type: String, default: "" },
  inline: { type: Boolean, default: false },
})

const html = computed(() => {
  if (!props.content) return ""
  return renderMd(props.content)
})
</script>

<template>
  <div v-if="content" class="markdown-body" :class="{ inline }" v-html="html"></div>
  <span v-else style="color:var(--text-muted);font-size:12px">未记录</span>
</template>

<style scoped>
.inline { display: inline; }
.inline :deep(p) { display: inline; margin: 0; }
</style>
