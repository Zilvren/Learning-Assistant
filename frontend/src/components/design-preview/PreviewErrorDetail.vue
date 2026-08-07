<script setup>
import {
  ArrowLeft,
  BookOpen,
  CheckCircle2,
  CircleAlert,
  FileQuestion,
  Lightbulb,
  NotebookPen,
  Tag,
} from "lucide-vue-next"
import { hasContent, reviewLabel } from "../../composables/useErrorLibrary.js"
import MarkdownRenderer from "../MarkdownRenderer.vue"

defineProps({
  item: Object,
  today: String,
  loading: Boolean,
  requestedId: [String, Number],
  notFound: Boolean,
})

defineEmits(["back", "tag"])

const missingTitles = new Set(["", "暂无", "未记录"])

function displayTitle(item) {
  const title = typeof item?.title === "string" ? item.title.trim() : ""
  return missingTitles.has(title) ? `错题 #${item?.id ?? "—"}` : title
}

function createdDate(value) {
  if (!value) return "录入时间未知"
  return `录入于 ${String(value).slice(0, 10)}`
}

function safeTags(value) {
  if (!Array.isArray(value)) return []
  return value.map((tag) => String(tag ?? "").trim()).filter(Boolean)
}
</script>

<template>
  <article
    class="kp-error-detail"
    :aria-labelledby="item ? 'kp-detail-title' : undefined"
    data-testid="preview-error-detail"
  >
    <div v-if="loading" class="kp-detail-loading" aria-busy="true" data-testid="preview-detail-loading">
      <span class="kp-sr-only" role="status">正在加载错题详情</span>
      <div class="kp-detail-skeleton" aria-hidden="true">
        <span class="kp-skeleton-line kp-skeleton-line--meta"></span>
        <span class="kp-skeleton-line kp-skeleton-line--heading"></span>
        <span class="kp-skeleton-line kp-skeleton-line--subheading"></span>
        <div v-for="n in 3" :key="n" class="kp-skeleton-section">
          <span class="kp-skeleton-line kp-skeleton-line--label"></span>
          <span class="kp-skeleton-line"></span>
          <span class="kp-skeleton-line kp-skeleton-line--body"></span>
          <span class="kp-skeleton-line kp-skeleton-line--short"></span>
        </div>
      </div>
    </div>

    <template v-else-if="item">
      <header class="kp-detail-header">
        <div class="kp-detail-header-inner">
          <button type="button" class="kp-mobile-back" aria-label="返回错题目录" @click="$emit('back')">
            <ArrowLeft :size="17" aria-hidden="true" />
            返回目录
          </button>

          <div class="kp-detail-heading-row">
            <div class="kp-detail-heading">
              <div class="kp-detail-meta">
                <span class="kp-detail-id">#{{ item.id }}</span>
                <span>{{ item.subject || "未分类" }}</span>
                <span>{{ reviewLabel(item, today) }}</span>
                <span>{{ createdDate(item.created) }}</span>
              </div>
              <h1 id="kp-detail-title" data-testid="preview-detail-title">{{ displayTitle(item) }}</h1>
            </div>
            <span class="kp-readonly-chip"><i aria-hidden="true"></i>只读资料</span>
          </div>
        </div>
      </header>

      <div class="kp-detail-scroll">
        <div class="kp-detail-content" data-testid="preview-detail-content">
          <section class="kp-content-section kp-content-section--question" aria-labelledby="kp-question-heading">
            <h2 id="kp-question-heading">
              <span class="kp-section-icon" aria-hidden="true"><NotebookPen :size="15" /></span>
              <span>题目</span><small>原始记录</small>
            </h2>
            <MarkdownRenderer :content="item.question || ''" />
          </section>

          <section
            v-if="hasContent(item.wrong)"
            class="kp-content-section kp-content-section--wrong"
            aria-labelledby="kp-wrong-heading"
          >
            <h2 id="kp-wrong-heading">
              <span class="kp-section-icon" aria-hidden="true"><CircleAlert :size="15" /></span>
              <span>错解</span><small>需要修正的思路</small>
            </h2>
            <MarkdownRenderer :content="item.wrong" />
          </section>

          <section
            v-if="hasContent(item.correct)"
            class="kp-content-section kp-content-section--correct"
            aria-labelledby="kp-correct-heading"
          >
            <h2 id="kp-correct-heading">
              <span class="kp-section-icon" aria-hidden="true"><CheckCircle2 :size="15" /></span>
              <span>正解</span><small>建议保留的解法</small>
            </h2>
            <MarkdownRenderer :content="item.correct" />
          </section>

          <section
            v-if="hasContent(item.reason)"
            class="kp-content-section kp-content-section--reason"
            aria-labelledby="kp-reason-heading"
          >
            <h2 id="kp-reason-heading">
              <span class="kp-section-icon" aria-hidden="true"><Lightbulb :size="15" /></span>
              <span>错因归纳</span><small>下一次先检查这里</small>
            </h2>
            <MarkdownRenderer :content="item.reason" />
          </section>

          <section
            v-if="safeTags(item.tags).length || safeTags(item.reason_tags).length"
            class="kp-detail-tags-section"
            aria-labelledby="kp-tags-heading"
          >
            <h2 id="kp-tags-heading">
              <span class="kp-section-icon" aria-hidden="true"><Tag :size="15" /></span>
              <span>相关标签</span>
            </h2>
            <div class="kp-detail-tags">
              <button
                v-for="tag in safeTags(item.tags)"
                :key="`topic-${tag}`"
                type="button"
                class="kp-tag"
                :aria-label="`按题目标签 ${tag} 筛选`"
                @click="$emit('tag', '题目标签', tag)"
              >
                {{ tag }}
              </button>
              <button
                v-for="tag in safeTags(item.reason_tags)"
                :key="`reason-${tag}`"
                type="button"
                class="kp-tag kp-tag--reason"
                :aria-label="`按错因标签 ${tag} 筛选`"
                @click="$emit('tag', '错因标签', tag)"
              >
                {{ tag }}
              </button>
            </div>
          </section>
        </div>
      </div>
    </template>

    <div v-else-if="notFound" class="kp-detail-state" data-testid="preview-detail-not-found">
      <span class="kp-state-icon" aria-hidden="true"><FileQuestion :size="25" /></span>
      <h1>没有找到这则错题</h1>
      <p>编号 {{ requestedId || "—" }} 可能不存在，或已被移除。</p>
      <button type="button" class="kp-state-action" @click="$emit('back')">
        <ArrowLeft :size="16" aria-hidden="true" />返回错题目录
      </button>
    </div>

    <div v-else class="kp-detail-state" data-testid="preview-detail-empty">
      <span class="kp-state-icon" aria-hidden="true"><BookOpen :size="25" /></span>
      <h1>选择一则错题开始阅读</h1>
      <p>从目录中选择内容，在这里查看完整题目、订正与错因归纳。</p>
    </div>
  </article>
