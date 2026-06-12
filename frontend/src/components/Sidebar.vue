<script setup>
import { ref, onMounted } from "vue"
import { useSubjects } from "../store/subjects.js"
import { useSettings } from "../store/settings.js"
import { api } from "../api/index.js"

defineProps({ active: String })

const emit = defineEmits(["navigate"])
const showSubDlg = ref(false)
const showSettingsDlg = ref(false)
const subjectList = ref([])
const newSubject = ref("")
const { username, setUsername, load: loadSettings } = useSettings()

const mineruToken = ref("")
const tokenConfigured = ref(false)
const tokenMasked = ref("")
const tokenSaved = ref(false)

onMounted(async () => {
  try {
    await loadSettings()
    const t = await api.getToken()
    tokenConfigured.value = t.configured
    tokenMasked.value = t.configured ? t.token : ""
    mineruToken.value = ""
  }
  catch(e){/*ignore*/}
})

function openSubjectDialog() { showSubDlg.value = true }

const { add: storeAdd, remove: storeRemove, subjectRef } = useSubjects()

onMounted(async () => {
  await useSubjects().load()
  subjectList.value = subjectRef.value
})

const usernameSaved = ref(false)
async function saveUsername() {
  try {
    await api.saveUsername(username.value)
    setUsername(username.value)
    usernameSaved.value = true
    setTimeout(() => usernameSaved.value = false, 2000)
  } catch(e) { alert('保存失败: ' + e.message) }
}
async function saveToken() {
  try {
    await api.saveToken(mineruToken.value)
    const t = await api.getToken()
    tokenConfigured.value = t.configured
    tokenMasked.value = t.configured ? t.token : ""
    mineruToken.value = ""
    tokenSaved.value = true
    setTimeout(() => tokenSaved.value = false, 2000)
  } catch(e) { alert('保存失败: ' + e.message) }
}

async function clearToken() {
  try {
    await api.clearToken()
    tokenConfigured.value = false
    tokenMasked.value = ""
    mineruToken.value = ""
    tokenSaved.value = true
    setTimeout(() => tokenSaved.value = false, 2000)
  } catch(e) { alert('清除失败: ' + e.message) }
}

async function addSubject() {
  const name = newSubject.value.trim()
  if (!name) return
  try {
    await storeAdd(name)
    subjectList.value = [...subjectRef.value]
    newSubject.value = ""
  } catch (e) { /* ignore */ }
}

async function delSubject(name) {
  try {
    await storeRemove(name)
    subjectList.value = subjectRef.value.filter(s => s !== name)
  } catch (e) { /* ignore */ }
}

const items = [
  { key: "home", icon: "💡", label: "每日推送" },
  { key: "list", icon: "📋", label: "错题列表" },
]
</script>

