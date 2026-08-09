import { describe, expect, it } from "vitest"
import { mount } from "@vue/test-utils"
import BaseDialog from "../components/ui/BaseDialog.vue"

describe("accessible dialogs", () => {
  it("gives nested dialogs independent accessible title references", () => {
    const wrapper = mount({
      components: { BaseDialog },
      template: `<div><BaseDialog open title="编辑错题">编辑器</BaseDialog><BaseDialog open title="舍弃修改">确认内容</BaseDialog></div>`,
    }, { global: { stubs: { teleport: true } } })
    const dialogs = wrapper.findAll('[role="dialog"]')
    expect(dialogs).toHaveLength(2)
    const titleIds = dialogs.map((dialog) => dialog.attributes("aria-labelledby"))
    expect(new Set(titleIds).size).toBe(2)
    expect(wrapper.get(`#${titleIds[0]}`).text()).toBe("编辑错题")
    expect(wrapper.get(`#${titleIds[1]}`).text()).toBe("舍弃修改")
  })

  it("connects the description and supplies a fallback name without a title", () => {
    const wrapper = mount(BaseDialog, {
      props: { open: true, description: "删除后无法恢复", showClose: false },
      slots: { default: "确认内容" },
      global: { stubs: { teleport: true } },
    })
    const dialog = wrapper.get('[role="dialog"]')
    const descriptionId = dialog.attributes("aria-describedby")
    expect(dialog.attributes("aria-label")).toBe("对话框")
    expect(wrapper.get(`#${descriptionId}`).text()).toBe("删除后无法恢复")
  })
})
