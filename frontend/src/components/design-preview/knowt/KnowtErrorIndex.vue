<script setup>
import {
  CalendarClock,
  ChevronRight,
  Clock3,
  Layers3,
  LoaderCircle,
  Tag,
} from "lucide-vue-next"
import MarkdownRenderer from "../../MarkdownRenderer.vue"
import { hasContent, isDue, reviewLabel } from "../../../composables/useErrorLibrary.js"

const props = defineProps({
  items: { type: Array, default: () => [] },
  selectedId: { type: Number, default: null },
  loading: Boolean,
  today: { type: String, required: true },
})

const emit = defineEmits(["select", "tag"])

function safeTags(value) {
  return Array.isArray(value) ? value.filter((item) => typeof item === "string" && item.trim()) : []
}

function displayTitle(item) {
  const title = typeof item?.title === "string" ? item.title.trim() : ""
  return title && title !== "暂无" && title !== "未记录" ? title : `错题 #${item?.id ?? "—"}`
}

function displaySubject(item) {
  const value = typeof item?.subject === "string" ? item.subject.trim() : ""
  return value && value !== "暂无" && value !== "未记录" ? value : "未分类"
}

function createdDate(item) {
  const value = typeof item?.created === "string" ? item.created.trim() : ""
  return value ? value.slice(0, 10) : "日期未记录"
}
</script>

<template>
  <div class="kt-error-index">
    <div
      v-if="props.loading"
      class="kt-list-loading"
      role="status"
      aria-live="polite"
      data-testid="knowt-list-loading"
    >
      <LoaderCircle :size="22" aria-hidden="true" />
      <span>正在整理你的错题卡组…</span>
    </div>

    <div v-else-if="!props.items.length" class="kt-list-empty" data-testid="knowt-list-empty">
      <span class="kt-list-empty__icon" aria-hidden="true"><Layers3 :size="24" /></span>
      <h3>这里还没有符合条件的错题</h3>
      <p>换一个科目或关键词，看看其他学习记录。</p>
    </div>

    <ol v-else class="kt-error-list" data-testid="knowt-error-list">
      <li
        v-for="(item, index) in props.items"
        :key="item.id"
        class="kt-error-row"
        :class="{ 'is-selected': item.id === props.selectedId }"
        :data-testid="`knowt-error-item-${item.id}`"
      >
        <span class="kt-error-row__number" aria-hidden="true">{{ String(index + 1).padStart(2, "0") }}</span>

        <div class="kt-error-row__body">
          <div class="kt-error-row__eyebrow">
            <span class="kt-subject-pill">{{ displaySubject(item) }}</span>
            <span class="kt-review-pill" :class="{ 'is-due': isDue(item, props.today) }">
              <CalendarClock :size="12" aria-hidden="true" />
              {{ reviewLabel(item, props.today) }}
            </span>
          </div>

          <button
            type="button"
            class="kt-error-row__open"
            :aria-label="`查看错题：${displayTitle(item)}`"
            @click="emit('select', item)"
          >
            <span class="kt-error-row__copy">
              <strong>{{ displayTitle(item) }}</strong>
              <span v-if="hasContent(item.question)" class="kt-error-row__question">
                <MarkdownRenderer :content="item.question" :inline="true" />
              </span>
              <span v-else class="kt-error-row__question kt-error-row__question--muted">题面尚未记录</span>
            </span>
            <span class="kt-error-row__arrow" aria-hidden="true"><ChevronRight :size="19" /></span>
          </button>

          <div class="kt-error-row__footer">
            <div class="kt-tag-row" aria-label="题目标签">
              <Tag v-if="safeTags(item.tags).length" :size="12" aria-hidden="true" />
              <button
                v-for="value in safeTags(item.tags)"
                :key="`topic-${value}`"
                type="button"
                :aria-label="`按题目标签 ${value} 筛选`"
                @click="emit('tag', '题目标签', value)"
              >
                {{ value }}
              </button>
              <button
                v-for="value in safeTags(item.reason_tags)"
                :key="`reason-${value}`"
                type="button"
                class="is-reason"
                :aria-label="`按错因标签 ${value} 筛选`"
                @click="emit('tag', '错因标签', value)"
              >
                {{ value }}
              </button>
            </div>
            <span class="kt-created"><Clock3 :size="12" aria-hidden="true" />{{ createdDate(item) }}</span>
          </div>
        </div>
      </li>
    </ol>

    <p v-if="!props.loading && props.items.length" class="kt-error-count" data-testid="knowt-error-count">
      已显示 {{ props.items.length }} 条错题记录
    </p>
  </div>
</template>

<style scoped>
.kt-error-index { min-width: 0; }

.kt-list-loading,
.kt-list-empty {
  min-height: 230px;
  display: grid;
  place-items: center;
  align-content: center;
  padding: 32px;
  border: 1px dashed var(--kt-line-strong);
  border-radius: 18px;
  color: var(--kt-ink-muted);
  background: var(--kt-surface-soft);
  text-align: center;
}

.kt-list-loading { gap: 10px; font-size: 12px; }
.kt-list-loading svg { color: var(--kt-accent); animation: kt-index-spin 850ms linear infinite; }

