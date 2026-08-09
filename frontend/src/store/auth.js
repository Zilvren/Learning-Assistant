import { ref } from "vue"
import { api } from "../api/index.js"

const enabledRef = ref(false)
const registrationEnabledRef = ref(false)
const emailVerificationEnabledRef = ref(false)
const updateEnabledRef = ref(true)
const readyRef = ref(false)
const userRef = ref(null)
const initErrorRef = ref("")
let initPromise = null

// useAuth 维护该模块的响应式前端状态。
export function useAuth() {
  // init 维护该模块的响应式前端状态。
  async function init() {
    if (initPromise) return initPromise
    readyRef.value = false
    initErrorRef.value = ""
    initPromise = (async () => {
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
          return true
        }
        try {
          const me = await api.me()
          userRef.value = me.user
        } catch {
          const refreshed = await api.refreshAuth().catch(() => null)
          userRef.value = refreshed?.user || null
        }
        return true
      } catch (error) {
        enabledRef.value = false
        registrationEnabledRef.value = false
        emailVerificationEnabledRef.value = false
        userRef.value = null
        initErrorRef.value = error?.message || "无法连接学习空间"
        return false
      } finally {
        readyRef.value = true
        initPromise = null
      }
    })()
    return initPromise
  }

  // login 维护该模块的响应式前端状态。
  async function login(account, password) {
    const result = await api.login(account, password)
    userRef.value = result.user
    return result.user
  }

  // register 维护该模块的响应式前端状态。
  async function register(username, email, password) {
    const result = await api.register(username, email, password)
    userRef.value = result.user || null
    return result
  }

  // verifyEmail 使用邮件令牌完成验证，并把返回的登录用户写入全局状态。
  async function verifyEmail(token) {
    const result = await api.verifyEmail(token)
    userRef.value = result.user
    return result.user
  }

  // resendVerification 请求向尚未验证的邮箱重新发送验证链接。
  async function resendVerification(email) {
    return api.resendVerification(email)
  }

  // logout 维护该模块的响应式前端状态。
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
    initError: initErrorRef,
    init,
    login,
    register,
    verifyEmail,
    resendVerification,
    logout,
  }
}
