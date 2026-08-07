<script setup>
import { computed } from "vue"
import { RouterLink, RouterView, useRoute } from "vue-router"
import { ArrowLeft, BookOpen, BookOpenCheck, Home, Settings } from "lucide-vue-next"
import { useAuth } from "../store/auth.js"

const route = useRoute()
const auth = useAuth()

const identityName = computed(() => {
  if (!auth.enabled.value) return "本地学习者"
  return auth.user.value?.username || auth.user.value?.email || "已登录用户"
})

const identityMode = computed(() => auth.enabled.value ? "云端账户" : "本地模式")
const identityInitial = computed(() => Array.from(identityName.value)[0]?.toUpperCase() || "学")

const legacyErrorsTarget = computed(() => {
  const rawId = route.params.id
  if (typeof rawId !== "string" || !/^[1-9]\d*$/.test(rawId)) return { name: "errors" }

  const parsedId = Number(rawId)
  if (!Number.isSafeInteger(parsedId)) return { name: "errors" }
  return { name: "errors", params: { id: String(parsedId) } }
})
</script>

<template>
  <div class="knowledge-preview">
    <a class="kp-skip-link" href="#preview-content">跳到主要内容</a>

    <aside class="kp-workspace-sidebar" aria-label="知识库导航" data-testid="preview-library-sidebar">
      <RouterLink class="kp-brand" :to="{ name: 'design-preview-errors' }" aria-label="错题本视觉样板">
        <span class="kp-brand__mark" aria-hidden="true"><BookOpenCheck :size="18" /></span>
        <span class="kp-brand__copy">
          <strong>错题本</strong>
          <small>视觉样板</small>
        </span>
      </RouterLink>

      <div class="kp-workspace-card" aria-hidden="true">
        <span class="kp-workspace-card__mark">知</span>
        <span class="kp-workspace-card__copy">
          <strong>我的学习空间</strong>
          <small>知识工作台</small>
        </span>
      </div>

      <nav class="kp-sidebar-nav" aria-label="视觉样板工作区导航">
        <RouterLink class="kp-nav-link" :to="{ name: 'home' }" aria-label="前往概览">
          <Home :size="17" aria-hidden="true" />
          <span>概览</span>
        </RouterLink>
        <RouterLink
          class="kp-nav-link kp-nav-link--active"
          :to="{ name: 'design-preview-errors' }"
          aria-label="错题库"
          aria-current="page"
        >
          <BookOpen :size="17" aria-hidden="true" />
          <span>错题库</span>
          <small>只读</small>
        </RouterLink>
        <RouterLink class="kp-nav-link" :to="{ name: 'settings' }" aria-label="前往设置">
          <Settings :size="17" aria-hidden="true" />
          <span>设置</span>
        </RouterLink>
      </nav>

      <div class="kp-sidebar-context">
        <span>当前视图</span>
        <p><i aria-hidden="true"></i>全部错题</p>
        <small>通过目录选择记录，在右侧安静阅读与复盘。</small>
      </div>

      <footer class="kp-sidebar-footer">
        <RouterLink class="kp-return-link" :to="legacyErrorsTarget" aria-label="返回现有错题库">
          <ArrowLeft :size="16" aria-hidden="true" />
          <span>返回现有版</span>
        </RouterLink>

        <div class="kp-user" role="group" :aria-label="`${identityName}，${identityMode}`">
          <span class="kp-user__avatar" aria-hidden="true">{{ identityInitial }}</span>
          <span class="kp-user__copy" aria-hidden="true">
            <strong>{{ identityName }}</strong>
            <small>{{ identityMode }}</small>
          </span>
        </div>
      </footer>
    </aside>

    <header class="kp-mobile-topbar">
      <RouterLink class="kp-mobile-brand" :to="{ name: 'design-preview-errors' }" aria-label="错题本视觉样板">
        <span aria-hidden="true"><BookOpenCheck :size="16" /></span>
        <strong>错题本</strong>
        <small>样板</small>
      </RouterLink>
      <div class="kp-mobile-actions">
        <span class="kp-user__avatar" role="img" :aria-label="`${identityName}，${identityMode}`">{{ identityInitial }}</span>
        <RouterLink class="kp-mobile-return" :to="legacyErrorsTarget" aria-label="返回现有错题库">
          <ArrowLeft :size="17" aria-hidden="true" />
        </RouterLink>
      </div>
    </header>

    <main id="preview-content" class="kp-preview-main" tabindex="-1">
      <RouterView />
    </main>

    <nav class="kp-mobile-nav" aria-label="视觉样板移动导航">
      <RouterLink :to="{ name: 'home' }" aria-label="前往概览">
        <Home :size="18" aria-hidden="true" />
        <span>概览</span>
      </RouterLink>
      <RouterLink
        class="kp-mobile-nav__active"
        :to="{ name: 'design-preview-errors' }"
        aria-label="错题库"
        aria-current="page"
      >
        <BookOpen :size="18" aria-hidden="true" />
        <span>错题库</span>
      </RouterLink>
      <RouterLink :to="{ name: 'settings' }" aria-label="前往设置">
        <Settings :size="18" aria-hidden="true" />
        <span>设置</span>
      </RouterLink>
    </nav>
  </div>
