import { describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import LibraryPage from "../components/library/LibraryPage.vue"
import { api } from "../api/index.js"
import { extractOutline, renderMd } from "../utils/markdown.js"
import { rememberLibraryPath, rememberedLibraryPath } from "../utils/libraryPath.js"

describe("personal library", () => {
  it("remembers the last opened library folder", () => {
    localStorage.removeItem("library:last-path")
    rememberLibraryPath(42)
    expect(rememberedLibraryPath()).toBe("/library/42")
    rememberLibraryPath(null)
    expect(rememberedLibraryPath()).toBe("/library")
  })

  it("shows folders and opens them through the library route", async () => {
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[{id:2,kind:"folder",name:"课程笔记",tags:[],updated_at:"2026-08-02"}]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/:folderId?",name:"library",component:LibraryPage},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/library");await router.isReady()
    const wrapper=mount(LibraryPage,{global:{plugins:[router],stubs:{BaseDialog:true}}});await flushPromises()
    expect(wrapper.text()).toContain("课程笔记")
    await wrapper.get(".library-card__body").trigger("click");await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/library/2")
  })

  it("loads a Markdown preview only when a note card is hovered", async () => {
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[{id:9,kind:"note",name:"Redis学习.md",size:128,updated_at:"2026-08-02"}]})
    const content=vi.spyOn(api,"getLibraryContent").mockResolvedValue({content:"# Redis\n\n缓存学习笔记",version:1})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/:folderId?",name:"library",component:LibraryPage},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/library");await router.isReady()
    const wrapper=mount(LibraryPage,{global:{plugins:[router],stubs:{BaseDialog:true}}});await flushPromises()
    expect(content).not.toHaveBeenCalled()
    await wrapper.get(".library-card").trigger("mouseenter");await flushPromises()
    expect(content).toHaveBeenCalledWith(9)
    expect(wrapper.get(".library-card__back").text()).toContain("缓存学习笔记")
    expect(wrapper.find('input[type="file"]').attributes("hidden")).toBeDefined()
  })

  it("selects whole cards and can select every visible item", async () => {
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[
      {id:11,kind:"note",name:"笔记一.md",updated_at:"2026-08-02"},
      {id:12,kind:"note",name:"笔记二.md",updated_at:"2026-08-02"},
      {id:13,kind:"folder",name:"课程",updated_at:"2026-08-02"},
    ]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/:folderId?",name:"library",component:LibraryPage},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/library");await router.isReady()
    const wrapper=mount(LibraryPage,{global:{plugins:[router],stubs:{BaseDialog:true}}});await flushPromises()
    const cards=wrapper.findAll(".library-card")
    expect(cards[0].attributes("style")).toContain("--library-card-enter-delay: 0ms")
    expect(cards[1].attributes("style")).toContain("--library-card-enter-delay: 42ms")
    await wrapper.findAll('input[type="checkbox"]')[0].trigger("click")
    await wrapper.findAll(".library-card")[1].trigger("click")
    expect(wrapper.find(".library-selection-bar").text()).toContain("已选择 2 项")
    await wrapper.get(".library-selection-bar .lib-btn").trigger("click")
    expect(wrapper.find(".library-selection-bar").text()).toContain("已选择 3 项")
  })

  it("syntax-highlights fenced code without exposing raw HTML", () => {
    const html=renderMd('```c\ntypedef struct Node {\n  // next node\n  int value;\n} Node;\n```')
    expect(html).toContain('hljs-keyword')
    expect(html).toContain('hljs-comment')
    expect(html).toContain('hljs-type')
    expect(html).not.toContain('<script>')
  })

  it("builds stable outline anchors for rendered Markdown headings", () => {
    const source='# Redis\n\n## 链表实现\n\n正文'
    const outline=extractOutline(source)
    const html=renderMd(source)
    expect(outline.map((item)=>item.text)).toEqual(['Redis','链表实现'])
    expect(html).toContain('data-outline-index="1"')
    expect(html).toContain(`id="${outline[1].id}"`)
  })
})
