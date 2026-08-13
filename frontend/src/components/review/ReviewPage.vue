<script setup>
import { onMounted, ref } from "vue"
import { BrainCircuit, CheckCircle2, CircleHelp, RotateCcw } from "lucide-vue-next"
import { api } from "../../api/index.js"
import { useToast } from "../../store/toast.js"
import MarkdownRenderer from "../MarkdownRenderer.vue"

const toast = useToast()
const items = ref([])
const active = ref(null)
const content = ref("")
const errorDetail = ref(null)
const loading = ref(true)
const busy = ref(false)
const contentLoading = ref(false)
let openSequence = 0

async function load() {
  loading.value = true
  try {
    const result = await api.getReviewInbox()
    items.value = result.items || []
    if (active.value && !items.value.some((item) => item.source_type === active.value.source_type && item.id === active.value.id)) active.value = null
  } catch (error) { toast.error(error.message || "复习列表加载失败") }
  finally { loading.value = false }
}

async function open(item) {
  const sequence = ++openSequence
  active.value = item
  content.value = ""
  errorDetail.value = null
  contentLoading.value = true
  try {
    if (item.source_type === "library") {
      const result = await api.getLibraryContent(item.id)
      if (sequence === openSequence && active.value?.id === item.id) content.value = result.content || ""
    } else {
      const result = await api.getError(item.id)
      if (sequence === openSequence && active.value?.id === item.id) errorDetail.value = result
    }
  } catch (error) {
    if (sequence === openSequence) toast.error(error.message || "复习内容读取失败")
  } finally {
    if (sequence === openSequence) contentLoading.value = false
  }
}

async function complete(rating) {
  if (!active.value || busy.value || contentLoading.value) return
  const target = active.value
  busy.value = true
  try {
    if (target.source_type === "library") await api.reviewLibraryNote(target.id, rating)
    else await api.reviewError(target.id, rating)
    toast.success(rating === "again" ? "已安排今天再次复习" : "复习计划已更新")
    if (active.value?.id === target.id && active.value?.source_type === target.source_type) {
      openSequence += 1
      active.value = null
      content.value = ""
      errorDetail.value = null
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
    <header class="library-head"><div class="library-title-row"><div><h1>统一复习</h1><p>笔记与错题按逾期程度排在同一个队列中。</p></div><span class="review-count">{{ items.length }} 项待复习</span></div></header>
    <div v-if="loading" class="library-empty"><RotateCcw :size="30"/><p>正在整理复习列表…</p></div>
    <section v-else-if="!items.length" class="library-empty"><CheckCircle2 :size="38"/><h2>今天已经完成</h2><p>暂无到期的笔记或错题。</p></section>
    <section v-else class="review-layout">
      <aside class="review-list"><button v-for="item in items" :key="`${item.source_type}-${item.id}`" :class="{active:active?.id===item.id&&active?.source_type===item.source_type}" @click="open(item)"><strong><BrainCircuit v-if="item.source_type==='error'" :size="14"/>{{ item.title }}</strong><small>{{ item.source_type==='error' ? `错题 · ${item.subject || '未分类'}` : '复习笔记' }}<template v-if="item.overdue_days"> · 逾期 {{ item.overdue_days }} 天</template></small></button></aside>
      <article class="review-reader">
        <template v-if="active"><header><div><h2>{{ active.title }}</h2><div class="library-tags"><span v-for="tag in active.tags" :key="tag"># {{ tag }}</span></div></div></header>
          <div v-if="contentLoading" class="library-empty"><RotateCcw :size="24"/><p>正在读取复习内容…</p></div>
          <template v-else-if="active.source_type==='error' && errorDetail"><section class="review-problem"><h3>题目</h3><MarkdownRenderer :content="errorDetail.question"/><h3>你的错解</h3><MarkdownRenderer :content="errorDetail.wrong"/><h3>正确思路</h3><MarkdownRenderer :content="errorDetail.correct"/><h3>错因复盘</h3><MarkdownRenderer :content="errorDetail.reason"/></section></template>
          <MarkdownRenderer v-else-if="active.source_type==='library'" :content="content"/>
          <div v-else class="library-empty"><CircleHelp :size="24"/><p>内容暂时不可用。</p></div>
          <footer class="review-rating" aria-label="选择掌握程度"><span>这次掌握得怎样？</span><div><button class="lib-btn" :disabled="busy" @click="complete('again')">忘了</button><button class="lib-btn" :disabled="busy" @click="complete('hard')">有点难</button><button class="lib-btn lib-btn--primary" :disabled="busy" @click="complete('good')">掌握</button><button class="lib-btn" :disabled="busy" @click="complete('easy')">很轻松</button></div></footer>
        </template>
        <div v-else class="library-empty"><h2>选择一项开始复习</h2><p>完成后选择掌握程度，系统会安排下次复习时间。</p></div>
      </article>
    </section>
  </div>
</template>