</template>

<style scoped>
.knowledge-preview {
  --kp-sidebar-width: 232px;
  --kp-bg: #f7f7f8;
  --kp-bg-deep: #eff0f3;
  --kp-surface: #ffffff;
  --kp-surface-muted: #f6f6f8;
  --kp-surface-soft: var(--kp-surface-muted);
  --kp-surface-raised: #ffffff;
  --kp-sidebar: #f0f1f4;
  --kp-ink: #222328;
  --kp-ink-secondary: #555861;
  --kp-ink-muted: #7c7f89;
  --kp-ink-faint: #a1a4ad;
  --kp-line: #e0e1e6;
  --kp-line-strong: #c8cad2;
  --kp-accent: #5964cf;
  --kp-accent-strong: #454fb4;
  --kp-accent-wash: #ebedff;
  --kp-accent-soft: var(--kp-accent-wash);
  --kp-success: #3f7963;
  --kp-warning: #9a6c35;
  --kp-danger: #ad5654;
  --kp-danger-wash: #fbf0ef;
  --kp-danger-line: #e8cfcd;
  --kp-focus: #505cc8;
  --kp-shadow-soft: 0 1px 2px rgba(25, 27, 35, .045), 0 8px 24px rgba(25, 27, 35, .04);
  --kp-shadow-panel: 0 18px 48px rgba(25, 27, 35, .11), 0 2px 8px rgba(25, 27, 35, .05);
  --kp-shadow-sm: var(--kp-shadow-soft);
  --kp-shadow-md: var(--kp-shadow-panel);
  --kp-radius-sm: 7px;
  --kp-radius-md: 10px;
  --kp-radius-lg: 14px;
  --kp-font-sans: "MiSans", "HarmonyOS Sans SC", "Segoe UI Variable Text", "PingFang SC", "Microsoft YaHei UI", sans-serif;
  --kp-font-mono: "Cascadia Code", "SFMono-Regular", Consolas, monospace;
  --kp-motion-fast: 160ms;
  --kp-motion: 210ms;

  --paper: var(--kp-bg);
  --paper-deep: var(--kp-bg-deep);
  --sheet: var(--kp-surface);
  --sheet-soft: var(--kp-surface-soft);
  --sheet-raised: var(--kp-surface-raised);
  --ink: var(--kp-ink);
  --ink-secondary: var(--kp-ink-secondary);
  --ink-muted: var(--kp-ink-muted);
  --line: var(--kp-line);
  --line-strong: var(--kp-line-strong);
  --navy: var(--kp-ink);
  --navy-deep: #0d1a18;
  --accent: var(--kp-accent);
  --accent-bright: var(--kp-accent-strong);
  --accent-soft: var(--kp-accent-soft);
  --teal: var(--kp-accent);
  --teal-soft: var(--kp-accent-soft);
  --success: var(--kp-success);
  --warning: var(--kp-warning);
  --danger: var(--kp-danger);
  --focus: var(--kp-focus);
  --shadow-sm: var(--kp-shadow-sm);
  --shadow-md: var(--kp-shadow-md);
  --radius-sm: var(--kp-radius-sm);
  --radius-md: var(--kp-radius-md);
  --radius-lg: var(--kp-radius-lg);
  --font-display: var(--kp-font-sans);
  --font-body: var(--kp-font-sans);
  --font-mono: var(--kp-font-mono);
  --motion-fast: var(--kp-motion-fast);
  --motion: var(--kp-motion);
  --text-muted: var(--kp-ink-muted);

  display: grid;
  grid-template-columns: var(--kp-sidebar-width) minmax(0, 1fr);
  width: 100%;
  height: 100dvh;
  min-height: 620px;
  overflow: hidden;
  isolation: isolate;
  color: var(--kp-ink);
  color-scheme: light;
  background: var(--kp-bg);
  font-family: var(--kp-font-sans);
  font-size: 14px;
  line-height: 1.55;
  text-rendering: optimizeLegibility;
}

