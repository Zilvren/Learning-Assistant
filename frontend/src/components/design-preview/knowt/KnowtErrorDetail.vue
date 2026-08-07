<script setup>
import { computed } from "vue"
import {
  ArrowLeft,
  BookOpen,
  CalendarDays,
  CheckCircle2,
  CircleAlert,
  Clock3,
  Lightbulb,
  LoaderCircle,
  Tags,
  XCircle,
} from "lucide-vue-next"
import MarkdownRenderer from "../../MarkdownRenderer.vue"
import { hasContent, isDue, reviewLabel } from "../../../composables/useErrorLibrary.js"

const props = defineProps({
  item: { type: Object, default: null },
  today: { type: String, required: true },
  loading: Boolean,
  requestedId: { type: [String, Number, Array], default: null },
  notFound: Boolean,
})

const emit = defineEmits(["back", "tag"])

const requestedLabel = computed(() => {
  const value = Array.isArray(props.requestedId) ? props.requestedId[0] : props.requestedId
  return value === undefined || value === null || value === "" ? "—" : String(value)
})

const title = computed(() => {
  const value = typeof props.item?.title === "string" ? props.item.title.trim() : ""
  return value && value !== "暂无" && value !== "未记录"
    ? value
    : `错题 #${props.item?.id ?? requestedLabel.value}`
})

const subject = computed(() => {
  const value = typeof props.item?.subject === "string" ? props.item.subject.trim() : ""
  return value && value !== "暂无" && value !== "未记录" ? value : "未分类"
})

const topicTags = computed(() => safeTags(props.item?.tags))
const reasonTags = computed(() => safeTags(props.item?.reason_tags))

function safeTags(value) {
  return Array.isArray(value) ? value.filter((item) => typeof item === "string" && item.trim()) : []
}

function createdDate(value) {
  if (typeof value !== "string" || !value.trim()) return "日期未记录"
  return value.trim().slice(0, 10)
}
</script>

