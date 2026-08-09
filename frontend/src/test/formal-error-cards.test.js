import { describe, expect, it, vi } from "vitest"
import { mount, flushPromises } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import ErrorList from "../components/ErrorList.vue"
import { api } from "../api/index.js"

const records = [
  {
    id: 4,
    subject: "高数",
    title: "曲线积分复盘",
    question: "计算 $\\int_C x\\,ds$",
    wrong: "参数化区间写错",
    correct: "按弧长公式重新计算",
    reason: "漏看方向条件",
    tags: ["曲线积分"],
    reason_tags: ["审题"],
    created: "2026-08-01",
    next_review: "2026-08-01",
    review_count: 1,
  },
  {
    id: 8,
    subject: "英语",
    title: "长难句定位",
    question: "找出句子主干",
    tags: ["阅读"],
    reason_tags: [],
    created: "2026-08-02",
    review_count: 0,
  },
]

// makeRouter 完成当前模块的局部交互。
function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: "/errors/:id?", name: "errors", component: ErrorList },
      { path: "/settings", name: "settings", component: { template: "<div />" } },
    ],
  })
}

// mountPage 完成当前模块的局部交互。
async function mountPage(path = "/errors") {
  vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: ["高数", "英语"] })
  vi.spyOn(api, "getErrors").mockResolvedValue({ errors: records })

  const router = makeRouter()
  await router.push(path)
  await router.isReady()

  const wrapper = mount({ template: "<RouterView />" }, {
    attachTo: document.body,
    global: {
      plugins: [router],
      stubs: {
        ErrorEditorDialog: true,
        ExportDialog: true,
        ConfirmDialog: true,
      },
    },
  })
  await flushPromises()
  return { wrapper, router }
}

describe("formal error card library", () => {
  it("shows the card grid on /errors and opens a card as a detail route", async () => {
    const { wrapper, router } = await mountPage()

    expect(wrapper.get('[data-testid="formal-error-card-grid"]')).toBeTruthy()
    const card = wrapper.get('[data-testid="formal-error-card-4"]')
    expect(card.text()).toContain("曲线积分复盘")
    expect(card.find(".error-card__side--front .error-card__title").text()).toBe("曲线积分复盘")
    expect(card.find(".error-card__side--front .error-card__front-excerpt").exists()).toBe(false)
    expect(card.find(".error-card__side--front").text()).not.toContain("计算")
    expect(card.find(".error-card__flip").exists()).toBe(true)
    const preview = card.get('[data-testid="formal-error-card-preview"]')
    expect(preview.text()).toContain("计算")
    expect(preview.find(".error-card__back-title").exists()).toBe(false)
    expect(preview.findAll("button.tag-pill")).toHaveLength(2)

    await preview.find("button.tag-pill").trigger("click")
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe("/errors")
    expect(api.getErrors).toHaveBeenLastCalledWith(null, null, "曲线积分", null)

    await card.trigger("click")
    await flushPromises()

    expect(router.currentRoute.value.fullPath).toBe("/errors/4")
    expect(wrapper.find('[data-testid="formal-error-card-grid"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="formal-error-detail-title"]').text()).toBe("曲线积分复盘")
  })

  it("shows a clear not-found state for a missing detail id", async () => {
    const { wrapper } = await mountPage("/errors/999")

    expect(wrapper.find('[data-testid="formal-error-card-grid"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="formal-error-detail-not-found"]').text()).toContain("没有找到错题 #999")
  })
})