</template>

<style scoped>
.kp-error-detail {
  --kp-detail-ink: var(--kp-ink, #24252b);
  --kp-detail-muted: var(--kp-ink-muted, #71727c);
  --kp-detail-border: var(--kp-line, #e7e7eb);
  --kp-detail-accent: var(--kp-accent, #5367d8);
  --kp-detail-accent-soft: var(--kp-accent-wash, #eef0ff);
  --kp-detail-danger: var(--kp-danger, #a44f52);
  --kp-detail-danger-soft: #fff8f7;
  --kp-detail-danger-line: #efdad7;
  display: flex;
  min-width: 0;
  height: 100%;
  flex-direction: column;
  overflow: hidden;
  color: var(--kp-detail-ink);
  background: var(--kp-surface, #fff);
  font-family: var(--kp-font-sans, "MiSans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif);
}

.kp-detail-header {
  z-index: 1;
  flex: 0 0 auto;
  background: var(--kp-surface, #fff);
}

.kp-detail-header-inner,
.kp-detail-content,
.kp-detail-skeleton {
  width: min(840px, calc(100% - 72px));
  margin-inline: auto;
}

.kp-detail-header-inner {
  padding: clamp(34px, 5vw, 62px) 0 24px;
}

.kp-detail-heading-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.kp-detail-heading {
  min-width: 0;
}

.kp-detail-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 7px 17px;
  color: var(--kp-detail-muted);
  font-size: 11.5px;
  line-height: 1.4;
}

.kp-detail-meta span:not(:first-child) {
  position: relative;
}

.kp-detail-meta span:not(:first-child)::before {
  position: absolute;
  top: 50%;
  left: -9px;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: #c6c6cc;
  content: "";
  transform: translateY(-50%);
}

.kp-detail-id {
  color: var(--kp-detail-accent);
  font-weight: 750;
  font-variant-numeric: tabular-nums;
}

.kp-detail-heading h1 {
  max-width: 760px;
  margin: 10px 0 0;
  color: var(--kp-detail-ink);
  font-size: clamp(27px, 3vw, 38px);
  font-weight: 735;
  line-height: 1.22;
  letter-spacing: -.04em;
  overflow-wrap: anywhere;
}

.kp-readonly-chip {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border: 1px solid #dddde4;
  border-radius: 6px;
  color: var(--kp-detail-accent);
  background: #fafaff;
  font-size: 11px;
  font-weight: 680;
  white-space: nowrap;
}

.kp-readonly-chip i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--kp-detail-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--kp-detail-accent) 12%, transparent);
}

.kp-mobile-back {
  display: none;
  align-items: center;
  gap: 6px;
  margin: -3px 0 17px;
  padding: 6px 0;
  border: 0;
  color: var(--kp-detail-accent);
  background: transparent;
  font: inherit;
  font-size: 13px;
  font-weight: 650;
  cursor: pointer;
}

.kp-detail-scroll {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-color: #bdc9c5 transparent;
  scrollbar-width: thin;
}

.kp-detail-content {
  padding-block: 0 88px;
}

.kp-content-section {
  position: relative;
  padding: 30px 0 34px;
  font-size: 15px;
}

.kp-content-section + .kp-content-section,
.kp-detail-tags-section {
  margin-top: 10px;
}

.kp-content-section h2,
.kp-detail-tags-section h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 16px;
  color: var(--kp-detail-ink);
  font-size: 14px;
  font-weight: 730;
  line-height: 1.4;
}

.kp-content-section h2 small {
  margin-left: 2px;
  color: var(--kp-detail-muted);
  font-size: 11.5px;
  font-weight: 450;
}

.kp-section-icon {
  display: inline-grid;
  width: 25px;
  height: 25px;
  place-items: center;
  border-radius: 7px;
  color: var(--kp-detail-accent);
  background: var(--kp-detail-accent-soft);
}

.kp-content-section--question {
  border-top: 1px solid var(--kp-detail-border);
  border-bottom: 1px solid var(--kp-detail-border);
}

.kp-content-section--wrong,
.kp-content-section--correct,
.kp-content-section--reason {
  padding: 23px 24px 26px;
  border: 1px solid transparent;
  border-left-width: 3px;
  border-radius: 9px;
}

.kp-content-section--wrong {
  border-color: var(--kp-detail-danger-line);
  background: var(--kp-detail-danger-soft);
}

.kp-content-section--wrong h2 {
  color: var(--kp-detail-danger);
}

.kp-content-section--wrong .kp-section-icon {
  color: var(--kp-detail-danger);
  background: #f9e8e6;
}

.kp-content-section--correct {
  border-color: #dfe2f8;
  background: #f8f8ff;
}

.kp-content-section--correct h2 {
  color: #4459c4;
}

.kp-content-section--reason {
  border-color: #eee1c8;
  background: #fffcf5;
}

.kp-content-section--reason h2 {
  color: #866329;
}

.kp-content-section--reason .kp-section-icon {
  color: #946f2f;
  background: #f8edcf;
}

.kp-detail-tags-section {
  padding: 27px 0 0;
  border-top: 1px solid var(--kp-detail-border);
}

.kp-detail-tags-section h2 {
  margin-bottom: 12px;
}

.kp-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.kp-tag {
  max-width: 100%;
  padding: 6px 10px;
  overflow: hidden;
  border: 1px solid var(--kp-detail-border);
  border-radius: 6px;
  color: #5f606a;
  background: #fafafa;
  font: inherit;
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 160ms ease, color 160ms ease, background-color 160ms ease;
}

.kp-tag:hover {
  border-color: #b8beec;
  color: var(--kp-detail-accent);
  background: var(--kp-detail-accent-soft);
}

.kp-tag--reason {
  border-color: var(--kp-detail-danger-line);
  color: #865d58;
  background: var(--kp-detail-danger-soft);
}

.kp-mobile-back:focus-visible,
.kp-tag:focus-visible,
.kp-state-action:focus-visible {
  outline: 3px solid color-mix(in srgb, var(--kp-detail-accent) 30%, transparent);
  outline-offset: 3px;
}

.kp-detail-state {
  display: grid;
  min-height: 100%;
  padding: 48px 24px;
  place-items: center;
  align-content: center;
  color: var(--kp-detail-muted);
  text-align: center;
}

.kp-state-icon {
  display: grid;
  width: 50px;
  height: 50px;
  margin-bottom: 17px;
  place-items: center;
  border-radius: 15px;
  color: var(--kp-detail-accent);
  background: var(--kp-detail-accent-soft);
}

.kp-detail-state h1,
.kp-detail-state p {
  margin: 0;
}

.kp-detail-state h1 {
  color: var(--kp-detail-ink);
  font-size: 18px;
  font-weight: 700;
}

.kp-detail-state p {
  max-width: 430px;
  margin-top: 9px;
  font-size: 14px;
  line-height: 1.65;
}

.kp-state-action {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  margin-top: 20px;
  padding: 9px 14px;
  border: 1px solid #b8d4ce;
  border-radius: 9px;
  color: var(--kp-detail-accent);
  background: #fff;
  font: inherit;
  font-size: 13px;
  font-weight: 650;
  cursor: pointer;
}

.kp-detail-loading {
  min-height: 0;
  overflow: hidden;
}

.kp-detail-skeleton {
  padding-top: 36px;
}

.kp-skeleton-section {
  margin-top: 48px;
  padding-top: 26px;
  border-top: 1px solid var(--kp-detail-border);
}

.kp-skeleton-line {
  display: block;
  width: 100%;
  height: 12px;
  border-radius: 999px;
  background: linear-gradient(90deg, #edf2f0 20%, #f7f9f8 45%, #edf2f0 70%);
  background-size: 220% 100%;
  animation: kp-detail-shimmer 1.4s ease-in-out infinite;
}

.kp-skeleton-line + .kp-skeleton-line {
  margin-top: 13px;
}

.kp-skeleton-line--meta { width: 41%; height: 9px; }
.kp-skeleton-line--heading { width: 66%; height: 25px; margin-top: 14px; }
.kp-skeleton-line--subheading { width: 31%; height: 9px; margin-top: 14px; }
.kp-skeleton-line--label { width: 14%; height: 11px; margin-bottom: 20px; }
.kp-skeleton-line--body { width: 94%; }
.kp-skeleton-line--short { width: 62%; }

.kp-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.kp-error-detail :deep(.markdown-body) {
  max-width: 100%;
  color: var(--kp-detail-ink);
  font-family: inherit;
  font-size: 15px;
  line-height: 1.82;
  overflow-wrap: anywhere;
}

.kp-error-detail :deep(.markdown-body > :first-child) { margin-top: 0; }
.kp-error-detail :deep(.markdown-body > :last-child) { margin-bottom: 0; }

.kp-error-detail :deep(.markdown-body p),
.kp-error-detail :deep(.markdown-body ul),
.kp-error-detail :deep(.markdown-body ol) {
  margin: 0 0 1em;
}

.kp-error-detail :deep(.markdown-body ul),
.kp-error-detail :deep(.markdown-body ol) {
  padding-left: 1.55em;
}

.kp-error-detail :deep(.markdown-body li + li) { margin-top: 0.35em; }

.kp-error-detail :deep(.markdown-body h1),
.kp-error-detail :deep(.markdown-body h2),
.kp-error-detail :deep(.markdown-body h3),
.kp-error-detail :deep(.markdown-body h4),
.kp-error-detail :deep(.markdown-body h5),
.kp-error-detail :deep(.markdown-body h6) {
  margin: 1.45em 0 0.62em;
  color: var(--kp-detail-ink);
  font-family: inherit;
  font-weight: 720;
  line-height: 1.4;
  letter-spacing: -0.015em;
}

.kp-error-detail :deep(.markdown-body h1) { font-size: 1.48em; }
.kp-error-detail :deep(.markdown-body h2) { font-size: 1.3em; }
.kp-error-detail :deep(.markdown-body h3) { font-size: 1.16em; }
.kp-error-detail :deep(.markdown-body h4),
.kp-error-detail :deep(.markdown-body h5),
.kp-error-detail :deep(.markdown-body h6) { font-size: 1em; }

.kp-error-detail :deep(.markdown-body a) {
  color: var(--kp-detail-accent);
  text-decoration-thickness: 1px;
  text-underline-offset: 3px;
}

.kp-error-detail :deep(.markdown-body strong) { color: #202126; font-weight: 750; }

.kp-error-detail :deep(.markdown-body mark) {
  padding: 0.05em 0.2em;
  border-radius: 3px;
  color: inherit;
  background: #fff0b8;
}

.kp-error-detail :deep(.markdown-body blockquote) {
  margin: 1.2em 0;
  padding: 0.15em 0 0.15em 1em;
  border-left: 3px solid #aeb7ec;
  color: #5e6070;
  background: transparent;
}

.kp-error-detail :deep(.markdown-body code) {
  padding: 0.16em 0.38em;
  border-radius: 5px;
  color: #4857ad;
  background: #f0f1f8;
  font-family: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
  font-size: 0.9em;
}

.kp-error-detail :deep(.markdown-body pre) {
  max-width: 100%;
  margin: 1.2em 0;
  padding: 16px 18px;
  overflow-x: auto;
  border: 1px solid #3b3d4a;
  border-radius: 10px;
  color: #ececf3;
  background: #292b35;
  line-height: 1.65;
  overscroll-behavior-inline: contain;
}

.kp-error-detail :deep(.markdown-body pre code) {
  padding: 0;
  color: inherit;
  background: transparent;
  font-size: 0.88em;
}

.kp-error-detail :deep(.markdown-body table) {
  display: block;
  width: max-content;
  max-width: 100%;
  margin: 1.3em 0;
  overflow-x: auto;
  border-spacing: 0;
  border-collapse: separate;
  border: 1px solid var(--kp-detail-border);
  border-radius: 9px;
  font-size: 0.93em;
  white-space: nowrap;
  overscroll-behavior-inline: contain;
}

.kp-error-detail :deep(.markdown-body th),
.kp-error-detail :deep(.markdown-body td) {
  padding: 9px 13px;
  border-right: 1px solid var(--kp-detail-border);
  border-bottom: 1px solid var(--kp-detail-border);
  text-align: left;
}

.kp-error-detail :deep(.markdown-body th) {
  color: #44454e;
  background: #f5f5f7;
  font-weight: 700;
}

.kp-error-detail :deep(.markdown-body tr:last-child td) { border-bottom: 0; }
.kp-error-detail :deep(.markdown-body th:last-child),
.kp-error-detail :deep(.markdown-body td:last-child) { border-right: 0; }

.kp-error-detail :deep(.markdown-body img) {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 1.25em auto;
  border-radius: 10px;
}

.kp-error-detail :deep(.markdown-body hr) {
  margin: 2em 0;
  border: 0;
  border-top: 1px solid var(--kp-detail-border);
}

.kp-error-detail :deep(.markdown-body .katex) {
  color: inherit;
  font-size: 1.04em;
}

.kp-error-detail :deep(.markdown-body .katex-display) {
  max-width: 100%;
  margin: 1.2em 0;
  padding: 4px 0;
  overflow-x: auto;
  overflow-y: hidden;
  text-align: left;
  overscroll-behavior-inline: contain;
}

@keyframes kp-detail-shimmer {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

@media (max-width: 920px) {
  .kp-detail-header-inner,
  .kp-detail-content,
  .kp-detail-skeleton {
    width: calc(100% - 48px);
    margin-inline: auto;
  }
}

@media (max-width: 767px) {
  .kp-detail-header-inner,
  .kp-detail-content,
  .kp-detail-skeleton {
    width: calc(100% - 32px);
  }

  .kp-detail-header-inner { padding-top: 18px; }
  .kp-mobile-back { display: inline-flex; }
  .kp-detail-heading-row { gap: 12px; }
  .kp-readonly-chip { padding: 5px 8px; font-size: 11px; }
  .kp-detail-meta { gap: 6px 13px; }
  .kp-detail-meta span:not(:first-child)::before { left: -7px; }
  .kp-detail-heading h1 { font-size: 25px; }
  .kp-detail-content { padding-bottom: 42px; }
  .kp-content-section { padding-block: 25px 28px; }
  .kp-content-section--wrong,
  .kp-content-section--correct,
  .kp-content-section--reason {
    padding: 20px 17px 22px;
    border-radius: 8px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .kp-tag { transition: none; }
  .kp-skeleton-line { animation: none; }
}
</style>