.knowledge-preview :focus-visible {
  outline: 2px solid var(--kp-focus);
  outline-offset: 3px;
}

.kp-skip-link {
  position: fixed;
  top: 8px;
  left: calc(var(--kp-sidebar-width) + 12px);
  z-index: 1000;
  padding: 8px 12px;
  border-radius: 6px;
  color: #ffffff;
  background: var(--kp-accent-strong);
  box-shadow: var(--kp-shadow-md);
  font-size: 13px;
  font-weight: 700;
  transform: translateY(-160%);
  transition: transform var(--kp-motion-fast) ease;
}

.kp-skip-link:focus {
  transform: translateY(0);
}

.kp-workspace-sidebar {
  position: relative;
  z-index: 20;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-right: 1px solid var(--kp-line);
  background: var(--kp-sidebar);
}

.kp-brand {
  min-height: 62px;
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 10px;
  padding: 0 10px;
  border-bottom: 1px solid color-mix(in srgb, var(--kp-line) 75%, transparent);
  white-space: nowrap;
}

.kp-brand__mark {
  width: 31px;
  height: 31px;
  display: grid;
  flex: none;
  place-items: center;
  border-radius: 8px;
  color: #ffffff;
  background: var(--kp-accent);
  box-shadow: 0 5px 14px rgba(82, 93, 199, .2);
}

.kp-brand__copy,
.kp-workspace-card__copy,
.kp-user__copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  line-height: 1.25;
}

.kp-brand__copy strong {
  color: var(--kp-ink);
  font-size: 15px;
  font-weight: 760;
  letter-spacing: -.015em;
}

.kp-brand__copy small {
  margin-top: 1px;
  color: var(--kp-ink-muted);
  font-size: 9px;
  font-weight: 650;
  letter-spacing: .04em;
}

.kp-workspace-card {
  display: flex;
  align-items: center;
  gap: 9px;
  margin: 12px 10px 5px;
  padding: 8px 9px;
  border: 1px solid color-mix(in srgb, var(--kp-line) 80%, transparent);
  border-radius: 9px;
  background: color-mix(in srgb, var(--kp-surface) 63%, transparent);
}

.kp-workspace-card__mark {
  width: 27px;
  height: 27px;
  display: grid;
  flex: none;
  place-items: center;
  border-radius: 7px;
  color: var(--kp-accent-strong);
  background: var(--kp-accent-wash);
  font-size: 11px;
  font-weight: 800;
}

