<script setup>
import { ref, onMounted } from "vue"
import { api } from "../api/index.js"
import { useSubjects } from "../store/subjects.js"

const emit = defineEmits(["snack"])
const data = ref({ knowledge: {}, weak_errors: [], total_errors: 0, reviewed: 0, advice: "" })
const loading = ref(true)

onMounted(async () => {
  try { const { subjectRef } = useSubjects()
      await useSubjects().load()
      subjectList.value = subjectRef.value
      data.value = await api.getDailyPush() } catch (e) { emit("snack", e.message, "#ef4444") }
  finally { loading.value = false }
})

const { subjectRef } = useSubjects()
const subjectList = subjectRef
const colors = {}
const colorPool = ["#0EA5E9","#8B5CF6","#10B981","#F97316","#EC4899","#F59E0B","#6366F1","#14B8A6","#F43F5E","#EAB308"]
function hashCode(s) { let h = 0; for (let i = 0; i < s.length; i++) h = ((h << 5) - h + s.charCodeAt(i)) | 0; return h }
function subjectColor(name) { return colors[name] || colorPool[Math.abs(hashCode(name)) % colorPool.length] }
const icons = {}

const username = ref("")
const hour = new Date().getHours()
const greeting = hour < 12 ? "早上好" : hour < 18 ? "下午好" : "晚上好"

onMounted(async () => {
  try { const t = await api.getToken(); username.value = t.username || "" } catch(e){}
})
const date = new Date().toISOString().slice(0, 10)
</script>

<template>
  <div>
    <!-- Hero -->
    <div class="card" style="background:linear-gradient(135deg,#6366f1,#8b5cf6);color:#fff;padding:24px 28px;margin-bottom:24px">
      <div style="display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:16px">
        <div>
          <h2 style="font-size:24px;margin-bottom:6px">{{ greeting }}，{{ username || '同学' }}</h2>
          <p style="font-size:14px;opacity:.6">{{ date }} · 今日学习推送</p>
        </div>
        <div style="display:flex;gap:10px">
          <div class="card" style="background:rgba(255,255,255,.08);padding:10px 18px;text-align:center;backdrop-filter:blur(8px)">
            <div style="font-size:22px;font-weight:700">{{ data.total_errors }}</div>
            <div style="font-size:11.5px;opacity:.5">错题总数</div>
          </div>
          <div class="card" style="background:rgba(255,255,255,.08);padding:10px 18px;text-align:center;backdrop-filter:blur(8px)">
            <div style="font-size:22px;font-weight:700">{{ data.total_errors ? Math.round(data.reviewed/data.total_errors*100) : 0 }}%</div>
            <div style="font-size:11.5px;opacity:.5">已复习</div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" style="text-align:center;padding:40px;color:var(--text-muted)">加载中...</div>

    <template v-if="!loading">
      <!-- Knowledge Cards -->
      <h3 style="font-size:18px;font-weight:600;margin-bottom:14px">今日知识点</h3>
      <div v-for="s in subjectList" :key="s">
        <div v-if="data.knowledge[s]" class="card" style="display:flex;gap:12px;padding:16px;margin-bottom:8px;align-items:flex-start">
          <div :style="{ width:4, background:subjectColor(s), borderRadius:2, flexShrink:0, alignSelf:'stretch' }"></div>
          <div>
            <div style="display:flex;gap:8px;align-items:center;margin-bottom:4px">
              <span>{{ icons[s] }}</span>
              <span style="font-weight:600;font-size:15px">{{ s }}</span>
            </div>
            <p style="font-size:13.5px;color:var(--text-sec)">{{ data.knowledge[s] }}</p>
          </div>
        </div>
      </div>

      <!-- Advice -->
      <div class="card" style="margin-top:16px;display:flex;gap:14px;align-items:center;padding:14px 20px">
        <div style="width:36px;height:36px;border-radius:10px;background:var(--accent)15;display:flex;align-items:center;justify-content:center;flex-shrink:0">
          💡
        </div>
        <span style="font-size:14.5px;color:var(--text-sec)">{{ data.advice }}</span>
      </div>
    </template>
  </div>
</template>
