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
const isFirstStart = ref(false)
const { username, darkMode, setUsername, setDarkMode, load: loadSettings } = useSettings()

const mineruToken = ref("")
const tokenConfigured = ref(false)
const tokenMasked = ref("")
const tokenSaved = ref(false)
const backupInput = ref(null)
const backupBusy = ref(false)

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
  if (!subjectList.value.length) {
    isFirstStart.value = true
    showSubDlg.value = true
  }
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

function toggleDarkMode() {
  setDarkMode(!darkMode.value)
}

function backupStamp() {
  return new Date().toISOString().slice(0, 10)
}

async function exportBackup() {
  backupBusy.value = true
  try {
    const blob = await api.exportBackup()
    const url = URL.createObjectURL(blob)
    const link = document.createElement("a")
    link.href = url
    link.download = `study-tracker-backup-${backupStamp()}.zip`
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  } catch (e) {
    alert("备份失败: " + e.message)
  } finally {
    backupBusy.value = false
  }
}

function chooseBackupFile() {
  backupInput.value?.click()
}

async function importBackupFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ""
  if (!file) return
  const ok = window.confirm("导入备份会覆盖当前错题、科目和设置，确定继续吗？")
  if (!ok) return
  backupBusy.value = true
  try {
    const result = await api.importBackup(file)
    await useSubjects().load()
    subjectList.value = [...subjectRef.value]
    const snapshot = result.snapshot ? `\n已自动保留导入前数据：data/backups/${result.snapshot}` : ""
    alert("导入成功，页面将刷新以加载备份数据。" + snapshot)
    window.location.reload()
  } catch (e) {
    alert("导入失败: " + e.message)
  } finally {
    backupBusy.value = false
  }
}

async function addSubject() {
  const name = newSubject.value.trim()
  if (!name) return
  try {
    await storeAdd(name)
    subjectList.value = [...subjectRef.value]
    newSubject.value = ""
    if (isFirstStart.value && subjectList.value.length) {
      isFirstStart.value = false
      showSubDlg.value = false
      emit("navigate", "list")
    }
  } catch (e) { /* ignore */ }
}

async function delSubject(name) {
  try {
    await storeRemove(name)
    subjectList.value = subjectRef.value.filter(s => s !== name)
    if (!subjectList.value.length) isFirstStart.value = true
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
        <div v-if="showSettingsDlg" class="dialog" style="width:440px">
          <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px">
            <h3 style="font-weight:600;font-size:15px">⚙️ 设置</h3>
            <button class="btn" style="font-size:18px;padding:4px 8px" @click="showSettingsDlg = false">✕</button>
          </div>
          <label class="form-label" style="font-size:12px;margin-bottom:4px">用户名</label>
          <div style="display:flex;gap:8px;margin-bottom:12px">
            <input v-model="username" class="form-input" placeholder="设置用户名" style="font-size:12px" />
            <button class="btn btn-ghost" style="white-space:nowrap;font-size:11px" @click="saveUsername" :style="usernameSaved ? 'color:#10b981' : ''">{{ usernameSaved ? '已保存' : '保存' }}</button>
          </div>
          <div class="setting-row">
            <div>
              <strong>夜间模式</strong>
              <span>切换为更暗的界面配色</span>
            </div>
            <button type="button" class="theme-switch" :class="{ active: darkMode }" :aria-pressed="darkMode" @click="toggleDarkMode">
              <span></span>
            </button>
          </div>
          <div class="setting-row" style="align-items:flex-start;gap:12px">
            <div>
              <strong>数据备份</strong>
              <span>备份包含错题、科目、设置和知识点。</span>
            </div>
            <div style="display:flex;gap:8px;flex-shrink:0">
              <button class="btn btn-ghost" style="white-space:nowrap;font-size:11px" :disabled="backupBusy" @click="exportBackup">
                {{ backupBusy ? '处理中' : '备份数据' }}
              </button>
              <button class="btn btn-ghost" style="white-space:nowrap;font-size:11px" :disabled="backupBusy" @click="chooseBackupFile">
                导入备份
              </button>
            </div>
            <input ref="backupInput" type="file" accept=".zip,application/zip" style="display:none" @change="importBackupFile" />
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
            <h3 style="font-weight:600;font-size:15px">{{ isFirstStart ? '开始使用错题本' : '管理科目' }}</h3>
            <button v-if="!isFirstStart" class="btn" style="font-size:18px;padding:4px 8px" @click="showSubDlg = false">✕</button>
          </div>
          <div v-if="isFirstStart" style="padding:12px;border:1px solid var(--border);border-radius:10px;background:var(--surface-muted);margin-bottom:14px">
            <p style="font-size:13px;color:var(--text);font-weight:600;margin-bottom:6px">先创建一个科目，才能添加第一道错题。</p>
            <ol style="padding-left:18px;color:var(--text-sec);font-size:12px;line-height:1.7">
              <li>输入科目名称，例如：高数、英语、专业课。</li>
              <li>点击添加，系统会进入错题列表。</li>
              <li>之后可继续在这里增删科目。</li>
            </ol>
          </div>
          <div style="display:flex;gap:6px;flex-wrap:wrap;margin-bottom:12px">
            <span v-for="s in subjectList" :key="s" style="display:inline-flex;align-items:center;gap:4px;padding:4px 10px;border-radius:12px;background:var(--accent)12;font-size:12px">
              {{ s }}
              <span @click="delSubject(s)" style="cursor:pointer;color:var(--danger);font-size:14px;line-height:1">&times;</span>
            </span>
            <span v-if="!subjectList.length" style="font-size:12px;color:var(--text-muted)">暂无科目</span>
          </div>
          <div style="display:flex;gap:8px;margin-bottom:16px">
            <input v-model="newSubject" class="form-input" :placeholder="isFirstStart ? '例如：高数' : '输入新科目名称'" @keyup.enter="addSubject" />
            <button class="btn btn-primary" style="white-space:nowrap;font-size:12px" @click="addSubject">{{ isFirstStart ? '创建科目' : '添加' }}</button>
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