<template>
  <aside class="sidebar">
    <!-- Settings Dialog -->
    <Teleport to="body">
      <div v-if="showSettingsDlg" class="dialog-overlay">
      <Transition name="dialog">
        <div v-if="showSettingsDlg" class="dialog" style="width:400px">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
            <h3 style="font-weight:600;font-size:15px">⚙️ 设置</h3>
            <button class="btn" style="font-size:18px;padding:4px 8px" @click="showSettingsDlg = false">✕</button>
          </div>
          <label class="form-label" style="font-size:12px;margin-bottom:4px">用户名</label>
          <div style="display:flex;gap:8px;margin-bottom:12px">
            <input v-model="username" class="form-input" placeholder="设置用户名" style="font-size:12px" />
            <button class="btn btn-ghost" style="white-space:nowrap;font-size:11px" @click="saveUsername" :style="usernameSaved ? 'color:#10b981' : ''">{{ usernameSaved ? '已保存' : '保存' }}</button>
          </div>
          <label class="form-label" style="font-size:12px;margin-bottom:4px">MinerU Token</label>
          <div v-if="tokenConfigured" style="font-size:11px;color:var(--text-sec);margin-bottom:6px">
            当前已配置：{{ tokenMasked }}。留空保存不会覆盖现有 Token。
          </div>
          <div style="display:flex;gap:8px">
            <input v-model="mineruToken" class="form-input" :placeholder="tokenConfigured ? '粘贴新 Token，留空则不修改' : '粘贴 Token'" style="font-size:12px" />
            <button class="btn btn-ghost" style="white-space:nowrap;font-size:11px" @click="saveToken" :style="tokenSaved ? 'color:#10b981' : ''">{{ tokenSaved ? '已保存' : '保存' }}</button>
            <button v-if="tokenConfigured" class="btn btn-ghost" style="white-space:nowrap;font-size:11px;color:var(--danger);background:rgba(239,68,68,.08)" @click="clearToken">清除</button>
          </div>
          <p style="font-size:10px;color:var(--text-muted);margin-top:4px">{{ tokenConfigured ? 'OCR 可用' : '配置 Token 后可使用 OCR' }}</p>
        </div>
      </Transition>
      </div>
    </Teleport>

    <!-- Subject Dialog -->
    <Teleport to="body">
      <div v-if="showSubDlg" class="dialog-overlay">
      <Transition name="dialog">
        <div v-if="showSubDlg" class="dialog" style="width:420px">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
            <h3 style="font-weight:600;font-size:15px">管理科目</h3>
            <button class="btn" style="font-size:18px;padding:4px 8px" @click="showSubDlg = false">✕</button>
          </div>
          <div style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px">
            <span v-for="s in subjectList" :key="s" style="display:inline-flex;align-items:center;gap:4px;padding:4px 10px;border-radius:12px;background:var(--accent)12;font-size:12px">
              {{ s }}
              <span @click="delSubject(s)" style="cursor:pointer;color:var(--danger);font-size:14px;line-height:1">&times;</span>
            </span>
            <span v-if="!subjectList.length" style="font-size:12px;color:var(--text-muted)">暂无科目</span>
          </div>
          <div style="display:flex;gap:8px;margin-bottom:16px">
            <input v-model="newSubject" class="form-input" placeholder="输入新科目名称" @keyup.enter="addSubject" />
            <button class="btn btn-primary" style="white-space:nowrap;font-size:12px" @click="addSubject">添加</button>
          </div>
        </div>
      </Transition>
      </div>
    </Teleport>
    <div class="sidebar-brand">
      <div class="logo">
        <svg viewBox="0 0 24 24"><path d="M12 3L1 9l4 2.18v6L12 21l7-3.82v-6l2-1.09V17h2V9L12 3zm6.82 6L12 12.72 5.18 9 12 5.28 18.82 9zM17 15.99l-5 2.73-5-2.73v-3.72L12 15l5-2.73v3.72z"/></svg>
      </div>
      <h1 style="font-size:24px">错题本</h1>
      <p style="font-size:12.5px">{{ username }}</p>
    </div>
    <nav class="sidebar-nav">
      <div v-for="item in items" :key="item.key"
           class="nav-item" :class="{ active: active === item.key }"
           @click="emit('navigate', item.key)">
        <span class="nav-icon">{{ item.icon }}</span>
        <span class="nav-label">{{ item.label }}</span>
      </div>
    </nav>
    <div class="sidebar-footer">
      <button class="btn btn-ghost" style="color:rgba(255,255,255,.55);font-size:13.5px"
              @click="showSettingsDlg = true">
        ⚙️ 设置
      </button>
      <button class="btn btn-ghost" style="color:rgba(255,255,255,.55);font-size:13.5px"
              @click="openSubjectDialog">
        📚 管理科目
      </button>
      <button class="btn btn-ghost" style="color:rgba(255,255,255,.55);font-size:13.5px"
              @click="$emit('navigate', 'home')">
        🏠 首页
      </button>
    </div>
  </aside>
</template>
