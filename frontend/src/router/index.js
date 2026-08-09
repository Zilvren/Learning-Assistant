import { createRouter, createWebHistory } from "vue-router"
import { useAuth } from "../store/auth.js"

// AppShell 配置页面跳转或认证守卫。
const AppShell = () => import("../layouts/AppShell.vue")
// AuthPage 配置页面跳转或认证守卫。
const AuthPage = () => import("../components/AuthPage.vue")
// VerifyEmailPage 配置页面跳转或认证守卫。
const VerifyEmailPage = () => import("../components/VerifyEmailPage.vue")
// HomePage 配置页面跳转或认证守卫。
const HomePage = () => import("../components/HomePage.vue")
// ReviewPage 配置页面跳转或认证守卫。
const ReviewPage = () => import("../components/review/ReviewPage.vue")
// SettingsPage 配置页面跳转或认证守卫。
const SettingsPage = () => import("../components/SettingsPage.vue")
// LibraryPage 配置页面跳转或认证守卫。
const LibraryPage = () => import("../components/library/LibraryPage.vue")
// LibraryItemPage 配置页面跳转或认证守卫。
const LibraryItemPage = () => import("../components/library/LibraryItemPage.vue")

export const routes = [
  { path: "/login", name: "login", component: AuthPage, meta: { public: true } },
  { path: "/verify-email", name: "verify-email", component: VerifyEmailPage, meta: { public: true } },
  {
    path: "/",
    component: AppShell,
    children: [
      { path: "", name: "home", component: HomePage },
      { path: "library/items/:itemId", name: "library-item", component: LibraryItemPage, props: true },
      { path: "library/:folderId?", name: "library", component: LibraryPage, props: true },
      { path: "trash/:folderId?", name: "trash", component: LibraryPage, props: (route) => ({ trash: true, folderId: route.params.folderId }) },
      { path: "review", name: "review", component: ReviewPage },
      { path: "errors/:id?", redirect: { name: "library" } },
      { path: "settings", name: "settings", component: SettingsPage },
    ],
  },
  { path: "/:pathMatch(.*)*", redirect: "/" },
]

// installAuthGuard 配置页面跳转或认证守卫。
export function installAuthGuard(targetRouter, auth = useAuth()) {
  targetRouter.beforeEach(async (to) => {
    if (to.meta.skipAuth) return true
    if (!auth.ready.value) await auth.init()

    if (to.meta.public) {
      if (to.name === "login" && (!auth.enabled.value || auth.user.value)) return { name: "home" }
      return true
    }

    if (auth.enabled.value && !auth.user.value) {
      return { name: "login", query: { redirect: to.fullPath } }
    }
    return true
  })
  return targetRouter
}

// createAppRouter 配置页面跳转或认证守卫。
export function createAppRouter(history = createWebHistory()) {
  return installAuthGuard(createRouter({ history, routes, scrollBehavior: () => ({ top: 0 }) }))
}

export const router = createAppRouter()
