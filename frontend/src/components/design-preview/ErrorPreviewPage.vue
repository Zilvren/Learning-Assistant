<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { AlertCircle, ArrowRight, ChevronRight, RefreshCw } from "lucide-vue-next"
import { useRoute, useRouter } from "vue-router"
import { isDue } from "../../composables/useErrorLibrary.js"
import { useErrorPreview } from "../../composables/useErrorPreview.js"
import PreviewErrorDetail from "./PreviewErrorDetail.vue"
import PreviewErrorIndex from "./PreviewErrorIndex.vue"
import PreviewFilterBar from "./PreviewFilterBar.vue"

const route = useRoute()
const router = useRouter()
const preview = useErrorPreview()

const subject = ref("全部")
const mode = ref("全部")
const keyword = ref("")
const appliedKeyword = ref("")
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
const selectedDocumentLabel = computed(() => {
  if (!selectedError.value) return selectedId.value ? `错题 #${selectedId.value}` : "选择一则错题"
  const title = typeof selectedError.value.title === "string" ? selectedError.value.title.trim() : ""
  return title && title !== "暂无" && title !== "未记录" ? title : `错题 #${selectedError.value.id}`
})
const errorMessage = computed(() => {
  const value = preview.error.value
  return typeof value === "string" ? value : value?.message || "读取错题时遇到问题，请稍后重试。"
})

function currentFilters() {
  return { subject: subject.value, mode: mode.value, keyword: appliedKeyword.value }
}

async function clearSelectedRoute() {
  if (!hasRequestedId.value) return
  await router.replace({ name: "design-preview-errors" })
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
}

function filterByTag(nextMode, value) {
  window.clearTimeout(keywordTimer)
  mode.value = nextMode
  keyword.value = value
  appliedKeyword.value = value
}

async function selectError(item) {
  await router.push({ name: "design-preview-errors", params: { id: item.id } })
}

async function backToIndex() {
  await router.push({ name: "design-preview-errors" })
}

onMounted(() => preview.loadSubjects())

onBeforeUnmount(() => {
  window.clearTimeout(keywordTimer)
  preview.dispose()
})
</script>

<template>
  <div
    class="rn-library-page"
    :class="{ 'rn-library-page--selected': hasRequestedId }"
    data-testid="preview-library-workbench"
  >
    <section class="rn-collection-pane" aria-labelledby="rn-library-title">
      <header class="rn-collection-head">
        <div class="rn-collection-head__context">
          <span>资料库</span>
          <span aria-hidden="true">/</span>
          <strong>只读样板</strong>
        </div>
        <div class="rn-collection-head__title">
          <h1 id="rn-library-title">错题库</h1>
          <span>{{ preview.errors.value.length }}</span>
        </div>
        <p><strong>{{ dueCount }}</strong> 道待复习 · 内容按最近复习状态排列</p>
      </header>

      <PreviewFilterBar
        :subjects="preview.subjects.value"
        :subject="subject"
        :mode="mode"
        :keyword="keyword"
        :loading="preview.loading.value"
        :count="preview.errors.value.length"
        :due-count="dueCount"
        @update:subject="subject = $event"
        @update:mode="mode = $event"
        @update:keyword="updateKeyword"
        @reset="resetFilters"
      />

      <div v-if="preview.error.value" class="rn-library-alert" role="alert">
        <AlertCircle :size="17" aria-hidden="true" />
        <div><strong>暂时无法更新错题</strong><span>{{ errorMessage }}</span></div>
        <button type="button" @click="preview.refresh(currentFilters())">
          <RefreshCw :size="14" aria-hidden="true" />重试
        </button>
      </div>

      <div v-if="preview.subjectsError.value" class="rn-library-subject-note" role="status">
        科目暂时不可用，仍可浏览全部错题。
      </div>

      <div class="rn-index-slot">
      <PreviewErrorIndex
        :items="preview.errors.value"
        :selected-id="selectedId"
        :loading="preview.loading.value && !preview.loaded.value"
        :today="today"
        @select="selectError"
        @tag="filterByTag"
      />
      </div>
    </section>

    <section class="rn-document-pane" aria-label="错题文档阅读区">
      <header class="rn-document-toolbar">
        <div class="rn-document-crumb" aria-label="当前位置">
          <span>错题库</span>
          <ChevronRight :size="14" aria-hidden="true" />
          <strong>{{ selectedDocumentLabel }}</strong>
        </div>
        <RouterLink
          class="rn-open-current"
          :to="{ name: 'errors', params: selectedId ? { id: selectedId } : {} }"
        >
          在现有版本中打开
          <ArrowRight :size="15" aria-hidden="true" />
        </RouterLink>
      </header>

      <PreviewErrorDetail
        :item="selectedError"
        :today="today"
        :loading="preview.loading.value && hasRequestedId && !selectedError"
        :requested-id="requestedId"
        :not-found="detailNotFound"
        @back="backToIndex"
        @tag="filterByTag"
      />
    </section>
  </div>