.kp-workspace-card__copy strong { font-size: 11.5px; font-weight: 720; }
.kp-workspace-card__copy small { color: var(--kp-ink-muted); font-size: 9.5px; }

.kp-sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 8px 10px;
}

.kp-nav-link {
  position: relative;
  min-height: 36px;
  display: grid;
  grid-template-columns: 19px minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  padding: 0 9px;
  border-radius: 7px;
  color: var(--kp-ink-secondary);
  font-size: 12.5px;
  font-weight: 620;
  white-space: nowrap;
  transition: color var(--kp-motion-fast) ease, background-color var(--kp-motion-fast) ease, transform var(--kp-motion-fast) ease;
}

.kp-nav-link:hover {
  color: var(--kp-ink);
  background: color-mix(in srgb, var(--kp-surface) 72%, transparent);
  transform: translateX(1px);
}

.kp-nav-link--active {
  color: var(--kp-accent-strong);
  background: var(--kp-accent-wash);
}

.kp-nav-link small {
  padding: 2px 5px;
  border-radius: 4px;
  color: var(--kp-accent-strong);
  background: color-mix(in srgb, var(--kp-surface) 66%, transparent);
  font-size: 8.5px;
  font-weight: 750;
}

.kp-sidebar-context {
  margin: 10px 18px 0;
  padding-top: 14px;
  border-top: 1px solid var(--kp-line);
}

.kp-sidebar-context > span {
  color: var(--kp-ink-faint);
  font-size: 9px;
  font-weight: 760;
  letter-spacing: .07em;
}

.kp-sidebar-context p {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 8px;
  color: var(--kp-ink-secondary);
  font-size: 11.5px;
  font-weight: 650;
}

.kp-sidebar-context p i {
  width: 6px;
  height: 6px;
  border: 2px solid var(--kp-accent);
  border-radius: 50%;
}

.kp-sidebar-context small {
  display: block;
  margin-top: 7px;
  color: var(--kp-ink-muted);
  font-size: 10px;
  line-height: 1.55;
}

.kp-sidebar-footer {
  display: flex;
  flex-direction: column;
  gap: 7px;
  margin-top: auto;
  padding: 10px;
  border-top: 1px solid var(--kp-line);
}

.kp-user {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 6px 8px;
}

.kp-user__avatar {
  width: 29px;
  height: 29px;
  display: grid;
  flex: none;
  place-items: center;
  border: 1px solid color-mix(in srgb, var(--kp-accent) 20%, var(--kp-line));
  border-radius: 8px;
  color: var(--kp-accent-strong);
  background: var(--kp-accent-wash);
  font-size: 11px;
  font-weight: 760;
}