<template>
  <section
    v-if="props.loading"
    class="kt-detail-state"
    role="status"
    aria-live="polite"
    data-testid="knowt-detail-loading"
  >
    <span class="kt-detail-state__icon is-loading" aria-hidden="true"><LoaderCircle :size="25" /></span>
    <h2>正在打开学习卡</h2>
    <p>题面、错解和解析很快就好。</p>
  </section>

  <section v-else-if="props.notFound" class="kt-detail-state" data-testid="knowt-detail-not-found">
    <span class="kt-detail-state__icon is-missing" aria-hidden="true"><CircleAlert :size="25" /></span>
    <h2>没有找到这条错题</h2>
    <p>编号 {{ requestedLabel }} 不在当前筛选结果中，它也可能已经被移除。</p>
    <button type="button" aria-label="返回错题目录" @click="emit('back')">
      <ArrowLeft :size="16" aria-hidden="true" />返回错题目录
    </button>
  </section>

  <section v-else-if="!props.item" class="kt-detail-state" data-testid="knowt-detail-empty">
    <span class="kt-detail-state__icon" aria-hidden="true"><BookOpen :size="25" /></span>
    <h2>选择一条错题开始回看</h2>
    <p>从目录中挑一条记录，这里会展示完整题面与订正。</p>
  </section>

  <article v-else class="kt-error-detail" data-testid="knowt-error-detail">
    <header class="kt-detail-head">
      <button type="button" class="kt-detail-back" aria-label="返回错题目录" @click="emit('back')">
        <ArrowLeft :size="17" aria-hidden="true" />
        <span>返回错题目录</span>
      </button>

      <div class="kt-detail-head__meta">
        <span class="kt-detail-subject">{{ subject }}</span>
        <span class="kt-detail-review" :class="{ 'is-due': isDue(props.item, props.today) }">
          <CalendarDays :size="13" aria-hidden="true" />{{ reviewLabel(props.item, props.today) }}
        </span>
      </div>

      <h1 data-testid="knowt-detail-title">{{ title }}</h1>
      <div class="kt-detail-stats" aria-label="错题记录信息">
        <span><Clock3 :size="13" aria-hidden="true" />记录于 {{ createdDate(props.item.created) }}</span>
        <span><BookOpen :size="13" aria-hidden="true" />已复习 {{ props.item.review_count || 0 }} 次</span>
      </div>
    </header>

    <div class="kt-detail-content" data-testid="knowt-detail-content">
      <section class="kt-question-card" aria-labelledby="kt-question-title">
        <div class="kt-card-label"><span aria-hidden="true">Q</span><strong id="kt-question-title">题目</strong></div>
        <div v-if="hasContent(props.item.question)" class="kt-question-copy">
          <MarkdownRenderer :content="props.item.question" />
        </div>
        <p v-else class="kt-unrecorded">这条记录还没有补充题面。</p>
        <span class="kt-card-corner" aria-hidden="true"></span>
      </section>

      <div v-if="topicTags.length || reasonTags.length" class="kt-detail-tags">
        <Tags :size="15" aria-hidden="true" />
        <span class="kt-detail-tags__label">相关标签</span>
        <button
          v-for="value in topicTags"
          :key="`topic-${value}`"
          type="button"
          :aria-label="`按题目标签 ${value} 筛选`"
          @click="emit('tag', '题目标签', value)"
        >
          {{ value }}
        </button>
        <button
          v-for="value in reasonTags"
          :key="`reason-${value}`"
          type="button"
          class="is-reason"
          :aria-label="`按错因标签 ${value} 筛选`"
          @click="emit('tag', '错因标签', value)"
        >
          {{ value }}
        </button>
      </div>

      <div v-if="hasContent(props.item.wrong) || hasContent(props.item.correct)" class="kt-answer-grid">
        <section v-if="hasContent(props.item.wrong)" class="kt-answer-card is-wrong" aria-labelledby="kt-wrong-title">
          <span class="kt-answer-card__icon" aria-hidden="true"><XCircle :size="20" /></span>
          <div>
            <span class="kt-answer-card__kicker">当时的解法</span>
            <h2 id="kt-wrong-title">错解</h2>
            <MarkdownRenderer :content="props.item.wrong" />
          </div>
        </section>

        <section v-if="hasContent(props.item.correct)" class="kt-answer-card is-correct" aria-labelledby="kt-correct-title">
          <span class="kt-answer-card__icon" aria-hidden="true"><CheckCircle2 :size="20" /></span>
          <div>
            <span class="kt-answer-card__kicker">订正后的思路</span>
            <h2 id="kt-correct-title">正确解法</h2>
            <MarkdownRenderer :content="props.item.correct" />
          </div>
        </section>
      </div>

      <section v-if="hasContent(props.item.reason)" class="kt-reason-card" aria-labelledby="kt-reason-title">
        <span class="kt-reason-card__icon" aria-hidden="true"><Lightbulb :size="21" /></span>
        <div>
          <span>复盘提示</span>
          <h2 id="kt-reason-title">这次为什么会错？</h2>
          <MarkdownRenderer :content="props.item.reason" />
        </div>
      </section>
    </div>
  </article>
</template>

<style scoped>
.kt-error-detail,
.kt-detail-state {
  min-width: 0;
  border: 1px solid var(--kt-line);
  border-radius: 23px;
  background: var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
}

.kt-detail-state {
  min-height: 410px;
  display: grid;
  place-items: center;
  align-content: center;
  padding: 42px;
  text-align: center;
}

.kt-detail-state__icon {
  width: 55px;
  height: 55px;
  display: grid;
  place-items: center;
  border-radius: 18px;
  color: var(--kt-accent-strong);
  background: var(--kt-accent-soft);
}

