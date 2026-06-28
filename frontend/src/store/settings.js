import { ref } from "vue"
import { api } from "../api/index.js"

const usernameRef = ref("")
const THEME_KEY = "studyTrackerThemeV2"

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

function applyTheme(dark) {
  document.documentElement.dataset.theme = dark ? "dark" : "light"
  document.documentElement.style.colorScheme = dark ? "dark" : "light"
}

applyTheme(darkModeRef.value)

export function useSettings() {
  async function load() {
    try {
      const t = await api.getToken()
      usernameRef.value = t.username || ""
      loaded = true
    } catch(e) { /* ignore */ }
  }

  function setUsername(name) { usernameRef.value = name }
  function setDarkMode(value) {
    darkModeRef.value = !!value
    try {
      localStorage.setItem(THEME_KEY, darkModeRef.value ? "dark" : "light")
    } catch { /* ignore */ }
    // 添加过渡 class，让颜色平滑渐变
    document.documentElement.classList.add('theme-transitioning')
    applyTheme(darkModeRef.value)
    setTimeout(() => {
      document.documentElement.classList.remove('theme-transitioning')
    }, 1000)
  }

  return { load, setUsername, setDarkMode, username: usernameRef, darkMode: darkModeRef }
}
