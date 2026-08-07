<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { RouterLink, RouterView, useRoute } from "vue-router"
import {
  ArrowLeft,
  BookOpenCheck,
  Boxes,
  CalendarCheck2,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  FolderKanban,
  Home,
  LibraryBig,
  Menu,
  Search,
  Settings,
  Sparkles,
  X,
} from "lucide-vue-next"
import { useAuth } from "../store/auth.js"

const route = useRoute()
const auth = useAuth()
const railPinned = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)
let mobileMediaQuery = null

const identityName = computed(() => {
  if (!auth.enabled.value) return "本地学习者"
  return auth.user.value?.username || auth.user.value?.email || "已登录用户"
})

const identityMode = computed(() => auth.enabled.value ? "云端账户" : "本地模式")
const identityInitial = computed(() => Array.from(identityName.value)[0]?.toUpperCase() || "学")

const preservedId = computed(() => {
  let value = route.params.id
  if (Array.isArray(value)) value = value[0]
  if (typeof value !== "string" && typeof value !== "number") return null

  const raw = String(value)
  if (!/^[1-9]\d*$/.test(raw)) return null

  const parsed = Number(raw)
  return Number.isSafeInteger(parsed) ? String(parsed) : null
})

function errorTarget(name) {
  return preservedId.value
    ? { name, params: { id: preservedId.value } }
    : { name }
}

const legacyErrorsTarget = computed(() => errorTarget("errors"))
const remnotePreviewTarget = computed(() => errorTarget("design-preview-errors"))

const navigationExpanded = computed(() => isMobile.value ? mobileMenuOpen.value : railPinned.value)
const navigationLabel = computed(() => {
  if (isMobile.value) return mobileMenuOpen.value ? "关闭导航菜单" : "打开导航菜单"
  return railPinned.value ? "取消固定侧栏" : "固定展开侧栏"
})

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

function toggleNavigation() {
  if (isMobile.value) {
    mobileMenuOpen.value = !mobileMenuOpen.value
    return
  }
  railPinned.value = !railPinned.value
}

function syncMobileViewport(event) {
  const matches = Boolean(event?.matches)
  isMobile.value = matches
  if (!matches) closeMobileMenu()
}

function handleEscape(event) {
  if (event.key === "Escape" && mobileMenuOpen.value) closeMobileMenu()
}

watch(() => route.fullPath, closeMobileMenu)

onMounted(() => {
  mobileMediaQuery = window.matchMedia("(max-width: 767px)")
  syncMobileViewport(mobileMediaQuery)
  mobileMediaQuery.addEventListener?.("change", syncMobileViewport)
  window.addEventListener("keydown", handleEscape)
})

onBeforeUnmount(() => {
  mobileMediaQuery?.removeEventListener?.("change", syncMobileViewport)
  window.removeEventListener("keydown", handleEscape)
})
</script>

