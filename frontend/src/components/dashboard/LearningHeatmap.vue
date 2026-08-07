<script setup>
import { computed, ref } from "vue"
import { CalendarDays, Flame, Sparkles } from "lucide-vue-next"

const props = defineProps({
  activity: { type: Object, default: () => ({ days: [] }) },
})

const dayMs = 24 * 60 * 60 * 1000

function parseDate(value) {
  if (!value) return null
  const [year, month, day] = value.split("-").map(Number)
  return Number.isFinite(year) && Number.isFinite(month) && Number.isFinite(day) ? new Date(year, month - 1, day) : null
}

function isoDate(date) {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, "0")
  const day = String(date.getDate()).padStart(2, "0")
  return `${year}-${month}-${day}`
}

function addDays(date, days) {
  const next = new Date(date)
  next.setDate(next.getDate() + days)
  return next
}

const startDate = computed(() => parseDate(props.activity?.start_date) || addDays(new Date(), -182))
const endDate = computed(() => parseDate(props.activity?.end_date) || new Date())
const activityMap = computed(() => new Map((props.activity?.days || []).map((day) => [day.date, day.count])))
const total = computed(() => Number(props.activity?.total || 0))
const activeDays = computed(() => Number(props.activity?.active_days || 0))
const selectedCell = ref(null)

const weeks = computed(() => {
  const start = startDate.value
  const end = endDate.value
  const paddedStart = addDays(start, -start.getDay())
  const paddedEnd = addDays(end, 6 - end.getDay())
  const totalDays = Math.round((paddedEnd - paddedStart) / dayMs) + 1
  const columns = []
  for (let index = 0; index < totalDays; index += 1) {
    const date = addDays(paddedStart, index)
    const key = isoDate(date)
    const inRange = date >= start && date <= end
    const count = inRange ? Number(activityMap.value.get(key) || 0) : 0
    const column = Math.floor(index / 7)
    if (!columns[column]) columns[column] = []
    columns[column].push({ key, date, inRange, count })
  }
  return columns
})

const monthLabels = computed(() => {
  const labels = []
  let previous = ""
  weeks.value.forEach((week, index) => {
    const marker = week.find((cell) => cell.inRange && (cell.date.getDate() === 1 || cell.key === isoDate(startDate.value)))
    if (!marker) return
    const label = marker.date.toLocaleDateString("zh-CN", { month: "short" }).replace("月", "月")
    if (label !== previous) labels.push({ label, column: index + 1 })
    previous = label
  })
  return labels
})

function level(count) {
  if (count <= 0) return 0
  if (count === 1) return 1
  if (count <= 3) return 2
  if (count <= 5) return 3
  return 4
}

function cellLabel(cell) {
  const formatted = cell.date.toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric", weekday: "short" })
  return `${formatted}：${cell.count ? `${cell.count} 次学习活动` : "暂无学习活动"}`
}

const selectedDescription = computed(() => selectedCell.value ? cellLabel(selectedCell.value) : "点击任意日期，查看当天的学习记录。")

function selectDay(cell) {
  if (cell.inRange) selectedCell.value = cell
}
</script>

<template>
  <section class="learning-heatmap paper-panel" aria-labelledby="learning-heatmap-title">
    <header class="learning-heatmap__header">
      <div class="learning-heatmap__overview">
        <div>
          <span class="page-eyebrow"><Sparkles :size="13" /> 学习足迹</span>
          <h2 id="learning-heatmap-title">近半年学习记录</h2>
          <p>{{ activeDays ? `你已在 ${activeDays} 天里留下学习痕迹。` : "从今天的第一条笔记或复习开始，点亮你的学习足迹。" }}</p>
        </div>
        <div class="learning-heatmap__metrics" aria-label="近半年学习统计">
          <div class="learning-heatmap__metric">
            <Flame :size="16" />
            <strong>{{ total }}</strong>
            <span>学习活动</span>
          </div>
          <div class="learning-heatmap__metric learning-heatmap__metric--quiet">
            <strong>{{ activeDays }}</strong>
            <span>活跃日</span>
          </div>
        </div>
      </div>
    </header>

    <div class="learning-heatmap__frame">
      <div class="learning-heatmap__frame-head">
        <span><i></i>最近 6 个月</span>
        <div class="learning-heatmap__legend" aria-label="活动量图例"><span>少</span><i v-for="item in 5" :key="item" :class="`level-${item - 1}`"></i><span>多</span></div>
      </div>
      <div class="learning-heatmap__scroll" tabindex="0" aria-label="近半年学习活动时间轴，可横向滑动浏览">
        <div class="learning-heatmap__weekday" aria-hidden="true"><span>日</span><span>二</span><span>四</span><span>六</span></div>
        <div class="learning-heatmap__calendar">
          <div class="learning-heatmap__months" :style="{ '--heatmap-weeks': weeks.length }">
            <span v-for="month in monthLabels" :key="`${month.label}-${month.column}`" :style="{ gridColumn: month.column }">{{ month.label }}</span>
          </div>
          <div class="learning-heatmap__days" :style="{ '--heatmap-weeks': weeks.length }">
            <template v-for="(week, weekIndex) in weeks" :key="weekIndex">
              <button
                v-for="cell in week"
                :key="cell.key"
                type="button"
                class="learning-heatmap__cell"
                :class="[`level-${level(cell.count)}`, { 'is-outside': !cell.inRange, 'is-selected': selectedCell?.key === cell.key }]"
                :aria-label="cellLabel(cell)"
                :aria-pressed="selectedCell?.key === cell.key"
                :title="cellLabel(cell)"
                @click="selectDay(cell)"
              ></button>
            </template>
          </div>
        </div>
      </div>
    </div>

    <footer><CalendarDays :size="15" /><span>{{ selectedDescription }}</span><em>记录新增、整理、修改与复习</em></footer>
  </section>
</template>
