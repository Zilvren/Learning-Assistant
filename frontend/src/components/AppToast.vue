<script setup>
import { AlertCircle, CheckCircle2, Info, TriangleAlert, X } from "lucide-vue-next"
import { useToast } from "../store/toast.js"

const toast = useToast()
const icons = { success: CheckCircle2, error: AlertCircle, warning: TriangleAlert, info: Info }
</script>

<template>
  <Teleport to="body">
    <div class="toast-region" aria-live="polite" aria-atomic="false">
      <TransitionGroup name="toast-list">
        <article v-for="item in toast.toasts.value" :key="item.id" class="app-toast" :class="`app-toast--${item.type}`">
          <component :is="icons[item.type] || Info" :size="19" aria-hidden="true" />
          <p>{{ item.message }}</p>
          <button type="button" aria-label="关闭通知" @click="toast.remove(item.id)"><X :size="16" /></button>
        </article>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
