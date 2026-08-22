import { afterEach, describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AIChatPage from "../components/AIChatPage.vue"
import { api } from "../api/index.js"

// createRouterForTest 为当前用例准备或验证测试场景。
function createRouterForTest() {
  return createRouter({ history: createMemoryHistory(), routes: [
    { path: "/ai", name: "ai", component: AIChatPage },
    { path: "/settings", name: "settings", component: { template: "<div />" } },
    { path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
  ] })
}

// mountAIPage 为当前用例准备或验证测试场景。
async function mountAIPage(path = "/ai") {
  const router = createRouterForTest()
  await router.push(path)
  await router.isReady()
  const wrapper = mount(AIChatPage, { global: { plugins: [router] } })
  await flushPromises()
  return { wrapper, router }
}

// mockReadyHarness 为当前用例准备或验证测试场景。
function mockReadyHarness() {
  vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
  vi.spyOn(api, "getAIHarnessStatus").mockResolvedValue({ enabled: true })
  vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [] })
  vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
  vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => Promise.resolve({ conversations }))
}

describe("Harness-only AI chat", () => {
  afterEach(() => {
    vi.restoreAllMocks()
    globalThis.localStorage?.clear()
  })

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

  it("keeps the current conversation visible until the target conversation data is ready", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
      { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }] },
      { id: "target", title: "目标对话", item_ids: [7], messages: [{ role: "assistant", content: "目标内容" }] },
    ] })
    let resolveItem
    vi.spyOn(api, "getLibraryItem").mockImplementation(() => new Promise((resolve) => { resolveItem = resolve }))

    const { wrapper } = await mountAIPage()
    const targetButton = wrapper.findAll("button.ai-history-item").find((button) => button.text().includes("目标对话"))
    await targetButton.trigger("click")

    expect(wrapper.text()).toContain("当前内容")
    expect(wrapper.find(".ai-chat-empty").exists()).toBe(false)

    resolveItem({ id: 7, kind: "note", name: "资料" })
    await flushPromises()

    expect(wrapper.text()).toContain("目标内容")
  })

  it("archives the current conversation and automatically selects the remaining active conversation", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
      { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }] },
      { id: "next", title: "下一对话", item_ids: [], messages: [{ role: "assistant", content: "下一内容" }] },
    ] })
    const archive = vi.spyOn(api, "archiveAIConversation").mockResolvedValue({ conversations: [
      { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }], archived_at: "2026-08-21T00:00:00Z" },
      { id: "next", title: "下一对话", item_ids: [], messages: [{ role: "assistant", content: "下一内容" }] },
    ] })

    const { wrapper, router } = await mountAIPage()
    await wrapper.get('button[aria-label="当前对话 的操作"]').trigger("click")
    await wrapper.get('button[role="menuitem"]').trigger("click")
    await flushPromises()

    expect(archive).toHaveBeenCalledWith("current")
    expect(wrapper.text()).toContain("下一内容")
    expect(router.currentRoute.value.query.conversation).toBe("next")
  })

  it("saves a newly created conversation with a null archive timestamp", async () => {
    mockReadyHarness()
    const save = vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => Promise.resolve({ conversations }))

    const { wrapper } = await mountAIPage()
    save.mockClear()

    await wrapper.get('button[aria-label="新对话"]').trigger("click")
    await flushPromises()

    expect(save).toHaveBeenCalledTimes(1)
    const savedConversations = save.mock.calls[0][0]
    expect(savedConversations).toEqual(expect.arrayContaining([
      expect.objectContaining({ archived_at: null }),
    ]))
    expect(savedConversations.every((conversation) => conversation.archived_at === null)).toBe(true)
    expect(wrapper.text()).not.toContain("对话上下文格式错误")
  })

  it("archives a current conversation that has saved folder and item context", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
      { id: "contextual", title: "资料对话", folder_id: 4, item_ids: [7], messages: [{ role: "assistant", content: "资料内容" }] },
    ] })
    vi.spyOn(api, "getLibraryItem").mockResolvedValue({ id: 7, kind: "note", name: "导数专题" })
    const save = vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => Promise.resolve({ conversations }))
    const archive = vi.spyOn(api, "archiveAIConversation").mockResolvedValue({ conversations: [
      { id: "contextual", title: "资料对话", folder_id: 4, item_ids: [7], messages: [{ role: "assistant", content: "资料内容" }], archived_at: "2026-08-21T00:00:00Z" },
    ] })

    const { wrapper } = await mountAIPage()
    await wrapper.get('button[aria-label="资料对话 的操作"]').trigger("click")
    await wrapper.get('button[role="menuitem"]').trigger("click")
    await flushPromises()

    expect(save).toHaveBeenCalledWith(expect.arrayContaining([
      expect.objectContaining({ id: "contextual", folder_id: 4, item_ids: [7], archived_at: null }),
    ]))
    expect(archive).toHaveBeenCalledWith("contextual")
  })

  it("saves a non-current conversation before archiving it", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
      { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }] },
      { id: "pending", title: "待归档对话", item_ids: [], messages: [{ role: "assistant", content: "待归档内容" }] },
    ] })
    let savedPending = false
    vi.spyOn(api, "saveAIConversation").mockImplementation((conversations) => {
      savedPending = conversations.some((conversation) => conversation.id === "pending")
      return Promise.resolve({ conversations })
    })
    const archive = vi.spyOn(api, "archiveAIConversation").mockImplementation((id) => {
      if (!savedPending) return Promise.reject(new Error("AI 对话不存在"))
      return Promise.resolve({ conversations: [
        { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }] },
        { id, title: "待归档对话", item_ids: [], messages: [{ role: "assistant", content: "待归档内容" }], archived_at: "2026-08-21T00:00:00Z" },
      ] })
    })

    const { wrapper } = await mountAIPage()
    await wrapper.get('button[aria-label="待归档对话 的操作"]').trigger("click")
    await wrapper.get('button[role="menuitem"]').trigger("click")
    await flushPromises()

    expect(archive).toHaveBeenCalledWith("pending")
    expect(wrapper.text()).not.toContain("待归档对话")
  })

  it("does not archive a current conversation when its save fails", async () => {
    mockReadyHarness()
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
      { id: "current", title: "当前对话", item_ids: [], messages: [{ role: "assistant", content: "当前内容" }] },
    ] })
    vi.spyOn(api, "saveAIConversation").mockRejectedValue(new Error("AI 对话上下文格式错误"))
    const archive = vi.spyOn(api, "archiveAIConversation")

    const { wrapper } = await mountAIPage()
    await wrapper.get('button[aria-label="当前对话 的操作"]').trigger("click")
    await wrapper.get('button[role="menuitem"]').trigger("click")
    await flushPromises()

    expect(archive).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain("当前内容")
  })

  it("can collapse the conversation list and keeps that preference in this browser", async () => {
    mockReadyHarness()
    const { wrapper } = await mountAIPage()

    await wrapper.get('button[aria-label="隐藏对话记录"]').trigger("click")

    expect(wrapper.find(".ai-history-rail.is-collapsed").exists()).toBe(true)
    expect(wrapper.get('button[aria-label="展开对话记录"]').exists()).toBe(true)
    expect(globalThis.localStorage.getItem("learning-assistant:ai-history-collapsed")).toBe("true")
  })
})
