import { describe, expect, it, vi } from "vitest"
import { api } from "../api/index.js"
import { mapPreviewQuery, useErrorPreview } from "../composables/useErrorPreview.js"

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

describe("error preview data", () => {
  it("maps preview filters to the existing error query arguments", () => {
    expect(mapPreviewQuery()).toEqual([null, null, null, null])
    expect(mapPreviewQuery({ subject: "高数", keyword: "  极限  ", mode: "全部" }))
      .toEqual(["高数", "极限", null, null])
    expect(mapPreviewQuery({ subject: "全部", keyword: "导数", mode: "题目" }))
      .toEqual([null, "导数", null, null])
    expect(mapPreviewQuery({ subject: "数学", keyword: "计算题", mode: "题目标签" }))
      .toEqual(["数学", null, "计算题", null])
    expect(mapPreviewQuery({ subject: "数学", keyword: "粗心", mode: "错因标签" }))
      .toEqual(["数学", null, null, "粗心"])
  })

  it("uses only the two read APIs and normalizes nullable arrays", async () => {
    vi.spyOn(api, "getErrors").mockResolvedValue({
      errors: [{ id: 4, tags: null, reason_tags: null }],
    })
    vi.spyOn(api, "getSubjects").mockResolvedValue({ subjects: null })
    const mutationSpies = ["addError", "updateError", "deleteError", "reviewError"]
      .map((name) => vi.spyOn(api, name).mockResolvedValue({}))

    const preview = useErrorPreview()
    await preview.refresh({ subject: "数学", keyword: "曲线", mode: "题目" })
    await preview.loadSubjects()

    expect(api.getErrors).toHaveBeenCalledWith("数学", "曲线", null, null)
    expect(api.getSubjects).toHaveBeenCalledOnce()
    expect(preview.errors.value).toEqual([{ id: 4, tags: [], reason_tags: [] }])
    expect(preview.subjects.value).toEqual([])
    mutationSpies.forEach((spy) => expect(spy).not.toHaveBeenCalled())
  })

  it("lets only the latest refresh update data, errors and loading state", async () => {
    const oldSuccess = deferred()
    const oldFailure = deferred()
    vi.spyOn(api, "getErrors")
      .mockImplementationOnce(() => oldSuccess.promise)
      .mockImplementationOnce(() => oldFailure.promise)
      .mockResolvedValueOnce({ errors: [{ id: 3, tags: [], reason_tags: [] }] })

    const preview = useErrorPreview()
    const first = preview.refresh({ keyword: "first" })
    const second = preview.refresh({ keyword: "second" })
    await preview.refresh({ keyword: "latest" })

    expect(preview.errors.value.map((item) => item.id)).toEqual([3])
    expect(preview.loading.value).toBe(false)
    oldSuccess.resolve({ errors: [{ id: 1 }] })
    oldFailure.reject(new Error("stale failure"))
    await Promise.all([first, second])

    expect(preview.errors.value.map((item) => item.id)).toEqual([3])
    expect(preview.error.value).toBeNull()
    expect(preview.loaded.value).toBe(true)
    expect(preview.loading.value).toBe(false)
  })

  it("keeps existing records on failure and clears the error after retry", async () => {
    vi.spyOn(api, "getErrors")
      .mockResolvedValueOnce({ errors: [{ id: 1, tags: [], reason_tags: [] }] })
      .mockRejectedValueOnce(new Error("数据目录正在被占用"))
      .mockResolvedValueOnce({ errors: [{ id: 2, tags: null, reason_tags: null }] })

    const preview = useErrorPreview()
    await preview.refresh()
    await expect(preview.refresh({ keyword: "失败" })).resolves.toBeNull()

    expect(preview.errors.value.map((item) => item.id)).toEqual([1])
    expect(preview.error.value).toBeInstanceOf(Error)
    expect(preview.error.value.message).toBe("数据目录正在被占用")
    expect(preview.loaded.value).toBe(true)
    expect(preview.loading.value).toBe(false)

    await preview.refresh({ keyword: "重试" })
    expect(preview.errors.value).toEqual([{ id: 2, tags: [], reason_tags: [] }])
    expect(preview.error.value).toBeNull()
  })

  it("invalidates pending error and subject requests when disposed", async () => {
    const pendingErrors = deferred()
    const pendingSubjects = deferred()
    vi.spyOn(api, "getErrors").mockReturnValue(pendingErrors.promise)
    vi.spyOn(api, "getSubjects").mockReturnValue(pendingSubjects.promise)

    const preview = useErrorPreview()
    const errorRequest = preview.refresh()
    const subjectRequest = preview.loadSubjects()
    expect(preview.loading.value).toBe(true)
    expect(preview.subjectsLoading.value).toBe(true)

    preview.dispose()
    pendingErrors.resolve({ errors: [{ id: 9 }] })
    pendingSubjects.resolve({ subjects: ["数学"] })
    await Promise.all([errorRequest, subjectRequest])

    expect(preview.loading.value).toBe(false)
    expect(preview.subjectsLoading.value).toBe(false)
    expect(preview.loaded.value).toBe(false)
    expect(preview.errors.value).toEqual([])
    expect(preview.subjects.value).toEqual([])
    expect(preview.error.value).toBeNull()
    expect(preview.subjectsError.value).toBeNull()
  })
})
