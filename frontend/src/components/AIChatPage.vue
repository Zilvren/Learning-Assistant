<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { Bot, BookOpenText, ChevronDown, CornerDownLeft, FileText, Folder, Lightbulb, LoaderCircle, RefreshCw, Settings2, Sparkles, X } from "lucide-vue-next"
import { api } from "../api/index.js"
import { useToast } from "../store/toast.js"
import MarkdownRenderer from "./MarkdownRenderer.vue"
import BaseButton from "./ui/BaseButton.vue"
import PageHeader from "./ui/PageHeader.vue"

const router = useRouter()
const route = useRoute()
const toast = useToast()
const composer = ref("")
const messages = ref([])
const sending = ref(false)
const configured = ref(null)
const conversationReady = ref(false)
const messageList = ref(null)
const folderOptions = ref([])
const folderOptionsLoading = ref(false)
const selectedFolderID = ref(readPositiveID(route.query.folder))
const folderSelection = ref(selectedFolderID.value)
const folderMenuOpen = ref(false)
const selectedItemIDs = ref(readPositiveIDs(route.query.items))
const selectedItems = ref([])

const quickPrompts = [
  "请分析我的资料库，给出本周最值得优先复习的内容。",
  "根据我的错题和笔记，列出 3 个薄弱知识点与行动建议。",
  "帮我根据当前资料和每日目标，制定一个 30 分钟学习计划。",
]
const continueInstruction = "请从你上一条回答结束的位置直接继续。不要重复任何已经输出的内容，保持原有格式，并完成剩余内容。"
const canSend = computed(() => Boolean(composer.value.trim()) && !sending.value && configured.value === true && conversationReady.value)
const selectedFolder = computed(() => folderOptions.value.find((folder) => folder.id === selectedFolderID.value) || null)
const selectedFolderOption = computed(() => folderOptions.value.find((folder) => folder.id === readPositiveID(folderSelection.value)) || null)
const selectedFolderLabel = computed(() => selectedFolderOption.value?.path || "整个资料库")
const hasScopedContext = computed(() => selectedFolderID.value !== null || selectedItemIDs.value.length > 0)
const scopeSummary = computed(() => {
  const parts = []
  if (selectedFolder.value) parts.push(`路径：${selectedFolder.value.path}`)
  if (selectedItems.value.length) parts.push(`已转发 ${selectedItems.value.length} 项资料`)
  return parts.join(" · ") || "整个资料库"
})

function readPositiveID(value) {
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : null
}

function readPositiveIDs(value) {
  const seen = new Set()
  return String(value || "").split(",").map((item) => readPositiveID(item)).filter((id) => id && !seen.has(id) && seen.add(id)).slice(0, 60)
}

function scrollToLatest() {
  nextTick(() => {
    const list = messageList.value
    if (list && typeof list.scrollTo === "function") list.scrollTo({ top: list.scrollHeight, behavior: "smooth" })
  })
}

async function resetConversation() {
  if (sending.value || !messages.value.length) {
    composer.value = ""
    return
  }
  try {
    await api.clearAIConversation()
    messages.value = []
    composer.value = ""
    toast.success("已开始新的对话")
  } catch (error) {
    toast.error(error.message || "对话清除失败，请重试")
  }
}

function setQuickPrompt(value) {
  composer.value = value
}

function chooseFolder(folderID) {
  folderSelection.value = folderID
  folderMenuOpen.value = false
}

function closeFolderMenu(event) {
  const target = event.target instanceof Element ? event.target : null
  if (!target?.closest(".ai-path-combobox")) folderMenuOpen.value = false
}

function closeFolderMenuOnEscape(event) {
  if (event.key === "Escape") folderMenuOpen.value = false
}

async function syncScopeQuery() {
  const query = { ...route.query }
  if (selectedFolderID.value) query.folder = String(selectedFolderID.value)
  else delete query.folder
  if (selectedItemIDs.value.length) query.items = selectedItemIDs.value.join(",")
  else delete query.items
  await router.replace({ name: "ai", query })
}

