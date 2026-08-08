import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { installAuthGuard, routes } from "../router/index.js"

const View = { template: "<div />" }
const testRoutes = [
  { path: "/", name: "home", component: View },
  { path: "/login", name: "login", component: View, meta: { public: true } },
  { path: "/errors/:id?", name: "errors", component: View },
  { path: "/design-preview/errors/:id?", name: "design-preview-errors", component: View },
  { path: "/design-preview/knowt/errors/:id?", name: "knowt-preview-errors", component: View },
]

function makeRouter({ enabled, user }) {
  const router = createRouter({ history: createMemoryHistory(), routes: testRoutes })
  const auth = { enabled: ref(enabled), ready: ref(true), user: ref(user), init: vi.fn() }
  return installAuthGuard(router, auth)
}

describe("authentication route guard", () => {
  it("redirects unauthenticated cloud users to login and keeps the destination", async () => {
    const router = makeRouter({ enabled: true, user: null })
    await router.push("/errors/12")
    await router.isReady()
    expect(router.currentRoute.value.name).toBe("login")
    expect(router.currentRoute.value.query.redirect).toBe("/errors/12")
  })

  it("allows authenticated users to open protected routes", async () => {
    const router = makeRouter({ enabled: true, user: { username: "tester" } })
    await router.push("/errors/12")
    await router.isReady()
    expect(router.currentRoute.value.fullPath).toBe("/errors/12")
  })

  it("protects the isolated design preview and preserves its full destination", async () => {
    const router = makeRouter({ enabled: true, user: null })
    await router.push("/design-preview/errors/4")
    await router.isReady()
    expect(router.currentRoute.value.name).toBe("login")
    expect(router.currentRoute.value.query.redirect).toBe("/design-preview/errors/4")
  })

  it("protects the Knowt preview and preserves its full destination", async () => {
    const router = makeRouter({ enabled: true, user: null })
    await router.push("/design-preview/knowt/errors/4")
    await router.isReady()
    expect(router.currentRoute.value.name).toBe("login")
    expect(router.currentRoute.value.query.redirect).toBe("/design-preview/knowt/errors/4")
  })

  it("bypasses login entirely in local mode", async () => {
    const router = makeRouter({ enabled: false, user: null })
    await router.push("/login")
    await router.isReady()
    expect(router.currentRoute.value.name).toBe("home")
  })

  it("registers the preview outside the legacy application shell", () => {
    const preview = routes.find((route) => route.path === "/design-preview")
    const legacyShell = routes.find((route) => route.path === "/")
    expect(preview).toBeTruthy()
    expect(preview.component).not.toBe(legacyShell.component)
    expect(preview.children.some((route) => route.name === "design-preview-errors")).toBe(true)
  })

  it("registers the Knowt preview as a second isolated shell", () => {
    const knowtPreview = routes.find((route) => route.path === "/design-preview/knowt")
    const remnotePreview = routes.find((route) => route.path === "/design-preview")
    const legacyShell = routes.find((route) => route.path === "/")

    expect(knowtPreview).toBeTruthy()
    expect(knowtPreview.component).not.toBe(remnotePreview.component)
    expect(knowtPreview.component).not.toBe(legacyShell.component)
    expect(knowtPreview.children.some((route) => route.name === "knowt-preview-errors")).toBe(true)
  })
})