.kt-detail-state__icon.is-missing { color: #af5960; background: #fff0f2; }
.kt-detail-state__icon.is-loading svg { animation: kt-detail-spin 850ms linear infinite; }
.kt-detail-state h2 { margin: 15px 0 0; font-size: 17px; font-weight: 780; }
.kt-detail-state p { max-width: 390px; margin: 7px 0 0; color: var(--kt-ink-muted); font-size: 12px; line-height: 1.6; }

.kt-detail-state button,
.kt-detail-back {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  border: 0;
  color: var(--kt-accent-strong);
  background: var(--kt-accent-soft);
  font: inherit;
  font-weight: 720;
  cursor: pointer;
}

.kt-detail-state button { height: 38px; margin-top: 18px; padding: 0 13px; border-radius: 11px; font-size: 11.5px; }

.kt-detail-head { padding: 22px 27px 21px; border-bottom: 1px solid var(--kt-line); }
.kt-detail-back { height: 34px; padding: 0 10px; border-radius: 10px; font-size: 11px; }
.kt-detail-back:hover,
.kt-detail-state button:hover { color: #fff; background: var(--kt-accent-strong); }

.kt-detail-head__meta { display: flex; flex-wrap: wrap; align-items: center; gap: 7px; margin-top: 20px; }
.kt-detail-subject,
.kt-detail-review {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 10px;
  font-weight: 730;
}
.kt-detail-subject { color: #5264ae; background: #eef1ff; }
.kt-detail-review { color: #34766e; background: #eaf8f4; }
.kt-detail-review.is-due { color: #af5660; background: #fff0f2; }

.kt-detail-head h1 {
  margin: 12px 0 0;
  color: var(--kt-ink);
  font-size: clamp(24px, 3.2vw, 36px);
  font-weight: 800;
  line-height: 1.16;
  letter-spacing: -.035em;
}

.kt-detail-stats { display: flex; flex-wrap: wrap; gap: 13px; margin-top: 12px; color: var(--kt-ink-muted); font-size: 10.5px; }
.kt-detail-stats span { display: inline-flex; align-items: center; gap: 5px; }

.kt-detail-content { display: grid; gap: 15px; padding: 24px 27px 29px; }

.kt-question-card {
  position: relative;
  min-height: 230px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: start;
  gap: 18px;
  padding: 28px;
  overflow: hidden;
  border: 1px solid #dce6e3;
  border-radius: 20px;
  background: linear-gradient(145deg, #ffffff 0%, #fbfefd 72%, #f0faf7 100%);
  box-shadow: 0 13px 33px rgba(26, 76, 64, .06);
}

.kt-card-label { display: grid; justify-items: center; gap: 6px; }
.kt-card-label > span { width: 37px; height: 37px; display: grid; place-items: center; border-radius: 12px; color: #fff; background: var(--kt-accent); font-size: 15px; font-weight: 850; box-shadow: 0 8px 20px rgba(22, 141, 128, .2); }
.kt-card-label strong { color: var(--kt-ink-muted); font-size: 9px; font-weight: 760; }

.kt-question-copy { min-width: 0; padding-top: 4px; color: var(--kt-ink); font-size: 15px; line-height: 1.78; }
.kt-question-copy :deep(.markdown-body > :first-child) { margin-top: 0; }
.kt-question-copy :deep(.markdown-body > :last-child) { margin-bottom: 0; }
.kt-unrecorded { margin: 8px 0 0; color: var(--kt-ink-faint); font-size: 13px; }

.kt-card-corner { position: absolute; right: -44px; bottom: -54px; width: 135px; height: 135px; border: 19px solid rgba(40, 169, 153, .07); border-radius: 50%; }

.kt-detail-tags { display: flex; flex-wrap: wrap; align-items: center; gap: 6px; padding: 4px 2px; color: var(--kt-ink-faint); }
.kt-detail-tags__label { margin-right: 2px; color: var(--kt-ink-muted); font-size: 10px; font-weight: 710; }
.kt-detail-tags button { max-width: 180px; overflow: hidden; padding: 4px 8px; border: 0; border-radius: 8px; color: #39756e; background: #eaf7f4; font: inherit; font-size: 10px; font-weight: 680; text-overflow: ellipsis; white-space: nowrap; cursor: pointer; }
.kt-detail-tags button:hover { color: var(--kt-accent-strong); background: #dcefeb; }
.kt-detail-tags button.is-reason { color: #9e5961; background: #fff0f2; }

.kt-answer-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.kt-answer-card { min-width: 0; display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; gap: 12px; padding: 20px; border: 1px solid var(--kt-line); border-radius: 18px; }
.kt-answer-card.is-wrong { border-color: #f0d7da; background: #fff8f8; }
.kt-answer-card.is-correct { border-color: #cfe7e1; background: #f5fbf9; }
.kt-answer-card__icon { width: 35px; height: 35px; display: grid; place-items: center; border-radius: 11px; }
.is-wrong .kt-answer-card__icon { color: #bd626a; background: #ffe8eb; }
.is-correct .kt-answer-card__icon { color: #238479; background: #dff3ee; }
.kt-answer-card__kicker { color: var(--kt-ink-muted); font-size: 9px; font-weight: 710; }
.kt-answer-card h2 { margin: 3px 0 11px; font-size: 14px; font-weight: 780; }
.kt-answer-card :deep(.markdown-body) { color: var(--kt-ink-secondary); font-size: 12px; line-height: 1.68; }
.kt-answer-card :deep(.markdown-body > :first-child) { margin-top: 0; }
.kt-answer-card :deep(.markdown-body > :last-child) { margin-bottom: 0; }

.kt-reason-card { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; gap: 13px; padding: 20px; border: 1px solid #f0e1c8; border-radius: 18px; background: #fffaf2; }
.kt-reason-card__icon { width: 38px; height: 38px; display: grid; place-items: center; border-radius: 12px; color: #b57925; background: #ffedcf; }
.kt-reason-card > div > span { color: #a47736; font-size: 9px; font-weight: 730; }
.kt-reason-card h2 { margin: 3px 0 10px; font-size: 14px; font-weight: 780; }
.kt-reason-card :deep(.markdown-body) { color: var(--kt-ink-secondary); font-size: 12px; line-height: 1.7; }
.kt-reason-card :deep(.markdown-body > :first-child) { margin-top: 0; }
.kt-reason-card :deep(.markdown-body > :last-child) { margin-bottom: 0; }

@keyframes kt-detail-spin { to { transform: rotate(360deg); } }

@media (max-width: 760px) {
  .kt-detail-head { padding: 17px 18px; }
  .kt-detail-content { padding: 17px 18px 21px; }
  .kt-question-card { min-height: 205px; padding: 22px; }
  .kt-answer-grid { grid-template-columns: 1fr; }
}

@media (max-width: 520px) {
  .kt-error-detail,
  .kt-detail-state { border-radius: 18px; }
  .kt-detail-head { padding: 14px 15px 16px; }
  .kt-detail-back span { display: none; }
  .kt-detail-back { width: 35px; padding: 0; }
  .kt-detail-head__meta { margin-top: 14px; }
  .kt-detail-head h1 { font-size: 24px; }
  .kt-detail-content { gap: 11px; padding: 13px 12px 17px; }
  .kt-question-card { grid-template-columns: 1fr; gap: 14px; min-height: 230px; padding: 20px 17px; border-radius: 16px; }
  .kt-card-label { display: flex; align-items: center; justify-items: initial; }
  .kt-card-label > span { width: 32px; height: 32px; border-radius: 10px; font-size: 13px; }
  .kt-question-copy { padding-top: 0; font-size: 14px; }
  .kt-answer-card,
  .kt-reason-card { padding: 16px 14px; border-radius: 15px; }
  .kt-detail-state { min-height: 330px; padding: 26px 18px; }
}

@media (prefers-reduced-motion: reduce) {
  .kt-detail-state__icon.is-loading svg { animation: none; }
}
</style>
