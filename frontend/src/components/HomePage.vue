<script setup>
import { computed, onMounted, ref } from "vue"
import { RouterLink } from "vue-router"
import { BookOpenCheck, CalendarCheck2, CalendarClock, FileText, Flame, Tags } from "lucide-vue-next"
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
const selectedActivityYear = ref(new Date().getFullYear())
const currentYearActivity = ref({ days: [], total: 0, active_days: 0 })
const loading = ref(true)
const hour = new Date().getHours()
const greeting = hour < 12 ? "早上好" : hour < 18 ? "下午好" : "晚上好"
const today = new Date()

function dateKey(value) {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ""
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`
}

function parseDay(value) {
  if (!value) return null
  const [year, month, day] = String(value).slice(0, 10).split("-").map(Number)
  const date = new Date(year, month - 1, day)
  return Number.isFinite(date.getTime()) ? date : null
}

function dayDifference(from, to) {
  return Math.round((to.getTime() - from.getTime()) / 86400000)
}

function streakStats(days) {
  const active = new Set((days || []).filter((day) => Number(day.count) > 0).map((day) => day.date))
  const ordered = [...active].sort()
  let longest = 0
  let run = 0
  let previous = null
  ordered.forEach((key) => {
    const current = parseDay(key)
    run = previous && dayDifference(previous, current) === 1 ? run + 1 : 1
    longest = Math.max(longest, run)
    previous = current
  })

  let cursor = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  if (!active.has(dateKey(cursor))) cursor.setDate(cursor.getDate() - 1)
  let current = 0
  while (active.has(dateKey(cursor))) {
    current += 1
    cursor.setDate(cursor.getDate() - 1)
  }
  return { current, longest }
}

const weekStartedAt = computed(() => {
  const start = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  start.setDate(start.getDate() - ((start.getDay() + 6) % 7))
  return start
})
const addedThisWeek = computed(() => notes.value.filter((item) => {
  const created = new Date(item.created_at)
  return !Number.isNaN(created.getTime()) && created >= weekStartedAt.value
}).length)
const reviewedToday = computed(() => notes.value.filter((item) => dateKey(item.last_review) === dateKey(today)).length)
const reviewGoal = computed(() => due.value.length + reviewedToday.value)
const reviewProgress = computed(() => reviewGoal.value ? Math.round((reviewedToday.value / reviewGoal.value) * 100) : 100)
const overdueNotes = computed(() => due.value.filter((item) => {
  const nextReview = parseDay(item.next_review)
  return nextReview && nextReview < new Date(today.getFullYear(), today.getMonth(), today.getDate())
}))
const earliestOverdueDays = computed(() => overdueNotes.value.reduce((days, item) => {
  const nextReview = parseDay(item.next_review)
  return nextReview ? Math.max(days, dayDifference(nextReview, new Date(today.getFullYear(), today.getMonth(), today.getDate()))) : days
}, 0))
const reviewContext = computed(() => {
  if (earliestOverdueDays.value) return `最早逾期 ${earliestOverdueDays.value} 天`
  if (due.value.length) return "均计划于今天完成"
  return "今天没有到期内容"
})
const learningStreak = computed(() => streakStats(currentYearActivity.value.days))
const recentNotes = computed(() => [...notes.value].sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at)).slice(0, 3))
const tagStats = computed(() => {
  const counts = new Map(tags.value.map((tag) => [tag, 0]))
  notes.value.forEach((item) => (item.tags || []).forEach((tag) => counts.set(tag, (counts.get(tag) || 0) + 1)))
  return [...counts.entries()]
    .map(([name, count]) => ({ name, count }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name, "zh-CN"))
    .slice(0, 8)
})

function relativeTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return "最近更新"
  const minutes = Math.max(0, Math.floor((Date.now() - date.getTime()) / 60000))
  if (minutes < 1) return "刚刚"
  if (minutes < 60) return `${minutes} 分钟前`
  if (minutes < 1440) return `${Math.floor(minutes / 60)} 小时前`
  if (minutes < 10080) return `${Math.floor(minutes / 1440)} 天前`
  return date.toLocaleDateString("zh-CN", { month: "short", day: "numeric" })
}

function dueLabel(item) {
  const nextReview = parseDay(item.next_review)
  if (!nextReview) return "今天"
  const difference = dayDifference(nextReview, new Date(today.getFullYear(), today.getMonth(), today.getDate()))
  return difference > 0 ? `逾期 ${difference} 天` : "今天"
}

// loadActivity 协调当前组件的状态和交互。
async function loadActivity(year = selectedActivityYear.value) {
  try {
    activity.value = await api.getLearningActivity(year)
  } catch (error) { toast.error(error.message || "学习记录加载失败") }
}

// selectActivityYear 协调当前组件的状态和交互。
function selectActivityYear(year) {
  if (year === selectedActivityYear.value) return
  selectedActivityYear.value = year
  loadActivity(year)
}

onMounted(async () => {
  try {
    await settings.load()
    const [all, reviews, tagResult, activityResult] = await Promise.all([
      api.getLibraryItems({ kind: "note", query: " " }),
      api.getLibraryReviews(),
      api.getLibraryTags(),
      api.getLearningActivity(selectedActivityYear.value),
    ])
    notes.value = all.items || []
    due.value = reviews.items || []
    tags.value = tagResult.tags || []
    activity.value = activityResult
    currentYearActivity.value = activityResult
  } catch (error) { toast.error(error.message || "概览加载失败") }
  finally { loading.value = false }
})
</script>

<template>
  <div class="dashboard-view home-overview page-stage">
    <header class="home-overview__hero">
      <div class="home-overview__hero-copy">
        <p class="page-eyebrow">{{ today.toLocaleDateString('zh-CN') }}</p>
        <h1>{{ greeting }}，{{ settings.username.value || '学习者' }}</h1>
        <p>从资料库继续整理，在复习队列里巩固重要笔记。</p>
      </div>
      <div class="home-overview__review-action">
        <div><strong>今日 {{ reviewedToday }} / {{ reviewGoal }}</strong><small>{{ overdueNotes.length ? `${overdueNotes.length} 篇已经逾期` : '按计划完成今日内容' }}</small></div>
        <RouterLink class="lib-btn lib-btn--primary" :to="{name:'review'}"><CalendarCheck2 :size="17"/>开始今日复习</RouterLink>
      </div>
    </header>
    <div v-if="loading" class="dashboard-skeleton"><span v-for="n in 4" :key="n"></span></div>
    <template v-else>
      <section class="home-note-stats">
        <article><span class="home-note-stats__icon"><BookOpenCheck :size="19"/></span><span><b>笔记总数</b><small>本周新增 {{ addedThisWeek }} 篇</small></span><strong>{{ notes.length }}</strong></article>
        <article><span class="home-note-stats__icon"><CalendarClock :size="19"/></span><span><b>待复习笔记</b><small>{{ reviewContext }}</small></span><strong>{{ due.length }}</strong></article>
        <article><span class="home-note-stats__icon"><Flame :size="19"/></span><span><b>连续学习</b><small>本年最长 {{ learningStreak.longest }} 天</small></span><strong>{{ learningStreak.current }}<em>天</em></strong></article>
      </section>
      <section class="home-overview__workspace">
        <div class="home-overview__activity">
          <LearningHeatmap compact :activity="activity" :selected-year="selectedActivityYear" @select-year="selectActivityYear" />
          <section class="home-recent" aria-labelledby="home-recent-title">
            <header><strong id="home-recent-title">最近更新</strong><small>继续上次进度</small></header>
            <RouterLink v-for="item in recentNotes" :key="item.id" :to="{name:'library-item',params:{itemId:item.id}}">
              <span><FileText :size="16"/></span>
              <div><strong>{{ item.name }}</strong><small>{{ relativeTime(item.updated_at) }}<template v-if="item.tags?.length"> · {{ item.tags[0] }}</template></small></div>
            </RouterLink>
            <p v-if="!recentNotes.length" class="home-recent__empty">新建笔记后，最近进度会出现在这里。</p>
          </section>
        </div>

        <aside class="home-today">
          <header class="home-today__head"><CalendarCheck2 :size="18"/><div><h2>今天要做</h2><p>先完成到期内容</p></div></header>
          <div class="home-today__progress">
            <div class="home-today__ring" :style="{'--review-progress': `${reviewProgress}%`}" :aria-label="`今日复习进度 ${reviewedToday}/${reviewGoal}`"><strong>{{ reviewedToday }}/{{ reviewGoal }}</strong></div>
            <div><strong>{{ due.length ? `还差 ${due.length} 篇` : '今日任务已清空' }}</strong><small v-if="due.length">预计约 {{ due.length * 2 }} 分钟</small><small v-if="overdueNotes.length" class="is-overdue">其中 {{ overdueNotes.length }} 篇已逾期</small><small v-else>保持现在的学习节奏</small></div>
          </div>

          <div v-if="due.length" class="home-today__queue" aria-label="复习队列">
            <RouterLink v-for="item in due.slice(0,3)" :key="item.id" :to="{name:'library-item',params:{itemId:item.id}}">
              <i></i><div><strong>{{ item.name }}</strong><small><template v-if="item.tags?.length"># {{ item.tags[0] }} · </template>第 {{ Number(item.review_count || 0) + 1 }} 轮</small></div><small :class="{'is-overdue': dueLabel(item).startsWith('逾期')}">{{ dueLabel(item) }}</small>
            </RouterLink>
          </div>
          <div v-else class="home-today__empty"><CalendarCheck2 :size="24"/><p>今天没有到期笔记，可以整理一条新内容。</p></div>

          <section class="home-index" aria-labelledby="home-index-title">
            <header><Tags :size="16"/><strong id="home-index-title">知识索引</strong><small>{{ tags.length }} 个标签</small></header>
            <div v-if="tagStats.length" class="home-index__tags"><RouterLink v-for="tag in tagStats" :key="tag.name" :to="{name:'library',query:{tag:tag.name}}"><span># {{ tag.name }}</span><small>{{ tag.count }}</small></RouterLink></div>
            <p v-else>给笔记添加标签后会显示在这里。</p>
          </section>
        </aside>
      </section>
    </template>
  </div>
</template>
