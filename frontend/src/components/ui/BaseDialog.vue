<script setup>
import { nextTick, onBeforeUnmount, ref, useId, watch } from "vue"
import { X } from "lucide-vue-next"
import IconButton from "./IconButton.vue"

const props = defineProps({
  open: { type: Boolean, default: false },
  title: { type: String, default: "" },
  description: { type: String, default: "" },
  size: { type: String, default: "md" },
  closeOnBackdrop: { type: Boolean, default: true },
  showClose: { type: Boolean, default: true },
})
const emit = defineEmits(["close"])
const panel = ref(null)
const titleId = `dialog-title-${useId()}`
let lastFocused = null

function focusableElements() {
  if (!panel.value) return []
  return [...panel.value.querySelectorAll('button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [href], [tabindex]:not([tabindex="-1"])')]
}

function onKeydown(event) {
  if (!props.open) return
  if (event.key === "Escape") {
    event.preventDefault()
    emit("close")
    return
  }
  if (event.key !== "Tab") return
  const items = focusableElements()
  if (!items.length) return
  const first = items[0]
  const last = items[items.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

watch(() => props.open, async (open) => {
  if (open) {
    lastFocused = document.activeElement
    document.body.classList.add("dialog-open")
    window.addEventListener("keydown", onKeydown)
    await nextTick()
    const target = panel.value?.querySelector("[autofocus]") || focusableElements()[0] || panel.value
    target?.focus()
  } else {
    document.body.classList.remove("dialog-open")
    window.removeEventListener("keydown", onKeydown)
    lastFocused?.focus?.()
  }
})

onBeforeUnmount(() => {
  document.body.classList.remove("dialog-open")
  window.removeEventListener("keydown", onKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog-fade">
      <div v-if="open" class="ui-dialog-backdrop" @mousedown.self="closeOnBackdrop && emit('close')">
        <section
          ref="panel"
          class="ui-dialog"
          :class="`ui-dialog--${size}`"
          role="dialog"
          aria-modal="true"
          :aria-labelledby="title ? titleId : undefined"
          tabindex="-1"
        >
          <header v-if="title || $slots.header || showClose" class="ui-dialog__header">
            <slot name="header">
              <div>
                <h2 v-if="title" :id="titleId">{{ title }}</h2>
                <p v-if="description">{{ description }}</p>
              </div>
            </slot>
            <IconButton v-if="showClose" label="关闭" @click="emit('close')"><X :size="19" /></IconButton>
          </header>
          <div class="ui-dialog__body"><slot></slot></div>
          <footer v-if="$slots.footer" class="ui-dialog__footer"><slot name="footer"></slot></footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>
