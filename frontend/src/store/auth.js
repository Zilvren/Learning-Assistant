import { ref } from "vue"
import { api } from "../api/index.js"

const enabledRef = ref(false)
const readyRef = ref(false)
const userRef = ref(null)

export function useAuth() {
  async function init() {
    readyRef.value = false
    try {
      const status = await api.authStatus()
      enabledRef.value = !!status.enabled
      if (!enabledRef.value) {
        userRef.value = null
        return
      }
      try {
        const me = await api.me()
        userRef.value = me.user
      } catch {
        const refreshed = await api.refreshAuth().catch(() => null)
        userRef.value = refreshed?.user || null
      }
    } finally {
      readyRef.value = true
    }
  }

  async function login(account, password) {
    const result = await api.login(account, password)
    userRef.value = result.user
    return result.user
  }

  async function register(username, email, password) {
    const result = await api.register(username, email, password)
    userRef.value = result.user
    return result.user
  }

  async function logout() {
    await api.logout().catch(() => {})
    userRef.value = null
  }

  return {
    enabled: enabledRef,
    ready: readyRef,
    user: userRef,
    init,
    login,
    register,
    logout,
  }
}