</template>

<style scoped>
.rn-library-page {
  display: grid;
  grid-template-columns: 356px minmax(0, 1fr);
  width: 100%;
  height: 100dvh;
  min-height: 0;
  overflow: hidden;
  background: var(--kp-surface);
  animation: rn-workbench-enter 260ms cubic-bezier(.22, .76, .3, 1) both;
}

@keyframes rn-workbench-enter {
  from { opacity: 0; }
  to { opacity: 1; }
}

.rn-collection-pane,
.rn-document-pane {
  min-width: 0;
  min-height: 0;
}

.rn-collection-pane {
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--kp-line);
  background: #fafafa;
}

.rn-collection-head {
  flex: 0 0 auto;
  padding: 24px 22px 15px;
}

.rn-collection-head__context {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--kp-ink-faint);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: .035em;
}

.rn-collection-head__context strong {
  color: var(--kp-accent-strong);
  font-weight: 720;
}

.rn-collection-head__title {
  display: flex;
  align-items: center;
  gap: 9px;
  margin-top: 7px;
}

.rn-collection-head h1 {
  margin: 0;
  color: var(--kp-ink);
  font-size: 23px;
  font-weight: 735;
  line-height: 1.25;
  letter-spacing: -.035em;
}

.rn-collection-head__title > span {
  display: inline-grid;
  min-width: 24px;
  height: 20px;
  padding-inline: 6px;
  place-items: center;
  border-radius: 6px;
  color: var(--kp-ink-muted);
  background: #ececef;
  font-size: 11px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.rn-collection-head p {
  margin: 6px 0 0;
  color: var(--kp-ink-muted);
  font-size: 11.5px;
}

.rn-collection-head p strong {
  color: #aa6a28;
  font-weight: 750;
}

.rn-library-alert {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 9px;
  margin: 8px 12px 0;
  padding: 10px 11px;
  border: 1px solid #ead4d1;
  border-radius: 9px;
  color: var(--kp-danger);
  background: #fff8f7;
}

.rn-library-alert div { display: grid; min-width: 0; gap: 1px; }
.rn-library-alert strong { color: #8a4f48; font-size: 12px; }
.rn-library-alert span { color: var(--kp-ink-muted); font-size: 11px; line-height: 1.45; }
.rn-library-alert button {
  grid-column: 2;
  display: inline-flex;
  justify-self: start;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: 0;
  border-radius: 6px;
  color: var(--kp-danger);
  background: #f8e9e7;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}

.rn-library-subject-note {
  margin: 7px 16px 0;
  color: var(--kp-ink-muted);
  font-size: 11px;
}

.rn-index-slot {
  flex: 1 1 auto;
  min-height: 0;
  overflow: hidden;
}

.rn-index-slot :deep(.kp-error-index) {
  width: 100%;
  height: 100%;
}

.rn-document-pane {
  display: flex;
  flex-direction: column;
  background: var(--kp-surface);
}

.rn-document-toolbar {
  display: flex;
  min-height: 46px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 18px;
  border-bottom: 1px solid var(--kp-line);
  background: rgba(255, 255, 255, .94);
}

.rn-document-crumb {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 5px;
  color: var(--kp-ink-faint);
  font-size: 11.5px;
}

.rn-document-crumb strong {
  max-width: min(46vw, 520px);
  overflow: hidden;
  color: var(--kp-ink-secondary);
  font-weight: 620;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rn-open-current {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 8px;
  border-radius: 6px;
  color: var(--kp-accent-strong);
  font-size: 11.5px;
  font-weight: 680;
  transition: background-color var(--kp-motion-fast) ease;
}

.rn-open-current:hover { background: var(--kp-accent-wash); }

.rn-document-pane :deep(.kp-error-detail) {
  flex: 1 1 auto;
  min-height: 0;
}

@media (max-width: 1120px) {
  .rn-library-page { grid-template-columns: 320px minmax(0, 1fr); }
  .rn-collection-head { padding-inline: 18px; }
}

@media (max-width: 767px) {
  .rn-library-page {
    display: block;
    height: 100%;
  }

  .rn-collection-pane,
  .rn-document-pane {
    height: 100%;
  }

  .rn-library-page:not(.rn-library-page--selected) .rn-document-pane { display: none; }
  .rn-library-page.rn-library-page--selected .rn-collection-pane { display: none; }
  .rn-collection-head { padding: 19px 17px 13px; }
  .rn-document-toolbar { min-height: 43px; padding-inline: 12px; }
  .rn-document-crumb strong { max-width: 42vw; }
  .rn-open-current { width: 32px; justify-content: center; padding: 0; }
  .rn-open-current { font-size: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .rn-library-page { animation: none; }
  .rn-open-current { transition: none; }
}
</style>