.kp-user__copy strong {
  max-width: 152px;
  overflow: hidden;
  font-size: 11.5px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.kp-user__copy small {
  color: var(--kp-ink-muted);
  font-size: 9.5px;
}

.kp-return-link {
  min-height: 33px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 9px;
  border: 0;
  border-radius: 7px;
  color: var(--kp-ink-secondary);
  background: transparent;
  font-size: 11.5px;
  font-weight: 650;
  white-space: nowrap;
  transition: color var(--kp-motion-fast) ease, background-color var(--kp-motion-fast) ease;
}

.kp-return-link:hover {
  color: var(--kp-accent-strong);
  background: color-mix(in srgb, var(--kp-surface) 74%, transparent);
}

.kp-preview-main {
  min-width: 0;
  min-height: 0;
  height: 100dvh;
  overflow: auto;
  outline: none;
}

.kp-mobile-topbar,
.kp-mobile-nav {
  display: none;
}

@media (max-width: 1120px) and (min-width: 768px) {
  .knowledge-preview { --kp-sidebar-width: 72px; }

  .kp-brand {
    justify-content: center;
    margin-inline: 8px;
    padding-inline: 0;
  }

  .kp-brand__copy,
  .kp-workspace-card__copy,
  .kp-nav-link span,
  .kp-nav-link small,
  .kp-sidebar-context,
  .kp-user__copy,
  .kp-return-link span {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    clip-path: inset(50%);
    white-space: nowrap;
  }

  .kp-workspace-card {
    justify-content: center;
    margin-inline: 9px;
    padding-inline: 0;
  }

  .kp-sidebar-nav { padding-inline: 9px; }
  .kp-nav-link { grid-template-columns: 1fr; place-items: center; padding: 0; }

  .kp-sidebar-footer { align-items: center; padding-inline: 9px; }
  .kp-return-link { width: 36px; justify-content: center; padding: 0; }
  .kp-user { justify-content: center; padding-inline: 0; }
}

@media (max-width: 767px) {
  .knowledge-preview {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: 52px minmax(0, 1fr);
    min-height: 0;
  }

  .kp-workspace-sidebar { display: none; }

  .kp-skip-link {
    top: 6px;
    left: 10px;
  }

  .kp-mobile-topbar {
    position: relative;
    z-index: 40;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 11px 0 13px;
    border-bottom: 1px solid var(--kp-line);
    background: color-mix(in srgb, var(--kp-surface) 95%, transparent);
    backdrop-filter: blur(14px);
  }

  .kp-mobile-brand {
    display: flex;
    align-items: center;
    gap: 7px;
    border-radius: 7px;
  }

  .kp-mobile-brand > span {
    width: 27px;
    height: 27px;
    display: grid;
    place-items: center;
    border-radius: 7px;
    color: #fff;
    background: var(--kp-accent);
  }

  .kp-mobile-brand strong { font-size: 13.5px; font-weight: 760; }

  .kp-mobile-brand small {
    padding: 2px 5px;
    border-radius: 4px;
    color: var(--kp-accent-strong);
    background: var(--kp-accent-wash);
    font-size: 8.5px;
    font-weight: 700;
  }

  .kp-mobile-actions {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .kp-mobile-return {
    width: 31px;
    height: 31px;
    display: grid;
    place-items: center;
    border: 1px solid var(--kp-line);
    border-radius: 7px;
    color: var(--kp-ink-secondary);
    background: var(--kp-surface);
  }

  .kp-preview-main {
    height: auto;
    min-height: 0;
    padding-bottom: calc(65px + env(safe-area-inset-bottom));
    overflow: auto;
  }

  .kp-mobile-nav {
    position: fixed;
    right: 10px;
    bottom: calc(8px + env(safe-area-inset-bottom));
    left: 10px;
    z-index: 60;
    height: 55px;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 3px;
    padding: 4px;
    border: 1px solid color-mix(in srgb, var(--kp-line-strong) 76%, transparent);
    border-radius: 13px;
    background: color-mix(in srgb, var(--kp-surface) 93%, transparent);
    box-shadow: 0 14px 34px rgba(31, 33, 42, .16);
    backdrop-filter: blur(18px) saturate(125%);
  }

  .kp-mobile-nav a {
    min-width: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    border-radius: 9px;
    color: var(--kp-ink-muted);
    font-size: 10.5px;
    font-weight: 650;
  }

  .kp-mobile-nav a:hover { color: var(--kp-ink); background: var(--kp-surface-muted); }

  .kp-mobile-nav .kp-mobile-nav__active {
    color: var(--kp-accent-strong);
    background: var(--kp-accent-wash);
  }
}

@media (max-width: 390px) {
  .kp-user__avatar {
    width: 27px;
    height: 27px;
  }

  .kp-mobile-nav a { gap: 4px; }
}

@media (prefers-reduced-motion: reduce) {
  .knowledge-preview *,
  .knowledge-preview *::before,
  .knowledge-preview *::after {
    scroll-behavior: auto !important;
    animation-duration: .01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: .01ms !important;
  }
}
</style>
