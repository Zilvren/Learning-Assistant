<script setup>
import { AlertTriangle } from "lucide-vue-next"
import BaseButton from "./BaseButton.vue"
import BaseDialog from "./BaseDialog.vue"

defineProps({
  open: Boolean,
  title: { type: String, default: "确认操作" },
  message: { type: String, default: "确定要继续吗？" },
  confirmText: { type: String, default: "确认" },
  danger: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})
defineEmits(["close", "confirm"])
</script>

<template>
  <BaseDialog :open="open" :title="title" size="sm" @close="$emit('close')">
    <div class="confirm-copy">
      <span class="confirm-copy__icon"><AlertTriangle :size="22" /></span>
      <p>{{ message }}</p>
    </div>
    <template #footer>
      <BaseButton @click="$emit('close')">取消</BaseButton>
      <BaseButton :variant="danger ? 'danger' : 'primary'" :busy="busy" @click="$emit('confirm')">{{ confirmText }}</BaseButton>
    </template>
  </BaseDialog>
</template>
