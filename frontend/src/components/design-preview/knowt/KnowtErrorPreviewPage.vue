<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import {
  AlertCircle,
  BookOpen,
  Brain,
  CalendarClock,
  CheckCircle2,
  RefreshCw,
  Shuffle,
  Sparkles,
} from "lucide-vue-next"
import { useRoute, useRouter } from "vue-router"
import { isDue, reviewLabel } from "../../../composables/useErrorLibrary.js"
import { useErrorPreview } from "../../../composables/useErrorPreview.js"
import KnowtErrorDetail from "./KnowtErrorDetail.vue"
import KnowtErrorIndex from "./KnowtErrorIndex.vue"
import KnowtFilterBar from "./KnowtFilterBar.vue"

const route = useRoute()
const router = useRouter()
const preview = useErrorPreview()

const subject = ref("全部")
const mode = ref("全部")
const keyword = ref("")
const appliedKeyword = ref("")
const studyView = ref("all")
const today = new Date().toISOString().slice(0, 10)

let keywordTimer = 0
let filtersReady = false

function strictPositiveId(value) {
  if (Array.isArray(value)) value = value[0]
  if (typeof value !== "string" && typeof value !== "number") return null
  const raw = String(value)
  if (!/^[1-9]\d*$/.test(raw)) return null
  const id = Number(raw)
  return Number.isSafeInteger(id) ? id : null
}

const hasRequestedId = computed(() => route.params.id !== undefined && route.params.id !== "")
const requestedId = computed(() => route.params.id)
const selectedId = computed(() => strictPositiveId(route.params.id))
const selectedError = computed(() => {
  if (selectedId.value === null) return null
  return preview.errors.value.find((item) => item.id === selectedId.value) || null
})
const detailNotFound = computed(() => (
  hasRequestedId.value
  && preview.loaded.value
  && !preview.loading.value
  && !preview.error.value
  && !selectedError.value
))
const dueCount = computed(() => preview.errors.value.filter((item) => isDue(item, today)).length)
const scheduledCount = computed(() => Math.max(0, preview.errors.value.length - dueCount.value))
const readyPercent = computed(() => {
  if (!preview.errors.value.length) return 0
  return Math.round((scheduledCount.value / preview.errors.value.length) * 100)
})
const visibleErrors = computed(() => {
  if (studyView.value === "due") return preview.errors.value.filter((item) => isDue(item, today))
  if (studyView.value === "scheduled") return preview.errors.value.filter((item) => !isDue(item, today))
  return preview.errors.value
})
const errorMessage = computed(() => {
  const value = preview.error.value
  return typeof value === "string" ? value : value?.message || "读取错题时遇到问题，请稍后重试。"
})
const studyModes = computed(() => [
  { id: "all", label: "浏览全部", meta: `${preview.errors.value.length} 条记录`, icon: BookOpen, color: "#26a69a", wash: "#e8f8f5" },
  { id: "due", label: "今日复习", meta: `${dueCount.value} 条待处理`, icon: Brain, color: "#e86f88", wash: "#fff0f3" },
  { id: "scheduled", label: "已安排", meta: `${scheduledCount.value} 条稳定`, icon: CheckCircle2, color: "#5b71cf", wash: "#eef1ff" },
  { id: "random", label: "随机抽查", meta: "打开一条记录", icon: Shuffle, color: "#d28b30", wash: "#fff6e8" },
])

function currentFilters() {
  return { subject: subject.value, mode: mode.value, keyword: appliedKeyword.value }
}

async function clearSelectedRoute() {
  if (!hasRequestedId.value) return
  await router.replace({ name: "knowt-preview-errors" })
}

async function applyFilters() {
  if (filtersReady) await clearSelectedRoute()
  filtersReady = true
  await preview.refresh(currentFilters())
}

watch([subject, mode, appliedKeyword], applyFilters, { immediate: true })

watch(keyword, (value) => {
  window.clearTimeout(keywordTimer)
  keywordTimer = window.setTimeout(() => {
    appliedKeyword.value = value.trim()
  }, 250)
})

