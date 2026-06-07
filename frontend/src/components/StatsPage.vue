<script setup>
import { ref, onMounted, computed } from "vue"
import { api } from "../api/index.js"
import { useSubjects } from "../store/subjects.js"
import { exportPdfReport } from "../utils/pdfExport.js"

const emit = defineEmits(["snack"])
const stats = ref({ total: 0, reviewed: 0, review_rate: 0, subject_counts: {}, weak_errors: [] })
const loading = ref(true)

const colors = {}
const colorPool = ["#0EA5E9","#8B5CF6","#10B981","#F97316","#EC4899","#F59E0B","#6366F1","#14B8A6","#F43F5E","#EAB308"]
function hashCode(s) { let h = 0; for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0; return h }
function subjectColor(name) { return colors[name] || colorPool[Math.abs(hashCode(name)) % colorPool.length] }
const { subjectRef } = useSubjects()
const subjectList = subjectRef

onMounted(async () => {
  try { const { subjectRef } = useSubjects()
      await useSubjects().load()
      subjectList.value = subjectRef.value
      stats.value = await api.getStats() } catch (e) { emit("snack", e.message, "#ef4444") }
  finally { loading.value = false }
})

const maxCount = computed(() => Math.max(...Object.values(stats.value.subject_counts), 1))
const weakest = computed(() => {
  if (!stats.value.total) return "暂无"
  return Object.entries(stats.value.subject_counts)
    .sort((a, b) => b[1] - a[1])[0][0]
})

async function exportPdf() {
  try {
    const all = await api.getErrors()
    await exportPdfReport(all.errors)
  } catch (e) {
    emit("snack", e.message || "导出失败", "#ef4444")
  }
}
</script>

<template>
  <div>
    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:20px">
      <h2 style="font-size:22px;font-weight:600">薄弱点分析</h2>
      <button class="btn btn-ghost" style="color:var(--accent)" @click="exportPdf">📄 导出 PDF</button>
    </div>

    <div v-if="loading" style="text-align:center;padding:40px;color:var(--text-muted)">加载中...</div>

    <template v-if="!loading">
      <!-- Overview Cards -->
      <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin-bottom:20px">
        <div class="card">
          <div style="width:40px;height:40px;border-radius:12px;background:var(--accent)18;display:flex;align-items:center;justify-content:center;margin-bottom:8px">❌</div>
          <div style="font-size:26px;font-weight:700">{{ stats.total }}</div>
          <div style="font-size:13.5px;color:var(--text-sec)">错题总数</div>
          <div style="font-size:12.5px;color:var(--text-muted)">已复习 {{ stats.reviewed }} 道</div>
        </div>
        <div class="card">
          <div style="width:40px;height:40px;border-radius:12px;background:rgba(16,185,129,.1);display:flex;align-items:center;justify-content:center;margin-bottom:8px">✅</div>
          <div style="font-size:26px;font-weight:700">{{ stats.review_rate }}%</div>
          <div style="font-size:13.5px;color:var(--text-sec)">复习率</div>
          <div style="font-size:12.5px;color:var(--text-muted)">{{ stats.reviewed }}/{{ stats.total }} 已完成</div>
        </div>
        <div class="card">
          <div style="width:40px;height:40px;border-radius:12px;background:rgba(245,158,11,.1);display:flex;align-items:center;justify-content:center;margin-bottom:8px">⚠️</div>
          <div style="font-size:26px;font-weight:700">{{ weakest }}</div>
          <div style="font-size:13.5px;color:var(--text-sec)">最薄弱</div>
          <div style="font-size:12.5px;color:var(--text-muted)">{{ stats.subject_counts[weakest] || 0 }} 道错题</div>
        </div>
      </div>

      <!-- Distribution -->
      <div class="card" style="margin-bottom:20px">
        <h3 style="font-size:16px;font-weight:600;margin-bottom:12px">各科错题分布</h3>
        <div v-for="s in subjectList" :key="s" style="margin-bottom:8px">
          <div style="display:flex;align-items:center;gap:8px">
            <span style="font-size:14px;font-weight:500;width:120px">{{ s }}</span>
            <span :style="{ fontSize:13, fontWeight:700, color: subjectColor(s), width:40 }">{{ stats.subject_counts[s] || 0 }}</span>
            <div style="flex:1;height:22px;border-radius:7px;background:var(--bg);position:relative;overflow:hidden">
              <div :style="{
                width: Math.round((stats.subject_counts[s]||0) / maxCount * 100) + '%',
                height: '100%',
                borderRadius: 7,
                background: `linear-gradient(90deg, ${subjectColor(s)}, ${subjectColor(s)}cc)`,
                transition: 'width .6s ease',
              }"></div>
            </div>
            <span v-if="(stats.subject_counts[s]||0) === Math.max(...Object.values(stats.subject_counts)) && stats.total > 0"
                  style="font-size:14px">🔥</span>
          </div>
        </div>
      </div>

      <!-- Weak Errors -->
      <div class="card">
        <div style="display:flex;gap:8px;align-items:center;margin-bottom:8px">
          <span style="font-size:18px">⚠️</span>
          <span style="font-size:15.5px;font-weight:600">需重点复习（复习 &lt; 2 次）</span>
        </div>
        <div v-for="e in stats.weak_errors.slice(0,10)" :key="e.id"
             style="display:flex;gap:8px;align-items:center;padding:8px 12px;border-radius:8px;background:var(--bg);margin-top:4px;font-size:12px">
          <div :style="{ width:8, height:8, borderRadius:4, background: e.review_count === 0 ? 'var(--danger)' : 'var(--warning)' }"></div>
          <span :style="{ fontWeight:700, color: colors[e.subject], width:36 }">#{{ e.id }}</span>
          <span style="color:var(--text-muted);width:110px">[{{ e.subject }}]</span>
          <span style="color:var(--text-sec)">{{ e.question?.slice(0,40) }}</span>
        </div>
        <div v-if="!stats.weak_errors.length" style="padding:8px;font-size:13px;color:var(--success)">
          暂无需要重点复习的错题
        </div>
        <div v-if="stats.weak_errors.length > 10" style="font-size:13.5px;color:var(--text-muted);padding:8px;margin-top:4px">
          ... 还有 {{ stats.weak_errors.length - 10 }} 道
        </div>
      </div>
    </template>
  </div>
</template>
