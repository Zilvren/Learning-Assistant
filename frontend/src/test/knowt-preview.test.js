import { flushPromises, mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { describe, expect, it, vi } from "vitest"
import { api } from "../api/index.js"
import KnowtErrorPreviewPage from "../components/design-preview/knowt/KnowtErrorPreviewPage.vue"
import KnowtPreviewShell from "../layouts/KnowtPreviewShell.vue"

const EmptyView = { template: "<div />" }

const mutatingApiMethods = [
  "login",
  "register",
  "refreshAuth",
  "logout",
  "addSubject",
  "deleteSubject",
  "ocrImage",
  "saveToken",
  "clearToken",
  "saveUsername",
  "importBackup",
  "applyUpdate",
  "addError",
  "updateError",
  "reviewError",
  "deleteError",
]

function spyOnMutatingApis() {
  return mutatingApiMethods.map((name) => vi.spyOn(api, name).mockResolvedValue({}))
}

function record(overrides = {}) {
  return {
    id: 4,
    subject: "数学",
    title: "曲线积分复盘",
    question: "真实题面内容",
    wrong: "遗漏了方向判断",
    correct: "先确定积分路径方向",
    reason: "符号判断错误",
    tags: ["曲线积分"],
    reason_tags: ["粗心"],
    created: "2026-07-30 09:30:00",
    review_count: 1,
    next_review: "2026-08-01",
    ...overrides,
  }
}

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

async function mountKnowt(path = "/design-preview/knowt/errors") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/design-preview/knowt/errors/:id?",
        name: "knowt-preview-errors",
        component: KnowtErrorPreviewPage,
      },
      { path: "/design-preview/errors/:id?", name: "design-preview-errors", component: EmptyView },
      { path: "/errors/:id?", name: "errors", component: EmptyView },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(KnowtErrorPreviewPage, { global: { plugins: [router] } })
  return { router, wrapper }
}

async function mountKnowtInShell(path = "/design-preview/knowt/errors") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/design-preview/knowt",
        component: KnowtPreviewShell,
        children: [
          { path: "errors/:id?", name: "knowt-preview-errors", component: KnowtErrorPreviewPage },
        ],
      },
      { path: "/design-preview/errors/:id?", name: "design-preview-errors", component: EmptyView },
      { path: "/errors/:id?", name: "errors", component: EmptyView },
      { path: "/", name: "home", component: EmptyView },
      { path: "/settings", name: "settings", component: EmptyView },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount({ template: "<RouterView />" }, { global: { plugins: [router] } })
  return { router, wrapper }
}

