<script setup>
import { CalendarClock, ChevronRight, FileText, Tags } from "lucide-vue-next"
import { renderMd } from "../../utils/markdown.js"
import { hasContent, isDue, reviewLabel, subjectColor } from "../../composables/useErrorLibrary.js"
import EmptyState from "../ui/EmptyState.vue"

defineProps({ items: { type: Array, default: () => [] }, selectedId: Number, loading: Boolean, today: String })
defineEmits(["select", "tag"])

function safeTags(value) {
  return Array.isArray(value) ? value.filter((item) => typeof item === "string" && item.trim()) : []
}

function displaySubject(item) {
  const subject = typeof item?.subject === "string" ? item.subject.trim() : ""
  return subject && subject !== "暂无" && subject !== "未记录" ? subject : "未分类"
}

function displayTitle(item) {
  const title = typeof item?.title === "string" ? item.title.trim() : ""
  if (title && title !== "暂无" && title !== "未记录") return title
  return `错题 #${item?.id ?? "—"}`
}

function displayExcerpt(item) {
  if (hasContent(item?.question)) return item.question
  if (hasContent(item?.reason)) return item.reason
  return "题面尚未记录"
}

function displayDate(item) {
  const updated = typeof item?.updated === "string" ? item.updated : ""
  const created = typeof item?.created === "string" ? item.created : ""
  return (updated || created || "").slice(0, 10) || "日期未记录"
}
</script>

<template>
  <section class="error-index error-index--cards" aria-labelledby="formal-error-card-title" data-testid="formal-error-card-library">
    <div class="error-index__rule">
      <span id="formal-error-card-title">你的错题卡片</span>
      <small>{{ items.length }} 张</small>
    </div>
    <div v-if="loading" class="list-skeleton error-card-skeleton" aria-busy="true" data-testid="formal-error-card-loading">
      <span v-for="n in 8" :key="n"></span>
    </div>
    <div v-else-if="items.length" class="error-index__list error-index__list--cards" data-testid="formal-error-card-grid">
      <article
        v-for="item in items" :key="item.id" class="error-card" :class="{ active: selectedId === item.id }"
        :style="{ '--subject-color': subjectColor(displaySubject(item)) }"
        :data-testid="`formal-error-card-${item.id}`"
        role="button" tabindex="0" @click="$emit('select', item)" @keydown.enter.prevent="$emit('select', item)"
      >
        <div class="error-card__flip">
          <div class="error-card__side error-card__side--front">
            <div class="error-card__top">
              <span class="error-card__icon" aria-hidden="true"><FileText :size="20" /></span>
              <span class="error-card__number">#{{ item.id }}</span>
            </div>

            <div class="error-card__meta">
              <span class="subject-dot">{{ displaySubject(item) }}</span>
              <span class="due-label" :class="{ due: isDue(item, today) }">
                <CalendarClock :size="12" />{{ reviewLabel(item, today) }}
              </span>
            </div>

            <h2 class="error-card__title">{{ displayTitle(item) }}</h2>

            <div v-if="safeTags(item.tags).length || safeTags(item.reason_tags).length" class="tag-row">
              <Tags :size="13" aria-hidden="true" />
              <button v-for="tag in safeTags(item.tags)" :key="tag" type="button" class="tag-pill" @click.stop="$emit('tag', '题目标签', tag)">{{ tag }}</button>
              <button v-for="tag in safeTags(item.reason_tags)" :key="tag" type="button" class="tag-pill tag-pill--reason" @click.stop="$emit('tag', '错因标签', tag)">{{ tag }}</button>
            </div>

            <footer class="error-card__footer">
              <span>{{ displayDate(item) }}</span>
              <span class="error-card__arrow" aria-hidden="true"><ChevronRight :size="17" /></span>
            </footer>
          </div>

          <div class="error-card__side error-card__side--back" data-testid="formal-error-card-preview">
            <div class="error-card__back-head">
              <span class="error-card__back-label">题面预览</span>
              <span class="error-card__number">#{{ item.id }}</span>
            </div>
            <div class="error-card__excerpt" v-html="renderMd(displayExcerpt(item))"></div>
            <div v-if="safeTags(item.tags).length || safeTags(item.reason_tags).length" class="tag-row tag-row--back">
              <Tags :size="13" aria-hidden="true" />
              <button v-for="tag in safeTags(item.tags)" :key="tag" type="button" class="tag-pill" @click.stop="$emit('tag', '题目标签', tag)">{{ tag }}</button>
              <button v-for="tag in safeTags(item.reason_tags)" :key="tag" type="button" class="tag-pill tag-pill--reason" @click.stop="$emit('tag', '错因标签', tag)">{{ tag }}</button>
            </div>
            <footer class="error-card__footer">
              <span>{{ displayDate(item) }}</span>
              <span class="error-card__arrow" aria-hidden="true"><ChevronRight :size="17" /></span>
            </footer>
          </div>
        </div>
      </article>
    </div>
    <EmptyState v-else title="没有找到错题" description="调整筛选条件，或录入一道新的错题。" />
  </section>
</template>
