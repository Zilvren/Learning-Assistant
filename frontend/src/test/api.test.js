import { describe, expect, it, vi } from "vitest"
import { ApiError, api } from "../api/index.js"

describe("API errors", () => {
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
})
