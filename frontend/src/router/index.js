import { createRouter, createWebHistory } from "vue-router"
import { useAuth } from "../store/auth.js"

const AppShell = () => import("../layouts/AppShell.vue")
const DesignPreviewShell = () => import("../layouts/DesignPreviewShell.vue")
const KnowtPreviewShell = () => import("../layouts/KnowtPreviewShell.vue")
const AuthPage = () => import("../components/AuthPage.vue")
const VerifyEmailPage = () => import("../components/VerifyEmailPage.vue")
const HomePage = () => import("../components/HomePage.vue")
const ReviewPage = () => import("../components/review/ReviewPage.vue")
const ErrorPreviewPage = () => import("../components/design-preview/ErrorPreviewPage.vue")
const KnowtErrorPreviewPage = () => import("../components/design-preview/knowt/KnowtErrorPreviewPage.vue")
const SettingsPage = () => import("../components/SettingsPage.vue")
const LibraryPage = () => import("../components/library/LibraryPage.vue")
const LibraryItemPage = () => import("../components/library/LibraryItemPage.vue")

export const routes = [
  { path: "/login", name: "login", component: AuthPage, meta: { public: true } },
  { path: "/verify-email", name: "verify-email", component: VerifyEmailPage, meta: { public: true } },
  {
    path: "/design-preview/knowt",
    component: KnowtPreviewShell,
    children: [
      { path: "", redirect: { name: "knowt-preview-errors" } },
      { path: "errors/:id?", name: "knowt-preview-errors", component: KnowtErrorPreviewPage, props: true },
    ],
  },
  {
    path: "/design-preview",
    component: DesignPreviewShell,
    children: [
      { path: "", redirect: { name: "design-preview-errors" } },
      { path: "errors/:id?", name: "design-preview-errors", component: ErrorPreviewPage, props: true },
    ],
  },
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

export function installAuthGuard(targetRouter, auth = useAuth()) {
  targetRouter.beforeEach(async (to) => {
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

export function createAppRouter(history = createWebHistory()) {
  return installAuthGuard(createRouter({ history, routes, scrollBehavior: () => ({ top: 0 }) }))
}

export const router = createAppRouter()
