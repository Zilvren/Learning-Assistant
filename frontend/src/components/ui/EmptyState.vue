<script setup>
import { BookOpen } from "lucide-vue-next"
defineProps({
  title: { type: String, default: "暂无内容" },
  description: { type: String, default: "" },
})
</script>

<template>
  <div class="ui-empty">
    <div class="ui-empty__mark"><slot name="icon"><BookOpen :size="24" /></slot></div>
    <h3>{{ title }}</h3>
    <p v-if="description">{{ description }}</p>
    <div v-if="$slots.default" class="ui-empty__actions"><slot></slot></div>
  </div>
</template>