<template>
  <div class="knowt-preview" data-testid="knowt-preview-shell">
    <a class="kt-skip-link" href="#knowt-preview-content">跳到主要内容</a>

    <button
      v-if="mobileMenuOpen"
      type="button"
      class="kt-mobile-backdrop"
      aria-label="关闭导航菜单"
      data-testid="knowt-mobile-backdrop"
      @click="closeMobileMenu"
    />

    <aside
      id="knowt-sidebar-navigation"
      class="kt-hover-rail"
      :class="{
        'is-pinned': railPinned,
        'is-mobile-open': mobileMenuOpen,
      }"
      :role="isMobile ? 'dialog' : undefined"
      :aria-modal="isMobile ? 'true' : undefined"
      :aria-hidden="isMobile && !mobileMenuOpen ? 'true' : undefined"
      :inert="isMobile && !mobileMenuOpen"
      aria-label="样板侧边导航"
      data-testid="knowt-hover-rail"
    >
      <RouterLink
        class="kt-rail-brand"
        :to="{ name: 'knowt-preview-errors' }"
        aria-label="错题本学习空间"
        @click="closeMobileMenu"
      >
        <span class="kt-rail-brand__mark" aria-hidden="true">
          <BookOpenCheck :size="20" :stroke-width="2.2" />
        </span>
        <span class="kt-rail-brand__copy">
          <strong>错题本</strong>
          <small>学习空间</small>
        </span>
      </RouterLink>

      <div id="knowt-sidebar-panel" class="kt-sidebar-panel" data-testid="knowt-sidebar-panel">
        <div class="kt-rail-section">
          <p class="kt-rail-heading">学习</p>
          <nav class="kt-rail-nav" aria-label="学习导航">
            <RouterLink class="kt-rail-link" :to="{ name: 'home' }" @click="closeMobileMenu">
              <span class="kt-rail-link__icon"><Home :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">概览</span>
            </RouterLink>
            <RouterLink
              class="kt-rail-link"
              :to="{ name: 'home', hash: '#today-review' }"
              @click="closeMobileMenu"
            >
              <span class="kt-rail-link__icon"><CalendarCheck2 :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">今日复习</span>
            </RouterLink>
          </nav>
        </div>

        <div class="kt-rail-section">
          <p class="kt-rail-heading">资料库</p>
          <nav class="kt-rail-nav" aria-label="资料库导航">
            <RouterLink
              class="kt-rail-link kt-rail-link--active"
              :to="{ name: 'knowt-preview-errors' }"
              aria-current="page"
              @click="closeMobileMenu"
            >
              <span class="kt-rail-link__icon"><LibraryBig :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">错题库</span>
            </RouterLink>
            <RouterLink
              class="kt-rail-link"
              :to="{ name: 'settings', hash: '#subjects' }"
              @click="closeMobileMenu"
            >
              <span class="kt-rail-link__icon"><FolderKanban :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">科目</span>
            </RouterLink>
            <RouterLink class="kt-rail-link" :to="{ name: 'settings' }" @click="closeMobileMenu">
              <span class="kt-rail-link__icon"><Settings :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">设置</span>
            </RouterLink>
          </nav>
        </div>

        <div class="kt-rail-section kt-rail-section--versions">
          <p class="kt-rail-heading">版本</p>
          <nav class="kt-rail-nav" aria-label="版本切换">
            <RouterLink
              class="kt-rail-link"
              :to="remnotePreviewTarget"
              aria-label="切换到 RemNote 视觉样板"
              @click="closeMobileMenu"
            >
              <span class="kt-rail-link__icon"><Boxes :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">RemNote 版</span>
            </RouterLink>
            <RouterLink
              class="kt-rail-link"
              :to="legacyErrorsTarget"
              aria-label="返回现有错题库"
              @click="closeMobileMenu"
            >
              <span class="kt-rail-link__icon"><ArrowLeft :size="19" aria-hidden="true" /></span>
              <span class="kt-rail-label">返回现有版</span>
            </RouterLink>
          </nav>
        </div>
      </div>

      <div class="kt-rail-footer">
        <RouterLink
          class="kt-rail-link"
          :to="{ name: 'settings', hash: '#help' }"
          @click="closeMobileMenu"
        >
          <span class="kt-rail-link__icon"><CircleHelp :size="19" aria-hidden="true" /></span>
          <span class="kt-rail-label">帮助</span>
        </RouterLink>

        <div class="kt-rail-identity" role="group" :aria-label="`${identityName}，${identityMode}`">
          <span class="kt-identity__avatar" aria-hidden="true">{{ identityInitial }}</span>
          <span class="kt-rail-identity__copy">
            <strong>{{ identityName }}</strong>
            <small>{{ identityMode }}</small>
          </span>
        </div>
      </div>
    </aside>

    <header class="kt-topbar">
      <button
        type="button"
        class="kt-navigation-toggle"
        :aria-label="navigationLabel"
        :aria-expanded="navigationExpanded"
        aria-controls="knowt-sidebar-panel"
        data-testid="knowt-mobile-menu-button"
        @click="toggleNavigation"
      >
        <X v-if="isMobile && mobileMenuOpen" :size="19" aria-hidden="true" />
        <Menu v-else-if="isMobile" :size="19" aria-hidden="true" />
        <ChevronLeft v-else-if="railPinned" :size="18" aria-hidden="true" />
        <ChevronRight v-else :size="18" aria-hidden="true" />
      </button>

      <div class="kt-context-label" aria-label="当前页面：错题库视觉样板">
        <Search :size="16" aria-hidden="true" />
        <span>错题库</span>
        <span class="kt-context-label__separator" aria-hidden="true">·</span>
        <small>视觉样板</small>
      </div>

      <div class="kt-topbar__actions">
        <span class="kt-preview-badge" aria-label="当前为轻快学习视觉样板">
          <Sparkles :size="14" aria-hidden="true" />
          <span>轻快样板</span>
        </span>
        <RouterLink
          class="kt-switch-link"
          :to="remnotePreviewTarget"
          aria-label="切换到 RemNote 视觉样板"
        >
          <Boxes :size="15" aria-hidden="true" />
          <span>RemNote 版</span>
        </RouterLink>
        <RouterLink class="kt-current-link" :to="legacyErrorsTarget" aria-label="返回现有错题库">
          <ArrowLeft :size="15" aria-hidden="true" />
          <span>返回现有版</span>
        </RouterLink>
      </div>
    </header>

    <main
      id="knowt-preview-content"
      class="kt-preview-main"
      tabindex="-1"
      data-testid="knowt-preview-main"
    >
      <RouterView />
    </main>
  </div>