function updateKeyword(value) {
  keyword.value = value
}

function resetFilters() {
  window.clearTimeout(keywordTimer)
  clearSelectedRoute()
  keyword.value = ""
  appliedKeyword.value = ""
  subject.value = "全部"
  mode.value = "全部"
  studyView.value = "all"
}

function filterByTag(nextMode, value) {
  window.clearTimeout(keywordTimer)
  studyView.value = "all"
  mode.value = nextMode
  keyword.value = value
  appliedKeyword.value = value
}

async function selectError(item) {
  await router.push({ name: "knowt-preview-errors", params: { id: item.id } })
}

async function backToIndex() {
  await router.push({ name: "knowt-preview-errors" })
}

async function chooseStudyView(nextView) {
  if (nextView === "random") {
    if (!visibleErrors.value.length) return
    const index = (new Date().getDate() - 1) % visibleErrors.value.length
    await selectError(visibleErrors.value[index])
    return
  }
  studyView.value = nextView
  await clearSelectedRoute()
}

onMounted(() => preview.loadSubjects())

onBeforeUnmount(() => {
  window.clearTimeout(keywordTimer)
  preview.dispose()
})
</script>

<template>
  <div
    class="kt-error-page"
    :class="{ 'kt-error-page--selected': hasRequestedId }"
    data-testid="knowt-error-workbench"
  >
    <nav class="kt-breadcrumb" aria-label="Knowt 样板当前位置">
      <span>学习主页</span><i aria-hidden="true">›</i><strong>错题集</strong>
    </nav>

    <section class="kt-set-hero" aria-labelledby="kt-set-title">
      <div class="kt-set-hero__copy">
        <span class="kt-set-kicker"><Sparkles :size="14" aria-hidden="true" />Knowt 方向视觉样板</span>
        <h1 id="kt-set-title">我的错题集</h1>
        <p>把错题当作一套可以浏览、筛选和回看的学习资料，而不是沉重的管理后台。</p>
        <div class="kt-set-meta" aria-label="错题集概况">
          <span><strong>{{ preview.errors.value.length }}</strong> 条记录</span>
          <i aria-hidden="true"></i>
          <span><strong>{{ dueCount }}</strong> 条待复习</span>
          <i aria-hidden="true"></i>
          <span>{{ preview.subjects.value.length || "—" }} 个科目</span>
        </div>
      </div>

      <div class="kt-progress" :style="{ '--kt-progress': `${readyPercent * 3.6}deg` }" aria-label="当前无需复习的记录比例">
        <div><strong>{{ readyPercent }}%</strong><span>当前稳定</span></div>
      </div>
    </section>

    <section class="kt-study-modes" aria-label="浏览方式">
      <button
        v-for="item in studyModes"
        :key="item.id"
        type="button"
        class="kt-mode-card"
        :class="{ 'is-active': studyView === item.id }"
        :style="{ '--mode-color': item.color, '--mode-wash': item.wash }"
        :aria-pressed="item.id === 'random' ? undefined : studyView === item.id"
        :disabled="item.id === 'random' && !visibleErrors.length"
        @click="chooseStudyView(item.id)"
      >
        <span class="kt-mode-icon" aria-hidden="true"><component :is="item.icon" :size="19" /></span>
        <span class="kt-mode-copy"><strong>{{ item.label }}</strong><small>{{ item.meta }}</small></span>
        <span class="kt-mode-arrow" aria-hidden="true">→</span>
      </button>
    </section>

    <KnowtFilterBar
      :subjects="preview.subjects.value"
      :subject="subject"
      :mode="mode"
      :keyword="keyword"
      :loading="preview.loading.value"
      :count="visibleErrors.length"
      @update:subject="subject = $event"
      @update:mode="mode = $event"
      @update:keyword="updateKeyword"
      @reset="resetFilters"
    />

    <div v-if="preview.error.value" class="kt-load-error" role="alert" data-testid="knowt-load-error">
      <AlertCircle :size="18" aria-hidden="true" />
      <div><strong>暂时无法读取错题集</strong><span>{{ errorMessage }}</span></div>
      <button type="button" @click="preview.refresh(currentFilters())">
        <RefreshCw :size="15" aria-hidden="true" />重试
      </button>
    </div>

    <div v-if="preview.subjectsError.value" class="kt-subject-note" role="status">
      科目列表暂时不可用，仍可浏览全部错题。
    </div>

    <section
      v-if="!hasRequestedId"
      class="kt-library-pane"
      aria-labelledby="kt-library-title"
      data-testid="knowt-library-pane"
    >
      <header class="kt-section-head">
        <div><h2 id="kt-library-title">错题条目</h2><p>像浏览学习卡组一样回看每一条错误记录。</p></div>
        <span>{{ visibleErrors.length }} 项</span>
      </header>
      <KnowtErrorIndex
        :items="visibleErrors"
        :selected-id="selectedId"
        :loading="preview.loading.value && !preview.loaded.value"
        :today="today"
        @select="selectError"
        @tag="filterByTag"
      />
    </section>

    <section v-else class="kt-detail-layout" aria-label="错题学习详情">
      <KnowtErrorDetail
        :item="selectedError"
        :today="today"
        :loading="preview.loading.value && hasRequestedId && !selectedError"
        :requested-id="requestedId"
        :not-found="detailNotFound"
        @back="backToIndex"
        @tag="filterByTag"
      />

      <aside v-if="selectedError" class="kt-study-summary" aria-label="当前错题学习摘要">
        <span class="kt-study-summary__icon" aria-hidden="true"><CalendarClock :size="19" /></span>
        <h2>学习状态</h2>
        <p>{{ reviewLabel(selectedError, today) }}</p>
        <div class="kt-study-summary__meter"><i :class="{ 'is-due': isDue(selectedError, today) }"></i></div>
        <dl>
          <div><dt>复习轮次</dt><dd>{{ selectedError.review_count || 0 }}</dd></div>
          <div><dt>所属科目</dt><dd>{{ selectedError.subject || "未分类" }}</dd></div>
        </dl>
        <button type="button" @click="backToIndex">查看全部条目</button>
      </aside>
    </section>
  </div>