.kt-list-empty__icon {
  width: 52px;
  height: 52px;
  display: grid;
  place-items: center;
  margin-bottom: 12px;
  border-radius: 17px;
  color: var(--kt-accent-strong);
  background: var(--kt-accent-soft);
}

.kt-list-empty h3 { margin: 0; color: var(--kt-ink); font-size: 15px; font-weight: 770; }
.kt-list-empty p { max-width: 300px; margin: 7px 0 0; font-size: 11.5px; }

.kt-error-list { display: grid; gap: 9px; margin: 0; padding: 0; list-style: none; }

.kt-error-row {
  position: relative;
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr);
  gap: 10px;
  padding: 14px 15px 13px 12px;
  border: 1px solid var(--kt-line);
  border-radius: 17px;
  background: var(--kt-surface);
  transition: border-color 180ms ease, box-shadow 180ms ease, transform 180ms ease;
}

.kt-error-row:hover {
  border-color: color-mix(in srgb, var(--kt-accent) 32%, var(--kt-line));
  box-shadow: 0 10px 27px rgba(25, 73, 62, .075);
  transform: translateY(-1px);
}

.kt-error-row.is-selected { border-color: color-mix(in srgb, var(--kt-accent) 46%, var(--kt-line)); }

.kt-error-row__number {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 11px;
  color: var(--kt-accent-strong);
  background: var(--kt-accent-soft);
  font-size: 10px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.kt-error-row__body { min-width: 0; }

.kt-error-row__eyebrow,
.kt-error-row__footer,
.kt-tag-row,
.kt-created {
  display: flex;
  align-items: center;
}

.kt-error-row__eyebrow { flex-wrap: wrap; gap: 6px; min-height: 20px; }

.kt-subject-pill,
.kt-review-pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 7px;
  border-radius: 999px;
  font-size: 9.5px;
  font-weight: 730;
  line-height: 1.25;
}

.kt-subject-pill { color: #5062ad; background: #eef1ff; }
.kt-review-pill { color: #39756e; background: #ecf8f5; }
.kt-review-pill.is-due { color: #b35b64; background: #fff0f2; }

.kt-error-row__open {
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 16px;
  margin-top: 8px;
  padding: 0;
  border: 0;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.kt-error-row__copy { min-width: 0; display: grid; gap: 5px; }
.kt-error-row__copy > strong { overflow: hidden; font-size: 14px; font-weight: 770; letter-spacing: -.015em; text-overflow: ellipsis; white-space: nowrap; }

.kt-error-row__question {
  min-width: 0;
  display: block;
  overflow: hidden;
  color: var(--kt-ink-muted);
  font-size: 11.5px;
  line-height: 1.55;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kt-error-row__question :deep(.markdown-body),
.kt-error-row__question :deep(.markdown-body *) { display: inline; margin: 0; font: inherit; line-height: inherit; }
.kt-error-row__question :deep(.katex-display) { display: inline; margin: 0; }
.kt-error-row__question--muted { color: var(--kt-ink-faint); }

.kt-error-row__arrow {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 11px;
  color: var(--kt-ink-faint);
  background: var(--kt-surface-soft);
  transition: color 180ms ease, background-color 180ms ease, transform 180ms ease;
}

.kt-error-row__open:hover .kt-error-row__arrow { color: var(--kt-accent-strong); background: var(--kt-accent-soft); transform: translateX(2px); }

.kt-error-row__footer { justify-content: space-between; gap: 14px; margin-top: 10px; }
.kt-tag-row { min-width: 0; flex-wrap: wrap; gap: 5px; color: var(--kt-ink-faint); }

.kt-tag-row button {
  max-width: 150px;
  overflow: hidden;
  padding: 3px 7px;
  border: 0;
  border-radius: 7px;
  color: #3c756f;
  background: #edf8f5;
  font: inherit;
  font-size: 9.5px;
  font-weight: 680;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
}

.kt-tag-row button:hover { color: var(--kt-accent-strong); background: #dff2ed; }
.kt-tag-row button.is-reason { color: #a35d65; background: #fff1f3; }
.kt-tag-row button.is-reason:hover { color: #8f4650; background: #fbe4e7; }
.kt-created { flex: none; gap: 4px; color: var(--kt-ink-faint); font-size: 9.5px; white-space: nowrap; }

.kt-error-count { margin: 13px 0 0; color: var(--kt-ink-faint); font-size: 10.5px; text-align: center; }

@keyframes kt-index-spin { to { transform: rotate(360deg); } }

@media (max-width: 560px) {
  .kt-error-row { grid-template-columns: 32px minmax(0, 1fr); gap: 7px; padding: 12px 10px 11px 9px; border-radius: 14px; }
  .kt-error-row__number { width: 29px; height: 29px; border-radius: 9px; }
  .kt-error-row__open { gap: 8px; }
  .kt-error-row__copy > strong { font-size: 13px; }
  .kt-error-row__arrow { width: 30px; height: 30px; }
  .kt-error-row__footer { align-items: flex-start; flex-direction: column; gap: 7px; }
  .kt-created { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .kt-error-row,
  .kt-error-row__arrow { transition: none; }
  .kt-list-loading svg { animation: none; }
}
</style>
