<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { Bot, ChevronDown, FileText, Folder, LoaderCircle, MessageSquare, Plus, SendHorizontal, Sparkles, X } from "lucide-vue-next"
import { api } from "../api/index.js"
import { useToast } from "../store/toast.js"
import MarkdownRenderer from "./MarkdownRenderer.vue"

const router = useRouter()
const route = useRoute()
const toast = useToast()
const composer = ref("")
const messages = ref([])
const conversations = ref([])
const activeConversationID = ref(String(route.query.conversation || ""))
const activeContextSummary = ref("")
const sending = ref(false)
const configured = ref(null)
const conversationReady = ref(false)
const messageList = ref(null)
const folderOptions = ref([])
const folderOptionsLoading = ref(false)
const selectedFolderID = ref(readPositiveID(route.query.folder))
const folderSelection = ref(selectedFolderID.value)
const folderMenuOpen = ref(false)
const scopeMenuOpen = ref(false)
const selectedItemIDs = ref(readPositiveIDs(route.query.items))
const selectedItems = ref([])

const continueInstruction = "请从你上一条回答结束的位置直接继续。不要重复任何已经输出的内容，保持原有格式，并完成剩余内容。"
const canSend = computed(() => Boolean(composer.value.trim()) && !sending.value && configured.value === true && conversationReady.value)
const selectedFolder = computed(() => folderOptions.value.find((folder) => folder.id === selectedFolderID.value) || null)
const selectedFolderOption = computed(() => folderOptions.value.find((folder) => folder.id === readPositiveID(folderSelection.value)) || null)
const selectedFolderLabel = computed(() => selectedFolderOption.value?.path || "整个资料库")
const hasScopedContext = computed(() => selectedFolderID.value !== null || selectedItemIDs.value.length > 0)
const activeConversation = computed(() => conversations.value.find((conversation) => conversation.id === activeConversationID.value) || null)
const conversationRows = computed(() => conversations.value.map((conversation) => ({
  id: conversation.id,
  title: conversation.title || "新对话",
  active: conversation.id === activeConversationID.value,
})))
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

function newConversationID() {
  const generated = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  return `chat-${generated}`
}

function conversationTitle(sourceMessages, fallback = "新对话") {
  const firstQuestion = sourceMessages.find((message) => message.role === "user")?.content
  const title = String(firstQuestion || "").replace(/\s+/g, " ").trim()
  return Array.from(title).slice(0, 48).join("") || fallback
}

function normalizeConversation(conversation) {
  const id = String(conversation?.id || "")
  if (!/^[A-Za-z0-9_-]{1,80}$/.test(id)) return null
  const restoredMessages = (conversation.messages || [])
    .filter((message) => message?.role === "user" || message?.role === "assistant")
    .map((message) => ({
      role: message.role,
      content: String(message.content || ""),
      scope: String(message.scope || ""),
      model: String(message.model || ""),
      sources: Array.isArray(message.sources) ? message.sources : [],
      incomplete: Boolean(message.incomplete),
      context_compacted: Boolean(message.context_compacted),
    }))
  return {
    id,
    title: String(conversation.title || conversationTitle(restoredMessages)),
    folder_id: readPositiveID(conversation.folder_id),
    item_ids: readPositiveIDs((conversation.item_ids || []).join(",")),
    messages: restoredMessages,
    context_summary: String(conversation.context_summary || ""),
  }
}

function createConversation() {
  return {
    id: newConversationID(),
    title: "新对话",
    folder_id: selectedFolderID.value,
    item_ids: [...selectedItemIDs.value],
    messages: [],
    context_summary: "",
  }
}

function restoreMessages(conversation) {
  return (conversation.messages || []).map((message, index) => ({ ...message, id: `saved-${conversation.id}-${index}` }))
}

async function syncConversationQuery() {
  if (!activeConversationID.value) return
  const query = { ...route.query, conversation: activeConversationID.value }
  delete query.folder
  delete query.items
  await router.replace({ name: "ai", query })
}

