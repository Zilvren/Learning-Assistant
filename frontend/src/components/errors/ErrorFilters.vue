<script setup>
import { Search, SlidersHorizontal } from "lucide-vue-next"
import BaseButton from "../ui/BaseButton.vue"

defineProps({
  subjects: { type: Array, default: () => [] },
  subject: { type: String, default: "全部" },
  keyword: { type: String, default: "" },
  mode: { type: String, default: "全部" },
  loading: Boolean,
})
const emit = defineEmits(["update:subject", "update:keyword", "update:mode", "search"])
const modes = ["全部", "题目", "题目标签", "错因标签"]
</script>

<template>
  <section class="error-filters paper-panel">
    <div class="subject-tabs" aria-label="科目筛选">
      <button
        v-for="item in ['全部', ...subjects]" :key="item" type="button"
        :class="{ active: subject === item }" @click="emit('update:subject', item); emit('search')"
      >{{ item }}</button>
    </div>
    <form class="search-bar" role="search" @submit.prevent="emit('search')">
      <label class="search-bar__mode"><SlidersHorizontal :size="15" /><select :value="mode" aria-label="搜索范围" @change="emit('update:mode', $event.target.value)"><option v-for="item in modes" :key="item">{{ item }}</option></select></label>
      <label class="search-bar__input"><Search :size="17" /><input :value="keyword" placeholder="搜索题目、标签或错因" @input="emit('update:keyword', $event.target.value)" /></label>
      <BaseButton type="submit" variant="ink" size="sm" :busy="loading">查询</BaseButton>
    </form>
  </section>
</template>