describe("Knowt error preview", () => {
  it("uses an independent hover sidebar with one main landmark and preserves the selected ID in exit links", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountKnowtInShell("/design-preview/knowt/errors/4")
    await flushPromises()

    expect(wrapper.get('[data-testid="knowt-preview-shell"]')).toBeTruthy()
    expect(wrapper.get('[data-testid="knowt-hover-rail"]')).toBeTruthy()
    const sidebar = wrapper.get('[data-testid="knowt-sidebar-panel"]')
    expect(sidebar.attributes("id")).toBe("knowt-sidebar-panel")
    expect(wrapper.findAll("main")).toHaveLength(1)
    expect(wrapper.get('[data-testid="knowt-preview-main"]').element.tagName).toBe("MAIN")
    expect(wrapper.get('[data-testid="knowt-error-workbench"]')).toBeTruthy()
    expect(wrapper.get('[data-testid="knowt-error-detail"]').element.tagName).toBe("ARTICLE")

    const currentLink = sidebar.get('a[aria-current="page"]')
    expect(currentLink.text()).toContain("错题库")

    const legacyLink = wrapper.get('a[aria-label="返回现有错题库"]')
    expect(legacyLink.attributes("href")).toBe("/errors/4")
    const remnoteLinks = wrapper.findAll('a[aria-label="切换到 RemNote 视觉样板"]')
    expect(remnoteLinks.length).toBeGreaterThan(0)
    remnoteLinks.forEach((link) => expect(link.attributes("href")).toBe("/design-preview/errors/4"))
    wrapper.unmount()
  })

  it("exposes an accessible mobile drawer that closes with Escape", async () => {
    vi.spyOn(window, "matchMedia").mockReturnValue({
      matches: true,
      media: "(max-width: 767px)",
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountKnowtInShell("/design-preview/knowt/errors/4")
    await flushPromises()

    const menuButton = wrapper.get('[data-testid="knowt-mobile-menu-button"]')
    expect(menuButton.element.tagName).toBe("BUTTON")
    expect(menuButton.attributes("aria-controls")).toBe("knowt-sidebar-panel")
    expect(menuButton.attributes("aria-expanded")).toBe("false")
    expect(wrapper.find('[data-testid="knowt-mobile-backdrop"]').exists()).toBe(false)

    await menuButton.trigger("click")
    await nextTick()
    expect(menuButton.attributes("aria-expanded")).toBe("true")
    expect(wrapper.get('[data-testid="knowt-mobile-backdrop"]')).toBeTruthy()

    window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))
    await nextTick()
    expect(menuButton.attributes("aria-expanded")).toBe("false")
    expect(wrapper.find('[data-testid="knowt-mobile-backdrop"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it("renders a selected real record and returns to the Knowt collection", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { router, wrapper } = await mountKnowt("/design-preview/knowt/errors/4")
    await flushPromises()

    expect(wrapper.get('[data-testid="knowt-detail-title"]').text()).toBe("曲线积分复盘")
    expect(wrapper.get('[data-testid="knowt-detail-content"]').text()).toContain("真实题面内容")
    await wrapper.get('button[aria-label="返回错题目录"]').trigger("click")
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/knowt/errors")
    expect(wrapper.get('[data-testid="knowt-library-pane"]').element.tagName).toBe("SECTION")
    wrapper.unmount()
  })

  it.each(["not-an-id", "99"])("shows not-found for %s only after the collection loads", async (id) => {
    const pending = deferred()
    vi.spyOn(api, "getErrors").mockReturnValue(pending.promise)
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountKnowt(`/design-preview/knowt/errors/${id}`)
    await nextTick()
    expect(wrapper.find('[data-testid="knowt-detail-not-found"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="knowt-detail-loading"]').exists()).toBe(true)

    pending.resolve({ errors: [record()] })
    await flushPromises()
    expect(wrapper.get('[data-testid="knowt-detail-not-found"]').text()).toContain("没有找到这条错题")
    wrapper.unmount()
  })

  it("syncs list selection to the Knowt route", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { router, wrapper } = await mountKnowt()
    await flushPromises()
    expect(wrapper.get('[data-testid="knowt-error-list"]')).toBeTruthy()
    await wrapper.get('[data-testid="knowt-error-item-4"] button[aria-label^="查看错题"]').trigger("click")
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe("/design-preview/knowt/errors/4")
    expect(wrapper.get('[data-testid="knowt-detail-title"]').text()).toBe("曲线积分复盘")
    wrapper.unmount()
  })

  it("maps filters, debounces keyword changes and never calls a mutating API", async () => {
    const getErrors = vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    const getSubjects = vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学", "数据结构"] })
    const mutationSpies = spyOnMutatingApis()
    const { router, wrapper } = await mountKnowt("/design-preview/knowt/errors/4")

    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith(null, null, null, null)
    expect(getSubjects).toHaveBeenCalledOnce()

    await wrapper.get('select[aria-label="按科目筛选"]').setValue("数学")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, null)
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/knowt/errors")

    const topicTag = wrapper.findAll('button[aria-label="按题目标签 曲线积分 筛选"]')[0]
    expect(topicTag).toBeTruthy()
    await topicTag.trigger("click")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, "曲线积分", null)

    await wrapper.get('select[aria-label="选择搜索范围"]').setValue("错因标签")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, "曲线积分")

    const search = wrapper.get('input[aria-label="搜索错题"]')
    const callsBeforeTyping = getErrors.mock.calls.length
    await search.setValue("粗心")
    await new Promise((resolve) => window.setTimeout(resolve, 200))
    await flushPromises()
    expect(getErrors).toHaveBeenCalledTimes(callsBeforeTyping)

    await new Promise((resolve) => window.setTimeout(resolve, 70))
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, "粗心")
    mutationSpies.forEach((spy) => expect(spy).not.toHaveBeenCalled())
    wrapper.unmount()
  })

  it("retries a failed read without replacing the selected route with not-found", async () => {
    vi.spyOn(api, "getErrors")
      .mockRejectedValueOnce(new Error("数据目录正在被其他操作占用"))
      .mockResolvedValueOnce({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { router, wrapper } = await mountKnowt("/design-preview/knowt/errors/4")
    await flushPromises()
    const alert = wrapper.get('[data-testid="knowt-load-error"]')
    expect(alert.attributes("role")).toBe("alert")
    expect(alert.text()).toContain("数据目录正在被其他操作占用")
    expect(wrapper.find('[data-testid="knowt-detail-not-found"]').exists()).toBe(false)

    await alert.get("button").trigger("click")
    await flushPromises()
    expect(wrapper.find('[data-testid="knowt-load-error"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="knowt-detail-title"]').text()).toBe("曲线积分复盘")
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/knowt/errors/4")
    wrapper.unmount()
  })

  it("normalizes nullable tags and omits unrecorded correction sections", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({
      errors: [record({ title: "暂无", wrong: "未记录", correct: "未记录", reason: "未记录", tags: null, reason_tags: null })],
    })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountKnowt("/design-preview/knowt/errors/4")
    await flushPromises()
    expect(wrapper.get('[data-testid="knowt-detail-title"]').text()).toBe("错题 #4")
    const detail = wrapper.get('[data-testid="knowt-detail-content"]')
    expect(detail.text()).toContain("真实题面内容")
    expect(detail.text()).not.toContain("未记录")
    wrapper.unmount()
  })
})
