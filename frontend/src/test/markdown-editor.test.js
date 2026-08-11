import { describe, expect, it, vi } from "vitest"
import { mount } from "@vue/test-utils"
import { nextTick } from "vue"
import MarkdownEditor from "../components/MarkdownEditor.vue"
import { renderMd } from "../utils/markdown.js"

describe("MarkdownEditor", () => {
  it("embeds a selected image as base64 Markdown", async () => {
    class FakeFileReader {
      readAsDataURL() {
        this.result = "data:image/png;base64,cGl4ZWw="
        this.onload()
      }
    }
    vi.stubGlobal("FileReader", FakeFileReader)

    const wrapper = mount(MarkdownEditor, { props: { modelValue: "前言" } })
    const textarea = wrapper.get("textarea")
    await textarea.trigger("focus")
    textarea.element.setSelectionRange(2, 2)

    const input = wrapper.get('input[type="file"]')
    const file = new File(["pixel"], "说明.png", { type: "image/png" })
    Object.defineProperty(input.element, "files", { configurable: true, value: [file] })
    await input.trigger("change")
    await nextTick()

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe('前言![说明](data:image/png;base64,cGl4ZWw= "width=400;align=left")')
    expect(wrapper.get('button[aria-label="插入图片"]').attributes("disabled")).toBeUndefined()
    vi.unstubAllGlobals()
  })

  it("embeds an image pasted from the clipboard at the cursor", async () => {
    class FakeFileReader {
      readAsDataURL() {
        this.result = "data:image/png;base64,cGl4ZWw="
        this.onload()
      }
    }
    vi.stubGlobal("FileReader", FakeFileReader)

    const wrapper = mount(MarkdownEditor, { props: { modelValue: "笔记：" } })
    const textarea = wrapper.get("textarea")
    await textarea.trigger("focus")
    textarea.element.setSelectionRange(3, 3)
    const file = new File(["pixel"], "", { type: "image/png" })
    await textarea.trigger("paste", { clipboardData: { items: [{ kind: "file", type: "image/png", getAsFile: () => file }] } })
    await nextTick()

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe('笔记：![剪贴板图片](data:image/png;base64,cGl4ZWw= "width=400;align=left")')
    vi.unstubAllGlobals()
  })

  it("rejects image types that cannot be safely embedded", async () => {
    const wrapper = mount(MarkdownEditor, { props: { modelValue: "" } })
    const input = wrapper.get('input[type="file"]')
    const file = new File(["<svg />"], "unsafe.svg", { type: "image/svg+xml" })
    Object.defineProperty(input.element, "files", { configurable: true, value: [file] })
    await input.trigger("change")

    expect(wrapper.emitted("update:modelValue")).toBeUndefined()
    expect(wrapper.text()).toContain("仅支持 PNG、JPG、GIF 或 WebP 图片")
  })

  it("shows embedded images as controls that save the edited name and width", async () => {
    const source = '开始\n![原始名称](data:image/png;base64,cGl4ZWw= "width=400")\n结束'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    expect(wrapper.find("textarea.md-textarea:not(.md-text-segment)").exists()).toBe(false)
    expect(wrapper.text()).not.toContain("<<IMG")
    expect(wrapper.get(".md-textarea--visual").attributes("role")).toBe("group")
    expect(wrapper.get(".md-text-segment").element.tagName).toBe("TEXTAREA")
    const imageNode = wrapper.get(".md-image-control")
    expect(imageNode.attributes("class")).not.toContain("is-active")
    const imageButton = wrapper.get('button[aria-label="编辑图片 1"]')
    expect(wrapper.findAll(".md-text-segment").length).toBeGreaterThan(0)
    await imageButton.trigger("click")
    expect(wrapper.get(".md-image-control").attributes("class")).toContain("is-active")
    expect(wrapper.get(".md-image-control__name").element.value).toBe("原始名称")

    const nameInput = wrapper.get(".md-image-control__name")
    await nameInput.setValue("获奖证书")
    await nameInput.trigger("blur")
    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toContain('![获奖证书](data:image/png;base64,cGl4ZWw= "width=400;align=left")')

    await wrapper.get('button[aria-label="图片 1：大"]').trigger("click")
    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toContain('![获奖证书](data:image/png;base64,cGl4ZWw= "width=640;align=left")')
    expect(renderMd('![获奖证书](data:image/png;base64,cGl4ZWw= "width=640;align=center")')).toContain('class="markdown-image--align-center"')
  })

  it("keeps image data intact while text is edited around its control", async () => {
    const source = '开始\n![证书](data:image/png;base64,cGl4ZWw= "width=400")\n结束'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })
    const openingText = wrapper.findAll(".md-text-segment").find((segment) => segment.element.value.includes("开始"))
    await openingText.setValue("更新的开始\n")

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe('更新的开始\n![证书](data:image/png;base64,cGl4ZWw= "width=400")\n结束')
  })

  it("sizes text blocks beside an image by their rendered content height", async () => {
    const source = '开始\n![证书](data:image/png;base64,cGl4ZWw= "width=400")\n结束'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    const firstTextBlock = wrapper.get('.md-text-segment')
    Object.defineProperty(firstTextBlock.element, "scrollHeight", { configurable: true, value: 212 })
    await firstTextBlock.setValue('一\n二\n三\n四\n五\n六\n七\n八')

    expect(firstTextBlock.element.style.height).toBe('212px')
    expect(wrapper.find('input[aria-label*="高度"]').exists()).toBe(false)
  })

  it("keeps an editable text anchor after a final image node", async () => {
    const source = '![证书](data:image/png;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })
    const anchor = wrapper.get(".md-text-segment")
    await anchor.setValue("图片旁的说明")

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe(`${source}图片旁的说明`)
  })

  it("adds an editable text line above or below an image from its controls", async () => {
    const source = '![证书](data:image/png;base64,cGl4ZWw= "width=400")'
    const above = mount(MarkdownEditor, { props: { modelValue: source } })
    await above.get('button[aria-label="编辑图片 1"]').trigger("click")
    await above.get('button[aria-label="在图片 1 上方添加文字"]').trigger("click")
    expect(above.emitted("update:modelValue")?.at(-1)?.[0]).toBe(` ${source}`)
    expect(above.get(".md-text-segment").element.value).toBe(" ")

    const below = mount(MarkdownEditor, { props: { modelValue: source } })
    await below.get('button[aria-label="编辑图片 1"]').trigger("click")
    await below.get('button[aria-label="在图片 1 下方添加文字"]').trigger("click")
    expect(below.emitted("update:modelValue")?.at(-1)?.[0]).toBe(`${source} `)
    expect(below.get(".md-text-segment").element.value).toBe(" ")
  })

  it("restores image controls when a reused note page returns to an image note", async () => {
    const source = '![证书](data:image/png;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    await wrapper.setProps({ modelValue: "" })
    expect(wrapper.find("textarea.md-textarea:not(.md-text-segment)").exists()).toBe(true)

    await wrapper.setProps({ modelValue: source })
    expect(wrapper.find("textarea.md-textarea:not(.md-text-segment)").exists()).toBe(false)
    expect(wrapper.get('button[aria-label="编辑图片 1"]').exists()).toBe(true)
  })

  it("aligns an active image square using the toolbar", async () => {
    const source = '![证书](data:image/png;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })
    await wrapper.get('button[aria-label="编辑图片 1"]').trigger("click")
    await wrapper.get('button[aria-label="居中"]').trigger("click")

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toContain('"width=400;align=center"')
  })

  it("deletes an active image from its visible action or keyboard", async () => {
    const source = '![证书](data:image/png;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })
    const imageButton = wrapper.get('button[aria-label="编辑图片 1"]')

    await imageButton.trigger("click")
    await wrapper.get('button[aria-label="删除图片 1"]').trigger("click")
    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe("")

    await wrapper.setProps({ modelValue: source })
    await imageButton.trigger("keydown", { key: "Delete" })
    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe("")
  })

  it("keeps remaining image controls visible after a deletion", async () => {
    // image 完成当前模块的局部交互。
    const image = (name) => `![${name}](data:image/png;base64,cGl4ZWw= "width=400")`
    const source = `${image("一")}\n${image("二")}\n${image("三")}`
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    await wrapper.get('button[aria-label="编辑图片 2"]').trigger("click")
    await wrapper.get('button[aria-label="删除图片 2"]').trigger("click")
    await nextTick()

    expect(wrapper.findAll(".md-image-control")).toHaveLength(2)
    expect(wrapper.get('button[aria-label="编辑图片 1"]').exists()).toBe(true)
    expect(wrapper.get('button[aria-label="编辑图片 2"]').exists()).toBe(true)
  })

  it("wraps selected text as an aligned block", async () => {
    const wrapper = mount(MarkdownEditor, { props: { modelValue: "学习记录" } })
    const textarea = wrapper.get("textarea")
    await textarea.trigger("focus")
    textarea.element.setSelectionRange(0, 4)

    await wrapper.get('button[aria-label="右对齐"]').trigger("click")

    expect(wrapper.emitted("update:modelValue")?.at(-1)?.[0]).toBe("[[align:right]]\n学习记录\n[[/align]]")
    expect(renderMd("[[align:right]]\n学习记录\n[[/align]]")).toContain('class="markdown-align markdown-align--right"')
  })

  it("never exposes a data URL as the editable image name", async () => {
    const source = '![data:image/jpeg;base64,not-a-name](data:image/png;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    const imageButton = wrapper.get('button[aria-label="编辑图片 1"]')
    await imageButton.trigger("click")
    expect(wrapper.get(".md-image-control__name").element.value).toBe("图片")
    expect(wrapper.text()).not.toContain("data:image/png;base64")
  })

  it("recognizes legacy image/jpg data URLs as image controls", () => {
    const source = '![旧图片](data:image/jpg;base64,cGl4ZWw= "width=400")'
    const wrapper = mount(MarkdownEditor, { props: { modelValue: source } })

    expect(wrapper.find("textarea.md-textarea:not(.md-text-segment)").exists()).toBe(false)
    expect(wrapper.get('button[aria-label="编辑图片 1"]').exists()).toBe(true)
  })
})
