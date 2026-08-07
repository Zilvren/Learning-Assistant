import { describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AppShell from "../layouts/AppShell.vue"

vi.mock("../store/auth.js", async () => {
  const { ref } = await import("vue")
  return {
    useAuth: () => ({
      enabled: ref(false),
      ready: ref(true),
      user: ref(null),
      logout: vi.fn(),
    }),
  }
})

vi.mock("../store/settings.js", async () => {
  const { ref } = await import("vue")
  return {
    useSettings: () => ({
      username: ref("自律人"),
      darkMode: ref(false),
      load: vi.fn().mockResolvedValue(undefined),
    }),
  }
})

vi.mock("../store/subjects.js", async () => {
  const { ref } = await import("vue")
  return {
    useSubjects: () => ({
      subjectRef: ref(["数学"]),
      load: vi.fn().mockResolvedValue(undefined),
    }),
  }
})

vi.mock("../store/toast.js", () => ({
  useToast: () => ({ info: vi.fn() }),
}))

const TestPage = { template: '<section data-testid="route-page">页面内容</section>' }

function mockViewport(matches) {
  vi.spyOn(window, "matchMedia").mockImplementation((query) => ({
    matches,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

async function mountShell(path = "/library") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/", name: "home", component: TestPage },
      { path: "/errors/:id?", name: "errors", component: TestPage },
      { path: "/review", name: "review", component: TestPage },
      { path: "/library/:folderId?", name: "library", component: TestPage },
      { path: "/library/items/:itemId", name: "library-item", component: TestPage },
      { path: "/trash", name: "trash", component: TestPage },
      { path: "/settings", name: "settings", component: TestPage },
      { path: "/login", name: "login", component: TestPage },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(AppShell, {
    global: {
      plugins: [router],
      stubs: { Transition: false },
    },
  })
  await flushPromises()
  return wrapper
}

describe("formal Knowt application shell", () => {
  it("keeps one main landmark and can pin and collapse the desktop rail", async () => {
    mockViewport(false)
    const wrapper = await mountShell()

    expect(wrapper.findAll("main")).toHaveLength(1)
    expect(wrapper.get('[data-testid="formal-hover-rail"]').classes()).not.toContain("is-pinned")
    expect(wrapper.get('[data-testid="formal-mobile-menu-button"]').attributes("aria-expanded")).toBe("false")

    await wrapper.get('[data-testid="formal-mobile-menu-button"]').trigger("click")
    expect(wrapper.get('[data-testid="formal-hover-rail"]').classes()).toContain("is-pinned")
    expect(wrapper.get('[data-testid="formal-rail-collapse-button"]').attributes("aria-label")).toBe("收起并取消固定侧栏")

    await wrapper.get('[data-testid="formal-rail-collapse-button"]').trigger("click")
    expect(wrapper.get('[data-testid="formal-hover-rail"]').classes()).not.toContain("is-pinned")
    expect(wrapper.find('[data-testid="formal-rail-collapse-button"]').exists()).toBe(false)
  })

  it("opens the mobile navigation as a dialog and closes it with Escape", async () => {
    mockViewport(true)
    const wrapper = await mountShell()
    const menuButton = wrapper.get('[data-testid="formal-mobile-menu-button"]')

    expect(menuButton.attributes("aria-label")).toBe("打开导航菜单")
    await menuButton.trigger("click")
    expect(wrapper.get('[data-testid="formal-hover-rail"]').classes()).toContain("is-mobile-open")
    expect(wrapper.get('[data-testid="formal-hover-rail"]').attributes("role")).toBe("dialog")
    expect(wrapper.find('[data-testid="formal-mobile-backdrop"]').exists()).toBe(true)

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))
    await flushPromises()
    expect(wrapper.get('[data-testid="formal-hover-rail"]').classes()).not.toContain("is-mobile-open")
    expect(wrapper.find('[data-testid="formal-mobile-backdrop"]').exists()).toBe(false)
  })
})
