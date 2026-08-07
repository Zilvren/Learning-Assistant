<script setup>
import { CalendarDays, Flame, LibraryBig, Target } from "lucide-vue-next"

defineProps({
  date: String,
  greeting: String,
  username: String,
  advice: String,
  total: Number,
  due: Number,
  overdue: Number,
  topSubject: String,
})

const stats = [
  { key: "total", label: "错题总数", icon: LibraryBig },
  { key: "due", label: "今日到期", icon: CalendarDays },
  { key: "overdue", label: "逾期复习", icon: Flame },
  { key: "topSubject", label: "当前薄弱", icon: Target },
]
</script>

<template>
  <section class="dashboard-hero paper-panel">
    <div class="dashboard-hero__intro">
      <span class="page-eyebrow">{{ date }} · 今日学习</span>
      <h1>{{ greeting }}<template v-if="username">，{{ username }}</template></h1>
      <p>{{ advice || "先完成今日复习，再补充新的错题批注。" }}</p>
    </div>
    <div class="dashboard-stats">
      <article v-for="item in stats" :key="item.key">
        <component :is="item.icon" :size="17" />
        <strong>{{ $props[item.key] ?? 0 }}</strong>
        <span>{{ item.label }}</span>
      </article>
    </div>
  </section>
</template>
