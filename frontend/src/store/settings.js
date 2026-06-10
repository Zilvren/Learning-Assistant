import { ref } from "vue"
import { api } from "../api/index.js"

const usernameRef = ref("")
let loaded = false

export function useSettings() {
  async function load() {
    try {
      const t = await api.getToken()
      usernameRef.value = t.username || ""
      loaded = true
    } catch(e) { /* ignore */ }
  }

  function setUsername(name) { usernameRef.value = name }

  return { load, setUsername, username: usernameRef }
}
