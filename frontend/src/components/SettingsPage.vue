<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import {
  ArchiveRestore, Brush, Check, CloudDownload, DatabaseBackup, Eye, EyeOff,
  KeyRound, LogOut, Moon, PackageCheck, RefreshCw, Sun, UserRound,
} from "lucide-vue-next"
import { api } from "../api/index.js"
import { useAuth } from "../store/auth.js"
import { useSettings } from "../store/settings.js"
import { useToast } from "../store/toast.js"
import BaseButton from "./ui/BaseButton.vue"
import ConfirmDialog from "./ui/ConfirmDialog.vue"
import PageHeader from "./ui/PageHeader.vue"

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const settings = useSettings()
const toast = useToast()
const { username, darkMode } = settings
const usernameBusy = ref(false)
const token = ref("")
const tokenConfigured = ref(false)
const tokenMasked = ref("")
const tokenVisible = ref(false)
const tokenBusy = ref(false)
const backupInput = ref(null)
const backupBusy = ref(false)
const pendingBackup = ref(null)
const updateInfo = ref(null)
const versionInfo = ref({ version: "", can_auto_update: false })
const updateBusy = ref(false)
const updateApplying = ref(false)
const updateStatus = ref("正在读取版本信息")
const confirmUpdate = ref(false)
let restartPollTimer = null
const LAST_UPDATE_CHECK_KEY = "studyTrackerLastUpdateCheck"

const canApplyUpdate = computed(() => !!updateInfo.value?.has_update && !!updateInfo.value?.asset_found && !!versionInfo.value.can_auto_update)

async function saveUsername() {
  if (usernameBusy.value) return
  usernameBusy.value = true
  try {
    await api.saveUsername(username.value.trim())
    settings.setUsername(username.value.trim())
    toast.success("称呼已保存")
  } catch (error) { toast.error(error.message || "保存失败") }
  finally { usernameBusy.value = false }
}

async function loadToken() {
  try {
    const result = await api.getToken()
    tokenConfigured.value = result.configured
    tokenMasked.value = result.configured ? result.token : ""
  } catch (error) { toast.error(error.message || "OCR 配置读取失败") }
}

async function saveToken() {
  if (!token.value.trim()) return toast.warning(tokenConfigured.value ? "粘贴新 Token 后再保存" : "请输入 MinerU Token")
  tokenBusy.value = true
  try {
    await api.saveToken(token.value.trim())
    token.value = ""
    await loadToken()
    toast.success("OCR Token 已安全保存")
  } catch (error) { toast.error(error.message || "Token 保存失败") }
  finally { tokenBusy.value = false }
}

async function clearToken() {
  tokenBusy.value = true
  try {
    await api.clearToken()
    token.value = ""
    await loadToken()
    toast.success("OCR Token 已清除")
  } catch (error) { toast.error(error.message || "Token 清除失败") }
  finally { tokenBusy.value = false }
}

function backupStamp() { return new Date().toISOString().slice(0, 10) }
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
    toast.success("备份文件已导出")
  } catch (error) { toast.error(error.message || "备份失败") }
  finally { backupBusy.value = false }
}

function chooseBackup(event) {
  pendingBackup.value = event.target.files?.[0] || null
  event.target.value = ""
}

async function importBackup() {
  if (!pendingBackup.value) return
  backupBusy.value = true
  try {
    const result = await api.importBackup(pendingBackup.value)
    pendingBackup.value = null
    const suffix = result.snapshot ? `；导入前快照：${result.snapshot}` : ""
    toast.success(`备份导入成功${suffix}`, { timeout: 7000 })
    window.setTimeout(() => window.location.reload(), 900)
  } catch (error) { toast.error(error.message || "导入失败"); pendingBackup.value = null }
  finally { backupBusy.value = false }
}

function todayKey() { return new Date().toISOString().slice(0, 10) }
async function loadVersion() {
  try {
    versionInfo.value = await api.getVersion()
    updateStatus.value = versionInfo.value.can_auto_update ? "可检查并自动安装更新" : "源码模式：可检查更新"
    if (localStorage.getItem(LAST_UPDATE_CHECK_KEY) !== todayKey()) {
      localStorage.setItem(LAST_UPDATE_CHECK_KEY, todayKey())
      await checkUpdate(false, true)
    }
  } catch { updateStatus.value = "版本信息读取失败" }
}

