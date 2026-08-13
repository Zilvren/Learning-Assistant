import { describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import AIChatPage from "../components/AIChatPage.vue"
import { api } from "../api/index.js"

describe("AI library scope", () => {
  it("indexes a selected path and sends forwarded items with the question", async () => {
    vi.spyOn(api, "getDeepSeekToken").mockResolvedValue({ configured: true })
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
    await wrapper.get("textarea").setValue("请总结这份资料")
    await wrapper.get("form").trigger("submit")
    await flushPromises()

    expect(chat).toHaveBeenCalledWith({ message: "请总结这份资料", history: [], folder_id: 4, item_ids: [7] })
    expect(wrapper.text()).toContain("先复习导数符号表")
  })
})
