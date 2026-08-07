import { describe, expect, it } from "vitest"
import { mount } from "@vue/test-utils"
import { nextTick } from "vue"
import ErrorEditorDialog from "../components/errors/ErrorEditorDialog.vue"

describe("error editor", () => {
  it("reports unsaved content and emits the unchanged backend payload shape", async () => {
    const wrapper = mount(ErrorEditorDialog, {
      attachTo: document.body,
      props: { open: true, mode: "add", subjects: ["高数"], busy: false },
      global: { stubs: { teleport: true } },
    })
    await nextTick()
    await nextTick()
    const textareas = wrapper.findAll("textarea")
    await textareas[0].setValue("求极限 $x \\to 0$")
    await nextTick()
    expect(wrapper.emitted("dirty-change")?.flat() || []).toContain(true)
    await wrapper.get("button.ui-button--primary").trigger("click")
    expect(wrapper.emitted("save")?.[0]?.[0]).toEqual({
      subject: "高数",
      title: "",
      question: "求极限 $x \\to 0$",
      wrong: "未记录",
      correct: "未记录",
      reason: "未记录",
      tags: [],
      reason_tags: [],
    })
    wrapper.unmount()
  })
})
