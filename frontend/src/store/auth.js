import { ref } from "vue"
import { api } from "../api/index.js"

const enabledRef = ref(false)
const registrationEnabledRef = ref(false)
const emailVerificationEnabledRef = ref(false)
const updateEnabledRef = ref(true)
const readyRef = ref(false)
const userRef = ref(null)

export function useAuth() {
  async function init() {
    readyRef.value = false
    try {
      const status = await api.authStatus()
      enabledRef.value = !!status.enabled
      registrationEnabledRef.value = !!status.registration_enabled
      emailVerificationEnabledRef.value = !!status.email_verification_enabled
      updateEnabledRef.value = !!status.update_enabled
      if (!enabledRef.value) {
        userRef.value = null
        registrationEnabledRef.value = false
        emailVerificationEnabledRef.value = false
        updateEnabledRef.value = true
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
    userRef.value = result.user || null
    return result
  }

  async function verifyEmail(token) {
    const result = await api.verifyEmail(token)
    userRef.value = result.user
    return result.user
  }

  async function resendVerification(email) {
    return api.resendVerification(email)
  }

  async function logout() {
    await api.logout().catch(() => {})
    userRef.value = null
  }

  return {
    enabled: enabledRef,
    registrationEnabled: registrationEnabledRef,
    emailVerificationEnabled: emailVerificationEnabledRef,
    updateEnabled: updateEnabledRef,
    ready: readyRef,
    user: userRef,
    init,
    login,
    register,
    verifyEmail,
    resendVerification,
    logout,
  }
}