async function checkUpdate(force = true, silent = false) {
  updateBusy.value = true
  if (!silent) updateStatus.value = "正在检查更新…"
  try {
    const result = await api.checkUpdate(force)
    updateInfo.value = result.ok ? result : null
    if (!result.ok) updateStatus.value = result.message || "检查失败"
    else if (!result.has_update) updateStatus.value = "当前已是最新版本"
    else updateStatus.value = result.asset_found ? `发现新版本 v${result.latest_version}` : `发现 v${result.latest_version}，但缺少更新包`
  } catch (error) { updateInfo.value = null; updateStatus.value = `检查失败：${error.message}` }
  finally { updateBusy.value = false }
}

async function applyUpdate() {
  confirmUpdate.value = false
  if (!canApplyUpdate.value) return
  updateApplying.value = true
  updateStatus.value = "正在下载并校验更新…"
  try {
    const result = await api.applyUpdate()
    updateStatus.value = result.message || "程序即将重启安装更新"
    waitForRestart(result.latest_version)
  } catch (error) { updateStatus.value = `更新失败：${error.message}`; updateApplying.value = false }
}

function waitForRestart(targetVersion) {
  const startedAt = Date.now()
  window.clearInterval(restartPollTimer)
  restartPollTimer = window.setInterval(async () => {
    try {
      const info = await api.getVersion()
      if (!targetVersion || info.version === targetVersion) {
        window.clearInterval(restartPollTimer)
        updateStatus.value = "更新完成，正在刷新…"
        window.location.reload()
      }
    } catch { updateStatus.value = "正在等待新版程序启动…" }
    if (Date.now() - startedAt > 90000) {
      window.clearInterval(restartPollTimer)
      updateApplying.value = false
      updateStatus.value = "更新可能仍在进行，请稍后手动刷新"
    }
  }, 1500)
}

async function logout() {
  await auth.logout()
  toast.info("已退出登录")
  await router.replace({ name: "login" })
}

onMounted(async () => {
  await Promise.all([settings.load(), loadToken(), loadVersion()])
  if (route.query.section) await nextTick(() => document.getElementById(String(route.query.section))?.scrollIntoView({ behavior: "smooth" }))
})
onBeforeUnmount(() => window.clearInterval(restartPollTimer))
</script>