async function activateConversation(conversation, updateURL = true) {
  if (!conversation || sending.value) return
  activeConversationID.value = conversation.id
  messages.value = restoreMessages(conversation)
  activeContextSummary.value = String(conversation.context_summary || "")
  selectedFolderID.value = readPositiveID(conversation.folder_id)
  folderSelection.value = selectedFolderID.value
  selectedItemIDs.value = readPositiveIDs((conversation.item_ids || []).join(","))
  await loadSelectedItems()
  if (updateURL) await syncConversationQuery()
  if (messages.value.length) scrollToLatest()
}

async function selectConversation(id) {
  if (id === activeConversationID.value || sending.value) return
  const conversation = conversations.value.find((item) => item.id === id)
  if (conversation) await activateConversation(conversation)
}

async function resetConversation() {
  if (sending.value) return
  composer.value = ""
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  activeContextSummary.value = ""
  const conversation = createConversation()
  conversations.value.unshift(conversation)
  await activateConversation(conversation)
  await saveConversation()
  toast.success("已开始新的独立对话")
}

function chooseFolder(folderID) {
  folderSelection.value = folderID
  folderMenuOpen.value = false
}

function closeFolderMenu(event) {
  const target = event.target instanceof Element ? event.target : null
  if (!target?.closest(".ai-scope-popover")) {
    folderMenuOpen.value = false
    scopeMenuOpen.value = false
  }
}

function closeFolderMenuOnEscape(event) {
  if (event.key === "Escape") {
    folderMenuOpen.value = false
    scopeMenuOpen.value = false
  }
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
      await saveConversation()
      await syncConversationQuery()
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
    toast.warning("已移除不存在或已删除的转发资料")
  }
}

async function applyFolderScope() {
  selectedFolderID.value = readPositiveID(folderSelection.value)
  folderMenuOpen.value = false
  scopeMenuOpen.value = false
  await saveConversation()
  await syncConversationQuery()
  toast.success(selectedFolderID.value ? `已索引路径：${selectedFolder.value?.path || "所选文件夹"}` : "已恢复为整个资料库")
}

async function removeSelectedItem(id) {
  selectedItemIDs.value = selectedItemIDs.value.filter((itemID) => itemID !== id)
  selectedItems.value = selectedItems.value.filter((item) => item.id !== id)
  await saveConversation()
  await syncConversationQuery()
}

async function clearScopedContext() {
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  await saveConversation()
  await syncConversationQuery()
  toast.info("已恢复为整个资料库")
}

function historyForRequest() {
  return messages.value
    .filter((message) => (message.role === "user" || message.role === "assistant") && !message.context_compacted)
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
      context_compacted: Boolean(message.context_compacted),
    }))
}

function updateActiveConversation(promote = false) {
  const currentIndex = conversations.value.findIndex((conversation) => conversation.id === activeConversationID.value)
  if (currentIndex < 0) return false
  const current = conversations.value[currentIndex]
  const updated = {
    ...current,
    title: conversationTitle(messages.value, current.title),
    folder_id: selectedFolderID.value,
    item_ids: [...selectedItemIDs.value],
    messages: persistableMessages(),
    context_summary: activeContextSummary.value,
  }
  if (promote && currentIndex > 0) {
    conversations.value.splice(currentIndex, 1)
    conversations.value.unshift(updated)
  } else {
    conversations.value.splice(currentIndex, 1, updated)
  }
  return true
}

async function saveConversation(promote = false) {
  if (!updateActiveConversation(promote)) return
  try {
    const result = await api.saveAIConversation(conversations.value)
    if (Array.isArray(result?.conversations)) {
      conversations.value = result.conversations.map(normalizeConversation).filter(Boolean)
    }
  } catch (error) {
    toast.warning(error.message || "本次对话暂未保存")
  }
}

