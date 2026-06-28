import { ref } from "vue"
import { api } from "../api/index.js"

const subjectRef = ref([])
let loaded = false

export function useSubjects() {
  async function load() {
    try {
      const r = await api.getSubjects()
      subjectRef.value = r.subjects
      loaded = true
    } catch (e) { /* ignore */ }
    return subjectRef
  }

  async function add(name) {
    await api.addSubject(name)
    subjectRef.value = [...subjectRef.value, name]
  }

  async function remove(name) {
    await api.deleteSubject(name)
    subjectRef.value = subjectRef.value.filter(s => s !== name)
  }

  function list() {
    if (!loaded) load()
    return subjectRef
  }

  return { load, add, remove, list, subjectRef }
}