</template>

<style scoped>
.knowt-preview {
  --kt-header-height: 60px;
  --kt-rail-width: 68px;
  --kt-rail-expanded: 232px;
  --kt-bg: #f3f7f6;
  --kt-surface: #ffffff;
  --kt-surface-soft: #f7faf9;
  --kt-surface-raised: #ffffff;
  --kt-ink: #172723;
  --kt-ink-secondary: #536660;
  --kt-ink-muted: #7b8c87;
  --kt-ink-faint: #a4b1ad;
  --kt-line: #dde7e4;
  --kt-line-strong: #c6d5d1;
  --kt-accent: #168d80;
  --kt-accent-strong: #0f7067;
  --kt-accent-soft: #e1f4ef;
  --kt-accent-bright: #25ac9b;
  --kt-danger: #bd5d56;
  --kt-danger-soft: #fff0ee;
  --kt-warning: #b7782d;
  --kt-focus: #087e74;
  --kt-shadow-sm: 0 1px 2px rgba(17, 52, 44, .04), 0 8px 24px rgba(17, 52, 44, .045);
  --kt-shadow-md: 0 18px 44px rgba(17, 52, 44, .15), 0 3px 10px rgba(17, 52, 44, .06);
  --kt-radius-sm: 10px;
  --kt-radius-md: 15px;
  --kt-radius-lg: 22px;
  --kt-font-sans: "MiSans", "HarmonyOS Sans SC", "Segoe UI Variable Text", "PingFang SC", "Microsoft YaHei UI", sans-serif;
  --kt-font-mono: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
  --kt-motion-fast: 160ms;
  --kt-motion: 220ms;

  --paper: var(--kt-bg);
  --paper-deep: #eaf1ef;
  --sheet: var(--kt-surface);
  --sheet-soft: var(--kt-surface-soft);
  --sheet-raised: var(--kt-surface-raised);
  --ink: var(--kt-ink);
  --ink-secondary: var(--kt-ink-secondary);
  --ink-muted: var(--kt-ink-muted);
  --line: var(--kt-line);
  --line-strong: var(--kt-line-strong);
  --accent: var(--kt-accent);
  --accent-bright: var(--kt-accent-bright);
  --accent-soft: var(--kt-accent-soft);
  --teal: var(--kt-accent);
  --teal-soft: var(--kt-accent-soft);
  --danger: var(--kt-danger);
  --focus: var(--kt-focus);
  --font-display: var(--kt-font-sans);
  --font-body: var(--kt-font-sans);
  --font-mono: var(--kt-font-mono);
  --motion-fast: var(--kt-motion-fast);
  --motion: var(--kt-motion);

  display: grid;
  grid-template-columns: var(--kt-rail-width) minmax(0, 1fr);
  grid-template-rows: var(--kt-header-height) minmax(0, 1fr);
  width: 100%;
  height: 100dvh;
  min-height: 560px;
  overflow: hidden;
  isolation: isolate;
  color: var(--kt-ink);
  color-scheme: light;
  background:
    radial-gradient(circle at 82% -18%, rgba(37, 172, 155, .075), transparent 31%),
    var(--kt-bg);
  font-family: var(--kt-font-sans);
  font-size: 14px;
  line-height: 1.5;
  text-rendering: optimizeLegibility;
}

