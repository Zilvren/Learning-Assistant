<script setup>
import { computed, ref, onMounted } from "vue"
import { api } from "../api/index.js"
import { useSettings } from "../store/settings.js"
import MarkdownRenderer from "./MarkdownRenderer.vue"

const emit = defineEmits(["snack"])
const data = ref({ knowledge: {}, weak_errors: [], total_errors: 0, reviewed: 0, advice: "" })
const loading = ref(true)
const selectedReview = ref(null)

const { username } = useSettings()
const hour = new Date().getHours()
const greeting = hour < 12 ? "早上好" : hour < 18 ? "下午好" : "晚上好"
const date = new Date().toISOString().slice(0, 10)

const dueReviews = computed(() => data.value.weak_errors)
const knowledgeItems = computed(() => Object.entries(data.value.knowledge || {}).slice(0, 6))
const topSubject = computed(() => {
  const counts = {}
  for (const item of data.value.weak_errors) counts[item.subject] = (counts[item.subject] || 0) + 1
  return Object.entries(counts).sort((a, b) => b[1] - a[1])[0]?.[0] || "暂无"
})

function reviewTitle(e) {
  return e.title || `未命名错题 #${e.id}`
}

function showText(v) {
  const t = (v || "").trim()
  return t && t !== "未记录"
}

function reviewPlanText(e) {
  const count = e.review_count || 0
  if (!e.next_review) return `第 ${count + 1} 轮复习`
  if (e.next_review < date) return `已逾期 · 第 ${count + 1} 轮`
  if (e.next_review === date) return `今日到期 · 第 ${count + 1} 轮`
  return `下次 ${e.next_review} · 第 ${count + 1} 轮`
}

function closeReviewDetail() {
  selectedReview.value = null
}

async function loadHomeData() {
  const [push, allErrors] = await Promise.all([
    api.getDailyPush(),
    api.getErrors(),
  ])
  const fullById = new Map((allErrors.errors || []).map(e => [e.id, e]))
  data.value = {
    ...push,
    weak_errors: (push.weak_errors || []).map(e => ({ ...e, ...(fullById.get(e.id) || {}) })),
  }
}

async function markReviewed(e) {
  try {
    const result = await api.reviewError(e.id)
    emit("snack", result.next_review ? `已复习 #${e.id}，下次 ${result.next_review}` : `已标记复习 #${e.id}`)
    if (selectedReview.value?.id === e.id) {
      closeReviewDetail()
    }
    await loadHomeData()
  } catch (err) {
    emit("snack", err.message, "#ef4444")
  }
}

