<script setup>
defineProps({
  label: { type: String, required: true },
  variant: { type: String, default: "ghost" },
})
</script>

<template>
  <button type="button" class="ui-icon-button" :class="`ui-icon-button--${variant}`" :aria-label="label" :title="label">
    <slot></slot>
  </button>
</template>
