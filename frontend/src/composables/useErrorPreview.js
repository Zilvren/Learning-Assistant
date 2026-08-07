import { ref } from "vue"
import { api } from "../api/index.js"

export function mapPreviewQuery({ subject = "全部", keyword = "", mode = "全部" } = {}) {
  const term = String(keyword ?? "").trim() || null
  return [
    subject === "全部" ? null : subject,
    mode === "题目" || mode === "全部" ? term : null,
    mode === "题目标签" ? term : null,
    mode === "错因标签" ? term : null,
  ]
}

function asError(value, fallback) {
  if (value instanceof Error) return value
  if (value && typeof value.message === "string") return new Error(value.message)
  return new Error(typeof value === "string" && value ? value : fallback)
}

function normalizeErrors(value) {
  if (!Array.isArray(value)) return []
  return value.map((item) => {
    if (!item || typeof item !== "object") return item
    return {
      ...item,
      tags: Array.isArray(item.tags) ? item.tags : [],
      reason_tags: Array.isArray(item.reason_tags) ? item.reason_tags : [],
    }
  })
}

export function useErrorPreview() {
  const errors = ref([])
  const subjects = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref(null)
  const subjectsLoading = ref(false)
  const subjectsError = ref(null)

  let requestSequence = 0
  let subjectsSequence = 0
  let disposed = false

  async function refresh(filters = {}) {
    if (disposed) return null
    const sequence = ++requestSequence
    loading.value = true
    error.value = null

    try {
      const response = await api.getErrors(...mapPreviewQuery(filters))
      if (disposed || sequence !== requestSequence) return null
      errors.value = normalizeErrors(response?.errors)
      loaded.value = true
      return errors.value
    } catch (reason) {
      if (disposed || sequence !== requestSequence) return null
      error.value = asError(reason, "错题加载失败")
      loaded.value = true
      return null
    } finally {
      if (!disposed && sequence === requestSequence) loading.value = false
    }
  }

  async function loadSubjects() {
    if (disposed) return null
    const sequence = ++subjectsSequence
    subjectsLoading.value = true
    subjectsError.value = null

    try {
      const response = await api.getSubjects()
      if (disposed || sequence !== subjectsSequence) return null
      subjects.value = Array.isArray(response?.subjects) ? response.subjects : []
      return subjects.value
    } catch (reason) {
      if (disposed || sequence !== subjectsSequence) return null
      subjectsError.value = asError(reason, "科目加载失败")
      return null
    } finally {
      if (!disposed && sequence === subjectsSequence) subjectsLoading.value = false
    }
  }

  function dispose() {
    disposed = true
    requestSequence += 1
    subjectsSequence += 1
    loading.value = false
    subjectsLoading.value = false
  }

  return {
    errors,
    subjects,
    loading,
    loaded,
    error,
    subjectsLoading,
    subjectsError,
    refresh,
    loadSubjects,
    dispose,
  }
}
