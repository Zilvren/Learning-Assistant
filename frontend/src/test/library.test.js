import { describe, expect, it, vi } from "vitest"
import { flushPromises, mount } from "@vue/test-utils"
import { createMemoryHistory, createRouter } from "vue-router"
import LibraryPage from "../components/library/LibraryPage.vue"
import LibraryItemPage from "../components/library/LibraryItemPage.vue"
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
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[{id:2,kind:"folder",name:"课程笔记",tags:[],created_at:"2026-08-01T09:30:00+08:00",updated_at:"2026-08-02"}]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/:folderId?",name:"library",component:LibraryPage},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/library");await router.isReady()
    const wrapper=mount(LibraryPage,{global:{plugins:[router],stubs:{BaseDialog:true}}});await flushPromises()
    expect(wrapper.text()).toContain("课程笔记")
    expect(wrapper.get(".library-card__created").text()).toContain("08/01")
    await wrapper.get(".library-card__body").trigger("click");await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/library/2")
  })

  it("opens a trashed folder inside the trash route", async () => {
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[{id:5,kind:"folder",name:"Note",tags:[],deleted_at:"2026-08-08T10:00:00+08:00"}]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/trash/:folderId?",name:"trash",component:LibraryPage,props:route=>({trash:true,folderId:route.params.folderId})},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/trash");await router.isReady()
    const wrapper=mount(LibraryPage,{props:{trash:true},global:{plugins:[router],stubs:{BaseDialog:true}}});await flushPromises()
    await wrapper.get(".library-card__body").trigger("click");await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/trash/5")
  })

  it("returns to the root when a saved folder URL no longer exists", async () => {
    vi.useFakeTimers()
    vi.spyOn(api,"getLibraryItem").mockRejectedValue(new Error("not found"))
    vi.spyOn(api,"getLibraryItems").mockResolvedValue({items:[{id:4724,kind:"folder",name:"daily",tags:[]}]})
    vi.spyOn(api,"getLibraryTags").mockResolvedValue({tags:[]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/:folderId?",name:"library",component:LibraryPage},{path:"/library/items/:itemId",name:"library-item",component:{template:"<div/>"}}]})
    await router.push("/library/3");await router.isReady()
    const wrapper=mount(LibraryPage,{global:{plugins:[router],stubs:{BaseDialog:true}}})
    await flushPromises()
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    expect(router.currentRoute.value.fullPath).toBe("/library")
    expect(wrapper.text()).toContain("daily")
    vi.useRealTimers()
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

  it("saves the first edit made to a newly created empty note", async () => {
    vi.useFakeTimers()
    vi.spyOn(api,"getLibraryItem").mockResolvedValue({id:42,kind:"note",name:"未命名笔记",tags:[],current_version:1})
    vi.spyOn(api,"getLibraryContent").mockResolvedValue({content:"",version:1})
    const save=vi.spyOn(api,"saveLibraryContent").mockResolvedValue({id:42,kind:"note",name:"未命名笔记",current_version:2,tags:[]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/items/:itemId",name:"library-item",component:LibraryItemPage,props:true},{path:"/library",name:"library",component:{template:"<div/>"}}]})
    await router.push("/library/items/42");await router.isReady()
    const wrapper=mount({template:"<router-view/>"},{global:{plugins:[router],stubs:{MarkdownEditor:{props:["modelValue"],emits:["update:modelValue"],template:'<button data-test="first-edit" @click="$emit(\'update:modelValue\', \'第一条学习记录\')"/>'},MarkdownRenderer:true}}})
    await flushPromises()

    await wrapper.findAll(".item-head-actions .lib-btn")[1].trigger("click")
    await wrapper.get('[data-test="first-edit"]').trigger("click")
    await vi.advanceTimersByTimeAsync(800)

    expect(save).toHaveBeenCalledWith("42",{content:"第一条学习记录",base_version:1,checkpoint:false,force:false})
    vi.useRealTimers()
  })

  it("persists text typed while an earlier autosave is still in flight", async () => {
    vi.useFakeTimers()
    vi.spyOn(api,"getLibraryItem").mockResolvedValue({id:42,kind:"note",name:"并发保存",tags:[],current_version:1})
    vi.spyOn(api,"getLibraryContent").mockResolvedValue({content:"",version:1})
    let resolveFirst
    const firstSave = new Promise((resolve) => { resolveFirst = resolve })
    const save = vi.spyOn(api,"saveLibraryContent")
      .mockReturnValueOnce(firstSave)
      .mockResolvedValueOnce({id:42,kind:"note",name:"并发保存",current_version:3,tags:[]})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:"/library/items/:itemId",name:"library-item",component:LibraryItemPage,props:true}]})
    await router.push("/library/items/42");await router.isReady()
    const wrapper=mount({template:"<router-view/>"},{global:{plugins:[router],stubs:{MarkdownEditor:{props:["modelValue"],emits:["update:modelValue"],template:'<div><button data-test="edit-a" @click="$emit(\'update:modelValue\', \'A\')"/><button data-test="edit-b" @click="$emit(\'update:modelValue\', \'B\')"/></div>'},MarkdownRenderer:true}}})
    await flushPromises()
    await wrapper.get(".item-head-actions .lib-btn:nth-child(2)").trigger("click")
    await wrapper.get('[data-test="edit-a"]').trigger("click")
    await vi.advanceTimersByTimeAsync(800)
    await wrapper.get('[data-test="edit-b"]').trigger("click")
    resolveFirst({id:42,kind:"note",name:"并发保存",current_version:2,tags:[]})
    await flushPromises()
    expect(save).toHaveBeenNthCalledWith(2,"42",{content:"B",base_version:2,checkpoint:false,force:false})
    vi.useRealTimers()
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
