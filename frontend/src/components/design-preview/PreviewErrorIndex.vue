<script setup>
import { ChevronRight, Clock3, SearchX } from "lucide-vue-next"
import { isDue, reviewLabel } from "../../composables/useErrorLibrary.js"

defineProps({
  items: { type: Array, default: () => [] },
  selectedId: Number,
  loading: Boolean,
  today: String,
})

defineEmits(["select", "tag"])

const missingTitles = new Set(["", "暂无", "未记录"])

function plainText(value) {
  if (typeof value !== "string") return ""
  return value
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/<[^>]+>/g, " ")
    .replace(/\\infty/g, "∞")
    .replace(/\\(?:leqslant|leq)/g, "≤")
    .replace(/\\(?:geqslant|geq)/g, "≥")
    .replace(/\\neq/g, "≠")
    .replace(/\\to/g, "→")
    .replace(/\\[a-zA-Z]+/g, " ")
    .replace(/[$ {}]/g, " ")
    .replace(/(^|\s)[#>*_~`-]+(?=\S)/g, "$1")
    .replace(/[\\*_~`]/g, "")
    .replace(/\s+/g, " ")
    .trim()
}

function itemTitle(item) {
  const title = typeof item?.title === "string" ? item.title.trim() : ""
  if (!missingTitles.has(title)) return plainText(title)

  const questionLine = typeof item?.question === "string"
    ? item.question.split(/\r?\n/).find((line) => line.trim()) || ""
    : ""
  const question = plainText(questionLine)
  if (!question) return `未命名错题 #${item?.id ?? "—"}`
  return question.length > 72 ? `${question.slice(0, 72)}…` : question
}

function safeTags(value) {
  if (!Array.isArray(value)) return []
  return value.map((tag) => String(tag ?? "").trim()).filter(Boolean)
}
</script>

<template>
  <section class="kp-error-index" aria-labelledby="kp-error-index-title" data-testid="preview-error-index">
    <header class="kp-index-header">
      <div>
        <h2 id="kp-error-index-title">全部错题</h2>
        <p>按记录浏览</p>
      </div>
      <span class="kp-index-count" data-testid="preview-error-count">{{ items.length }} 则</span>
    </header>

    <div v-if="loading" class="kp-index-skeleton" aria-busy="true" data-testid="preview-index-loading">
      <span class="kp-sr-only" role="status">正在加载错题目录</span>
      <div v-for="n in 6" :key="n" class="kp-skeleton-row" aria-hidden="true">
        <span class="kp-skeleton-line kp-skeleton-line--meta"></span>
        <span class="kp-skeleton-line kp-skeleton-line--title"></span>
        <span class="kp-skeleton-line kp-skeleton-line--short"></span>
      </div>
    </div>

    <ul v-else-if="items.length" class="kp-index-list" data-testid="preview-error-list">
      <li
        v-for="item in items"
        :key="item?.id"
        class="kp-index-item"
        :class="{ 'is-selected': selectedId === item?.id }"
        :data-testid="`preview-error-item-${item?.id}`"
      >
        <button
          type="button"
          class="kp-index-select"
          :aria-current="selectedId === item?.id ? 'true' : undefined"
          :aria-label="`查看错题 ${item?.id}：${itemTitle(item)}`"
          @click="$emit('select', item)"
        >
          <span class="kp-item-meta">
            <span class="kp-item-id">#{{ item?.id }}</span>
            <span class="kp-item-subject">{{ item?.subject || "未分类" }}</span>
            <span class="kp-item-review" :class="{ 'is-due': isDue(item || {}, today) }">
              <Clock3 :size="11" aria-hidden="true" />
              {{ reviewLabel(item || {}, today) }}
            </span>
          </span>
          <span class="kp-item-title">{{ itemTitle(item) }}</span>
          <ChevronRight class="kp-item-arrow" :size="15" aria-hidden="true" />
        </button>

        <div
          v-if="safeTags(item?.tags).length || safeTags(item?.reason_tags).length"
          class="kp-item-tags"
          aria-label="错题标签"
        >
          <button
            v-for="tag in safeTags(item?.tags)"
            :key="`topic-${tag}`"
            type="button"
            class="kp-tag"
            :aria-label="`按题目标签 ${tag} 筛选`"
            @click="$emit('tag', '题目标签', tag)"
          >
            {{ tag }}
          </button>
          <button
            v-for="tag in safeTags(item?.reason_tags)"
            :key="`reason-${tag}`"
            type="button"
            class="kp-tag kp-tag--reason"
            :aria-label="`按错因标签 ${tag} 筛选`"
            @click="$emit('tag', '错因标签', tag)"
          >
            {{ tag }}
          </button>
        </div>
      </li>
    </ul>

    <div v-else class="kp-index-empty" data-testid="preview-index-empty">
      <span class="kp-empty-icon" aria-hidden="true"><SearchX :size="22" /></span>
      <h3>没有找到错题</h3>
      <p>换一个科目、关键词或标签试试。</p>
    </div>
  </section>
</template>

<style scoped>
.kp-error-index {
  --kp-index-ink: var(--kp-ink, #222328);
  --kp-index-muted: var(--kp-ink-muted, #7c7f89);
  --kp-index-soft: var(--kp-surface-muted, #f6f6f8);
  --kp-index-border: var(--kp-line, #e0e1e6);
  --kp-index-accent: var(--kp-accent, #5964cf);
  --kp-index-accent-soft: var(--kp-accent-wash, #ebedff);
  display: flex;
  min-width: 0;
  height: 100%;
  flex-direction: column;
  overflow: hidden;
  color: var(--kp-index-ink);
  background: var(--kp-surface, #fff);
  font-family: var(--kp-font-sans, "MiSans", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif);
}

.kp-index-header {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 49px;
  padding: 8px 13px;
  border-bottom: 1px solid var(--kp-index-border);
  background: color-mix(in srgb, var(--kp-surface, #fff) 97%, var(--kp-index-soft));
}

.kp-index-header h2,
.kp-index-header p,
.kp-index-empty h3,
.kp-index-empty p {
  margin: 0;
}

.kp-index-header h2 {
  color: var(--kp-index-ink);
  font-size: 12.5px;
  font-weight: 720;
  letter-spacing: -.01em;
}

.kp-index-header p {
  margin-top: 1px;
  color: var(--kp-index-muted);
  font-size: 9.5px;
}

.kp-index-count {
  flex: 0 0 auto;
  color: var(--kp-index-muted);
  font-size: 10px;
  font-weight: 620;
  font-variant-numeric: tabular-nums;
}

.kp-index-list {
  min-height: 0;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  list-style: none;
  scrollbar-color: #c9cad1 transparent;
  scrollbar-width: thin;
}

.kp-index-item {
  position: relative;
  border-bottom: 1px solid color-mix(in srgb, var(--kp-index-border) 76%, transparent);
  background: transparent;
  transition: background-color var(--kp-motion-fast, 160ms) ease;
}

.kp-index-item::before {
  position: absolute;
  z-index: 1;
  top: 8px;
  bottom: 8px;
  left: 0;
  width: 2px;
  border-radius: 0 2px 2px 0;
  background: var(--kp-index-accent);
  content: "";
  opacity: 0;
  transform: scaleY(.4);
  transition: opacity var(--kp-motion-fast, 160ms) ease, transform var(--kp-motion-fast, 160ms) ease;
}

.kp-index-item:hover {
  background: color-mix(in srgb, var(--kp-index-soft) 82%, transparent);
}

.kp-index-item.is-selected {
  background: color-mix(in srgb, var(--kp-index-accent-soft) 82%, var(--kp-surface, #fff));
}

.kp-index-item.is-selected::before {
  opacity: 1;
  transform: scaleY(1);
}

.kp-index-select {
  position: relative;
  display: block;
  width: 100%;
  padding: 10px 31px 5px 14px;
  border: 0;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.kp-index-select:focus-visible,
.kp-tag:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--kp-index-accent) 42%, transparent);
  outline-offset: -2px;
}

.kp-item-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--kp-index-muted);
  font-size: 9.5px;
  line-height: 1.25;
}

.kp-item-id {
  color: var(--kp-index-muted);
  font-weight: 650;
  font-variant-numeric: tabular-nums;
}

.kp-item-subject {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kp-item-review {
  display: inline-flex;
  min-width: 0;
  align-items: center;
  gap: 3px;
  margin-left: auto;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kp-item-review.is-due {
  color: var(--kp-danger, #ad5654);
}

.kp-item-title {
  display: -webkit-box;
  margin-top: 5px;
  overflow: hidden;
  color: var(--kp-index-ink);
  font-size: 12.5px;
  font-weight: 620;
  line-height: 1.42;
  overflow-wrap: anywhere;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.kp-item-arrow {
  position: absolute;
  top: 50%;
  right: 10px;
  color: var(--kp-index-muted);
  opacity: .42;
  transform: translateY(-50%);
  transition: color var(--kp-motion-fast, 160ms) ease, opacity var(--kp-motion-fast, 160ms) ease, transform var(--kp-motion-fast, 160ms) ease;
}

.kp-index-item.is-selected .kp-item-arrow,
.kp-index-select:hover .kp-item-arrow {
  color: var(--kp-index-accent);
  opacity: 1;
  transform: translate(2px, -50%);
}

.kp-item-tags {
  display: flex;
  gap: 3px;
  padding: 0 14px 9px;
  overflow-x: auto;
  overscroll-behavior-inline: contain;
  scrollbar-width: none;
}

.kp-item-tags::-webkit-scrollbar { display: none; }

.kp-tag {
  max-width: 132px;
  flex: 0 0 auto;
  padding: 2px 4px;
  overflow: hidden;
  border: 0;
  border-radius: 4px;
  color: var(--kp-index-muted);
  background: transparent;
  font: inherit;
  font-size: 9px;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
  transition: color var(--kp-motion-fast, 160ms) ease, background-color var(--kp-motion-fast, 160ms) ease;
}

.kp-tag::before { color: var(--kp-index-accent); content: "#"; opacity: .62; }

.kp-tag:hover {
  color: var(--kp-index-accent);
  background: var(--kp-index-accent-soft);
}

.kp-tag--reason {
  color: color-mix(in srgb, var(--kp-danger, #ad5654) 82%, var(--kp-index-muted));
  background: transparent;
}

.kp-tag--reason::before { color: var(--kp-danger, #ad5654); }
.kp-tag--reason:hover { color: var(--kp-danger, #ad5654); background: var(--kp-danger-wash, #fbf0ef); }

.kp-index-skeleton {
  min-height: 0;
  overflow: hidden;
}

.kp-skeleton-row {
  padding: 13px 14px 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--kp-index-border) 76%, transparent);
}

.kp-skeleton-line {
  display: block;
  height: 7px;
  border-radius: 999px;
  background: linear-gradient(90deg, #ececf0 20%, #f7f7f9 45%, #ececf0 70%);
  background-size: 220% 100%;
  animation: kp-index-shimmer 1.4s ease-in-out infinite;
}

.kp-skeleton-line + .kp-skeleton-line {
  margin-top: 8px;
}

.kp-skeleton-line--meta { width: 58%; }
.kp-skeleton-line--title { width: 92%; height: 11px; }
.kp-skeleton-line--short { width: 36%; height: 6px; }

.kp-index-empty {
  display: grid;
  place-items: center;
  align-content: center;
  min-height: 240px;
  padding: 34px 20px;
  color: var(--kp-index-muted);
  text-align: center;
}

.kp-empty-icon {
  display: grid;
  width: 38px;
  height: 38px;
  margin-bottom: 12px;
  place-items: center;
  border-radius: 10px;
  color: var(--kp-index-accent);
  background: var(--kp-index-accent-soft);
}

.kp-index-empty h3 {
  color: var(--kp-index-ink);
  font-size: 13px;
}

.kp-index-empty p {
  margin-top: 5px;
  font-size: 11px;
  line-height: 1.6;
}

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

@keyframes kp-index-shimmer {
  from { background-position: 100% 0; }
  to { background-position: -100% 0; }
}

@media (max-width: 767px) {
  .kp-index-header { min-height: 51px; padding-inline: 14px; }
  .kp-index-select { padding: 12px 34px 6px 15px; }
  .kp-item-title { font-size: 13px; }
  .kp-item-tags { padding-right: 15px; padding-bottom: 10px; padding-left: 15px; }
}

@media (prefers-reduced-motion: reduce) {
  .kp-index-item,
  .kp-index-item::before,
  .kp-item-arrow,
  .kp-tag {
    transition: none;
  }

  .kp-skeleton-line {
    animation: none;
  }
}
</style>
