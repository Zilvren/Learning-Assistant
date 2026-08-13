import { afterEach, describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AIChatPage from "../components/AIChatPage.vue"
import { api } from "../api/index.js"

describe("AI library scope", () => {
	afterEach(() => vi.restoreAllMocks())

  it("indexes a selected path and sends forwarded items with the question", async () => {
    vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
    vi.spyOn(api, "getAIConversation").mockResolvedValue({ messages: [] })
    const saveConversation = vi.spyOn(api, "saveAIConversation").mockResolvedValue({ messages: [] })
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
      expect.objectContaining({ role: "user", content: "请总结这份资料", scope: expect.stringContaining("路径：数学") }),
      expect.objectContaining({ role: "assistant", content: "先复习导数符号表。", model: "deepseek-v4-flash" }),
    ]))
    expect(wrapper.text()).toContain("先复习导数符号表")
  })

	it("restores saved conversation as the next request context", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ messages: [
			{ role: "user", content: "我在导数上总出错" },
			{ role: "assistant", content: "先从导数符号表开始复习。", model: "deepseek-v4-flash" },
		] })
		vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
		vi.spyOn(api, "saveAIConversation").mockResolvedValue({ messages: [] })
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

	it("offers continuation when DeepSeek reaches the completion limit", async () => {
		vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
		vi.spyOn(api, "getAIConversation").mockResolvedValue({ messages: [] })
		vi.spyOn(api, "getLibraryItems").mockResolvedValue({ items: [] })
		vi.spyOn(api, "saveAIConversation").mockResolvedValue({ messages: [] })
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