async function loadFolderOptions() {
  folderOptionsLoading.value = true
  const folders = []
  try {
    async function visit(parentID, parentPath) {
      const result = await api.getLibraryItems({ parentId: parentID, kind: "folder" })
      const children = [...(result.items || [])].sort((left, right) => String(left.name || "").localeCompare(String(right.name || ""), "zh-CN"))
      for (const folder of children) {
        const path = parentPath ? `${parentPath} / ${folder.name}` : folder.name
        folders.push({ id: folder.id, path })
        await visit(folder.id, path)
      }
    }
    await visit(null, "")
    folderOptions.value = folders
    if (selectedFolderID.value && !folders.some((folder) => folder.id === selectedFolderID.value)) {
      selectedFolderID.value = null
      folderSelection.value = null
      await syncScopeQuery()
      toast.warning("原资料路径已不可用，已恢复为整个资料库")
    }
  } catch (error) {
    toast.error(error.message || "资料路径读取失败")
  } finally {
    folderOptionsLoading.value = false
  }
}

async function loadSelectedItems() {
  const requestedIDs = [...selectedItemIDs.value]
  if (!requestedIDs.length) {
    selectedItems.value = []
    return
  }
  const loaded = await Promise.all(requestedIDs.map((id) => api.getLibraryItem(id).catch(() => null)))
  const available = loaded.filter(Boolean)
  const availableIDs = available.map((item) => item.id)
  selectedItems.value = available
  if (availableIDs.length !== requestedIDs.length) {
    selectedItemIDs.value = availableIDs
    await syncScopeQuery()
    toast.warning("已移除不存在或已删除的转发资料")
  }
}

async function applyFolderScope() {
  selectedFolderID.value = readPositiveID(folderSelection.value)
  folderMenuOpen.value = false
  await syncScopeQuery()
  toast.success(selectedFolderID.value ? `已索引路径：${selectedFolder.value?.path || "所选文件夹"}` : "已恢复为整个资料库")
}

async function removeSelectedItem(id) {
  selectedItemIDs.value = selectedItemIDs.value.filter((itemID) => itemID !== id)
  selectedItems.value = selectedItems.value.filter((item) => item.id !== id)
  await syncScopeQuery()
}

async function clearScopedContext() {
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  await syncScopeQuery()
  toast.info("已恢复为整个资料库")
}

function historyForRequest() {
  return messages.value
    .filter((message) => message.role === "user" || message.role === "assistant")
    .slice(-8)
    .map((message) => ({ role: message.role, content: message.content }))
}

function persistableMessages() {
  return messages.value
    .filter((message) => message.role === "user" || message.role === "assistant")
    .map((message) => ({
      role: message.role,
      content: message.content,
      scope: message.scope || "",
      model: message.model || "",
      sources: message.sources || [],
      incomplete: Boolean(message.incomplete),
    }))
}

async function saveConversation() {
  try {
    await api.saveAIConversation(persistableMessages())
  } catch (error) {
    toast.warning(error.message || "本次对话暂未保存")
  }
}

async function loadConversation() {
  try {
    const result = await api.getAIConversation()
    const restored = (result.messages || [])
      .filter((message) => message?.role === "user" || message?.role === "assistant")
      .map((message, index) => ({ ...message, id: `saved-${Date.now()}-${index}` }))
    messages.value = restored
    if (restored.length) scrollToLatest()
  } catch (error) {
    toast.warning(error.message || "无法恢复已保存的对话")
  } finally {
    conversationReady.value = true
  }
}

async function send() {
  const content = composer.value.trim()
  if (!content || sending.value) return
  if (configured.value === false) return toast.warning("请先在设置中配置 DeepSeek API Key")
  if (configured.value !== true) return
  const history = historyForRequest()
  messages.value.push({ id: `user-${Date.now()}`, role: "user", content, scope: hasScopedContext.value ? scopeSummary.value : "" })
  composer.value = ""
  sending.value = true
  scrollToLatest()
  let shouldSaveConversation = false
  try {
    const result = await api.aiChat({ message: content, history, folder_id: selectedFolderID.value, item_ids: selectedItemIDs.value })
    messages.value.push({ id: `assistant-${Date.now()}`, role: "assistant", content: result.answer, model: result.model, sources: result.sources || [], incomplete: Boolean(result.incomplete) })
    shouldSaveConversation = true
  } catch (error) {
    const needsSetup = error?.detail?.code === "deepseek_not_configured" || error?.code === "deepseek_not_configured"
    if (needsSetup) configured.value = false
    messages.value.push({ id: `error-${Date.now()}`, role: "error", content: error.message || "AI 暂时无法回答，请稍后重试。", setup: needsSetup })
  } finally {
    sending.value = false
    if (shouldSaveConversation) await saveConversation()
    scrollToLatest()
  }
}