</template>

<style scoped>
.kt-error-page {
  width: min(1160px, calc(100% - 40px));
  margin: 0 auto;
  padding: 24px 0 64px;
  color: var(--kt-ink);
  animation: kt-page-enter 300ms cubic-bezier(.22, .76, .3, 1) both;
}

.kt-breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 14px;
  color: var(--kt-ink-muted);
  font-size: 12px;
}

.kt-breadcrumb i { color: var(--kt-ink-faint); font-size: 17px; font-style: normal; }
.kt-breadcrumb strong { color: var(--kt-ink-secondary); font-weight: 720; }

.kt-set-hero {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 32px;
  min-height: 220px;
  padding: 34px 38px;
  overflow: hidden;
  border: 1px solid var(--kt-line);
  border-radius: 26px;
  background:
    radial-gradient(circle at 92% 8%, rgba(74, 201, 187, .14), transparent 15rem),
    var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
}

.kt-set-hero::after {
  position: absolute;
  right: 115px;
  bottom: -82px;
  width: 180px;
  height: 180px;
  border: 26px solid rgba(95, 118, 213, .06);
  border-radius: 50%;
  content: "";
}

.kt-set-hero__copy { position: relative; z-index: 1; min-width: 0; }

.kt-set-kicker {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 9px;
  border-radius: 999px;
  color: #23776f;
  background: #e9f8f5;
  font-size: 10.5px;
  font-weight: 780;
}

.kt-set-hero h1 {
  margin: 13px 0 0;
  font-size: clamp(30px, 4.4vw, 48px);
  font-weight: 790;
  line-height: 1.08;
  letter-spacing: -.045em;
}

.kt-set-hero p {
  max-width: 620px;
  margin: 11px 0 0;
  color: var(--kt-ink-muted);
  font-size: 14px;
  line-height: 1.65;
}

.kt-set-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 19px;
  color: var(--kt-ink-muted);
  font-size: 12px;
}

