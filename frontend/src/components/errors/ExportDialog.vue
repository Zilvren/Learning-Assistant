<script setup>
import { BookOpen, FileDown, ListChecks } from "lucide-vue-next"
import BaseButton from "../ui/BaseButton.vue"
import BaseDialog from "../ui/BaseDialog.vue"

defineProps({ open: Boolean, modelValue: String, busy: Boolean })
const emit = defineEmits(["close", "update:modelValue", "export"])
const options = [
  { key: "detailed", title: "详细复盘", description: "完整展开题目、错答、正解与错因，适合整理归档。", icon: BookOpen },
  { key: "compact", title: "紧凑打印", description: "压缩间距与字号，适合一次打印较多错题。", icon: FileDown },
  { key: "practice", title: "练习自测", description: "题目与答案分册排列，适合遮住答案重新作答。", icon: ListChecks },
]
</script>

<template>
  <BaseDialog :open="open" title="编排错题册" description="选择本次 PDF 的阅读与打印方式。" size="md" @close="emit('close')">
    <div class="export-grid">
      <button v-for="option in options" :key="option.key" type="button" :class="{ active: modelValue === option.key }" @click="emit('update:modelValue', option.key)">
        <component :is="option.icon" :size="22" /><strong>{{ option.title }}</strong><p>{{ option.description }}</p><span></span>
      </button>
    </div>
    <template #footer><BaseButton @click="emit('close')">取消</BaseButton><BaseButton variant="primary" :busy="busy" @click="emit('export')"><template #icon><FileDown :size="16" /></template>导出 PDF</BaseButton></template>
  </BaseDialog>
</template>
