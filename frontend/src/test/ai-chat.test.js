import { afterEach, describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AIChatPage from "../components/AIChatPage.vue"
import { api } from "../api/index.js"

function createRouterForTest() {
  return createRouter({ history: createMemoryHistory(), routes: [
    { path: "/ai", name: "ai", component: AIChatPage },
    { path: "/settings", name: "settings", component: { template: "<div />" } },
    { path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
  ] })
}

async function mountAIPage(path = "/ai") {
  const router = createRouterForTest()
  await router.push(path)
  await router.isReady()
  const wrapper = mount(AIChatPage, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

function mockReadyHarness() {
  vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
  vi.spyOn(api, "getAIHarnessStatus").mockResolvedValue({ enabled: true })
  vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [] })
  vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
  vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => Promise.resolve({ conversations }))
}

describe("Harness-only AI chat", () => {
  afterEach(() => vi.restoreAllMocks())

  it("disables the assistant when the required Harness runtime is unavailable", async () => {
    vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
    vi.spyOn(api, "getAIHarnessStatus").mockResolvedValue({ enabled: false, reason: "请在 harness 目录执行 npm install" })
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [] })
    vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })

    const { wrapper } = await mountAIPage()

    expect(wrapper.get("textarea").attributes("disabled")).toBeDefined()
    expect(wrapper.text()).toContain("Harness 未就绪")
    expect(wrapper.text()).toContain("npm install")
  })

  it("sends every note-write request through Harness instead of preview endpoints", async () => {
    mockReadyHarness()
    const chat = vi.spyOn(api, "aiChat").mockResolvedValue({
      answer: "已创建 daily / 今日整理.md。",
      model: "deepseek-v4-flash",
      sources: [],
      harness_session_id: "harness-daily",
    })

    const { wrapper } = await mountAIPage()
    await wrapper.get("textarea").setValue("把今天的复习整理写在 daily/今日整理.md 中")
    await wrapper.get("form").trigger("submit")
    await flushPromises()

    expect(chat).toHaveBeenCalledWith(expect.objectContaining({
      message: "把今天的复习整理写在 daily/今日整理.md 中",
      history: [],
    }))
    expect(wrapper.text()).toContain("已创建 daily / 今日整理.md")
    expect(wrapper.text()).toContain("Harness Agent")
    expect(wrapper.find('[aria-label="AI 写入预览"]').exists()).toBe(false)
  })

  it("passes the selected scope to the Harness chat and persists its session id", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getLibraryItems").mockImplementation(({ parentId, kind }) => {
      if (kind === "folder" && parentId == null) return Promise.resolve({ items: [{ id: 4, kind: "folder", name: "数学" }] })
      return Promise.resolve({ items: [] })
    })
    vi.spyOn(api, "getLibraryItem").mockResolvedValue({ id: 7, kind: "note", name: "导数专题" })
    const chat = vi.spyOn(api, "aiChat").mockResolvedValue({
      answer: "先复习导数符号表。",
      model: "deepseek-v4-flash",
      sources: [],
      harness_session_id: "harness-math",
    })
    const saveConversation = vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => Promise.resolve({ conversations }))

    const { wrapper } = await mountAIPage("/ai?folder=4&items=7")
    await wrapper.get("textarea").setValue("请总结这份资料")
    await wrapper.get("form").trigger("submit")
    await flushPromises()

    expect(chat).toHaveBeenCalledWith(expect.objectContaining({
      message: "请总结这份资料",
      history: [],
      folder_id: 4,
      item_ids: [7],
    }))
    expect(saveConversation).toHaveBeenLastCalledWith(expect.arrayContaining([
      expect.objectContaining({ harness_session_id: "harness-math" }),
    ]))
  })

  it("continues an existing Harness session and keeps prior messages as fallback history", async () => {
    vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
    vi.spyOn(api, "getAIHarnessStatus").mockResolvedValue({ enabled: true })
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [{
      id: "derivative-review",
      title: "导数复习",
      item_ids: [],
      harness_session_id: "harness-existing",
      messages: [
        { role: "user", content: "我在导数上总出错" },
        { role: "assistant", content: "先从导数符号表开始复习。", model: "deepseek-v4-flash" },
      ],
    }] })
    vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
    vi.spyOn(api, "saveAIConversation").mockResolvedValue({ conversations: [] })
    const chat = vi.spyOn(api, "aiChat").mockResolvedValue({ answer: "再做两道单调性题。", model: "deepseek-v4-flash", sources: [], harness_session_id: "harness-existing" })

    const { wrapper } = await mountAIPage()
    await wrapper.get("textarea").setValue("下一步怎么练？")
    await wrapper.get("form").trigger("submit")
    await flushPromises()

    expect(chat).toHaveBeenCalledWith(expect.objectContaining({
      message: "下一步怎么练？",
      harness_session_id: "harness-existing",
      history: [
        { role: "user", content: "我在导数上总出错" },
        { role: "assistant", content: "先从导数符号表开始复习。" },
      ],
    }))
  })
})