.kt-set-meta strong { color: var(--kt-ink); font-size: 13px; }
.kt-set-meta i { width: 3px; height: 3px; border-radius: 50%; background: #c7c9d0; }

.kt-progress {
  position: relative;
  z-index: 1;
  width: 118px;
  height: 118px;
  display: grid;
  flex: none;
  place-items: center;
  border-radius: 50%;
  background: conic-gradient(var(--kt-accent) var(--kt-progress), #edf0f3 0);
  box-shadow: 0 14px 34px rgba(41, 158, 148, .12);
}

.kt-progress::before {
  position: absolute;
  width: 92px;
  height: 92px;
  border-radius: 50%;
  background: #fff;
  content: "";
}

.kt-progress > div { position: relative; display: grid; place-items: center; line-height: 1.15; }
.kt-progress strong { font-size: 24px; font-weight: 790; font-variant-numeric: tabular-nums; }
.kt-progress span { margin-top: 4px; color: var(--kt-ink-muted); font-size: 9.5px; font-weight: 680; }

.kt-study-modes {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-top: 16px;
}

.kt-mode-card {
  min-width: 0;
  min-height: 78px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 12px 13px;
  border: 1px solid var(--kt-line);
  border-radius: 17px;
  color: var(--kt-ink);
  background: var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: transform 180ms ease, border-color 180ms ease, box-shadow 180ms ease;
}

.kt-mode-card:hover:not(:disabled) { transform: translateY(-2px); border-color: color-mix(in srgb, var(--mode-color) 35%, var(--kt-line)); box-shadow: var(--kt-shadow-md); }
.kt-mode-card.is-active { border-color: color-mix(in srgb, var(--mode-color) 42%, var(--kt-line)); background: color-mix(in srgb, var(--mode-wash) 64%, #fff); }
.kt-mode-card:disabled { opacity: .45; cursor: default; }

.kt-mode-icon {
  width: 39px;
  height: 39px;
  display: grid;
  place-items: center;
  border-radius: 12px;
  color: var(--mode-color);
  background: var(--mode-wash);
}

.kt-mode-copy { min-width: 0; display: grid; gap: 2px; }
.kt-mode-copy strong { overflow: hidden; font-size: 12.5px; font-weight: 760; text-overflow: ellipsis; white-space: nowrap; }
.kt-mode-copy small { overflow: hidden; color: var(--kt-ink-muted); font-size: 10px; text-overflow: ellipsis; white-space: nowrap; }
.kt-mode-arrow { color: var(--kt-ink-faint); font-size: 17px; transition: transform 180ms ease; }
.kt-mode-card:hover .kt-mode-arrow { color: var(--mode-color); transform: translateX(2px); }

.kt-load-error {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  margin-top: 14px;
  padding: 12px 14px;
  border: 1px solid #efd4d2;
  border-radius: 14px;
  color: #a55250;
  background: #fff6f5;
}

.kt-load-error div { min-width: 0; display: grid; gap: 1px; }
.kt-load-error strong { font-size: 12.5px; }
.kt-load-error span { color: var(--kt-ink-muted); font-size: 11.5px; }
.kt-load-error button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  border: 0;
  border-radius: 9px;
  color: #9f4f4d;
  background: #f8e6e4;
  font: inherit;
  font-size: 11.5px;
  font-weight: 720;
  cursor: pointer;
}

.kt-subject-note { margin: 10px 4px 0; color: var(--kt-ink-muted); font-size: 11.5px; }

.kt-library-pane {
  margin-top: 20px;
  padding: 25px;
  border: 1px solid var(--kt-line);
  border-radius: 23px;
  background: var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
}

.kt-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.kt-section-head h2 { margin: 0; font-size: 20px; font-weight: 780; letter-spacing: -.025em; }
.kt-section-head p { margin: 4px 0 0; color: var(--kt-ink-muted); font-size: 12px; }
.kt-section-head > span { padding: 5px 9px; border-radius: 999px; color: #23776f; background: var(--kt-accent-soft); font-size: 10.5px; font-weight: 750; }

.kt-detail-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 250px;
  align-items: start;
  gap: 18px;
  margin-top: 20px;
}

.kt-study-summary {
  position: sticky;
  top: 84px;
  padding: 22px;
  border: 1px solid var(--kt-line);
  border-radius: 20px;
  background: var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
}

.kt-study-summary__icon { width: 42px; height: 42px; display: grid; place-items: center; border-radius: 13px; color: #586cc5; background: #eef1ff; }
.kt-study-summary h2 { margin: 14px 0 0; font-size: 15px; font-weight: 780; }
.kt-study-summary > p { margin: 5px 0 0; color: var(--kt-ink-muted); font-size: 11.5px; }
.kt-study-summary__meter { height: 7px; margin-top: 15px; overflow: hidden; border-radius: 999px; background: #edf0f2; }
.kt-study-summary__meter i { display: block; width: 78%; height: 100%; border-radius: inherit; background: var(--kt-accent); }
.kt-study-summary__meter i.is-due { width: 34%; background: #e67a85; }
.kt-study-summary dl { display: grid; gap: 9px; margin: 18px 0 0; }
.kt-study-summary dl div { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.kt-study-summary dt { color: var(--kt-ink-muted); font-size: 11px; }
.kt-study-summary dd { margin: 0; font-size: 11.5px; font-weight: 720; }
.kt-study-summary button { width: 100%; height: 38px; margin-top: 18px; border: 0; border-radius: 11px; color: #155e58; background: var(--kt-accent-soft); font: inherit; font-size: 11.5px; font-weight: 740; cursor: pointer; }

@keyframes kt-page-enter { from { opacity: 0; transform: translateY(5px); } to { opacity: 1; transform: translateY(0); } }

@media (max-width: 920px) {
  .kt-error-page { width: min(100% - 30px, 900px); }
  .kt-study-modes { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .kt-detail-layout { grid-template-columns: minmax(0, 1fr) 220px; }
}

@media (max-width: 700px) {
  .kt-error-page { width: min(100% - 24px, 680px); padding-top: 16px; padding-bottom: 88px; }
  .kt-set-hero { min-height: 0; padding: 25px 23px; border-radius: 21px; }
  .kt-progress { width: 92px; height: 92px; }
  .kt-progress::before { width: 70px; height: 70px; }
  .kt-progress strong { font-size: 19px; }
  .kt-detail-layout { display: block; }
  .kt-study-summary { display: none; }
}

@media (max-width: 520px) {
  .kt-breadcrumb { margin-bottom: 10px; }
  .kt-set-hero { grid-template-columns: 1fr; gap: 20px; }
  .kt-set-hero p { font-size: 12.5px; }
  .kt-progress { position: absolute; top: 22px; right: 20px; width: 72px; height: 72px; box-shadow: none; }
  .kt-progress::before { width: 55px; height: 55px; }
  .kt-progress strong { font-size: 15px; }
  .kt-progress span { font-size: 8px; }
  .kt-set-hero__copy { padding-right: 68px; }
  .kt-set-hero h1 { font-size: 30px; }
  .kt-set-meta { margin-top: 15px; gap: 6px; font-size: 10.5px; }
  .kt-study-modes { gap: 8px; margin-top: 10px; }
  .kt-mode-card { min-height: 68px; padding: 9px; border-radius: 14px; }
  .kt-mode-icon { width: 34px; height: 34px; border-radius: 10px; }
  .kt-mode-arrow { display: none; }
  .kt-library-pane { margin-top: 14px; padding: 17px 13px; border-radius: 18px; }
  .kt-error-page--selected .kt-breadcrumb,
  .kt-error-page--selected .kt-set-hero,
  .kt-error-page--selected .kt-study-modes,
  .kt-error-page--selected :deep(.kt-filter-bar) { display: none; }
  .kt-error-page--selected { padding-top: 10px; }
}

@media (prefers-reduced-motion: reduce) {
  .kt-error-page { animation: none; }
  .kt-mode-card,
  .kt-mode-arrow { transition: none; }
}
</style>
