<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, RouterView, useRoute, useRouter } from "vue-router"
import {
  BookOpenCheck,
	Bot,
  CalendarCheck2,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Files,
  Home,
  LogOut,
  Menu,
  Settings,
  Sparkles,
  Trash2,
  X,
} from "lucide-vue-next"
import { useAuth } from "../store/auth.js"
import { useSettings } from "../store/settings.js"
import { useToast } from "../store/toast.js"
import { rememberedLibraryPath } from "../utils/libraryPath.js"

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const settings = useSettings()
const toast = useToast()
const railPinned = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)
const topbarCollapsed = ref(false)
let mobileMediaQuery = null

const navSections = [
  {
    label: "学习",
    items: [
      { to: { name: "home" }, label: "概览", icon: Home, match: "home" },
      { to: { name: "review" }, label: "今日复习", icon: CalendarCheck2, match: "review" },
      { to: "/ai", label: "AI 助手", icon: Bot, match: "ai" },
    ],
  },
  {
    label: "资料库",
    items: [
      { to: "/library", label: "资料库", icon: Files, match: "library" },
      { to: "/trash", label: "回收站", icon: Trash2, match: "trash" },
      { to: { name: "settings" }, label: "设置", icon: Settings, match: "settings" },
    ],
  },
]

const pageMeta = computed(() => {
  if (route.name === "library" || route.name === "library-item") return { title: "资料库", description: "整理笔记、标签与文件", icon: Files }
  if (route.name === "review") return { title: "今日复习", description: "复习今天到期的笔记", icon: CalendarCheck2 }
	if (route.name === "ai") return { title: "AI 学习助手", description: "分析资料并制定下一步行动", icon: Bot }
  if (route.name === "trash") return { title: "回收站", description: "恢复或永久删除资料", icon: Trash2 }
  if (route.name === "settings") return { title: "设置", description: "账户、外观与数据", icon: Settings }
  return { title: route.hash === "#today-review" ? "今日复习" : "学习概览", description: "查看今天的学习进度", icon: route.hash === "#today-review" ? CalendarCheck2 : Home }
})

const identity = computed(() => auth.enabled.value && auth.user.value
  ? auth.user.value.username
  : settings.username.value || "本地学习者")
const identityMode = computed(() => auth.enabled.value ? "云端账户" : "本地模式")
const identityInitial = computed(() => Array.from(identity.value)[0]?.toUpperCase() || "学")
const navigationExpanded = computed(() => isMobile.value ? mobileMenuOpen.value : railPinned.value)
const navigationLabel = computed(() => {
  if (isMobile.value) return mobileMenuOpen.value ? "关闭导航菜单" : "打开导航菜单"
  return railPinned.value ? "取消固定侧栏" : "固定展开侧栏"
})
const libraryDestination = computed(() => {
  // 路由变化会重新读取已保存的位置；在当前目录内再次点击不会把路径重置为根目录。
  if (route.name === "library") return route.path
  return rememberedLibraryPath()
})

// isNavActive 协调当前组件的状态和交互。
function isNavActive(item) {
  if (item.match === "library") return route.name === "library" || route.name === "library-item"
  if (item.match === "home") return route.name === "home" && route.hash !== "#today-review"
  if (item.match === "settings") return route.name === "settings" && route.hash !== "#subjects"
  return route.name === item.match
}

// closeMobileMenu 协调当前组件的状态和交互。
function closeMobileMenu() {
  mobileMenuOpen.value = false
}

// toggleNavigation 协调当前组件的状态和交互。
function toggleNavigation() {
  if (isMobile.value) mobileMenuOpen.value = !mobileMenuOpen.value
  else railPinned.value = !railPinned.value
}

// syncMobileViewport 协调当前组件的状态和交互。
function syncMobileViewport(event) {
  const matches = Boolean(event?.matches)
  isMobile.value = matches
  if (!matches) closeMobileMenu()
}

// handleEscape 协调当前组件的状态和交互。
function handleEscape(event) {
  if (event.key === "Escape" && mobileMenuOpen.value) closeMobileMenu()
}

// setTopbarCollapsed 接收编辑页滚动状态，在沉浸编辑时收起全局顶栏。
function setTopbarCollapsed(event) {
  topbarCollapsed.value = Boolean(event?.detail?.collapsed)
}

// signOut 协调当前组件的状态和交互。
async function signOut() {
  await auth.logout()
  toast.info("已退出登录")
  await router.replace({ name: "login" })
}

watch(() => route.fullPath, () => {
  closeMobileMenu()
  topbarCollapsed.value = false
})

onMounted(async () => {
  mobileMediaQuery = window.matchMedia?.("(max-width: 767px)") || null
  syncMobileViewport(mobileMediaQuery)
  mobileMediaQuery?.addEventListener?.("change", syncMobileViewport)
  window.addEventListener("keydown", handleEscape)
  window.addEventListener("learning-space:editor-chrome", setTopbarCollapsed)
  await settings.load()
})

onBeforeUnmount(() => {
  mobileMediaQuery?.removeEventListener?.("change", syncMobileViewport)
  window.removeEventListener("keydown", handleEscape)
  window.removeEventListener("learning-space:editor-chrome", setTopbarCollapsed)
})
</script>

