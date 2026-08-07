import { ref } from "vue"
import { api } from "../api/index.js"

const colors = ["#2f6f73", "#855f3e", "#8a4b46", "#496a8f", "#6f5b8f", "#4f7557", "#a06a32", "#58606d"]

function hash(value) {
  let result = 0
  for (let index = 0; index < value.length; index++) result = ((result << 5) - result + value.charCodeAt(index)) | 0
  return result
}

export function subjectColor(name = "") {
  return colors[Math.abs(hash(name)) % colors.length]
}

export function hasContent(value) {
  const text = (value || "").trim()
  return !!text && text !== "未记录"
}

export function reviewLabel(item, today = new Date().toISOString().slice(0, 10)) {
  const next = item.next_review || item.created?.slice(0, 10)
  const round = (item.review_count || 0) + 1
  if (!next) return `第 ${round} 轮复习`
  if (next < today) return `逾期 ${next} · 第 ${round} 轮`
  if (next === today) return `今日到期 · 第 ${round} 轮`
  return `下次 ${next} · 第 ${round} 轮`
}

export function isDue(item, today = new Date().toISOString().slice(0, 10)) {
  return (item.next_review || item.created?.slice(0, 10) || today) <= today
}

export function useErrorLibrary() {
  const errors = ref([])
  const loading = ref(false)

  async function refresh({ subject = "全部", keyword = "", mode = "全部" } = {}) {
    loading.value = true
    try {
      const term = keyword.trim() || null
      const response = await api.getErrors(
        subject === "全部" ? null : subject,
        mode === "题目" || mode === "全部" ? term : null,
        mode === "题目标签" ? term : null,
        mode === "错因标签" ? term : null,
      )
      errors.value = response.errors || []
      return errors.value
    } finally {
      loading.value = false
    }
  }

  async function create(payload) { return api.addError(payload) }
  async function update(id, payload) { return api.updateError(id, payload) }
  async function remove(id) { return api.deleteError(id) }
  async function review(id) { return api.reviewError(id) }

  return { errors, loading, refresh, create, update, remove, review }
}
