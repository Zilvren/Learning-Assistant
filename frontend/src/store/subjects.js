import { ref } from "vue"
import { api } from "../api/index.js"

const subjectRef = ref([])
let loaded = false

// useSubjects 维护该模块的响应式前端状态。
export function useSubjects() {
  // load 维护该模块的响应式前端状态。
  async function load() {
    try {
      const r = await api.getSubjects()
      subjectRef.value = r.subjects
      loaded = true
    } catch (e) { /* ignore */ }
    return subjectRef
  }

  // add 维护该模块的响应式前端状态。
  async function add(name) {
    await api.addSubject(name)
    subjectRef.value = [...subjectRef.value, name]
  }

  // remove 维护该模块的响应式前端状态。
  async function remove(name) {
    await api.deleteSubject(name)
    subjectRef.value = subjectRef.value.filter(s => s !== name)
  }

  // list 维护该模块的响应式前端状态。
  function list() {
    if (!loaded) load()
    return subjectRef
  }

  return { load, add, remove, list, subjectRef }
}
