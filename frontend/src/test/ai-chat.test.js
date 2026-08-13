import { afterEach, describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AIChatPage from "../components/AIChatPage.vue"
import { api } from "../api/index.js"

describe("AI library scope", () => {
	afterEach(() => vi.restoreAllMocks())

  it("indexes a selected path and sends forwarded items with the question", async () => {
    vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [] })
    const saveConversation = vi.spyOn(api, "saveAIConversation").mockResolvedValue({ conversations: [] })
    vi.spyOn(api, "getLibraryItems").mockImplementation(({ parentId, kind }) => {
      if (kind === "folder" && parentId == null) return Promise.resolve({ items: [{ id: 4, kind: "folder", name: "数学" }] })
      return Promise.resolve({ items: [] })
    })
    vi.spyOn(api, "getLibraryItem").mockResolvedValue({ id: 7, kind: "note", name: "导数专题" })
    const chat = vi.spyOn(api, "aiChat").mockResolvedValue({ answer: "先复习导数符号表。", model: "deepseek-v4-flash", sources: [] })
    const router = createRouter({ history: createMemoryHistory(), routes: [
      { path: "/ai", name: "ai", component: AIChatPage },
      { path: "/settings", name: "settings", component: { template: "<div />" } },
      { path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
    ] })
    await router.push("/ai?folder=4&items=7")
    await router.isReady()
    const wrapper = mount(AIChatPage, { global: { plugins: [router], stubs: { PageHeader: { template: "<div><slot /><slot name='actions' /></div>" } } } })
    await flushPromises()

    const scopeButton = wrapper.get('.ai-composer__scope-button')
    expect(scopeButton.attributes("aria-expanded")).toBe("false")
    await scopeButton.trigger("click")
    expect(wrapper.text()).toContain("数学")
    expect(wrapper.text()).toContain("导数专题")
    const pathTrigger = wrapper.get(".ai-path-combobox__trigger")
    expect(pathTrigger.attributes("aria-expanded")).toBe("false")
    await pathTrigger.trigger("click")
    expect(wrapper.get('[role="listbox"]').text()).toContain("数学")
    expect(wrapper.get('[role="option"][aria-selected="true"]').text()).toContain("数学")
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }))
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[role="listbox"]').exists()).toBe(false)
    await wrapper.get("textarea").setValue("请总结这份资料")
    await wrapper.get("form").trigger("submit")
    await flushPromises()

    expect(chat).toHaveBeenCalledWith({ message: "请总结这份资料", history: [], folder_id: 4, item_ids: [7] })
    expect(saveConversation).toHaveBeenCalledWith(expect.arrayContaining([
      expect.objectContaining({
        folder_id: 4,
        item_ids: [7],
        messages: expect.arrayContaining([
          expect.objectContaining({ role: "user", content: "请总结这份资料", scope: expect.stringContaining("路径：数学") }),
          expect.objectContaining({ role: "assistant", content: "先复习导数符号表。", model: "deepseek-v4-flash" }),
        ]),
      }),
    ]))
    expect(wrapper.text()).toContain("先复习导数符号表")
  })

	it("restores saved conversation as the next request context", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [{
			id: "derivative-review",
			title: "导数复习",
			folder_id: null,
			item_ids: [],
			messages: [
				{ role: "user", content: "我在导数上总出错" },
				{ role: "assistant", content: "先从导数符号表开始复习。", model: "deepseek-v4-flash" },
			],
		}] })
		vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
		vi.spyOn(api, "saveAIConversation").mockResolvedValue({ conversations: [] })
		const chat = vi.spyOn(api, "aiChat").mockResolvedValue({ answer: "再做两道单调性题。", model: "deepseek-v4-flash", sources: [] })
		const router = createRouter({ history: createMemoryHistory(), routes: [
			{ path: "/ai", name: "ai", component: AIChatPage },
			{ path: "/settings", name: "settings", component: { template: "<div />" } },
			{ path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
		] })
		await router.push("/ai")
		await router.isReady()
		const wrapper = mount(AIChatPage, { global: { plugins: [router], stubs: { PageHeader: { template: "<div><slot /><slot name='actions' /></div>" } } } })
		await flushPromises()

		expect(wrapper.text()).toContain("我在导数上总出错")
		expect(wrapper.text()).toContain("先从导数符号表开始复习")
		await wrapper.get("textarea").setValue("下一步怎么练？")
		await wrapper.get("form").trigger("submit")
		await flushPromises()

		expect(chat).toHaveBeenCalledWith({
			message: "下一步怎么练？",
			history: [
				{ role: "user", content: "我在导数上总出错" },
				{ role: "assistant", content: "先从导数符号表开始复习。" },
			],
			folder_id: null,
			item_ids: [],
		})
	})

	it("keeps auto-compacted memory while excluding compacted messages from the next request", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [{
			id: "long-study-chat",
			title: "长期复习",
			folder_id: null,
			item_ids: [],
			messages: [
				{ role: "user", content: "我在第一章学了什么？" },
				{ role: "assistant", content: "你完成了函数基础。" },
			],
		}] })
		vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
		const saveConversation = vi.spyOn(api, "saveAIConversation").mockImplementation((payload) => Promise.resolve({ conversations: payload }))
		const chat = vi.spyOn(api, "aiChat")
			.mockResolvedValueOnce({ answer: "接着复习导数。", model: "deepseek-v4-flash", sources: [], context_summary: "已完成函数基础。", compacted_messages: 1 })
			.mockResolvedValueOnce({ answer: "再做两道题。", model: "deepseek-v4-flash", sources: [] })
		const router = createRouter({ history: createMemoryHistory(), routes: [
			{ path: "/ai", name: "ai", component: AIChatPage },
			{ path: "/settings", name: "settings", component: { template: "<div />" } },
			{ path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
		] })
		await router.push("/ai?conversation=long-study-chat")
		await router.isReady()
		const wrapper = mount(AIChatPage, { global: { plugins: [router] } })
		await flushPromises()

		await wrapper.get("textarea").setValue("下一阶段怎么安排？")
		await wrapper.get("form").trigger("submit")
		await flushPromises()
		expect(saveConversation).toHaveBeenLastCalledWith(expect.arrayContaining([
			expect.objectContaining({
				context_summary: "已完成函数基础。",
				messages: expect.arrayContaining([expect.objectContaining({ content: "我在第一章学了什么？", context_compacted: true })]),
			}),
		]))

		await wrapper.get("textarea").setValue("再具体一点")
		await wrapper.get("form").trigger("submit")
		await flushPromises()
		expect(chat.mock.calls[1][0]).toMatchObject({
			context_summary: "已完成函数基础。",
			history: [
				{ role: "assistant", content: "你完成了函数基础。" },
				{ role: "user", content: "下一阶段怎么安排？" },
				{ role: "assistant", content: "接着复习导数。" },
			],
		})
	})

	it("switches between independent conversations without mixing their context or scope", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [
			{
				id: "algebra-chat",
				title: "代数错题",
				folder_id: null,
				item_ids: [],
				messages: [{ role: "user", content: "帮我看看这个方程" }, { role: "assistant", content: "先移项再合并同类项。" }],
			},
			{
				id: "geometry-chat",
				title: "几何证明",
				folder_id: 9,
				item_ids: [],
				messages: [{ role: "user", content: "三角形全等怎么证？" }, { role: "assistant", content: "先找对应边角条件。" }],
			},
		] })
		vi.spyOn(api, "getLibraryItems").mockImplementation(({ parentId, kind }) => {
			if (kind === "folder" && parentId == null) return Promise.resolve({ items: [{ id: 9, kind: "folder", name: "几何" }] })
			return Promise.resolve({ items: [] })
		})
		const saveConversation = vi.spyOn(api, "saveAIConversation").mockImplementation((payload) => Promise.resolve({ conversations: payload }))
		const chat = vi.spyOn(api, "aiChat").mockResolvedValue({ answer: "再画一张辅助线。", model: "deepseek-v4-flash", sources: [] })
		const router = createRouter({ history: createMemoryHistory(), routes: [
			{ path: "/ai", name: "ai", component: AIChatPage },
			{ path: "/settings", name: "settings", component: { template: "<div />" } },
			{ path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
		] })
		await router.push("/ai?conversation=algebra-chat")
		await router.isReady()
		const wrapper = mount(AIChatPage, { global: { plugins: [router] } })
		await flushPromises()

		expect(wrapper.text()).toContain("帮我看看这个方程")
		await wrapper.findAll(".ai-history-item").find((item) => item.text().includes("几何证明")).trigger("click")
		await flushPromises()
		expect(wrapper.text()).toContain("三角形全等怎么证？")
		expect(wrapper.text()).not.toContain("帮我看看这个方程")
		await wrapper.get("textarea").setValue("下一步呢？")
		await wrapper.get("form").trigger("submit")
		await flushPromises()

		expect(chat).toHaveBeenCalledWith({
			message: "下一步呢？",
			history: [
				{ role: "user", content: "三角形全等怎么证？" },
				{ role: "assistant", content: "先找对应边角条件。" },
			],
			folder_id: 9,
			item_ids: [],
		})
		expect(saveConversation).toHaveBeenLastCalledWith(expect.arrayContaining([
			expect.objectContaining({ id: "algebra-chat", messages: expect.arrayContaining([expect.objectContaining({ content: "帮我看看这个方程" })]) }),
			expect.objectContaining({ id: "geometry-chat", messages: expect.arrayContaining([expect.objectContaining({ content: "下一步呢？" })]) }),
		]))
	})

	it("offers continuation when DeepSeek reaches the completion limit", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ conversations: [] })
		vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
		vi.spyOn(api, "saveAIConversation").mockResolvedValue({ conversations: [] })
		const chat = vi.spyOn(api, "aiChat")
			.mockResolvedValueOnce({ answer: "目录的第一部分", model: "deepseek-v4-flash", sources: [], incomplete: true })
			.mockResolvedValueOnce({ answer: "目录的后续部分", model: "deepseek-v4-flash", sources: [], incomplete: false })
		const router = createRouter({ history: createMemoryHistory(), routes: [
			{ path: "/ai", name: "ai", component: AIChatPage },
			{ path: "/settings", name: "settings", component: { template: "<div />" } },
			{ path: "/library/items/:itemId", name: "library-item", component: { template: "<div />" } },
		] })
		await router.push("/ai")
		await router.isReady()
		const wrapper = mount(AIChatPage, { global: { plugins: [router], stubs: { PageHeader: { template: "<div><slot /><slot name='actions' /></div>" } } } })
		await flushPromises()

		await wrapper.get("textarea").setValue("整理全部资料")
		await wrapper.get("form").trigger("submit")
		await flushPromises()
		expect(wrapper.text()).toContain("回答达到长度上限")
		await wrapper.get(".ai-message__continuation button").trigger("click")
		await flushPromises()

		expect(chat).toHaveBeenCalledTimes(2)
		expect(chat.mock.calls[1][0]).toMatchObject({
			folder_id: null,
			item_ids: [],
			history: [
				{ role: "user", content: "整理全部资料" },
				{ role: "assistant", content: "目录的第一部分" },
			],
		})
		expect(wrapper.text()).not.toContain("回答达到长度上限")
		expect(wrapper.text()).toContain("目录的后续部分")
	})
})
