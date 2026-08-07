import { describe, expect, it, vi } from "vitest"
import { api } from "../api/index.js"
import { isDue, reviewLabel, useErrorLibrary } from "../composables/useErrorLibrary.js"

describe("error library", () => {
  it("maps all-search and tag-search filters to the existing API contract", async () => {
    const spy = vi.spyOn(api, "getErrors").mockResolvedValue({ errors: [{ id: 1 }] })
    const library = useErrorLibrary()
    await library.refresh({ subject: "高数", keyword: "极限", mode: "全部" })
    expect(spy).toHaveBeenLastCalledWith("高数", "极限", null, null)
    await library.refresh({ subject: "全部", keyword: "粗心", mode: "错因标签" })
    expect(spy).toHaveBeenLastCalledWith(null, null, null, "粗心")
    expect(library.errors.value).toEqual([{ id: 1 }])
  })

  it("delegates create, edit, review and delete without changing payloads", async () => {
    const payload = { subject: "英语", question: "Q" }
    vi.spyOn(api, "addError").mockResolvedValue({ id: 9 })
    vi.spyOn(api, "updateError").mockResolvedValue({ ok: true })
    vi.spyOn(api, "reviewError").mockResolvedValue({ next_review: "2026-08-02" })
    vi.spyOn(api, "deleteError").mockResolvedValue({ ok: true })
    const library = useErrorLibrary()
    await library.create(payload)
    await library.update(9, payload)
    await library.review(9)
    await library.remove(9)
    expect(api.addError).toHaveBeenCalledWith(payload)
    expect(api.updateError).toHaveBeenCalledWith(9, payload)
    expect(api.reviewError).toHaveBeenCalledWith(9)
    expect(api.deleteError).toHaveBeenCalledWith(9)
  })

  it("labels due and overdue review rounds", () => {
    expect(isDue({ next_review: "2026-08-01" }, "2026-08-01")).toBe(true)
    expect(reviewLabel({ next_review: "2026-07-30", review_count: 2 }, "2026-08-01")).toContain("第 3 轮")
  })
})
