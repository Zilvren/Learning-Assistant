<script setup>
import { ListFilter, LoaderCircle, RotateCcw, Search } from "lucide-vue-next"

const props = defineProps({
  subjects: { type: Array, default: () => [] },
  subject: { type: String, default: "全部" },
  mode: { type: String, default: "全部" },
  keyword: { type: String, default: "" },
  loading: Boolean,
  count: { type: Number, default: 0 },
  dueCount: { type: Number, default: 0 },
})

const emit = defineEmits(["update:subject", "update:mode", "update:keyword", "reset"])
const searchModes = ["全部", "题目", "题目标签", "错因标签"]

function valueOf(event) {
  return event.target.value
}
</script>

<template>
  <section class="kp-filter-bar" aria-label="错题筛选">
    <label class="kp-filter-bar__search">
      <Search :size="18" aria-hidden="true" />
      <span class="kp-sr-only">搜索错题</span>
      <input
        type="search"
        :value="props.keyword"
        placeholder="搜索题目、知识点或错因"
        autocomplete="off"
        @input="emit('update:keyword', valueOf($event))"
      />
      <LoaderCircle v-if="props.loading" class="kp-filter-bar__spinner" :size="17" aria-label="正在更新结果" />
    </label>

    <label class="kp-filter-bar__select">
      <span>科目</span>
      <select :value="props.subject" aria-label="按科目筛选" @change="emit('update:subject', valueOf($event))">
        <option value="全部">全部科目</option>
        <option v-for="item in props.subjects" :key="item" :value="item">{{ item }}</option>
      </select>
    </label>

    <label class="kp-filter-bar__select">
      <span>范围</span>
      <select :value="props.mode" aria-label="选择搜索范围" @change="emit('update:mode', valueOf($event))">
        <option v-for="item in searchModes" :key="item" :value="item">{{ item === '全部' ? '全部内容' : item }}</option>
      </select>
    </label>

    <button
      type="button"
      class="kp-filter-bar__reset"
      aria-label="重置筛选"
      :disabled="props.subject === '全部' && props.mode === '全部' && !props.keyword"
      @click="emit('reset')"
    >
      <RotateCcw :size="16" aria-hidden="true" />
      <span>重置</span>
    </button>

    <div class="kp-filter-bar__summary" aria-live="polite">
      <ListFilter :size="16" aria-hidden="true" />
      <span><strong>{{ props.count }}</strong> 道错题</span>
      <i aria-hidden="true"></i>
      <span><strong>{{ props.dueCount }}</strong> 道待复习</span>
    </div>
  </section>
</template>

<style scoped>
.kp-filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  padding: 12px 12px 10px;
  border-bottom: 1px solid var(--kp-line);
  background: color-mix(in srgb, var(--kp-surface) 95%, var(--kp-bg));
}

.kp-filter-bar__search,
.kp-filter-bar__select {
  min-width: 0;
  height: 36px;
  border: 1px solid var(--kp-line);
  border-radius: 7px;
  background: var(--kp-surface);
  transition: border-color var(--kp-motion-fast, 160ms) ease, box-shadow var(--kp-motion-fast, 160ms) ease;
}

.kp-filter-bar__search:focus-within,
.kp-filter-bar__select:focus-within {
  border-color: var(--kp-accent);
  box-shadow: 0 0 0 2px var(--kp-accent-wash);
}

.kp-filter-bar__search {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px;
  color: var(--kp-ink-muted);
}

.kp-filter-bar__search > svg:first-child { flex: none; }

.kp-filter-bar__search input {
  width: 100%;
  min-width: 0;
  border: 0;
  outline: 0;
  color: var(--kp-ink);
  background: transparent;
  font-size: 12.5px;
}

.kp-filter-bar__search input::placeholder { color: var(--kp-ink-faint); }
.kp-filter-bar__search input::-webkit-search-cancel-button { display: none; }

.kp-filter-bar__spinner {
  flex: 0 0 auto;
  color: var(--kp-accent);
  animation: kp-filter-spin 800ms linear infinite;
}

@keyframes kp-filter-spin { to { transform: rotate(360deg); } }

.kp-filter-bar__select {
  position: relative;
  display: grid;
  grid-template-columns: 1fr;
  align-items: center;
  padding: 0 8px;
}

.kp-filter-bar__select span {
  position: absolute;
  top: 4px;
  left: 9px;
  z-index: 1;
  color: var(--kp-ink-faint);
  font-size: 7.5px;
  font-weight: 720;
  line-height: 1;
  pointer-events: none;
}

.kp-filter-bar__select select {
  width: 100%;
  height: 100%;
  padding: 10px 19px 0 0;
  border: 0;
  outline: 0;
  color: var(--kp-ink);
  background: transparent;
  font-size: 10.5px;
  font-weight: 640;
  text-overflow: ellipsis;
  cursor: pointer;
}

.kp-filter-bar__reset {
  grid-column: 3;
  grid-row: 3;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  height: 28px;
  padding: 0 7px;
  border: 0;
  border-radius: 6px;
  color: var(--kp-ink-muted);
  background: transparent;
  font-size: 10px;
  font-weight: 650;
  cursor: pointer;
  transition: color var(--kp-motion-fast, 160ms) ease, background-color var(--kp-motion-fast, 160ms) ease, opacity var(--kp-motion-fast, 160ms) ease;
}

.kp-filter-bar__reset:hover:not(:disabled) { color: var(--kp-accent-strong); background: var(--kp-accent-wash); }
.kp-filter-bar__reset:disabled { opacity: .32; cursor: default; }

.kp-filter-bar__summary {
  grid-column: 1 / 3;
  grid-row: 3;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 2px 1px 0;
  overflow: hidden;
  color: var(--kp-ink-muted);
  font-size: 9.5px;
  white-space: nowrap;
}

.kp-filter-bar__summary svg { flex: none; color: var(--kp-accent); }
.kp-filter-bar__summary strong { color: var(--kp-ink-secondary); font-size: 10px; font-variant-numeric: tabular-nums; }
.kp-filter-bar__summary i { width: 3px; height: 3px; border-radius: 50%; background: var(--kp-line-strong); }

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

@media (max-width: 767px) {
  .kp-filter-bar { gap: 8px; padding: 11px 12px 10px; }
  .kp-filter-bar__search { height: 38px; }
  .kp-filter-bar__select { height: 37px; }
  .kp-filter-bar__summary { font-size: 10px; }
}

@media (prefers-reduced-motion: reduce) {
  .kp-filter-bar__search,
  .kp-filter-bar__select,
  .kp-filter-bar__reset { transition: none; }
}
</style>
