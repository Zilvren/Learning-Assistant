import { flushPromises, mount } from "@vue/test-utils"
import { nextTick } from "vue"
import { createMemoryHistory, createRouter } from "vue-router"
import { describe, expect, it, vi } from "vitest"
import { api } from "../api/index.js"
import ErrorPreviewPage from "../components/design-preview/ErrorPreviewPage.vue"
import DesignPreviewShell from "../layouts/DesignPreviewShell.vue"

const LegacyErrors = { template: "<div>现有错题库</div>" }

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

async function mountPreview(path = "/design-preview/errors") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/design-preview/errors/:id?", name: "design-preview-errors", component: ErrorPreviewPage },
      { path: "/errors/:id?", name: "errors", component: LegacyErrors },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(ErrorPreviewPage, { global: { plugins: [router] } })
  return { router, wrapper }
}

async function mountPreviewInMain(path = "/design-preview/errors") {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: "/design-preview",
        component: DesignPreviewShell,
        children: [
          { path: "errors/:id?", name: "design-preview-errors", component: ErrorPreviewPage },
        ],
      },
      { path: "/errors/:id?", name: "errors", component: LegacyErrors },
      { path: "/", name: "home", component: { template: "<div>概览</div>" } },
      { path: "/settings", name: "settings", component: { template: "<div>设置</div>" } },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount({
    template: "<RouterView />",
  }, { global: { plugins: [router] } })
  return { router, wrapper }
}

describe("modern error preview page", () => {
  it("exposes one main landmark and the three read-only workspace regions", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountPreviewInMain("/design-preview/errors/4")
    await flushPromises()

    expect(wrapper.findAll("main")).toHaveLength(1)
    const sidebar = wrapper.get('[data-testid="preview-library-sidebar"]')
    expect(["ASIDE", "NAV"]).toContain(sidebar.element.tagName)
    expect(sidebar.attributes("aria-label") || sidebar.attributes("aria-labelledby")).toBeTruthy()
    expect(wrapper.get('[data-testid="preview-error-index"]').element.tagName).toBe("SECTION")
    expect(wrapper.get('[data-testid="preview-error-detail"]').element.tagName).toBe("ARTICLE")
    wrapper.unmount()
  })

  it("renders a real selected record and links back to the same legacy record", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountPreview("/design-preview/errors/4")
    await flushPromises()

    expect(wrapper.get('[data-testid="preview-detail-title"]').text()).toBe("曲线积分复盘")
    expect(wrapper.get('[data-testid="preview-detail-content"]').text()).toContain("真实题面内容")
    const legacyLink = wrapper.findAll("a").find((link) => link.text().includes("在现有版本中打开"))
    expect(legacyLink?.attributes("href")).toBe("/errors/4")
    wrapper.unmount()
  })

  it.each(["not-an-id", "99"])("shows not-found for %s only after the collection loads", async (id) => {
    const pending = deferred()
    vi.spyOn(api, "getErrors").mockReturnValue(pending.promise)
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountPreview(`/design-preview/errors/${id}`)
    await nextTick()
    expect(wrapper.find('[data-testid="preview-detail-not-found"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="preview-detail-loading"]').exists()).toBe(true)

    pending.resolve({ errors: [record()] })
    await flushPromises()
    expect(wrapper.get('[data-testid="preview-detail-not-found"]').text()).toContain("没有找到这则错题")
    wrapper.unmount()
  })

  it("syncs list selection to the preview route", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { router, wrapper } = await mountPreview()
    await flushPromises()
    await wrapper.get('[data-testid="preview-error-item-4"] button[aria-label^="查看错题"]').trigger("click")
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors/4")
    expect(wrapper.get('[data-testid="preview-detail-title"]').text()).toBe("曲线积分复盘")
    wrapper.unmount()
  })

  it("maps subject, mode and tag filters and debounces keyword routing changes", async () => {
    const getErrors = vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学", "数据结构"] })
    const mutationSpies = spyOnMutatingApis()
    const { router, wrapper } = await mountPreview("/design-preview/errors/4")

    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith(null, null, null, null)

    await wrapper.get('select[aria-label="按科目筛选"]').setValue("数学")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, null)
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors")

    await wrapper.get('[data-testid="preview-error-item-4"] button[aria-label^="查看错题"]').trigger("click")
    await flushPromises()
    const topicTag = wrapper.findAll('button[aria-label="按题目标签 曲线积分 筛选"]')[0]
    expect(topicTag).toBeTruthy()
    await topicTag.trigger("click")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, "曲线积分", null)
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors")

    await wrapper.get('select[aria-label="选择搜索范围"]').setValue("错因标签")
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, "曲线积分")

    await wrapper.get('[data-testid="preview-error-item-4"] button[aria-label^="查看错题"]').trigger("click")
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors/4")

    const search = wrapper.get('input[type="search"]')
    const callsBeforeTyping = getErrors.mock.calls.length
    await search.setValue("粗心")
    await new Promise((resolve) => window.setTimeout(resolve, 200))
    await flushPromises()
    expect(getErrors).toHaveBeenCalledTimes(callsBeforeTyping)
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors/4")

    await new Promise((resolve) => window.setTimeout(resolve, 70))
    await flushPromises()
    expect(getErrors).toHaveBeenLastCalledWith("数学", null, null, "粗心")
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors")
    mutationSpies.forEach((spy) => expect(spy).not.toHaveBeenCalled())
    wrapper.unmount()
  })

  it("shows a failed refresh with retry without discarding the selected route", async () => {
    vi.spyOn(api, "getErrors")
      .mockRejectedValueOnce(new Error("数据目录正在被其他操作占用"))
      .mockResolvedValueOnce({ errors: [record()] })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { router, wrapper } = await mountPreview("/design-preview/errors/4")
    await flushPromises()
    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toContain("数据目录正在被其他操作占用")
    expect(wrapper.find('[data-testid="preview-detail-not-found"]').exists()).toBe(false)

    await alert.get("button").trigger("click")
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="preview-detail-title"]').text()).toBe("曲线积分复盘")
    expect(router.currentRoute.value.fullPath).toBe("/design-preview/errors/4")
    wrapper.unmount()
  })

  it("handles nullable reason tags and unrecorded detail sections", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({
      errors: [record({ title: "暂无", wrong: "未记录", correct: "未记录", reason: "未记录", reason_tags: null })],
    })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["数学"] })

    const { wrapper } = await mountPreview("/design-preview/errors/4")
    await flushPromises()
    expect(wrapper.get('[data-testid="preview-detail-title"]').text()).toBe("错题 #4")
    const detail = wrapper.get('[data-testid="preview-detail-content"]')
    expect(detail.text()).toContain("真实题面内容")
    expect(detail.find("#kp-wrong-heading").exists()).toBe(false)
    expect(detail.find("#kp-correct-heading").exists()).toBe(false)
    expect(detail.find("#kp-reason-heading").exists()).toBe(false)
    wrapper.unmount()
  })
})
