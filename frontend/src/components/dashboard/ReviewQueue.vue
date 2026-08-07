<script setup>
import { ArrowRight, Check, Clock3 } from "lucide-vue-next"
import BaseButton from "../ui/BaseButton.vue"
import EmptyState from "../ui/EmptyState.vue"

defineProps({ items: { type: Array, default: () => [] }, today: String, busyId: Number })
defineEmits(["select", "review"])

function title(item) { return item.title || `未命名错题 #${item.id}` }
function plan(item, today) {
  const count = (item.review_count || 0) + 1
  if (!item.next_review) return `第 ${count} 轮复习`
  if (item.next_review < today) return `已逾期 · 第 ${count} 轮`
  if (item.next_review === today) return `今日到期 · 第 ${count} 轮`
  return `下次 ${item.next_review} · 第 ${count} 轮`
}
</script>

<template>
  <section class="folio-section review-queue">
    <header class="folio-section__head">
      <div><span class="section-number">01</span><h2>今日优先复习</h2></div>
      <p><Clock3 :size="14" /> {{ items.length }} 道按复习曲线到期</p>
    </header>
    <div v-if="items.length" class="review-queue__list">
      <article
        v-for="item in items"
        :key="item.id"
        class="review-entry"
        :class="{ 'review-entry--overdue': item.next_review && item.next_review < today }"
        role="button"
        tabindex="0"
        @click="$emit('select', item)"
        @keydown.enter.prevent="$emit('select', item)"
        @keydown.space.prevent="$emit('select', item)"
      >
        <span class="review-entry__id">#{{ item.id }}</span>
        <div class="review-entry__copy">
          <span class="subject-caption">{{ item.subject }}</span>
          <h3>{{ title(item) }}</h3>
          <p>{{ plan(item, today) }}</p>
        </div>
        <div class="review-entry__actions">
          <BaseButton
            size="sm"
            variant="success"
            :busy="busyId === item.id"
            @click.stop="$emit('review', item)"
          ><template #icon><Check :size="15" /></template>完成</BaseButton>
          <ArrowRight class="review-entry__arrow" :size="17" />
        </div>
      </article>
    </div>
    <EmptyState v-else title="今日复习已清空" description="做得漂亮。你可以去错题库继续整理，或稍作休息。" />
  </section>
</template>