async function loadConversation() {
  try {
    const result = await api.getAIConversation()
    conversations.value = (result.conversations || []).map(normalizeConversation).filter(Boolean)
    const requestedID = String(route.query.conversation || "")
    const hasIncomingScope = selectedFolderID.value !== null || selectedItemIDs.value.length > 0
    const initial = conversations.value.find((conversation) => conversation.id === requestedID) || (hasIncomingScope ? null : conversations.value[0])
    if (initial) await activateConversation(initial, requestedID !== initial.id)
  } catch (error) {
    toast.warning(error.message || "无法恢复已保存的对话")
  } finally {
    conversationReady.value = true
  }
}

async function ensureActiveConversation() {
  if (activeConversation.value) return
  const conversation = createConversation()
  conversations.value.unshift(conversation)
  await activateConversation(conversation)
}

function chatRequest(message, history) {
  const request = { message, history, folder_id: selectedFolderID.value, item_ids: selectedItemIDs.value }
  if (activeContextSummary.value) request.context_summary = activeContextSummary.value
  return request
}

function applyCompactionResult(result) {
  if (typeof result.context_summary === "string") activeContextSummary.value = result.context_summary
  let remaining = Number(result.compacted_messages) || 0
  if (remaining <= 0) return
  for (const message of messages.value) {
    if (remaining <= 0) break
    if ((message.role !== "user" && message.role !== "assistant") || message.context_compacted) continue
    message.context_compacted = true
    remaining -= 1
  }
  toast.info("较早对话已自动整理，后续回答会保留其中的关键信息")
}

async function send() {
  const content = composer.value.trim()
  if (!content || sending.value) return
  if (configured.value === false) return toast.warning("请先在设置中配置 DeepSeek API Key")
  if (configured.value !== true) return
  await ensureActiveConversation()
  const history = historyForRequest()
  messages.value.push({ id: `user-${Date.now()}`, role: "user", content, scope: hasScopedContext.value ? scopeSummary.value : "" })
  composer.value = ""
  sending.value = true
  scrollToLatest()
  let shouldSaveConversation = true
  try {
    const result = await api.aiChat(chatRequest(content, history))
    applyCompactionResult(result)
    messages.value.push({ id: `assistant-${Date.now()}`, role: "assistant", content: result.answer, model: result.model, sources: result.sources || [], incomplete: Boolean(result.incomplete) })
  } catch (error) {
    const needsSetup = error?.detail?.code === "deepseek_not_configured" || error?.code === "deepseek_not_configured"
    if (needsSetup) configured.value = false
    messages.value.push({ id: `error-${Date.now()}`, role: "error", content: error.message || "AI 暂时无法回答，请稍后重试。", setup: needsSetup })
  } finally {
    sending.value = false
    if (shouldSaveConversation) await saveConversation(true)
    scrollToLatest()
  }
}

async function continueGeneration(message) {
  if (!message?.incomplete || sending.value || configured.value !== true) return
  const history = historyForRequest()
  sending.value = true
  scrollToLatest()
  try {
    const result = await api.aiChat(chatRequest(continueInstruction, history))
    applyCompactionResult(result)
    const messageIndex = messages.value.findIndex((item) => item.id === message.id)
    if (messageIndex >= 0) messages.value[messageIndex] = { ...messages.value[messageIndex], incomplete: false }
    messages.value.push({ id: `assistant-${Date.now()}`, role: "assistant", content: result.answer, model: result.model, sources: result.sources || [], incomplete: Boolean(result.incomplete) })
    await saveConversation(true)
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
  await loadFolderOptions()
  await loadSelectedItems()
  await loadConversation()
})

onBeforeUnmount(() => {
  document.removeEventListener("pointerdown", closeFolderMenu)
  document.removeEventListener("keydown", closeFolderMenuOnEscape)
})

watch(() => String(route.query.conversation || ""), async (nextID) => {
  if (!conversationReady.value || !nextID || nextID === activeConversationID.value) return
  const conversation = conversations.value.find((item) => item.id === nextID)
  if (conversation) await activateConversation(conversation, false)
})
</script>

