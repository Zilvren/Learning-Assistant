<script setup>
import { LoaderCircle, RotateCcw, Search, SlidersHorizontal } from "lucide-vue-next"

const props = defineProps({
  subjects: { type: Array, default: () => [] },
  subject: { type: String, default: "全部" },
  mode: { type: String, default: "全部" },
  keyword: { type: String, default: "" },
  loading: Boolean,
  count: { type: Number, default: 0 },
})

const emit = defineEmits(["update:subject", "update:mode", "update:keyword", "reset"])
const searchModes = ["全部", "题目", "题目标签", "错因标签"]

function valueOf(event) {
  return event.target.value
}
</script>

<template>
  <form class="kt-filter-bar" role="search" aria-label="筛选错题" data-testid="knowt-filter-bar" @submit.prevent>
    <label class="kt-filter-search">
      <Search :size="18" aria-hidden="true" />
      <span class="kt-sr-only">搜索错题</span>
      <input
        type="search"
        :value="props.keyword"
        aria-label="搜索错题"
        placeholder="搜索题目、知识点或错因"
        autocomplete="off"
        @input="emit('update:keyword', valueOf($event))"
      />
      <LoaderCircle v-if="props.loading" class="kt-filter-spinner" :size="16" aria-label="正在更新结果" />
    </label>

    <label class="kt-filter-select">
      <span>科目</span>
      <select :value="props.subject" aria-label="按科目筛选" @change="emit('update:subject', valueOf($event))">
        <option value="全部">全部科目</option>
        <option v-for="item in props.subjects" :key="item" :value="item">{{ item }}</option>
      </select>
    </label>

    <label class="kt-filter-select">
      <span>范围</span>
      <select :value="props.mode" aria-label="选择搜索范围" @change="emit('update:mode', valueOf($event))">
        <option v-for="item in searchModes" :key="item" :value="item">{{ item === "全部" ? "全部内容" : item }}</option>
      </select>
    </label>

    <button
      type="button"
      class="kt-filter-reset"
      aria-label="重置筛选"
      :disabled="props.subject === '全部' && props.mode === '全部' && !props.keyword"
      @click="emit('reset')"
    >
      <RotateCcw :size="15" aria-hidden="true" />
      <span>重置</span>
    </button>

    <div class="kt-filter-count" aria-live="polite">
      <SlidersHorizontal :size="15" aria-hidden="true" />
      <span><strong>{{ props.count }}</strong> 条结果</span>
    </div>
  </form>
</template>

<style scoped>
.kt-filter-bar {
  display: grid;
  grid-template-columns: minmax(260px, 1fr) 170px 170px auto auto;
  align-items: center;
  gap: 10px;
  margin-top: 22px;
  padding: 12px;
  border: 1px solid var(--kt-line);
  border-radius: 18px;
  background: var(--kt-surface);
  box-shadow: var(--kt-shadow-sm);
}

.kt-filter-search,
.kt-filter-select {
  min-width: 0;
  height: 44px;
  border: 1px solid var(--kt-line);
  border-radius: 12px;
  background: var(--kt-surface-soft);
  transition: border-color 170ms ease, box-shadow 170ms ease, background-color 170ms ease;
}

.kt-filter-search:focus-within,
.kt-filter-select:focus-within {
  border-color: var(--kt-accent);
  background: #fff;
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--kt-accent) 14%, transparent);
}

.kt-filter-search {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 0 13px;
  color: var(--kt-ink-muted);
}

.kt-filter-search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  color: var(--kt-ink);
  background: transparent;
  font: inherit;
  font-size: 13px;
}

.kt-filter-search input::placeholder { color: var(--kt-ink-faint); }
.kt-filter-search input::-webkit-search-cancel-button { display: none; }

.kt-filter-spinner {
  flex: 0 0 auto;
  color: var(--kt-accent-strong);
  animation: kt-filter-spin 800ms linear infinite;
}

.kt-filter-select {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding-left: 12px;
}

.kt-filter-select span {
  color: var(--kt-ink-faint);
  font-size: 10px;
  font-weight: 760;
}

.kt-filter-select select {
  width: 100%;
  height: 100%;
  padding: 0 28px 0 0;
  border: 0;
  outline: 0;
  color: var(--kt-ink);
  background: transparent;
  font: inherit;
  font-size: 12.5px;
  font-weight: 680;
  cursor: pointer;
}

.kt-filter-reset {
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 0 10px;
  border: 0;
  border-radius: 10px;
  color: var(--kt-ink-secondary);
  background: transparent;
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}

.kt-filter-reset:hover:not(:disabled) { color: var(--kt-accent-strong); background: var(--kt-accent-wash); }
.kt-filter-reset:disabled { opacity: .3; cursor: default; }

.kt-filter-count {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  color: var(--kt-ink-muted);
  font-size: 11.5px;
  white-space: nowrap;
}

.kt-filter-count svg { color: var(--kt-accent-strong); }
.kt-filter-count strong { color: var(--kt-ink); font-size: 12.5px; }

.kt-sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  overflow: hidden;
  clip: rect(0 0 0 0);
  clip-path: inset(50%);
  white-space: nowrap;
  border: 0;
}

@keyframes kt-filter-spin { to { transform: rotate(360deg); } }

@media (max-width: 920px) {
  .kt-filter-bar { grid-template-columns: minmax(220px, 1fr) 150px 150px auto; }
  .kt-filter-count { grid-column: 1 / -1; justify-content: flex-start; padding-inline: 3px; }
}

@media (max-width: 640px) {
  .kt-filter-bar { grid-template-columns: 1fr 1fr auto; gap: 8px; margin-top: 16px; padding: 9px; border-radius: 15px; }
  .kt-filter-search { grid-column: 1 / -1; }
  .kt-filter-select { min-width: 0; }
  .kt-filter-reset { width: 42px; padding: 0; }
  .kt-filter-reset span { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
  .kt-filter-count { grid-column: 1 / -1; }
}

@media (prefers-reduced-motion: reduce) {
  .kt-filter-search,
  .kt-filter-select { transition: none; }
  .kt-filter-spinner { animation: none; }
}
</style>
