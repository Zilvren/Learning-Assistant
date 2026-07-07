import { ref } from "vue"
import { api } from "../api/index.js"

const usernameRef = ref("")
const THEME_KEY = "studyTrackerThemeV2"
const THEME_TRANSITION_MS = 1900

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
    window.clearTimeout(themeTransitionTimer)
    document.documentElement.classList.add('theme-transitioning')
    window.requestAnimationFrame(() => {
      applyTheme(darkModeRef.value)
    })
    themeTransitionTimer = window.setTimeout(() => {
      document.documentElement.classList.remove('theme-transitioning')
    }, THEME_TRANSITION_MS)
  }

  return { load, setUsername, setDarkMode, username: usernameRef, darkMode: darkModeRef }
}