.knowt-preview :focus-visible {
  outline: 3px solid color-mix(in srgb, var(--kt-focus) 32%, transparent);
  outline-offset: 3px;
}

.kt-skip-link {
  position: fixed;
  top: 8px;
  left: 76px;
  z-index: 300;
  padding: 9px 13px;
  border-radius: 10px;
  color: #ffffff;
  background: var(--kt-accent-strong);
  box-shadow: var(--kt-shadow-md);
  font-size: 12px;
  font-weight: 750;
  transform: translateY(-150%);
  transition: transform var(--kt-motion-fast) ease;
}

.kt-skip-link:focus { transform: translateY(0); }

.kt-hover-rail {
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 120;
  width: var(--kt-rail-width);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--kt-line);
  background: var(--kt-surface);
  box-shadow: 1px 0 0 rgba(255, 255, 255, .8);
  transition: width var(--kt-motion) cubic-bezier(.2, .72, .25, 1), box-shadow var(--kt-motion) ease;
}

.kt-hover-rail:hover,
.kt-hover-rail:focus-within,
.kt-hover-rail.is-pinned {
  width: var(--kt-rail-expanded);
  box-shadow: var(--kt-shadow-md);
}

.kt-rail-brand {
  min-height: var(--kt-header-height);
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  align-items: center;
  gap: 4px;
  padding: 0 11px;
  border-bottom: 1px solid var(--kt-line);
  white-space: nowrap;
}

.kt-rail-brand__mark {
  width: 38px;
  height: 38px;
  display: grid;
  justify-self: center;
  place-items: center;
  border-radius: 13px;
  color: #ffffff;
  background: var(--kt-accent);
  box-shadow: 0 7px 18px rgba(22, 141, 128, .2);
}

.kt-rail-brand__copy,
.kt-rail-identity__copy {
  min-width: 0;
  max-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  opacity: 0;
  line-height: 1.15;
  transform: translateX(-5px);
  transition: max-width var(--kt-motion) ease, opacity var(--kt-motion-fast) ease, transform var(--kt-motion) ease;
}

.kt-hover-rail:hover .kt-rail-brand__copy,
.kt-hover-rail:focus-within .kt-rail-brand__copy,
.kt-hover-rail.is-pinned .kt-rail-brand__copy,
.kt-hover-rail:hover .kt-rail-identity__copy,
.kt-hover-rail:focus-within .kt-rail-identity__copy,
.kt-hover-rail.is-pinned .kt-rail-identity__copy {
  max-width: 150px;
  opacity: 1;
  transform: translateX(0);
}

.kt-rail-brand__copy strong { color: var(--kt-ink); font-size: 15px; font-weight: 780; }
.kt-rail-brand__copy small { margin-top: 3px; color: var(--kt-ink-muted); font-size: 9px; font-weight: 680; letter-spacing: .06em; }

.kt-sidebar-panel {
  min-height: 0;
  flex: 1;
  overflow-x: hidden;
  overflow-y: auto;
  padding: 14px 8px;
  scrollbar-width: thin;
}

.kt-rail-section + .kt-rail-section { margin-top: 12px; }

.kt-rail-heading {
  max-height: 0;
  margin: 0;
  padding: 0 10px;
  overflow: hidden;
  color: var(--kt-ink-muted);
  font-size: 10px;
  font-weight: 720;
  letter-spacing: .04em;
  opacity: 0;
  white-space: nowrap;
  transition: max-height var(--kt-motion) ease, margin var(--kt-motion) ease, opacity var(--kt-motion-fast) ease;
}

