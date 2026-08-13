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

  it("persists a selected color palette without affecting the light or dark mode", async () => {
    const settings = useSettings()
    settings.setDarkMode(true)
    settings.setColorTheme("sunset")
    await new Promise((resolve) => window.setTimeout(resolve, 30))
    expect(localStorage.getItem("studyTrackerColorThemeV1")).toBe("sunset")
    expect(document.documentElement.dataset.palette).toBe("sunset")
    expect(document.documentElement.dataset.theme).toBe("dark")
    settings.setColorTheme("verdant")
    settings.setDarkMode(false)
  })
})
