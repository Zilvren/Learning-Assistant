import { describe, expect, it, vi } from "vitest"
import { ApiError, api } from "../api/index.js"

describe("API errors", () => {
  it("requests the full library when a global tag index is needed", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ items: [] }), {
      status: 200, headers: { "content-type": "application/json" },
    })))

    await api.getLibraryItems({ all: true, tag: "链表" })
    expect(fetch).toHaveBeenCalledWith("/api/library/items?all=true&tag=%E9%93%BE%E8%A1%A8", expect.any(Object))
  })

  it("turns a busy data directory response into an identifiable error", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ detail: "数据目录正在被其他操作占用" }),
      { status: 503, headers: { "content-type": "application/json" } },
    )))

    await expect(api.getSubjects()).rejects.toMatchObject({
      name: "ApiError",
      status: 503,
      message: "数据目录正在被其他操作占用",
    })
    await expect(api.getSubjects()).rejects.toBeInstanceOf(ApiError)
  })

  it("retries a binary backup request only once after a 401", async () => {
// unauthorized 为当前用例准备或验证测试场景。
    const unauthorized = () => new Response(JSON.stringify({ detail: "登录已过期" }), {
      status: 401,
      headers: { "content-type": "application/json" },
    })
    const fetch = vi.fn()
      .mockResolvedValueOnce(unauthorized())
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(unauthorized())
    vi.stubGlobal("fetch", fetch)

    await expect(api.exportBackup()).rejects.toMatchObject({ status: 401 })
    expect(fetch).toHaveBeenCalledTimes(3)
    expect(fetch.mock.calls.map(([url]) => url)).toEqual([
      "/api/backup/export",
      "/api/auth/refresh",
      "/api/backup/export",
    ])
  })
})
