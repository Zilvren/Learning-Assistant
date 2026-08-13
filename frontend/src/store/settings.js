import { ref } from "vue"
import { api } from "../api/index.js"

const usernameRef = ref("")
const THEME_KEY = "studyTrackerThemeV2"
const COLOR_THEME_KEY = "studyTrackerColorThemeV1"
const THEME_TRANSITION_MS = 220

export const colorThemes = Object.freeze([
  { id: "verdant", emoji: "🌿", name: "翡翠书房", description: "青绿、珊瑚与靛蓝的平衡搭配" },
  { id: "ocean", emoji: "🌊", name: "海湾蓝", description: "海湾蓝、日落橙与金黄点缀" },
  { id: "sunset", emoji: "🌅", name: "暖阳纸页", description: "陶土橙、雾蓝与鼠尾草绿" },
  { id: "violet", emoji: "🪻", name: "紫藤暮光", description: "紫藤、琥珀与湖蓝的层次" },
  { id: "rose", emoji: "🌸", name: "玫瑰晨雾", description: "玫瑰、薄荷绿与靛青点缀" },
])

const COLOR_THEME_IDS = new Set(colorThemes.map((theme) => theme.id))

// initialDarkMode 维护该模块的响应式前端状态。
function initialDarkMode() {
  try {
    const theme = localStorage.getItem(THEME_KEY)
    return theme === "dark"
  } catch {
    return false
  }
}

// initialColorTheme 读取已保存的配色；无效或旧值会回退到默认方案。
function initialColorTheme() {
  try {
    const palette = localStorage.getItem(COLOR_THEME_KEY)
    return COLOR_THEME_IDS.has(palette) ? palette : "verdant"
  } catch {
    return "verdant"
  }
}

const darkModeRef = ref(initialDarkMode())
const colorThemeRef = ref(initialColorTheme())
let loaded = false
let themeTransitionTimer = 0

// applyTheme 维护该模块的响应式前端状态。
function applyTheme(dark) {
  document.documentElement.dataset.theme = dark ? "dark" : "light"
  document.documentElement.style.colorScheme = dark ? "dark" : "light"
}

// applyColorTheme 将所选配色写入根元素，所有页面通过 CSS token 自动同步。
function applyColorTheme(palette) {
  document.documentElement.dataset.palette = palette
}

applyTheme(darkModeRef.value)
applyColorTheme(colorThemeRef.value)

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

  // setColorTheme 只变更视觉 token，不触碰学习数据或页面结构。
  function setColorTheme(value) {
    if (!COLOR_THEME_IDS.has(value)) return
    colorThemeRef.value = value
    try {
      localStorage.setItem(COLOR_THEME_KEY, value)
    } catch { /* ignore */ }
    transitionTheme(() => applyColorTheme(colorThemeRef.value))
  }

  return {
    load,
    setUsername,
    setDarkMode,
    setColorTheme,
    username: usernameRef,
    darkMode: darkModeRef,
    colorTheme: colorThemeRef,
    colorThemes,
  }
}
