import { mount } from "@vue/test-utils"
import { describe, expect, it } from "vitest"
import LearningHeatmap from "../components/dashboard/LearningHeatmap.vue"

describe("learning heatmap", () => {
  const activity = {
    start_date: "2025-01-01",
    end_date: "2025-12-31",
    total: 8,
    active_days: 2,
    available_years: [2025, 2024],
    days: [{ date: "2025-01-03", count: 3 }, { date: "2025-08-07", count: 5 }],
  }

  it("renders a full year and emits a new year selection", async () => {
    const wrapper = mount(LearningHeatmap, { props: { activity, selectedYear: 2025 } })

    expect(wrapper.get("h2").text()).toBe("学习记录")
    expect(wrapper.get(".learning-heatmap__frame-head").text()).toContain("2025 年全年学习活动")
    expect(wrapper.findAll(".learning-heatmap__cell").length).toBeGreaterThanOrEqual(365)
    expect(wrapper.get('.learning-heatmap__year-picker button[aria-pressed="true"]').text()).toBe("2025")

    const previousYear = wrapper.findAll(".learning-heatmap__year-picker button").find((button) => button.text() === "2024")
    await previousYear.trigger("click")

    expect(wrapper.emitted("select-year")).toEqual([[2024]])
  })

  it("uses the same month and day as the default right-edge target in every year", () => {
    const now = new Date()
    const currentYear = now.getFullYear()
    const previousYear = currentYear - 1
    const day = Math.min(now.getDate(), new Date(previousYear, now.getMonth() + 1, 0).getDate())
    const anchor = `${previousYear}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(day).padStart(2, "0")}`
    const wrapper = mount(LearningHeatmap, {
      props: { activity: { ...activity, start_date: `${previousYear}-01-01`, end_date: `${previousYear}-12-31` }, selectedYear: previousYear },
    })

    expect(wrapper.get(`[data-activity-date="${anchor}"]`)).toBeTruthy()
  })
})