.kt-hover-rail:hover .kt-rail-heading,
.kt-hover-rail:focus-within .kt-rail-heading,
.kt-hover-rail.is-pinned .kt-rail-heading {
  max-height: 22px;
  margin-bottom: 5px;
  opacity: 1;
}

.kt-rail-nav { display: grid; gap: 3px; }

.kt-rail-link {
  width: 100%;
  min-height: 42px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  align-items: center;
  gap: 3px;
  padding: 0 5px;
  overflow: hidden;
  border-radius: 11px;
  color: var(--kt-ink-secondary);
  font-size: 12.5px;
  font-weight: 670;
  white-space: nowrap;
  transition: color var(--kt-motion-fast) ease, background-color var(--kt-motion-fast) ease;
}

.kt-rail-link:hover { color: var(--kt-ink); background: var(--kt-surface-soft); }
.kt-rail-link--active { color: var(--kt-accent-strong); background: var(--kt-accent-soft); }

.kt-rail-link__icon {
  width: 42px;
  height: 38px;
  display: grid;
  place-items: center;
}

.kt-rail-label {
  max-width: 0;
  overflow: hidden;
  opacity: 0;
  transform: translateX(-5px);
  transition: max-width var(--kt-motion) ease, opacity var(--kt-motion-fast) ease, transform var(--kt-motion) ease;
}

.kt-hover-rail:hover .kt-rail-label,
.kt-hover-rail:focus-within .kt-rail-label,
.kt-hover-rail.is-pinned .kt-rail-label {
  max-width: 145px;
  opacity: 1;
  transform: translateX(0);
}

.kt-rail-section--versions { padding-top: 12px; border-top: 1px solid var(--kt-line); }

.kt-rail-footer {
  display: grid;
  gap: 3px;
  padding: 8px;
  border-top: 1px solid var(--kt-line);
  background: var(--kt-surface);
}

.kt-rail-identity {
  min-height: 48px;
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr);
  align-items: center;
  gap: 3px;
  padding: 3px 5px;
  overflow: hidden;
}

.kt-identity__avatar {
  width: 34px;
  height: 34px;
  display: grid;
  place-self: center;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--kt-accent) 24%, var(--kt-line));
  border-radius: 12px;
  color: var(--kt-accent-strong);
  background: #f0faf7;
  font-size: 12px;
  font-weight: 800;
}