<template>
  <div class="settings-view page-stage">
    <PageHeader eyebrow="个人与数据" title="设置中心" description="管理你的身份、外观、数据安全与工具连接。" />

    <div class="settings-layout">
      <nav class="settings-toc" aria-label="设置目录">
        <a href="#account"><UserRound :size="16" />账户与会话</a>
        <a href="#appearance"><Brush :size="16" />外观</a>
        <a href="#backup"><DatabaseBackup :size="16" />备份恢复</a>
        <a href="#ocr"><KeyRound :size="16" />OCR Token</a>
        <a href="#updates"><PackageCheck :size="16" />版本更新</a>
      </nav>

      <div class="settings-sections">
        <section id="account" class="settings-section paper-panel">
          <header><span>01</span><div><h2>账户与会话</h2><p>这个称呼会出现在首页问候和应用身份区。</p></div><UserRound :size="21" /></header>
          <div class="setting-control-row">
            <label><span class="field-label">学习者称呼</span><input v-model="username" class="field-control" placeholder="如何称呼你" /></label>
            <BaseButton :busy="usernameBusy" @click="saveUsername"><template #icon><Check :size="16" /></template>保存称呼</BaseButton>
          </div>
          <div v-if="auth.enabled.value" class="setting-subrow"><div><strong>当前账户</strong><p>{{ auth.user.value?.username }}</p></div><BaseButton variant="quiet-danger" @click="logout"><template #icon><LogOut :size="16" /></template>退出登录</BaseButton></div>
          <div v-else class="setting-subrow"><div><strong>本地模式</strong><p>数据仅由当前运行实例管理，无需登录。</p></div><span class="status-chip status-chip--success">已启用</span></div>
        </section>

        <section id="appearance" class="settings-section paper-panel">
          <header><span>02</span><div><h2>外观</h2><p>主题会保存在当前浏览器，并沿用已有设置。</p></div><Brush :size="21" /></header>
          <div class="theme-choice">
            <button type="button" :class="{ active: !darkMode }" @click="settings.setDarkMode(false)"><Sun :size="22" /><strong>明亮模式</strong><span>清爽浅色学习空间</span></button>
            <button type="button" :class="{ active: darkMode }" @click="settings.setDarkMode(true)"><Moon :size="22" /><strong>深色模式</strong><span>低干扰夜间学习</span></button>
          </div>
        </section>

        <section id="backup" class="settings-section paper-panel">
          <header><span>03</span><div><h2>备份与恢复</h2><p>备份包含资料库、笔记标签、附件、设置和每日知识札记。</p></div><DatabaseBackup :size="21" /></header>
          <div class="setting-action-grid"><article><CloudDownload :size="22" /><div><strong>导出完整备份</strong><p>下载带日期的 ZIP 文件，建议在重要整理后执行。</p></div><BaseButton :busy="backupBusy" @click="exportBackup">导出备份</BaseButton></article><article><ArchiveRestore :size="22" /><div><strong>从备份恢复</strong><p>导入会覆盖当前数据，系统会自动保留导入前快照。</p></div><BaseButton :disabled="backupBusy" @click="backupInput?.click()">选择文件</BaseButton><input ref="backupInput" type="file" accept=".zip,application/zip" hidden @change="chooseBackup" /></article></div>
        </section>

        <section id="ocr" class="settings-section paper-panel">
          <header><span>04</span><div><h2>OCR 工具连接</h2><p>MinerU Token 只由后端保存；页面仅展示掩码。</p></div><KeyRound :size="21" /></header>
          <div v-if="tokenConfigured" class="token-status"><span class="status-chip status-chip--success">已配置</span><code>{{ tokenVisible ? tokenMasked : '••••••••••••' }}</code><button type="button" :aria-label="tokenVisible ? '隐藏 Token 掩码' : '显示 Token 掩码'" @click="tokenVisible = !tokenVisible"><component :is="tokenVisible ? EyeOff : Eye" :size="16" /></button></div>
          <div class="setting-control-row"><label><span class="field-label">{{ tokenConfigured ? '替换 Token' : 'MinerU Token' }}</span><input v-model="token" class="field-control" type="password" :placeholder="tokenConfigured ? '粘贴新 Token' : '粘贴 Token 后保存'" autocomplete="off" /></label><BaseButton :busy="tokenBusy" @click="saveToken">保存连接</BaseButton><BaseButton v-if="tokenConfigured" variant="quiet-danger" :busy="tokenBusy" @click="clearToken">清除</BaseButton></div>
        </section>

        <section id="updates" class="settings-section paper-panel">
          <header><span>05</span><div><h2>版本更新</h2><p>当前版本 {{ versionInfo.version || '读取中' }} · {{ updateStatus }}</p></div><PackageCheck :size="21" /></header>
          <div class="setting-subrow"><div><strong>{{ updateInfo?.has_update ? `可更新至 v${updateInfo.latest_version}` : '检查发行版本' }}</strong><p v-if="updateInfo?.published_at">发布于 {{ updateInfo.published_at.slice(0, 10) }}</p><p v-if="updateInfo?.has_update && !versionInfo.can_auto_update">源码模式需要手动拉取新版本。</p></div><div class="button-row"><BaseButton :busy="updateBusy" :disabled="updateApplying" @click="checkUpdate(true, false)"><template #icon><RefreshCw :size="16" /></template>检查更新</BaseButton><BaseButton v-if="canApplyUpdate" variant="primary" :busy="updateApplying" @click="confirmUpdate = true">立即更新</BaseButton></div></div>
        </section>
      </div>
    </div>

    <ConfirmDialog :open="!!pendingBackup" title="用备份覆盖当前数据？" message="资料库、笔记、标签与设置会被替换。系统会先自动创建 pre-import 快照。" confirm-text="确认导入" danger :busy="backupBusy" @close="pendingBackup = null" @confirm="importBackup" />
    <ConfirmDialog :open="confirmUpdate" title="立即安装更新？" message="程序会先备份数据，然后下载更新并自动重启。" confirm-text="开始更新" :busy="updateApplying" @close="confirmUpdate = false" @confirm="applyUpdate" />
  </div>
</template>
