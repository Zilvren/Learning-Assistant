<script setup>
import { Check } from "lucide-vue-next"
import MarkdownRenderer from "../MarkdownRenderer.vue"
import BaseButton from "../ui/BaseButton.vue"
import BaseDialog from "../ui/BaseDialog.vue"

const props = defineProps({ open: Boolean, item: Object, today: String, busy: Boolean })
defineEmits(["close", "review"])

// showText 协调当前组件的状态和交互。
function showText(value) {
  const text = (value || "").trim()
  return text && text !== "未记录"
}

// title 协调当前组件的状态和交互。
function title() { return props.item?.title || `未命名错题 #${props.item?.id}` }
// plan 协调当前组件的状态和交互。
function plan() {
  const item = props.item || {}
  const count = (item.review_count || 0) + 1
  if (!item.next_review) return `第 ${count} 轮复习`
  if (item.next_review < props.today) return `已逾期 · 第 ${count} 轮`
  if (item.next_review === props.today) return `今日到期 · 第 ${count} 轮`
  return `下次 ${item.next_review} · 第 ${count} 轮`
}
</script>

<template>
  <BaseDialog :open="open" size="lg" @close="$emit('close')">
    <template #header>
      <div class="review-sheet__heading">
        <span class="page-eyebrow">#{{ item?.id }} · {{ item?.subject }}</span>
        <h2>{{ title() }}</h2>
        <p>{{ plan() }}<template v-if="item?.created"> · 录入于 {{ item.created.slice(0, 10) }}</template></p>
      </div>
      <BaseButton variant="success" :busy="busy" @click="$emit('review', item)"><template #icon><Check :size="16" /></template>标记已复习</BaseButton>
    </template>
    <div v-if="item" class="review-sheet">
      <section><h3>题目</h3><MarkdownRenderer :content="item.question" /></section>
      <section class="answer-note answer-note--wrong"><h3>错解批注</h3><MarkdownRenderer :content="showText(item.wrong) ? item.wrong : '未记录'" /></section>
      <section class="answer-note answer-note--correct"><h3>正解订正</h3><MarkdownRenderer :content="showText(item.correct) ? item.correct : '未记录'" /></section>
      <section v-if="showText(item.reason)"><h3>错因归纳</h3><MarkdownRenderer :content="item.reason" /></section>
      <div class="tag-row"><span v-for="tag in item.tags || []" :key="tag" class="tag-pill">{{ tag }}</span><span v-for="tag in item.reason_tags || []" :key="tag" class="tag-pill tag-pill--reason">{{ tag }}</span></div>
    </div>
  </BaseDialog>
</template>
