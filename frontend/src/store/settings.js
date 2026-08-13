import { ref } from "vue"
import { api } from "../api/index.js"

const usernameRef = ref("")
const THEME_KEY = "studyTrackerThemeV2"
const THEME_TRANSITION_MS = 220

// initialDarkMode 维护该模块的响应式前端状态。
function initialDarkMode() {
  try {
    const theme = localStorage.getItem(THEME_KEY)
    return theme === "dark"
  } catch {
    return false
  }
}

const darkModeRef = ref(initialDarkMode())
let loaded = false
let themeTransitionTimer = 0

// applyTheme 维护该模块的响应式前端状态。
function applyTheme(dark) {
  document.documentElement.dataset.theme = dark ? "dark" : "light"
  document.documentElement.style.colorScheme = dark ? "dark" : "light"
}

applyTheme(darkModeRef.value)
try {
  localStorage.removeItem("studyTrackerColorThemeV1")
} catch { /* ignore */ }
delete document.documentElement.dataset.palette

// transitionTheme 使用同一段短过渡，避免明暗与配色切换的视觉反馈不一致。
function transitionTheme(apply) {
  window.clearTimeout(themeTransitionTimer)
  document.documentElement.classList.add("theme-transitioning")
  window.requestAnimationFrame(apply)
  themeTransitionTimer = window.setTimeout(() => {
    document.documentElement.classList.remove("theme-transitioning")
  }, THEME_TRANSITION_MS)
}

// useSettings 维护该模块的响应式前端状态。
export function useSettings() {
  // load 维护该模块的响应式前端状态。
  async function load() {
    try {
      const t = await api.getToken()
      usernameRef.value = t.username || ""
      loaded = true
    } catch(e) { /* ignore */ }
  }

  // setUsername 维护该模块的响应式前端状态。
  function setUsername(name) { usernameRef.value = name }
  // setDarkMode 维护该模块的响应式前端状态。
  function setDarkMode(value) {
    darkModeRef.value = !!value
    try {
      localStorage.setItem(THEME_KEY, darkModeRef.value ? "dark" : "light")
    } catch { /* ignore */ }
    transitionTheme(() => applyTheme(darkModeRef.value))
  }

  return {
    load,
    setUsername,
    setDarkMode,
    username: usernameRef,
    darkMode: darkModeRef,
  }
}
