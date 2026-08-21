import { readonly, ref } from "vue"

const toasts = ref([])
let nextId = 1
const timers = new Map()

// clearTimer 在前端状态层中维护当前状态或副作用。
function clearTimer(state) {
  if (state?.timer) window.clearTimeout(state.timer)
  if (state) state.timer = null
}

// schedule 在前端状态层中维护当前状态或副作用。
function schedule(id) {
  const state = timers.get(id)
  if (!state || state.reasons.size || state.remaining <= 0) return
  state.startedAt = Date.now()
  state.timer = window.setTimeout(() => remove(id), state.remaining)
}

// remove 维护该模块的响应式前端状态。
function remove(id) {
  clearTimer(timers.get(id))
  timers.delete(id)
  toasts.value = toasts.value.filter((toast) => toast.id !== id)
}

// pause/resume 让用户在阅读或操作通知时保留剩余展示时间。
function pause(id, reason = "interaction") {
  const state = timers.get(id)
  if (!state || state.reasons.has(reason)) return
  state.reasons.add(reason)
  if (state.timer) {
    state.remaining = Math.max(0, state.remaining - (Date.now() - state.startedAt))
    clearTimer(state)
  }
}

// resume 在前端状态层中维护当前状态或副作用。
function resume(id, reason = "interaction") {
  const state = timers.get(id)
  if (!state) return
  state.reasons.delete(reason)
  if (!state.reasons.size) schedule(id)
}

// syncDocumentVisibility 在前端状态层中维护当前状态或副作用。
function syncDocumentVisibility() {
  for (const { id } of toasts.value) {
    if (document.hidden) pause(id, "document")
    else resume(id, "document")
  }
}

if (typeof document !== "undefined") {
  document.addEventListener("visibilitychange", syncDocumentVisibility)
  import.meta.hot?.dispose(() => document.removeEventListener("visibilitychange", syncDocumentVisibility))
}

// notify 维护该模块的响应式前端状态。
function notify(message, type = "success", options = {}) {
  if (!message) return null
  const id = nextId++
  const timeout = options.timeout ?? (type === "error" ? 5200 : 3400)
  toasts.value = [...toasts.value, { id, message, type }]
  if (timeout > 0 && typeof window !== "undefined") {
    const reasons = new Set()
    if (typeof document !== "undefined" && document.hidden) reasons.add("document")
    timers.set(id, { timer: null, startedAt: 0, remaining: timeout, reasons })
    schedule(id)
  }
  return id
}

// useToast 维护该模块的响应式前端状态。
export function useToast() {
  return {
    toasts: readonly(toasts),
    notify,
    // success 维护该模块的响应式前端状态。
    success: (message, options) => notify(message, "success", options),
    // error 维护该模块的响应式前端状态。
    error: (message, options) => notify(message, "error", options),
    // warning 维护该模块的响应式前端状态。
    warning: (message, options) => notify(message, "warning", options),
    // info 维护该模块的响应式前端状态。
    info: (message, options) => notify(message, "info", options),
    remove,
    pause,
    resume,
  }
}
