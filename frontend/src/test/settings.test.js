import { describe, expect, it } from "vitest"
import { useSettings } from "../store/settings.js"

describe("appearance settings", () => {
  it("keeps the existing theme storage key and applies the dark palette", async () => {
    const settings = useSettings()
    settings.setDarkMode(true)
    await new Promise((resolve) => window.setTimeout(resolve, 30))
    expect(localStorage.getItem("studyTrackerThemeV2")).toBe("dark")
    expect(document.documentElement.dataset.theme).toBe("dark")
    settings.setDarkMode(false)
  })
})