async function continueGeneration(message) {
  if (!message?.incomplete || sending.value || configured.value !== true) return
  const history = historyForRequest()
  sending.value = true
  scrollToLatest()
  try {
    const result = await api.aiChat({ message: continueInstruction, history, folder_id: selectedFolderID.value, item_ids: selectedItemIDs.value })
    const messageIndex = messages.value.findIndex((item) => item.id === message.id)
    if (messageIndex >= 0) messages.value[messageIndex] = { ...messages.value[messageIndex], incomplete: false }
    messages.value.push({ id: `assistant-${Date.now()}`, role: "assistant", content: result.answer, model: result.model, sources: result.sources || [], incomplete: Boolean(result.incomplete) })
    await saveConversation()
  } catch (error) {
    const needsSetup = error?.detail?.code === "deepseek_not_configured" || error?.code === "deepseek_not_configured"
    if (needsSetup) configured.value = false
    toast.error(error.message || "继续生成失败，请重试")
  } finally {
    sending.value = false
    scrollToLatest()
  }
}

function keydown(event) {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault()
    void send()
  }
}

function openSettings() {
  router.push({ name: "settings", hash: "#ai" })
}

function openSource(source) {
  if (source.source_type === "library") {
    router.push({ name: "library-item", params: { itemId: source.id } })
    return
  }
  toast.info(`这条建议参考了错题 #${source.id}：${source.title || "未命名错题"}`)
}

onMounted(async () => {
  document.addEventListener("pointerdown", closeFolderMenu)
  document.addEventListener("keydown", closeFolderMenuOnEscape)
  try {
    configured.value = Boolean((await api.getDeepSeekToken()).configured)
  } catch {
    configured.value = false
  }
  await Promise.all([loadConversation(), loadFolderOptions(), loadSelectedItems()])
})

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", closeFolderMenu)
  document.removeEventListener("keydown", closeFolderMenuOnEscape)
})

watch(() => route.query, async () => {
  const nextFolderID = readPositiveID(route.query.folder)
  const nextItemIDs = readPositiveIDs(route.query.items)
  if (nextFolderID === selectedFolderID.value && nextItemIDs.join(",") === selectedItemIDs.value.join(",")) return
  selectedFolderID.value = nextFolderID
  folderSelection.value = nextFolderID
  selectedItemIDs.value = nextItemIDs
  await loadSelectedItems()
})
</script>

