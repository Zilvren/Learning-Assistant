import { describe, expect, it, vi } from "vitest"
import { ref } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { installAuthGuard } from "../router/index.js"

const View = { template: "<div />" }
const testRoutes = [
  { path: "/", name: "home", component: View },
  { path: "/login", name: "login", component: View, meta: { public: true } },
  { path: "/errors/:id?", name: "errors", component: View },
]

// makeRouter 完成当前模块的局部交互。
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

  it("bypasses login entirely in local mode", async () => {
    const router = makeRouter({ enabled: false, user: null })
    await router.push("/login")
    await router.isReady()
    expect(router.currentRoute.value.name).toBe("home")
  })
})
