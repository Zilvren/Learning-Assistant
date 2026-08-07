<script setup>
import { computed, nextTick, onMounted, ref, watch } from "vue"
import { CalendarDays, Flame, Sparkles } from "lucide-vue-next"

const props = defineProps({
  activity: { type: Object, default: () => ({ days: [], available_years: [] }) },
  selectedYear: { type: Number, default: () => new Date().getFullYear() },
})
const emit = defineEmits(["select-year"])

const dayMs = 24 * 60 * 60 * 1000
const frame = ref(null)
const selectedCell = ref(null)

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

function startOfYear(year) { return new Date(year, 0, 1) }
function endOfYear(year) { return new Date(year, 11, 31) }

const year = computed(() => Number(props.selectedYear) || new Date().getFullYear())
const today = computed(() => {
  const now = new Date()
  return new Date(now.getFullYear(), now.getMonth(), now.getDate())
})
const startDate = computed(() => parseDate(props.activity?.start_date) || startOfYear(year.value))
const endDate = computed(() => parseDate(props.activity?.end_date) || endOfYear(year.value))
const activityMap = computed(() => new Map((props.activity?.days || []).map((day) => [day.date, day.count])))
const total = computed(() => Number(props.activity?.total || 0))
const activeDays = computed(() => Number(props.activity?.active_days || 0))
const isCurrentYear = computed(() => year.value === today.value.getFullYear())
const todayKey = computed(() => isoDate(today.value))
const rangeLabel = computed(() => isCurrentYear.value ? `${year.value} 年全年视图 · 未来日期留白` : `${year.value} 年全年学习活动`)
const yearOptions = computed(() => {
  const currentYear = today.value.getFullYear()
  const years = new Set([year.value, ...(props.activity?.available_years || [])])
  for (let offset = 0; offset < 4; offset += 1) years.add(currentYear - offset)
  return [...years].filter((item) => item >= 2000 && item <= currentYear).sort((a, b) => b - a)
})

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
    const future = inRange && date > today.value
    const count = inRange && !future ? Number(activityMap.value.get(key) || 0) : 0
    const column = Math.floor(index / 7)
    if (!columns[column]) columns[column] = []
    columns[column].push({ key, date, inRange, future, count })
  }
  return columns
})

const monthLabels = computed(() => {
  const labels = []
  let previous = ""
  weeks.value.forEach((week, index) => {
    const marker = week.find((cell) => cell.inRange && (cell.date.getDate() === 1 || cell.key === isoDate(startDate.value)))
    if (!marker) return
    const label = marker.date.toLocaleDateString("zh-CN", { month: "short" })
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
  if (cell.future) return `${formatted}：尚未到达`
  return `${formatted}：${cell.count ? `${cell.count} 次学习活动` : "暂无学习活动"}`
}

const selectedDescription = computed(() => selectedCell.value ? cellLabel(selectedCell.value) : "点击任意日期，查看当天的学习记录。")

function selectDay(cell) {
  if (cell.inRange && !cell.future) selectedCell.value = cell
}

function selectYear(nextYear) {
  if (nextYear !== year.value) emit("select-year", nextYear)
}

function positionTimeline() {
  const container = frame.value
  if (!container) return
  if (!isCurrentYear.value) {
    container.scrollLeft = 0
    return
  }

  const currentDay = container.querySelector(`[data-activity-date="${todayKey.value}"]`)
  if (!currentDay) return
  const frameRect = container.getBoundingClientRect()
  const dayRect = currentDay.getBoundingClientRect()
  if (!frameRect.width || !dayRect.width) return

  const rightInset = 18
  const offset = dayRect.right - frameRect.right + rightInset
  container.scrollLeft = Math.max(0, Math.round(container.scrollLeft + offset))
}

async function queueTimelinePosition() {
  await nextTick()
  if (typeof requestAnimationFrame === "function") requestAnimationFrame(positionTimeline)
  else positionTimeline()
}

onMounted(queueTimelinePosition)

watch(() => props.selectedYear, async () => {
  selectedCell.value = null
  await queueTimelinePosition()
})
</script>

<template>
  <section class="learning-heatmap paper-panel" aria-labelledby="learning-heatmap-title">
    <header class="learning-heatmap__header">
      <div class="learning-heatmap__overview">
        <div>
          <span class="page-eyebrow"><Sparkles :size="13" /> 学习足迹</span>
          <h2 id="learning-heatmap-title">学习记录</h2>
          <p>{{ activeDays ? `你已在 ${activeDays} 天里留下学习痕迹。` : "从今天的第一条笔记或复习开始，点亮你的学习足迹。" }}</p>
          <div class="learning-heatmap__year-picker" role="group" aria-label="切换学习记录年份">
            <span>年份</span>
            <button v-for="option in yearOptions" :key="option" type="button" :class="{ 'is-active': option === year }" :aria-pressed="option === year" @click="selectYear(option)">{{ option }}</button>
          </div>
        </div>
        <div class="learning-heatmap__metrics" :aria-label="`${year} 年学习统计`">
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

    <div ref="frame" class="learning-heatmap__frame" :style="{ '--heatmap-weeks': weeks.length, '--heatmap-width': `${weeks.length * 21 + 26}px` }">
      <div class="learning-heatmap__frame-head">
        <span><i></i>{{ rangeLabel }}</span>
        <div class="learning-heatmap__legend" aria-label="活动量图例"><span>少</span><i v-for="item in 5" :key="item" :class="`level-${item - 1}`"></i><span>多</span></div>
      </div>
      <div class="learning-heatmap__scroll" tabindex="0" :aria-label="`${year} 年学习活动时间轴，可横向滑动浏览全年`">
        <div class="learning-heatmap__weekday" aria-hidden="true"><span>日</span><span>二</span><span>四</span><span>六</span></div>
        <div class="learning-heatmap__calendar">
          <div class="learning-heatmap__months">
            <span v-for="month in monthLabels" :key="`${month.label}-${month.column}`" :style="{ gridColumn: month.column }">{{ month.label }}</span>
          </div>
          <div class="learning-heatmap__days">
            <template v-for="(week, weekIndex) in weeks" :key="weekIndex">
              <button
                v-for="cell in week"
                :key="cell.key"
                type="button"
                class="learning-heatmap__cell"
                :class="[`level-${level(cell.count)}`, { 'is-outside': !cell.inRange, 'is-future': cell.future, 'is-selected': selectedCell?.key === cell.key }]"
                :data-activity-date="cell.key"
                :aria-label="cellLabel(cell)"
                :aria-pressed="selectedCell?.key === cell.key"
                :disabled="!cell.inRange || cell.future"
                :title="cellLabel(cell)"
                @click="selectDay(cell)"
              ></button>
            </template>
          </div>
        </div>
      </div>
    </div>

    <footer><CalendarDays :size="15" /><span>{{ selectedDescription }}</span><em>横向滑动浏览全年 · 记录新增、整理、修改与复习</em></footer>
  </section>
</template>