.kt-rail-identity__copy strong {
  max-width: 145px;
  overflow: hidden;
  color: var(--kt-ink);
  font-size: 11.5px;
  font-weight: 720;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kt-rail-identity__copy small { margin-top: 2px; color: var(--kt-ink-muted); font-size: 9.5px; }

.kt-topbar {
  position: relative;
  z-index: 40;
  grid-column: 2;
  grid-row: 1;
  min-width: 0;
  display: grid;
  grid-template-columns: auto minmax(180px, 1fr) auto;
  align-items: center;
  gap: 14px;
  padding: 0 clamp(14px, 2vw, 28px) 0 12px;
  border-bottom: 1px solid color-mix(in srgb, var(--kt-line) 88%, transparent);
  background: color-mix(in srgb, var(--kt-surface) 94%, transparent);
  backdrop-filter: blur(18px) saturate(120%);
}

.kt-navigation-toggle {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border: 1px solid var(--kt-line);
  border-radius: 9px;
  color: var(--kt-ink-secondary);
  background: var(--kt-surface);
  transition: color var(--kt-motion-fast) ease, border-color var(--kt-motion-fast) ease, background-color var(--kt-motion-fast) ease;
}

.kt-navigation-toggle:hover { color: var(--kt-accent-strong); border-color: var(--kt-line-strong); background: var(--kt-surface-soft); }

.kt-context-label {
  justify-self: center;
  width: min(100%, 700px);
  min-height: 38px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 14px;
  border: 1px solid color-mix(in srgb, var(--kt-line) 72%, transparent);
  border-radius: 999px;
  color: var(--kt-ink-muted);
  background: var(--kt-surface-soft);
  font-size: 12px;
}

.kt-context-label > span:first-of-type { color: var(--kt-ink-secondary); font-weight: 670; }
.kt-context-label__separator { color: var(--kt-ink-faint); }
.kt-context-label small { color: var(--kt-ink-muted); font-size: 10.5px; }

.kt-topbar__actions { display: flex; align-items: center; justify-content: flex-end; gap: 7px; }

.kt-preview-badge,
.kt-switch-link,
.kt-current-link {
  min-height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border-radius: 10px;
  white-space: nowrap;
  font-size: 10.5px;
  font-weight: 710;
}

.kt-preview-badge { color: var(--kt-accent-strong); background: var(--kt-accent-soft); }
.kt-switch-link { border: 1px solid var(--kt-line); color: var(--kt-ink-secondary); background: var(--kt-surface); }
.kt-switch-link:hover { border-color: color-mix(in srgb, var(--kt-accent) 42%, var(--kt-line)); color: var(--kt-accent-strong); }
.kt-current-link { color: #ffffff; background: var(--kt-ink); }
.kt-current-link:hover { background: var(--kt-accent-strong); }

.kt-preview-main {
  grid-column: 2;
  grid-row: 2;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  outline: none;
}

.kt-mobile-backdrop { display: none; }

@media (max-width: 1120px) {
  .kt-topbar { grid-template-columns: auto minmax(140px, 1fr) auto; padding-right: 14px; }
  .kt-preview-badge span,
  .kt-switch-link span,
  .kt-current-link span { display: none; }
  .kt-preview-badge,
  .kt-switch-link,
  .kt-current-link { width: 34px; justify-content: center; padding: 0; }
}

@media (max-width: 767px) {
  .knowt-preview {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: 56px minmax(0, 1fr);
    min-height: 0;
  }

  .kt-skip-link { left: 12px; }

  .kt-hover-rail,
  .kt-hover-rail:hover,
  .kt-hover-rail:focus-within,
  .kt-hover-rail.is-pinned {
    z-index: 160;
    width: min(84vw, 300px);
    box-shadow: var(--kt-shadow-md);
    transform: translateX(-102%);
    transition: transform var(--kt-motion) cubic-bezier(.2, .72, .25, 1);
  }

  .kt-hover-rail.is-mobile-open { transform: translateX(0); }

  .kt-hover-rail .kt-rail-brand__copy,
  .kt-hover-rail .kt-rail-identity__copy,
  .kt-hover-rail .kt-rail-label {
    max-width: 180px;
    opacity: 1;
    transform: translateX(0);
  }

  .kt-hover-rail .kt-rail-heading {
    max-height: 22px;
    margin-bottom: 5px;
    opacity: 1;
  }

  .kt-mobile-backdrop {
    position: fixed;
    inset: 0;
    z-index: 150;
    display: block;
    width: 100%;
    height: 100%;
    border: 0;
    border-radius: 0;
    background: rgba(15, 30, 27, .38);
    backdrop-filter: blur(2px);
  }

  .kt-topbar {
    grid-column: 1;
    grid-row: 1;
    grid-template-columns: auto minmax(0, 1fr) auto;
    gap: 8px;
    padding: 0 10px;
  }

  .kt-context-label {
    min-width: 0;
    min-height: 34px;
    padding-inline: 11px;
  }

  .kt-context-label small,
  .kt-context-label__separator,
  .kt-preview-badge,
  .kt-switch-link { display: none; }

  .kt-current-link { width: 34px; justify-content: center; padding: 0; color: var(--kt-ink-secondary); background: var(--kt-surface-soft); }
  .kt-current-link span { display: none; }

  .kt-preview-main { grid-column: 1; grid-row: 2; }
}

@media (max-width: 420px) {
  .kt-context-label svg { display: none; }
}

@media (prefers-reduced-motion: reduce) {
  .knowt-preview *,
  .knowt-preview *::before,
  .knowt-preview *::after {
    scroll-behavior: auto !important;
    animation-duration: .01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: .01ms !important;
  }
}
</style>