onMounted(async () => {
  try {
    await useSettings().load()
    await loadHomeData()
  } catch (e) {
    emit("snack", e.message, "#ef4444")
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="home-workbench">
    <section class="home-hero">
      <div>
        <div class="eyebrow">{{ date }}</div>
        <h2>{{ greeting }}<template v-if="username">，{{ username }}</template></h2>
        <p>{{ data.advice || "先完成今日复习，再补新增错题。" }}</p>
      </div>
      <div class="hero-stats">
        <div>
          <strong>{{ data.total_errors }}</strong>
          <span>错题总数</span>
        </div>
        <div>
          <strong>{{ data.due_count || data.weak_errors.length }}</strong>
          <span>今日到期</span>
        </div>
        <div>
          <strong>{{ data.overdue_count || 0 }}</strong>
          <span>逾期复习</span>
        </div>
        <div>
          <strong>{{ topSubject }}</strong>
          <span>当前薄弱</span>
        </div>
      </div>
    </section>

    <div v-if="loading" class="empty-state">加载中...</div>

    <template v-else>
      <section class="home-grid">
        <div class="panel panel-main">
          <div class="panel-head">
            <div>
              <h3>今日优先复习</h3>
              <p>{{ data.weak_errors.length }} 道按艾宾浩斯曲线到期</p>
            </div>
          </div>
          <div v-if="dueReviews.length" class="review-list review-list-scroll">
            <article v-for="e in dueReviews" :key="e.id" class="review-item" :class="{ overdue: e.next_review && e.next_review < date }">
              <div class="review-id">#{{ e.id }}</div>
              <div class="review-copy">
                <div class="review-title">
                  <span class="review-subject">[{{ e.subject }}]</span>
                  <span>{{ reviewTitle(e) }}</span>
                </div>
                <div class="review-meta">{{ reviewPlanText(e) }}</div>
              </div>
              <div class="review-actions">
                <button type="button" class="review-link" @click="selectedReview = e">查看详情</button>
                <button type="button" class="review-done" @click="markReviewed(e)">标记复习</button>
              </div>
            </article>
          </div>
          <div v-else class="empty-state">暂无待复习错题</div>
        </div>

        <div class="side-stack">
        <div class="panel compact-panel">
          <div class="panel-head">
            <div>
              <h3>复习状态</h3>
              <p>今日队列概览</p>
            </div>
          </div>
          <div class="status-grid">
            <div>
              <strong>{{ data.reviewed }}</strong>
              <span>已复习</span>
            </div>
            <div>
              <strong>{{ Math.max((data.total_errors || 0) - (data.reviewed || 0), 0) }}</strong>
              <span>未复习</span>
            </div>
          </div>
        </div>

        <div class="panel compact-panel">
          <div class="panel-head">
            <div>
              <h3>今日知识点</h3>
              <p>按当前科目随机抽取</p>
            </div>
          </div>
          <div class="knowledge-list">
            <article v-for="[subject, tip] in knowledgeItems" :key="subject" class="knowledge-item">
              <span>{{ subject }}</span>
              <p>{{ tip }}</p>
            </article>
            <div v-if="!knowledgeItems.length" class="empty-state">暂无知识点</div>
          </div>
        </div>
        </div>
      </section>
    </template>

    <Teleport to="body">
      <div v-if="selectedReview" class="dialog-overlay review-overlay" @click.self="closeReviewDetail">
        <div class="review-dialog">
          <div class="review-dialog-head">
            <div>
              <span class="review-dialog-id">#{{ selectedReview.id }} · {{ selectedReview.subject }}</span>
              <h3>{{ reviewTitle(selectedReview) }}</h3>
              <p>{{ reviewPlanText(selectedReview) }}<template v-if="selectedReview.created"> · 创建 {{ selectedReview.created.slice(0, 10) }}</template></p>
            </div>
            <div class="review-dialog-actions">
              <button type="button" class="review-done review-done-hero" @click="markReviewed(selectedReview)">标记复习</button>
              <button type="button" class="close-btn" @click="closeReviewDetail">×</button>
            </div>
          </div>
          <div class="review-dialog-body">
            <section class="detail-section">
              <h4>题目</h4>
              <MarkdownRenderer :content="selectedReview.question" />
            </section>
            <section class="detail-section wrong-block">
              <h4>错解</h4>
              <MarkdownRenderer :content="showText(selectedReview.wrong) ? selectedReview.wrong : '未记录'" />
            </section>
            <section class="detail-section correct-block">
              <h4>正解</h4>
              <MarkdownRenderer :content="showText(selectedReview.correct) ? selectedReview.correct : '未记录'" />
            </section>
            <section v-if="showText(selectedReview.reason)" class="detail-section">
              <h4>错因</h4>
              <MarkdownRenderer :content="selectedReview.reason" />
            </section>
            <div v-if="selectedReview.tags?.length || selectedReview.reason_tags?.length" class="detail-tags">
              <span v-for="t in selectedReview.tags || []" :key="t" class="mini-chip">{{ t }}</span>
              <span v-for="t in selectedReview.reason_tags || []" :key="t" class="mini-chip reason">{{ t }}</span>
            </div>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.home-workbench {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
}
.home-hero {
  display: flex; justify-content: space-between; gap: 24px; align-items: stretch;
  padding: 18px 20px; border: 1px solid var(--border); border-radius: 10px;
  background: var(--surface);
}
.eyebrow { font-size: 12px; color: var(--text-muted); margin-bottom: 8px; }
.home-hero h2 { font-size: 24px; line-height: 1.25; margin-bottom: 8px; letter-spacing: 0; }
.home-hero p { color: var(--text-sec); font-size: 14px; }
.hero-stats { display: grid; grid-template-columns: repeat(4, minmax(88px, 1fr)); gap: 10px; min-width: 480px; }
.hero-stats div { padding: 12px; border-radius: 8px; border: 1px solid var(--border); background: var(--surface-soft); }
.hero-stats strong { display: block; font-size: 20px; color: var(--text); margin-bottom: 4px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.hero-stats span { color: var(--text-muted); font-size: 12px; }
.home-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.55fr) minmax(300px, .75fr);
  gap: 14px;
  align-items: start;
  min-height: 0;
}
.panel { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 16px; }
.panel-main {
  min-height: 0;
  max-height: calc(100vh - 220px);
  display: flex;
  flex-direction: column;
}
.side-stack {
  display: flex;
  flex-direction: column;
  gap: 14px;
  max-height: calc(100vh - 220px);
  min-height: 0;
}
.compact-panel { flex-shrink: 0; }
.panel-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.panel-head h3 { font-size: 16px; margin-bottom: 3px; }
.panel-head p { font-size: 12px; color: var(--text-muted); }
.review-list, .knowledge-list { display: flex; flex-direction: column; gap: 8px; }
.review-list-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 4px;
}
.review-item {
  width: 100%;
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-soft);
  color: inherit;
  text-align: left;
  transition: border-color .15s, background .15s, box-shadow .15s;
}
.review-item.overdue {
  border-left: 3px solid var(--warning);
}
.review-item:hover {
  border-color: rgba(37, 99, 235, .28);
  background: var(--surface-muted);
  box-shadow: 0 4px 12px rgba(15,23,42,.06);
}
.review-id { font-weight: 700; color: var(--accent); }
.review-copy { min-width: 0; }
.review-title { display: flex; gap: 8px; align-items: baseline; color: var(--text); font-size: 14px; line-height: 1.45; min-width: 0; }
.review-title > span:last-child { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.review-subject { flex-shrink: 0; color: var(--text-sec); font-weight: 600; }
.review-meta { color: var(--text-muted); font-size: 12px; margin-top: 4px; }
.review-actions { display: flex; gap: 8px; align-items: center; }
.review-link, .review-done {
  height: 32px;
  padding: 0 10px;
  border-radius: 7px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: background .15s, color .15s, border-color .15s, transform .15s;
}
.review-link {
  border: 1px solid transparent;
  background: transparent;
  color: var(--accent);
}
.review-link:hover { background: color-mix(in srgb, var(--accent) 14%, transparent); }
.review-done {
  border: 1px solid rgba(16,185,129,.24);
  background: rgba(16,185,129,.08);
  color: #047857;
}
.review-done:hover {
  background: rgba(16,185,129,.14);
  border-color: rgba(16,185,129,.38);
}
.review-link:focus-visible, .review-done:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, .14);
}
.review-link:active, .review-done:active { transform: translateY(1px); }
.status-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.status-grid div {
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface-soft);
}
.status-grid strong {
  display: block;
  font-size: 22px;
  margin-bottom: 4px;
  color: var(--text);
}
.status-grid span { color: var(--text-muted); font-size: 12px; }
.knowledge-list {
  max-height: min(360px, calc(100vh - 395px));
  overflow-y: auto;
  padding-right: 4px;
}
.knowledge-item { padding: 11px; border: 1px solid var(--border); border-radius: 8px; }
.knowledge-item span { display: inline-block; font-size: 12px; color: var(--accent); font-weight: 700; margin-bottom: 6px; }
.knowledge-item p { color: var(--text-sec); font-size: 13px; line-height: 1.65; }
.empty-state { padding: 28px; text-align: center; color: var(--text-muted); font-size: 13px; }
.review-overlay { align-items: center; }
.review-dialog {
  width: min(920px, 92vw);
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--surface);
  border-radius: 12px;
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.review-dialog-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 18px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--border);
}
.review-dialog-id { color: var(--accent); font-size: 13px; font-weight: 700; }
.review-dialog-head h3 { margin-top: 6px; font-size: 18px; line-height: 1.45; }
.review-dialog-head p { margin-top: 5px; color: var(--text-muted); font-size: 12px; }
.review-dialog-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.review-done-hero {
  height: 40px;
  min-width: 96px;
  padding: 0 16px;
  font-size: 13px;
}
.close-btn {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface-muted);
  color: var(--text-sec);
  font-size: 26px;
  line-height: 1;
  cursor: pointer;
}
.close-btn:hover { background: var(--surface-hover); color: var(--accent); }
.review-dialog-body { overflow: auto; padding: 18px 20px 22px; }
.detail-section { padding: 13px 0; border-bottom: 1px solid var(--border); }
.detail-section:first-child { padding-top: 0; }
.detail-section h4 { font-size: 13px; color: var(--text-sec); margin-bottom: 8px; }
.wrong-block, .correct-block { border-radius: 10px; padding: 12px; margin: 12px 0; border: 1px solid; }
.wrong-block { background: rgba(239,68,68,.04); border-color: rgba(239,68,68,.18); }
.correct-block { background: rgba(16,185,129,.04); border-color: rgba(16,185,129,.16); }
.detail-tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 14px; }
.mini-chip { border: 1px solid var(--border); background: var(--surface-muted); color: var(--text-sec); border-radius: 6px; padding: 3px 7px; font-size: 11px; }
.mini-chip.reason { color: #b45309; background: #fffbeb; border-color: #fde68a; }
@media (max-width: 980px) {
  .home-hero, .home-grid { grid-template-columns: 1fr; flex-direction: column; }
  .hero-stats { min-width: 0; grid-template-columns: repeat(2, 1fr); }
  .panel-main, .side-stack { max-height: none; }
  .review-list-scroll, .knowledge-list { max-height: none; overflow: visible; }
  .review-item { grid-template-columns: 48px minmax(0, 1fr); }
  .review-actions { grid-column: 2; justify-content: flex-start; }
  .review-dialog-head { flex-direction: column; }
  .review-dialog-actions { width: 100%; justify-content: flex-end; }
}
</style>
