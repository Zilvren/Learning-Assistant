<script setup>
defineProps({
  variant: { type: String, default: "default" },
  size: { type: String, default: "md" },
  busy: { type: Boolean, default: false },
  type: { type: String, default: "button" },
})
</script>

<template>
  <button
    :type="type"
    class="ui-button"
    :class="[`ui-button--${variant}`, `ui-button--${size}`]"
    :disabled="busy || $attrs.disabled"
    :aria-busy="busy || undefined"
  >
    <span v-if="busy" class="ui-spinner" aria-hidden="true"></span>
    <slot name="icon"></slot>
    <slot></slot>
  </button>
</template>
