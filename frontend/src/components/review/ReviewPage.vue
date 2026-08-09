<script setup>
import { onMounted, ref } from "vue"
import { CheckCircle2, RotateCcw } from "lucide-vue-next"
import { api } from "../../api/index.js"
import { useToast } from "../../store/toast.js"
import MarkdownRenderer from "../MarkdownRenderer.vue"

const toast = useToast()
const items = ref([])
const active = ref(null)
const content = ref("")
const loading = ref(true)
const busy = ref(false)
const contentLoading = ref(false)
let openSequence = 0

// load 协调当前组件的状态和交互。
async function load() {
  loading.value = true
  try {
    const result = await api.getLibraryReviews()
    items.value = result.items || []
    if (active.value && !items.value.some((item) => item.id === active.value.id)) active.value = null
  } catch (error) { toast.error(error.message || "复习列表加载失败") }
  finally { loading.value = false }
}

// open 协调当前组件的状态和交互。
async function open(item) {
  const sequence = ++openSequence
  active.value = item
  content.value = ""
  contentLoading.value = true
  try {
    const result = await api.getLibraryContent(item.id)
    if (sequence === openSequence && active.value?.id === item.id) content.value = result.content || ""
  } catch (error) {
    if (sequence === openSequence) toast.error(error.message || "笔记读取失败")
  } finally {
    if (sequence === openSequence) contentLoading.value = false
  }
}

// complete 协调当前组件的状态和交互。
async function complete() {
  if (!active.value || busy.value || contentLoading.value) return
  const targetId = active.value.id
  busy.value = true
  try {
    await api.reviewLibraryNote(targetId)
    toast.success("本次复习已完成")
    if (active.value?.id === targetId) {
      openSequence += 1
      active.value = null
      content.value = ""
      contentLoading.value = false
    }
    await load()
  } catch (error) { toast.error(error.message || "复习保存失败") }
  finally { busy.value = false }
}

onMounted(load)
</script>

<template>
  <div class="review-page page-stage">
    <header class="library-head"><div class="library-title-row"><div><h1>今日复习</h1><p>所有到期的复习笔记都会汇总在这里。</p></div><span class="review-count">{{ items.length }} 篇待复习</span></div></header>
    <div v-if="loading" class="library-empty"><RotateCcw :size="30"/><p>正在整理复习列表…</p></div>
    <section v-else-if="!items.length" class="library-empty"><CheckCircle2 :size="38"/><h2>今天已经完成</h2><p>暂无到期的复习笔记。</p></section>
    <section v-else class="review-layout">
      <aside class="review-list"><button v-for="item in items" :key="item.id" :class="{active:active?.id===item.id}" @click="open(item)"><strong>{{ item.name }}</strong><small>{{ item.tags?.join(' · ') || '无标签' }}</small></button></aside>
      <article class="review-reader">
        <template v-if="active"><header><div><h2>{{ active.name }}</h2><div class="library-tags"><span v-for="tag in active.tags" :key="tag"># {{ tag }}</span></div></div><button class="lib-btn lib-btn--primary" :disabled="busy || contentLoading" @click="complete">{{ busy?'保存中…':contentLoading?'读取中…':'完成复习' }}</button></header><div v-if="contentLoading" class="library-empty"><RotateCcw :size="24"/><p>正在读取笔记正文…</p></div><MarkdownRenderer v-else :content="content"/></template>
        <div v-else class="library-empty"><h2>选择一篇笔记开始复习</h2><p>正文会完整显示在这里。</p></div>
      </article>
    </section>
  </div>
</template>