<template>
  <div class="ai-chat-page page-stage">
    <PageHeader eyebrow="资料库智能分析" title="AI 学习助手" description="让 DeepSeek 基于可读笔记、Office 文本、错题与学习进度给出建议；对话会自动保存。">
      <template #actions><BaseButton :disabled="sending || !conversationReady" @click="resetConversation"><template #icon><RefreshCw :size="16" /></template>新对话</BaseButton></template>
    </PageHeader>

    <section v-if="configured === false" class="ai-setup-callout">
      <div><Bot :size="22" /><div><strong>先连接 DeepSeek</strong><p>API Key 只保存在后端并加密存储；不会显示在聊天页面。</p></div></div>
      <BaseButton variant="primary" @click="openSettings"><template #icon><Settings2 :size="16" /></template>去配置</BaseButton>
    </section>

    <section class="ai-context-picker paper-panel" aria-label="AI 资料范围">
      <div class="ai-context-picker__head"><div><Folder :size="20" /><div><strong>索引资料库路径</strong><p>选择路径后，AI 只会阅读该目录及其下的可读文件；也可叠加从资料库转发的文件。</p></div></div><button v-if="hasScopedContext" type="button" class="ai-context-picker__clear" @click="clearScopedContext"><X :size="15" />清除范围</button></div>
      <div class="ai-context-picker__controls">
        <div class="ai-context-picker__field"><span id="ai-path-label">资料路径</span><div class="ai-path-combobox"><button type="button" class="ai-path-combobox__trigger" aria-labelledby="ai-path-label" aria-haspopup="listbox" :aria-expanded="folderMenuOpen" :disabled="folderOptionsLoading" @click="folderMenuOpen = !folderMenuOpen"><span>{{ selectedFolderLabel }}</span><ChevronDown :size="18" :class="{ 'is-open': folderMenuOpen }" /></button><div v-if="folderMenuOpen" class="ai-path-combobox__menu" role="listbox" aria-labelledby="ai-path-label"><button type="button" role="option" :aria-selected="folderSelection === null" @click="chooseFolder(null)"><span>整个资料库</span><small>不限定路径</small></button><button v-for="folder in folderOptions" :key="folder.id" type="button" role="option" :aria-selected="readPositiveID(folderSelection) === folder.id" @click="chooseFolder(folder.id)"><span>{{ folder.path }}</span></button></div></div></div>
        <BaseButton class="ai-context-picker__apply" :disabled="folderOptionsLoading" @click="applyFolderScope"><template #icon><Folder :size="16" /></template>{{ selectedFolderID === readPositiveID(folderSelection) ? (selectedFolderID ? '已索引' : '使用整个库') : '索引路径' }}</BaseButton>
        <span class="ai-context-picker__status">当前：{{ scopeSummary }}</span>
      </div>
      <div v-if="selectedItems.length" class="ai-context-picker__items"><span>已从资料库转发</span><button v-for="item in selectedItems" :key="item.id" type="button" @click="removeSelectedItem(item.id)"><FileText :size="14" />{{ item.name }}<X :size="13" /></button></div>
    </section>

    <section class="ai-chat-layout">
      <aside class="ai-chat-guide paper-panel">
        <div class="ai-chat-guide__icon"><Sparkles :size="20" /></div>
        <h2>把资料变成行动</h2>
        <p>AI 会选取与问题关联的可读内容，结合错题和学习统计提出建议。</p>
        <div class="ai-privacy-note"><BookOpenText :size="15" /><span>只发送文本内容和摘要；图片、PDF 原件及密钥不会发送。</span></div>
        <div class="ai-quick-prompts"><strong>试试这样问</strong><button v-for="prompt in quickPrompts" :key="prompt" type="button" :disabled="sending || configured !== true" @click="setQuickPrompt(prompt)"><Lightbulb :size="14" />{{ prompt }}</button></div>
      </aside>

      <section class="ai-chat-panel paper-panel">
        <div ref="messageList" class="ai-message-list" aria-live="polite">
          <div v-if="!messages.length" class="ai-chat-empty"><Bot :size="29" /><strong>今天想从哪里开始？</strong><p>例如：帮我梳理错题中的薄弱点，或根据资料库安排复习优先级。</p></div>
          <article v-for="message in messages" :key="message.id" class="ai-message" :class="`ai-message--${message.role}`">
            <div class="ai-message__avatar"><component :is="message.role === 'assistant' ? Bot : message.role === 'user' ? FileText : Sparkles" :size="16" /></div>
            <div class="ai-message__body">
              <span class="ai-message__label">{{ message.role === 'assistant' ? 'AI 学习助手' : message.role === 'user' ? '你' : '提示' }}</span>
              <MarkdownRenderer v-if="message.role === 'assistant'" :content="message.content" />
              <p v-else>{{ message.content }}</p>
              <small v-if="message.scope" class="ai-message__scope">{{ message.scope }}</small>
              <div v-if="message.sources?.length" class="ai-message__sources"><span>本次参考</span><button v-for="source in message.sources" :key="`${source.source_type}-${source.id}`" type="button" @click="openSource(source)">{{ source.source_type === 'error' ? '错题' : '资料' }} · {{ source.title }}</button></div>
              <div v-if="message.incomplete" class="ai-message__continuation"><span>回答达到长度上限，可能尚未完成。</span><button type="button" :disabled="sending || configured !== true" @click="continueGeneration(message)">继续生成</button></div>
              <button v-if="message.setup" type="button" class="ai-message__setup" @click="openSettings">去配置 DeepSeek</button>
              <small v-if="message.model">{{ message.model }}</small>
            </div>
          </article>
          <article v-if="sending" class="ai-message ai-message--assistant"><div class="ai-message__avatar"><Bot :size="16" /></div><div class="ai-message__body ai-message__thinking"><LoaderCircle :size="17" />正在阅读资料库并组织建议…</div></article>
        </div>
        <form class="ai-composer" @submit.prevent="send"><textarea v-model="composer" rows="3" maxlength="2000" placeholder="问问你的资料库：哪些知识点最薄弱？下一步怎么复习？" :disabled="sending || configured !== true" @keydown="keydown" /><div><small>Enter 发送 · Shift + Enter 换行</small><BaseButton type="submit" variant="primary" :disabled="!canSend" :busy="sending"><template #icon><CornerDownLeft :size="16" /></template>发送</BaseButton></div></form>
      </section>
    </section>
  </div>
</template>