<template>
  <div class="app-shell" :class="{ 'is-topbar-collapsed': topbarCollapsed }" data-testid="formal-app-shell">
    <a class="app-skip-link" href="#app-main-content">跳到主要内容</a>

    <button
      v-if="mobileMenuOpen"
      type="button"
      class="app-mobile-backdrop"
      aria-label="关闭导航菜单"
      data-testid="formal-mobile-backdrop"
      @click="closeMobileMenu"
    />

    <aside
      id="app-sidebar-navigation"
      class="app-rail"
      :class="{ 'is-pinned': railPinned, 'is-mobile-open': mobileMenuOpen }"
      :role="isMobile ? 'dialog' : undefined"
      :aria-modal="isMobile ? 'true' : undefined"
      :aria-hidden="isMobile && !mobileMenuOpen ? 'true' : undefined"
      :inert="isMobile && !mobileMenuOpen"
      aria-label="应用侧边导航"
      data-testid="formal-hover-rail"
    >
      <RouterLink class="brand" :to="{ name: 'home' }" aria-label="学习空间首页" @click="closeMobileMenu">
        <span class="brand__seal" aria-hidden="true"><BookOpenCheck :size="20" /></span>
        <span class="brand__copy"><strong>学习空间</strong><small>个人资料库</small></span>
      </RouterLink>

      <button
        v-if="!isMobile && railPinned"
        type="button"
        class="rail-pin-toggle"
        aria-label="收起并取消固定侧栏"
        data-testid="formal-rail-collapse-button"
        @click="toggleNavigation"
      >
        <ChevronLeft :size="17" aria-hidden="true" />
      </button>

      <div id="app-sidebar-panel" class="app-rail__panel">
        <section v-for="section in navSections" :key="section.label" class="app-nav-section">
          <p class="app-nav-heading">{{ section.label }}</p>
          <nav class="primary-nav" :aria-label="`${section.label}导航`">
            <RouterLink
              v-for="item in section.items"
              :key="item.label"
              :to="item.match === 'library' ? libraryDestination : item.to"
              class="primary-nav__item"
              :class="{ 'app-nav-link--active': isNavActive(item) }"
              :aria-current="isNavActive(item) ? 'page' : undefined"
              @click="closeMobileMenu"
            >
              <span class="app-nav-icon" aria-hidden="true"><component :is="item.icon" :size="19" /></span>
              <span class="app-nav-label">{{ item.label }}</span>
            </RouterLink>
          </nav>
        </section>
      </div>

      <footer class="rail-footer">
        <RouterLink class="primary-nav__item app-help-link" :to="{ name: 'settings', hash: '#updates' }" @click="closeMobileMenu">
          <span class="app-nav-icon" aria-hidden="true"><CircleHelp :size="19" /></span>
          <span class="app-nav-label">帮助与版本</span>
        </RouterLink>

        <div class="identity-card" role="group" :aria-label="`${identity}，${identityMode}`">
          <span class="identity-card__avatar" aria-hidden="true">{{ identityInitial }}</span>
          <div class="identity-card__copy"><strong>{{ identity }}</strong><small>{{ identityMode }}</small></div>
        </div>

        <button v-if="auth.enabled.value" type="button" class="rail-logout" aria-label="退出登录" @click="signOut">
          <span class="app-nav-icon" aria-hidden="true"><LogOut :size="18" /></span>
          <span class="app-nav-label">退出登录</span>
        </button>
      </footer>
    </aside>

    <header class="app-topbar">
      <button
        type="button"
        class="app-navigation-toggle"
        :aria-label="navigationLabel"
        :aria-expanded="navigationExpanded"
        aria-controls="app-sidebar-panel"
        data-testid="formal-mobile-menu-button"
        @click="toggleNavigation"
      >
        <X v-if="isMobile && mobileMenuOpen" :size="19" aria-hidden="true" />
        <Menu v-else-if="isMobile" :size="19" aria-hidden="true" />
        <ChevronLeft v-else-if="railPinned" :size="18" aria-hidden="true" />
        <ChevronRight v-else :size="18" aria-hidden="true" />
      </button>

      <div class="app-context" :aria-label="`当前页面：${pageMeta.title}`">
        <component :is="pageMeta.icon" :size="16" aria-hidden="true" />
        <strong>{{ pageMeta.title }}</strong>
        <span aria-hidden="true">·</span>
        <small>{{ pageMeta.description }}</small>
      </div>

      <div class="app-topbar__identity" :title="`${identity} · ${identityMode}`">
        <span class="app-live-status"><Sparkles :size="13" aria-hidden="true" />专注学习</span>
        <span class="app-topbar__avatar" aria-hidden="true">{{ identityInitial }}</span>
      </div>
    </header>

    <main id="app-main-content" class="app-main" :class="{ 'app-main--ai': route.name === 'ai' }" tabindex="-1" data-testid="formal-app-main">
      <RouterView v-slot="{ Component, route: currentRoute }">
        <Transition name="page-shift" mode="out-in">
          <component :is="Component" :key="currentRoute.path" />
        </Transition>
      </RouterView>
    </main>
  </div>
</template>
