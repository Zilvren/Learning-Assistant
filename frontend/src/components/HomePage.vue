<script setup>
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { BookOpenCheck, CalendarCheck2 } from "lucide-vue-next"
import { api } from "../api/index.js"
import LearningHeatmap from "./dashboard/LearningHeatmap.vue"
import { useSettings } from "../store/settings.js"
import { useToast } from "../store/toast.js"

const settings = useSettings()
const toast = useToast()
const notes = ref([])
const due = ref([])
const tags = ref([])
const activity = ref({ days: [], total: 0, active_days: 0 })
const loading = ref(true)
const hour = new Date().getHours()
const greeting = hour < 12 ? "早上好" : hour < 18 ? "下午好" : "晚上好"
const reviewed = computed(() => notes.value.filter((item) => item.review_enabled && item.review_count > 0).length)
const commonTags = computed(() => tags.value.slice(0, 8))

onMounted(async () => {
  try {
    await settings.load()
    const [all, reviews, tagResult, activityResult] = await Promise.all([
      api.getLibraryItems({ kind: "note", query: " " }),
      api.getLibraryReviews(),
      api.getLibraryTags(),
      api.getLearningActivity(),
    ])
    notes.value = all.items || []
    due.value = reviews.items || []
    tags.value = tagResult.tags || []
    activity.value = activityResult
  } catch (error) { toast.error(error.message || "概览加载失败") }
  finally { loading.value = false }
})
</script>

<template>
  <div class="dashboard-view page-stage">
    <header class="library-head"><div class="library-title-row"><div><p class="page-eyebrow">{{ new Date().toLocaleDateString('zh-CN') }}</p><h1>{{ greeting }}，{{ settings.username.value || '学习者' }}</h1><p>从资料库继续整理，在复习队列里巩固重要笔记。</p></div><RouterLink class="lib-btn lib-btn--primary" :to="{name:'review'}"><CalendarCheck2 :size="17"/>开始今日复习</RouterLink></div></header>
    <div v-if="loading" class="dashboard-skeleton"><span v-for="n in 4" :key="n"></span></div>
    <template v-else>
      <section class="home-note-stats">
        <article><BookOpenCheck :size="21"/><span>笔记总数</span><strong>{{ notes.length }}</strong></article>
        <article><CalendarCheck2 :size="21"/><span>待复习笔记</span><strong>{{ due.length }}</strong></article>
        <article><BookOpenCheck :size="21"/><span>已复习笔记</span><strong>{{ reviewed }}</strong></article>
      </section>
      <section class="home-activity-layout">
        <LearningHeatmap :activity="activity" />
        <aside class="home-tag-panel">
          <header><div><span class="page-eyebrow"><Tags :size="13" /> 常用标签</span><h2>知识索引</h2><p>按标签快速回到相关笔记</p></div></header>
          <div v-if="commonTags.length" class="library-tags home-tags"><RouterLink v-for="tag in commonTags" :key="tag" :to="{name:'library',query:{tag}}"># {{ tag }}</RouterLink></div>
          <div v-else class="home-tag-panel__empty"><Tags :size="23"/><p>给笔记添加标签后会显示在这里。</p></div>
        </aside>
      </section>
      <section class="review-layout home-review-layout home-review-layout--wide">
        <div class="review-reader"><header><div><h2>今日待复习</h2><p>按计划到期的可复习笔记</p></div><RouterLink class="lib-btn" :to="{name:'review'}">查看全部</RouterLink></header><div v-if="due.length" class="home-due-list"><RouterLink v-for="item in due.slice(0,6)" :key="item.id" :to="{name:'library-item',params:{itemId:item.id}}"><strong>{{ item.name }}</strong><small>{{ item.next_review || '今天' }}</small></RouterLink></div><div v-else class="library-empty"><CalendarCheck2 :size="30"/><p>今天没有到期笔记</p></div></div>
      </section>
    </template>
  </div>
</template>