<template>
  <div class="ai-chat-page">
    <section class="ai-workspace" aria-label="AI 学习助手">
      <aside class="ai-history-rail" aria-label="当前对话记录">
        <header class="ai-history-rail__head"><div><MessageSquare :size="18" /><strong>对话记录</strong></div><button type="button" title="新对话" aria-label="新对话" :disabled="sending || !conversationReady" @click="resetConversation"><Plus :size="17" /></button></header>
        <div class="ai-history-rail__list">
          <button v-for="conversation in conversationRows" :key="conversation.id" type="button" class="ai-history-item" :class="{ 'is-active': conversation.active }" :disabled="sending" @click="selectConversation(conversation.id)"><MessageSquare :size="14" /><span>{{ conversation.title }}</span></button>
          <div v-if="!conversationRows.length" class="ai-history-rail__empty"><Sparkles :size="16" /><span>开始一段新对话吧</span></div>
        </div>
      </aside>

      <section class="ai-chat-panel">
        <div ref="messageList" class="ai-message-list" aria-live="polite" role="log">
          <div v-if="!messages.length" class="ai-chat-empty"><h1>有什么可以帮你的？</h1></div>
          <article v-for="message in messages" :key="message.id" :data-message-id="message.id" class="ai-message" :class="`ai-message--${message.role}`">
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

        <form class="ai-composer" @submit.prevent="send">
          <div class="ai-composer__entry">
            <div class="ai-scope-popover">
              <button type="button" class="ai-composer__scope-button" aria-label="选择 AI 资料范围" :aria-expanded="scopeMenuOpen" :class="{ 'is-active': hasScopedContext }" @click="scopeMenuOpen = !scopeMenuOpen"><Folder :size="19" /><i v-if="hasScopedContext"></i></button>
              <section v-if="scopeMenuOpen" class="ai-scope-popover__card" aria-label="AI 资料范围">
                <header><strong>资料范围</strong><button v-if="hasScopedContext" type="button" aria-label="清除资料范围" @click="clearScopedContext"><X :size="14" /></button></header>
                <div class="ai-path-combobox"><button type="button" class="ai-path-combobox__trigger" aria-label="资料路径" aria-haspopup="listbox" :aria-expanded="folderMenuOpen" :disabled="folderOptionsLoading" @click="folderMenuOpen = !folderMenuOpen"><Folder :size="14" /><span>{{ selectedFolderLabel }}</span><ChevronDown :size="15" :class="{ 'is-open': folderMenuOpen }" /></button><div v-if="folderMenuOpen" class="ai-path-combobox__menu" role="listbox" aria-label="资料路径"><button type="button" role="option" :aria-selected="folderSelection === null" @click="chooseFolder(null)"><span>整个资料库</span><small>不限定路径</small></button><button v-for="folder in folderOptions" :key="folder.id" type="button" role="option" :aria-selected="readPositiveID(folderSelection) === folder.id" @click="chooseFolder(folder.id)"><span>{{ folder.path }}</span></button></div></div>
                <footer><small>仅使用可读文本</small><button type="button" class="ai-scope-popover__apply" :disabled="folderOptionsLoading" @click="applyFolderScope">完成</button></footer>
                <div v-if="selectedItems.length" class="ai-context-picker__items"><span>已转发资料</span><button v-for="item in selectedItems" :key="item.id" type="button" @click="removeSelectedItem(item.id)"><FileText :size="14" />{{ item.name }}<X :size="13" /></button></div>
              </section>
            </div>
            <textarea v-model="composer" rows="1" maxlength="2000" placeholder="输入你的问题…" :disabled="sending || configured !== true" @keydown="keydown" />
            <button type="submit" class="ai-composer__send" :disabled="!canSend" :aria-label="sending ? '正在发送' : '发送消息'"><LoaderCircle v-if="sending" :size="18" /><SendHorizontal v-else :size="18" /></button>
          </div>
          <small class="ai-composer__context-status">1M 上下文 · 接近上限时自动整理早期对话</small>
          <div v-if="configured === false" class="ai-composer__setup">尚未连接 DeepSeek <button type="button" @click="openSettings">去配置</button></div>
        </form>
      </section>
    </section>
  </div>
</template>
