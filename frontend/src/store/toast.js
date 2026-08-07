import { readonly, ref } from "vue"

const toasts = ref([])
let nextId = 1

function remove(id) {
  toasts.value = toasts.value.filter((toast) => toast.id !== id)
}

function notify(message, type = "success", options = {}) {
  if (!message) return null
  const id = nextId++
  const timeout = options.timeout ?? (type === "error" ? 5200 : 3400)
  toasts.value = [...toasts.value, { id, message, type }]
  if (timeout > 0) window.setTimeout(() => remove(id), timeout)
  return id
}

export function useToast() {
  return {
    toasts: readonly(toasts),
    notify,
    success: (message, options) => notify(message, "success", options),
    error: (message, options) => notify(message, "error", options),
    warning: (message, options) => notify(message, "warning", options),
    info: (message, options) => notify(message, "info", options),
    remove,
  }
}
