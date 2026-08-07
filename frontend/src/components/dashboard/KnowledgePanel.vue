<script setup>
import { CheckCircle2, CircleDashed, Quote } from "lucide-vue-next"
import EmptyState from "../ui/EmptyState.vue"

defineProps({ reviewed: Number, total: Number, items: { type: Array, default: () => [] } })
</script>

<template>
  <div class="dashboard-side">
    <section class="folio-section progress-note">
      <header class="folio-section__head"><div><span class="section-number">02</span><h2>复习状态</h2></div></header>
      <div class="progress-note__grid">
        <article><CheckCircle2 :size="18" /><strong>{{ reviewed || 0 }}</strong><span>已复习</span></article>
        <article><CircleDashed :size="18" /><strong>{{ Math.max((total || 0) - (reviewed || 0), 0) }}</strong><span>待掌握</span></article>
      </div>
    </section>

    <section class="folio-section knowledge-notes">
      <header class="folio-section__head"><div><span class="section-number">03</span><h2>今日知识札记</h2></div></header>
      <div v-if="items.length" class="knowledge-notes__list">
        <article v-for="[subject, tip] in items" :key="subject">
          <Quote :size="15" />
          <span>{{ subject }}</span>
          <p>{{ tip }}</p>
        </article>
      </div>
      <EmptyState v-else title="暂无知识札记" description="添加科目后，这里会出现每日知识点。" />
    </section>
  </div>
</template>
