<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { Archive, ArchiveRestore, Bot, ChevronDown, Ellipsis, FileText, Folder, LoaderCircle, MessageSquare, PanelLeftClose, PanelLeftOpen, Plus, SendHorizontal, Sparkles, Trash2, X } from "lucide-vue-next"
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
const sending = ref(false)
const configured = ref(null)
const harnessReady = ref(null)
const harnessReason = ref("")
const conversationReady = ref(false)
const conversationSwitching = ref(false)
const pendingConversationID = ref("")
const conversationOperationID = ref("")
const historyCollapsed = ref(readHistoryCollapsed())
const historyView = ref("active")
const historyMenuID = ref("")
const historyMenuPosition = ref({ top: 0, left: 0 })
const chatPanel = ref(null)
const messageList = ref(null)
const folderOptions = ref([])
const folderOptionsLoading = ref(false)
const selectedFolderID = ref(readPositiveID(route.query.folder))
const folderSelection = ref(selectedFolderID.value)
const folderMenuOpen = ref(false)
const scopeMenuOpen = ref(false)
const chatOnly = ref(route.query.chat_only === "1")
const selectedItemIDs = ref(readPositiveIDs(route.query.items))
const selectedItems = ref([])
const activeTurnID = ref("")
const activeTurnStatus = ref("")
const activeApproval = ref(null)
const activeApprovalResolving = ref(false)
let activeTurnStreamController = null

const continueInstruction = "请从你上一条回答结束的位置直接继续。不要重复任何已经输出的内容，保持原有格式，并完成剩余内容。"
const contextSummaryWarmupMessages = 120
const contextSummaryIntervalMessages = 40
const canSend = computed(() => Boolean(composer.value.trim()) && !sending.value && !conversationSwitching.value && configured.value === true && harnessReady.value === true && conversationReady.value)
const selectedFolder = computed(() => folderOptions.value.find((folder) => folder.id === selectedFolderID.value) || null)
const selectedFolderOption = computed(() => folderOptions.value.find((folder) => folder.id === readPositiveID(folderSelection.value)) || null)
const selectedFolderLabel = computed(() => selectedFolderOption.value?.path || "整个资料库")
const hasScopedContext = computed(() => selectedFolderID.value !== null || selectedItemIDs.value.length > 0)
const activeConversations = computed(() => conversations.value.filter((conversation) => !conversation.archived_at))
const archivedConversations = computed(() => conversations.value.filter((conversation) => conversation.archived_at))
const visibleConversations = computed(() => historyView.value === "archived" ? archivedConversations.value : activeConversations.value)
const activeConversation = computed(() => activeConversations.value.find((conversation) => conversation.id === activeConversationID.value) || null)
const conversationOperationPending = computed(() => Boolean(conversationOperationID.value))
const waitingForWriteApproval = computed(() => Boolean(activeApproval.value?.id) && activeTurnStatus.value === "waiting_approval")
const historyMenuConversation = computed(() => conversations.value.find((conversation) => conversation.id === historyMenuID.value) || null)
const historyMenuStyle = computed(() => ({
  top: `${historyMenuPosition.value.top}px`,
  left: `${historyMenuPosition.value.left}px`,
}))
const conversationRows = computed(() => visibleConversations.value.map((conversation) => ({
  id: conversation.id,
  title: conversation.title || "新对话",
  active: !conversation.archived_at && conversation.id === activeConversationID.value,
  archived: Boolean(conversation.archived_at),
  loading: conversation.id === pendingConversationID.value || conversation.id === conversationOperationID.value,
})))
const scopeSummary = computed(() => {
  if (chatOnly.value) return "纯聊天 · 不主动使用资料库"
  const parts = []
  if (selectedFolder.value) parts.push(`路径：${selectedFolder.value.path}`)
  if (selectedItems.value.length) parts.push(`已转发 ${selectedItems.value.length} 项资料`)
  return parts.join(" · ") || "整个资料库"
})

// readPositiveID 在当前界面组件中完成交互或数据处理。
function readPositiveID(value) {
  const id = Number(value)
  return Number.isSafeInteger(id) && id > 0 ? id : null
}

// readPositiveIDs 在当前界面组件中完成交互或数据处理。
function readPositiveIDs(value) {
  const seen = new Set()
  return String(value || "").split(",").map((item) => readPositiveID(item)).filter((id) => id && !seen.has(id) && seen.add(id)).slice(0, 60)
}

// readHistoryCollapsed 读取本浏览器保存的对话侧栏展示偏好，读取失败时保持默认展开。
function readHistoryCollapsed() {
  try {
    return globalThis.localStorage?.getItem("learning-assistant:ai-history-collapsed") === "true"
  } catch {
    return false
  }
}

// toggleHistoryCollapsed 切换对话侧栏的折叠状态，并只写入当前浏览器的本地存储。
function toggleHistoryCollapsed() {
  const panel = chatPanel.value
  const before = panel?.getBoundingClientRect?.()
  panel?.getAnimations?.().forEach((animation) => animation.cancel())
  historyCollapsed.value = !historyCollapsed.value
  try {
    globalThis.localStorage?.setItem("learning-assistant:ai-history-collapsed", String(historyCollapsed.value))
  } catch {
    // 无痕模式或受限环境不能保存偏好时，仍允许本次页面内切换。
  }
  nextTick(() => animateHistoryLayout(panel, before))
}

// animateHistoryLayout 用 FLIP 位移衔接侧栏切换后的聊天画布，避免直接改变宽度产生生硬跳变。
function animateHistoryLayout(panel, before) {
  if (!panel || !before || typeof panel.animate !== "function" || globalThis.matchMedia?.("(prefers-reduced-motion: reduce)").matches) return
  const after = panel.getBoundingClientRect()
  const offset = before.left - after.left
  if (Math.abs(offset) < 1) return
  const animation = panel.animate([
    { opacity: 0.985, transform: `translateX(${offset}px)` },
    { opacity: 1, transform: "translateX(0)" },
  ], {
    duration: 240,
    easing: "cubic-bezier(0.23, 1, 0.32, 1)",
    fill: "both",
  })
  animation.addEventListener?.("finish", () => animation.cancel(), { once: true })
}

// scrollToLatest 在当前界面组件中完成交互或数据处理。
function scrollToLatest() {
  nextTick(() => {
    const list = messageList.value
    if (list && typeof list.scrollTo === "function") list.scrollTo({ top: list.scrollHeight, behavior: "smooth" })
  })
}

// newConversationID 在当前界面组件中完成交互或数据处理。
function newConversationID() {
  const generated = globalThis.crypto?.randomUUID?.() || `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  return `chat-${generated}`
}

// conversationTitle 在当前界面组件中完成交互或数据处理。
function conversationTitle(sourceMessages, fallback = "新对话") {
  const firstQuestion = sourceMessages.find((message) => message.role === "user")?.content
  const title = String(firstQuestion || "").replace(/\s+/g, " ").trim()
  return Array.from(title).slice(0, 48).join("") || fallback
}

function normalizeContextMemory(memory) {
  if (!memory || typeof memory !== "object") return null
  const list = (value) => Array.isArray(value) ? value.map((item) => String(item || "").trim()).filter(Boolean).slice(0, 16) : []
  const normalized = {
    goal: String(memory.goal || "").trim(), completed: list(memory.completed), decisions: list(memory.decisions),
    references: list(memory.references), blockers: list(memory.blockers), next_step: String(memory.next_step || "").trim(),
  }
  return normalized.goal || normalized.next_step || normalized.completed.length || normalized.decisions.length || normalized.references.length || normalized.blockers.length ? normalized : null
}

// normalizeConversation 在当前界面组件中完成交互或数据处理。
function normalizeConversation(conversation) {
  const id = String(conversation?.id || "")
  if (!/^[A-Za-z0-9_-]{1,80}$/.test(id)) return null
  // 旧版本可能留下未完成的空消息；不把它再次写回服务端，以免整组对话无法保存或归档。
  const restoredMessages = (Array.isArray(conversation.messages) ? conversation.messages : [])
    .filter((message) => (message?.role === "user" || message?.role === "assistant") && String(message.content || "").trim())
    .map((message) => ({
      role: message.role,
      content: String(message.content || "").trim(),
      scope: String(message.scope || ""),
      model: String(message.model || ""),
      sources: Array.isArray(message.sources) ? message.sources : [],
      audit: Array.isArray(message.audit) ? message.audit : [],
      incomplete: Boolean(message.incomplete),
    }))
  return {
    id,
    title: String(conversation.title || conversationTitle(restoredMessages)),
    folder_id: readPositiveID(conversation.folder_id),
    item_ids: readPositiveIDs((Array.isArray(conversation.item_ids) ? conversation.item_ids : []).join(",")),
    chat_only: Boolean(conversation.chat_only),
    messages: restoredMessages,
    context_summary: String(conversation.context_summary || "").trim(),
    context_memory: normalizeContextMemory(conversation.context_memory),
    harness_session_id: /^[A-Za-z0-9_-]{1,80}$/.test(String(conversation.harness_session_id || "")) ? String(conversation.harness_session_id) : "",
    // Go 的 archived_at 是 *time.Time；空字符串无法反序列化，应以 null 表示未归档。
    archived_at: conversation.archived_at ? String(conversation.archived_at) : null,
  }
}

// createConversation 在当前界面组件中完成交互或数据处理。
function createConversation() {
  return {
    id: newConversationID(),
    title: "新对话",
    folder_id: selectedFolderID.value,
    item_ids: [...selectedItemIDs.value],
    chat_only: chatOnly.value,
    messages: [],
    context_summary: "",
    context_memory: null,
    harness_session_id: "",
    archived_at: null,
  }
}

// restoreMessages 在当前界面组件中完成交互或数据处理。
function restoreMessages(conversation) {
  return (conversation.messages || []).map((message, index) => ({ ...message, id: `saved-${conversation.id}-${index}` }))
}

// syncConversationQuery 在当前界面组件中完成交互或数据处理。
async function syncConversationQuery() {
  if (!activeConversationID.value) return
  const query = { ...route.query, conversation: activeConversationID.value }
  delete query.folder
  delete query.items
  delete query.chat_only
  if (chatOnly.value) query.chat_only = "1"
  await router.replace({ name: "ai", query })
}

// activateConversation 在当前界面组件中完成交互或数据处理。
async function activateConversation(conversation, updateURL = true) {
  if (!conversation || conversation.archived_at || sending.value) return false
  conversationSwitching.value = true
  pendingConversationID.value = conversation.id
  try {
    const conversationChatOnly = Boolean(conversation.chat_only)
    const folderID = conversationChatOnly ? null : readPositiveID(conversation.folder_id)
    const requestedItemIDs = conversationChatOnly ? [] : readPositiveIDs((conversation.item_ids || []).join(","))
    const resolvedItems = await resolveSelectedItems(requestedItemIDs)
    activeConversationID.value = conversation.id
    messages.value = restoreMessages(conversation)
    selectedFolderID.value = folderID
    folderSelection.value = folderID
    chatOnly.value = conversationChatOnly
    selectedItemIDs.value = resolvedItems.ids
    selectedItems.value = resolvedItems.items
    if (resolvedItems.ids.length !== requestedItemIDs.length) toast.warning("已移除不存在或已删除的转发资料")
    if (updateURL) await syncConversationQuery()
    if (messages.value.length) scrollToLatest()
    return true
  } finally {
    conversationSwitching.value = false
    pendingConversationID.value = ""
  }
}

// selectConversation 在当前界面组件中完成交互或数据处理。
async function selectConversation(id) {
  if (id === activeConversationID.value || sending.value || conversationSwitching.value) return
  historyMenuID.value = ""
  const conversation = activeConversations.value.find((item) => item.id === id)
  if (conversation) await activateConversation(conversation)
}

// resetConversation 在当前界面组件中完成交互或数据处理。
async function resetConversation() {
  if (sending.value) return
  if (activeConversations.value.length >= 24) {
    toast.warning("活跃对话已达 24 条，请先归档一条对话")
    return false
  }
  composer.value = ""
  chatOnly.value = false
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  const conversation = createConversation()
  conversations.value.unshift(conversation)
  await activateConversation(conversation)
  await saveConversation()
  toast.success("已开始新的独立对话")
  return true
}

// toggleConversationMenu 打开或关闭指定对话的操作菜单，并将菜单挂到页面顶层以避开所有滚动裁切。
function toggleConversationMenu(id, event) {
  if (historyMenuID.value === id) {
    historyMenuID.value = ""
    return
  }
  const trigger = event?.currentTarget
  const triggerRect = trigger?.getBoundingClientRect?.()
  const conversation = conversations.value.find((item) => item.id === id)
  const menuWidth = 126
  const menuHeight = conversation?.archived_at ? 104 : 56
  const padding = 8
  let top = padding
  let left = padding
  if (triggerRect) {
    const lowerTop = triggerRect.bottom + 4
    const upperTop = triggerRect.top - menuHeight - 4
    const viewportHeight = globalThis.innerHeight || 0
    const viewportWidth = globalThis.innerWidth || 0
    top = lowerTop + menuHeight <= viewportHeight - padding || upperTop < padding ? lowerTop : upperTop
    left = Math.max(padding, Math.min(viewportWidth - menuWidth - padding, triggerRect.right - menuWidth))
  }
  historyMenuPosition.value = { top, left }
  historyMenuID.value = id
}

// applySavedConversations 用服务端返回的标准化对话集合替换本地状态。
function applySavedConversations(result) {
  if (!Array.isArray(result?.conversations)) return false
  const saved = result.conversations.map(normalizeConversation).filter(Boolean)
  if (saved.length !== result.conversations.length) return false
  conversations.value = saved
  return true
}

// archiveConversation 归档一条对话；当前对话归档后自动切到最近的活跃对话或新建对话。
async function archiveConversation(id) {
  if (sending.value || conversationSwitching.value || conversationOperationID.value) return
  const archivingCurrent = id === activeConversationID.value
  historyMenuID.value = ""
  conversationOperationID.value = id
  try {
    // 无论是否为当前对话，都要先同步整个集合；非当前的本地新对话此前会直接触发“AI 对话不存在”。
    if (!(await saveConversation(false, id))) {
      toast.error("当前对话未保存，已取消归档；请刷新页面后重试")
      return
    }
    const result = await api.archiveAIConversation(id)
    if (!applySavedConversations(result)) throw new Error("归档后的对话数据无效")
    if (archivingCurrent) {
      const nextConversation = activeConversations.value[0]
      if (nextConversation) await activateConversation(nextConversation)
      else await resetConversation()
    }
    toast.success("对话已归档")
  } catch (error) {
    toast.error(error.message || "归档对话失败")
  } finally {
    conversationOperationID.value = ""
  }
}

// restoreConversation 恢复一条归档对话；服务端会拒绝超过活跃对话上限的操作。
async function restoreConversation(id) {
  if (sending.value || conversationSwitching.value || conversationOperationID.value) return
  historyMenuID.value = ""
  conversationOperationID.value = id
  try {
    const result = await api.restoreAIConversation(id)
    if (!applySavedConversations(result)) throw new Error("恢复后的对话数据无效")
    historyView.value = "active"
    toast.success("对话已恢复")
  } catch (error) {
    toast.error(error.message || "恢复对话失败")
  } finally {
    conversationOperationID.value = ""
  }
}

// deleteConversation 永久删除一条已归档对话，并在执行前要求用户确认。
async function deleteConversation(id) {
  if (sending.value || conversationSwitching.value || conversationOperationID.value) return
  if (typeof globalThis.confirm === "function" && !globalThis.confirm("永久删除后无法恢复，确定继续吗？")) return
  historyMenuID.value = ""
  conversationOperationID.value = id
  try {
    const result = await api.deleteAIConversation(id)
    if (!applySavedConversations(result)) throw new Error("删除后的对话数据无效")
    toast.success("归档对话已永久删除")
  } catch (error) {
    toast.error(error.message || "删除对话失败")
  } finally {
    conversationOperationID.value = ""
  }
}

// chooseFolder 在当前界面组件中完成交互或数据处理。
function chooseFolder(folderID) {
  folderSelection.value = folderID
  folderMenuOpen.value = false
}

// closeFolderMenu 在当前界面组件中完成交互或数据处理。
function closeFolderMenu(event) {
  const target = event.target instanceof Element ? event.target : null
  if (!target?.closest(".ai-scope-popover")) {
    folderMenuOpen.value = false
    scopeMenuOpen.value = false
  }
  if (!target?.closest(".ai-history-item-row, .ai-history-item__menu")) historyMenuID.value = ""
}

// closeFolderMenuOnEscape 在当前界面组件中完成交互或数据处理。
function closeFolderMenuOnEscape(event) {
  if (event.key === "Escape") {
    folderMenuOpen.value = false
    scopeMenuOpen.value = false
    historyMenuID.value = ""
  }
}

// loadFolderOptions 在当前界面组件中完成交互或数据处理。
async function loadFolderOptions() {
  folderOptionsLoading.value = true
  const folders = []
  try {
// visit 在当前界面组件中完成交互或数据处理。
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

// resolveSelectedItems 读取指定资料，并返回仍可用的资料及其标识而不立即替换当前界面状态。
async function resolveSelectedItems(requestedIDs) {
  if (!requestedIDs.length) {
    return { ids: [], items: [] }
  }
  const loaded = await Promise.all(requestedIDs.map((id) => api.getLibraryItem(id).catch(() => null)))
  const items = loaded.filter(Boolean)
  return { ids: items.map((item) => item.id), items }
}

// loadSelectedItems 读取当前选择的资料，并移除已经不存在的资料标识。
async function loadSelectedItems() {
  const requestedIDs = [...selectedItemIDs.value]
  const resolved = await resolveSelectedItems(requestedIDs)
  selectedItems.value = resolved.items
  if (resolved.ids.length !== requestedIDs.length) {
    selectedItemIDs.value = resolved.ids
    toast.warning("已移除不存在或已删除的转发资料")
  }
}

// applyFolderScope 在当前界面组件中完成交互或数据处理。
async function applyFolderScope() {
  chatOnly.value = false
  selectedFolderID.value = readPositiveID(folderSelection.value)
  folderMenuOpen.value = false
  scopeMenuOpen.value = false
  await saveConversation()
  await syncConversationQuery()
  toast.success(selectedFolderID.value ? `已索引路径：${selectedFolder.value?.path || "所选文件夹"}` : "已恢复为整个资料库")
}

// removeSelectedItem 在当前界面组件中完成交互或数据处理。
async function removeSelectedItem(id) {
  selectedItemIDs.value = selectedItemIDs.value.filter((itemID) => itemID !== id)
  selectedItems.value = selectedItems.value.filter((item) => item.id !== id)
  await saveConversation()
  await syncConversationQuery()
}

// clearScopedContext 在当前界面组件中完成交互或数据处理。
async function clearScopedContext() {
  chatOnly.value = false
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  await saveConversation()
  await syncConversationQuery()
  toast.info("已恢复为整个资料库")
}

// chooseChatFirstScope 切换为默认不主动检索资料的聊天模式；用户明确提出资料需求时仍可调用资料库工具。
async function chooseChatFirstScope() {
  chatOnly.value = true
  selectedFolderID.value = null
  folderSelection.value = null
  selectedItemIDs.value = []
  selectedItems.value = []
  folderMenuOpen.value = false
  scopeMenuOpen.value = false
  await saveConversation()
  await syncConversationQuery()
  toast.success("已切换为纯聊天；需要时仍可让我查资料")
}

// chooseLibraryScopeMode 恢复默认资料库模式，同时保留用户接下来选择整库或具体路径的空间。
async function chooseLibraryScopeMode() {
  chatOnly.value = false
  await saveConversation()
  await syncConversationQuery()
}

// historyForRequest 在当前界面组件中完成交互或数据处理。
function historyForRequest() {
  return messages.value
    .filter((message) => message.role === "user" || message.role === "assistant")
    .map((message) => ({ role: message.role, content: message.content }))
}

// persistableMessages 在当前界面组件中完成交互或数据处理。
function persistableMessages() {
  return messages.value
    .filter((message) => (message.role === "user" || message.role === "assistant") && String(message.content || "").trim())
    .map((message) => ({
      role: message.role,
      content: String(message.content || "").trim(),
      scope: message.scope || "",
      model: message.model || "",
      sources: message.sources || [],
      audit: message.audit || [],
      incomplete: Boolean(message.incomplete),
    }))
}

// updateActiveConversation 在当前界面组件中完成交互或数据处理。
function updateActiveConversation(promote = false) {
  const currentIndex = conversations.value.findIndex((conversation) => conversation.id === activeConversationID.value)
  if (currentIndex < 0) return false
  const current = conversations.value[currentIndex]
  const updated = {
    ...current,
    title: conversationTitle(messages.value, current.title),
    folder_id: selectedFolderID.value,
    item_ids: [...selectedItemIDs.value],
    chat_only: chatOnly.value,
    messages: persistableMessages(),
  }
  if (promote && currentIndex > 0) {
    conversations.value.splice(currentIndex, 1)
    conversations.value.unshift(updated)
  } else {
    conversations.value.splice(currentIndex, 1, updated)
  }
  return true
}

// saveConversation 在当前界面组件中完成交互或数据处理。
async function saveConversation(promote = false, requiredConversationID = activeConversationID.value) {
  if (!updateActiveConversation(promote)) return false
	const previous = conversations.value
  try {
    const result = await api.saveAIConversation(conversations.value)
    if (!applySavedConversations(result) || !conversations.value.some((conversation) => conversation.id === requiredConversationID)) {
		conversations.value = previous
      throw new Error("服务端未确认保存该对话")
    }
    return true
  } catch (error) {
    toast.warning(error.message || "本次对话暂未保存")
    return false
  }
}

// loadConversation 在当前界面组件中完成交互或数据处理。
async function loadConversation() {
  try {
    const result = await api.getAIConversation()
    conversations.value = (result.conversations || []).map(normalizeConversation).filter(Boolean)
    const requestedID = String(route.query.conversation || "")
    const hasIncomingScope = chatOnly.value || selectedFolderID.value !== null || selectedItemIDs.value.length > 0
    const initial = activeConversations.value.find((conversation) => conversation.id === requestedID) || (hasIncomingScope ? null : activeConversations.value[0])
    if (initial) await activateConversation(initial, requestedID !== initial.id)
	} catch (error) {
		toast.warning(error.message || "无法恢复已保存的对话")
	} finally {
		conversationReady.value = true
		void resumeLatestActiveTurn()
	}
}

// ensureActiveConversation 在当前界面组件中完成交互或数据处理。
async function ensureActiveConversation() {
  if (activeConversation.value) return true
  if (activeConversations.value.length >= 24) {
    toast.warning("活跃对话已达 24 条，请先归档一条对话")
    return false
  }
  const conversation = createConversation()
  conversations.value.unshift(conversation)
  await activateConversation(conversation)
  // 立即持久化，确保首条已发送消息拥有独立对话，即使 AI 请求缓慢、被取消或失败也不受影响。
  await saveConversation()
  return true
}

// shouldCompactContext 控制自动滚动摘要频率：长对话首次达到阈值后每 40 条消息更新一次。
function shouldCompactContext(history) {
  return history.length >= contextSummaryWarmupMessages && history.length % contextSummaryIntervalMessages === 0
}

// chatRequest 将应用级长期摘要与本轮上下文一并交给 Harness；摘要到达阈值时请求生成下一版。
function chatRequest(message, history) {
  const current = activeConversation.value
  return {
    message,
    history,
    folder_id: selectedFolderID.value,
    item_ids: selectedItemIDs.value,
    chat_only: chatOnly.value,
    conversation_id: activeConversationID.value,
    harness_session_id: current?.harness_session_id || "",
    context_summary: current?.context_summary || "",
    context_memory: normalizeContextMemory(current?.context_memory),
    compact_context: shouldCompactContext(history),
  }
}

// applyHarnessResult 在当前界面组件中完成交互或数据处理。
function applyHarnessResult(result) {
  const sessionID = String(result?.harness_session_id || "")
  const contextSummary = String(result?.context_summary || "").trim()
  const contextMemory = normalizeContextMemory(result?.context_memory)
  const validSessionID = /^[A-Za-z0-9_-]{1,80}$/.test(sessionID)
  if (!validSessionID && !contextSummary && !contextMemory) return
  const index = conversations.value.findIndex((conversation) => conversation.id === activeConversationID.value)
  if (index >= 0) {
    const current = conversations.value[index]
    conversations.value.splice(index, 1, {
      ...current,
      ...(validSessionID ? { harness_session_id: sessionID } : {}),
      ...(contextSummary ? { context_summary: contextSummary } : {}),
      ...(contextMemory ? { context_memory: contextMemory } : {}),
    })
  }
}

// updateStreamingAnswer 用服务端推送的完整可见答案快照替换同一条消息，不会显示模型推理内容。
function updateStreamingAnswer(id, answer) {
	const index = messages.value.findIndex((message) => message.id === id)
	if (index < 0) {
		messages.value.push({ id, role: "assistant", content: String(answer || ""), streaming: true })
  } else {
    messages.value[index] = { ...messages.value[index], content: String(answer || ""), streaming: true }
  }
	scrollToLatest()
}

// applyTurnResult 将终态任务结果写回当前对话；任务事件中不夹带模型思考，完整终态会从受保护的任务资源读取。
function applyTurnResult(result) {
	applyHarnessResult(result)
  activeTurnStatus.value = String(result?.status || "")
  if (result?.approval?.status === "pending") activeApproval.value = result.approval
  else activeApproval.value = null
}

function attachTurnAudit(assistantID, audit) {
  const index = messages.value.findIndex((message) => message.id === assistantID)
  if (index < 0 || !Array.isArray(audit)) return
  messages.value[index] = { ...messages.value[index], audit }
}

// applyTurnEvent 将可重放事件映射为页面状态；它只反映答案、确认和审计，不呈现工具参数或隐藏推理。
function applyTurnEvent(assistantID, payload) {
  const event = String(payload?.event || "")
  const data = payload?.data || {}
  if (data.status) activeTurnStatus.value = String(data.status)
  if (event === "answer.updated") updateStreamingAnswer(assistantID, data.answer)
  if (event === "approval.required") activeApproval.value = data.approval || null
  if (event === "approval.resolved") activeApproval.value = null
  if (event === "tool.completed" && data.audit) {
    const index = messages.value.findIndex((message) => message.id === assistantID)
    if (index >= 0) attachTurnAudit(assistantID, [...(messages.value[index].audit || []), data.audit])
  }
}

function applyCompletedTurn(assistantID, result) {
  applyTurnResult(result)
  const answer = String(result?.answer || "").trim()
  if (!answer) return
  const index = messages.value.findIndex((message) => message.id === assistantID)
  const finalMessage = {
    id: assistantID, role: "assistant", content: answer, model: result.model,
    sources: result.sources || [], audit: result.audit || [], incomplete: Boolean(result.incomplete),
  }
  if (index >= 0) messages.value[index] = finalMessage
  else messages.value.push(finalMessage)
}

// followAITurn 订阅后台任务并在网络中断后从持久化任务状态取回最终结果。关闭页面不会取消任务。
async function followAITurn(turn, assistantID) {
  activeTurnID.value = String(turn.id || "")
  activeTurnStatus.value = String(turn.status || "queued")
  activeApproval.value = turn.approval?.status === "pending" ? turn.approval : null
  activeTurnStreamController?.abort()
  activeTurnStreamController = new AbortController()
  try {
    await api.streamAITurn(activeTurnID.value, (payload) => applyTurnEvent(assistantID, payload), { signal: activeTurnStreamController.signal })
  } catch (error) {
    if (error?.name !== "AbortError") {
      const latest = await api.getAITurn(activeTurnID.value).catch(() => null)
      if (!latest || ["queued", "running", "waiting_approval"].includes(latest.status)) {
        throw new Error("与 AI 任务的连接已中断；任务仍在后台进行，可刷新页面后继续接收。")
      }
      applyTurnResult(latest)
    }
  } finally {
    if (activeTurnStreamController?.signal.aborted) return
  }
  const result = await api.getAITurn(activeTurnID.value)
  applyTurnResult(result)
  if (result.status === "completed") applyCompletedTurn(assistantID, result)
  return result
}

async function resolveWriteApproval(approved) {
  const approval = activeApproval.value
  if (!approval?.id || !activeTurnID.value || activeApprovalResolving.value) return
  activeApprovalResolving.value = true
  try {
    const turn = await api.resolveAITurnApproval(activeTurnID.value, approval.id, approved)
    applyTurnResult(turn)
    toast.info(approved ? "已确认写入，AI 正在继续完成回复" : "已拒绝写入，AI 会保留资料并继续回复")
  } catch (error) {
    toast.error(error.message || "处理写入确认失败")
  } finally {
    activeApprovalResolving.value = false
  }
}

async function cancelCurrentTurn() {
  if (!activeTurnID.value || !sending.value) return
  try {
    const turn = await api.cancelAITurn(activeTurnID.value)
    applyTurnResult(turn)
    toast.info("已取消本次生成")
  } catch (error) {
    toast.error(error.message || "取消 AI 任务失败")
  }
}

async function startTurn(message, history, assistantID) {
  const turn = await api.startAITurn(chatRequest(message, history))
  const result = await followAITurn(turn, assistantID)
  if (result?.status === "failed") throw new Error(result.error || "AI 暂时无法回答，请稍后重试。")
  return result
}

// send 为用户消息先建立持久化对话，再启动可恢复任务；刷新或网络断开不会导致正在执行的 Agent 被取消。
async function send() {
  const content = composer.value.trim()
  if (!content || sending.value) return
  if (configured.value === false) return toast.warning("请先在设置中配置 DeepSeek API Key")
  if (configured.value !== true) return
  if (harnessReady.value !== true) return toast.warning(harnessReason.value || "Harness 运行环境尚未就绪")
  if (!(await ensureActiveConversation())) return
  const history = historyForRequest()
  messages.value.push({ id: `user-${Date.now()}`, role: "user", content, scope: chatOnly.value || hasScopedContext.value ? scopeSummary.value : "" })
  composer.value = ""
  await saveConversation(true)
  sending.value = true
  scrollToLatest()
  const assistantID = `assistant-${Date.now()}`
  try {
    await startTurn(content, history, assistantID)
  } catch (error) {
    const needsSetup = error?.detail?.code === "deepseek_not_configured" || error?.code === "deepseek_not_configured"
    if (needsSetup) configured.value = false
    messages.value = messages.value.filter((message) => message.id !== assistantID)
    messages.value.push({ id: `error-${Date.now()}`, role: "error", content: error.message || "AI 暂时无法回答，请稍后重试。", setup: needsSetup })
  } finally {
    sending.value = false
    activeTurnID.value = ""
    activeTurnStatus.value = ""
    activeApproval.value = null
    await saveConversation(true)
    scrollToLatest()
  }
}

// continueGeneration 使用同一任务模型继续一条被长度上限截断的回答。
async function continueGeneration(message) {
  if (!message?.incomplete || sending.value || configured.value !== true) return
  const history = historyForRequest()
  sending.value = true
  const assistantID = `assistant-${Date.now()}`
  try {
    await startTurn(continueInstruction, history, assistantID)
    const index = messages.value.findIndex((item) => item.id === message.id)
    if (index >= 0) messages.value[index] = { ...messages.value[index], incomplete: false }
  } catch (error) {
    toast.error(error.message || "继续生成失败，请重试")
  } finally {
    sending.value = false
    activeTurnID.value = ""
    activeTurnStatus.value = ""
    activeApproval.value = null
    await saveConversation(true)
    scrollToLatest()
  }
}

async function resumeLatestActiveTurn() {
  if (!activeConversationID.value || sending.value) return
  const result = await api.listAITurns(activeConversationID.value).catch(() => null)
  const turns = Array.isArray(result?.turns) ? result.turns : []
  const turn = [...turns].reverse().find((item) => ["queued", "running", "waiting_approval"].includes(item.status))
  if (!turn) return
  const assistantID = `assistant-${turn.id}`
  if (turn.answer) updateStreamingAnswer(assistantID, turn.answer)
  sending.value = true
  try {
    await followAITurn(turn, assistantID)
  } catch (error) {
    toast.warning(error.message || "无法恢复 AI 任务")
  } finally {
    sending.value = false
    activeTurnID.value = ""
    activeTurnStatus.value = ""
    activeApproval.value = null
    await saveConversation(true)
  }
}

// keydown 在当前界面组件中完成交互或数据处理。
function keydown(event) {
  if (event.key === "Enter" && !event.shiftKey) {
    event.preventDefault()
    void send()
  }
}

// openSettings 在当前界面组件中完成交互或数据处理。
function openSettings() {
  router.push({ name: "settings", hash: "#ai" })
}

// openSource 在当前界面组件中完成交互或数据处理。
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
  try {
    const status = await api.getAIHarnessStatus()
    harnessReady.value = Boolean(status.enabled)
    harnessReason.value = String(status.reason || "")
  } catch {
    harnessReady.value = false
    harnessReason.value = "无法检查 Harness 运行环境"
  }
  await loadFolderOptions()
  await loadSelectedItems()
  await loadConversation()
})

onBeforeUnmount(() => {
	document.removeEventListener("pointerdown", closeFolderMenu)
	document.removeEventListener("keydown", closeFolderMenuOnEscape)
	activeTurnStreamController?.abort()
})

watch(() => String(route.query.conversation || ""), async (nextID) => {
  if (!conversationReady.value || !nextID || nextID === activeConversationID.value) return
  const conversation = activeConversations.value.find((item) => item.id === nextID)
  if (conversation) await activateConversation(conversation, false)
})

</script>

<template>
  <div class="ai-chat-page">
    <section class="ai-workspace" :class="{ 'is-history-collapsed': historyCollapsed }" aria-label="AI 学习助手">
      <aside class="ai-history-rail" :class="{ 'is-collapsed': historyCollapsed }" :aria-label="historyCollapsed ? '已折叠的对话记录' : '当前对话记录'">
        <Transition name="ai-history-content" mode="out-in">
          <button v-if="historyCollapsed" key="collapsed" type="button" class="ai-history-collapsed__button" title="展开对话记录" aria-label="展开对话记录" @click="toggleHistoryCollapsed"><PanelLeftOpen :size="18" /><span>展开对话记录</span></button>
          <div v-else key="expanded" class="ai-history-rail__content">
            <header class="ai-history-rail__head">
              <div><MessageSquare :size="18" /><strong>对话记录</strong></div>
              <div class="ai-history-rail__tools">
                <button type="button" title="新对话" aria-label="新对话" :disabled="sending || conversationOperationPending || !conversationReady || historyView === 'archived'" @click="resetConversation"><Plus :size="17" /></button>
                <button type="button" title="隐藏对话记录" aria-label="隐藏对话记录" @click="toggleHistoryCollapsed"><PanelLeftClose :size="17" /></button>
              </div>
            </header>
            <button type="button" class="ai-history-archive-toggle" :title="historyView === 'active' ? '查看已归档对话' : '返回活跃对话'" :aria-label="historyView === 'active' ? '查看已归档对话' : '返回活跃对话'" @click="historyView = historyView === 'active' ? 'archived' : 'active'"><Archive :size="14" />{{ historyView === 'active' ? '查看已归档' : '返回活跃对话' }}</button>
            <div class="ai-history-rail__list">
              <p>{{ historyView === 'active' ? `活跃对话 · ${activeConversations.length}/24` : `已归档 · ${archivedConversations.length}/100` }}</p>
              <TransitionGroup name="ai-history-row" tag="div" class="ai-history-rows">
                <div v-for="conversation in conversationRows" :key="conversation.id" class="ai-history-item-row" :class="{ 'is-active': conversation.active, 'is-loading': conversation.loading }">
                  <button type="button" class="ai-history-item" :class="{ 'is-active': conversation.active }" :disabled="sending || conversation.archived || conversationSwitching || conversationOperationPending" @click="selectConversation(conversation.id)"><MessageSquare :size="14" /><span>{{ conversation.title }}</span><LoaderCircle v-if="conversation.loading" :size="13" /></button>
                  <button type="button" class="ai-history-item__menu-button" :aria-label="`${conversation.title} 的操作`" :aria-expanded="historyMenuID === conversation.id" :disabled="sending || conversationSwitching || conversationOperationPending" @click.stop="toggleConversationMenu(conversation.id, $event)"><Ellipsis :size="16" /></button>
                </div>
              </TransitionGroup>
              <div v-if="!conversationRows.length" class="ai-history-rail__empty"><Sparkles :size="16" /><span>{{ historyView === 'active' ? '开始一段新对话吧' : '还没有已归档的对话' }}</span></div>
            </div>
          </div>
        </Transition>
      </aside>

      <Teleport to="body">
        <div v-if="historyMenuConversation" class="ai-history-item__menu ai-history-item__menu--floating" role="menu" :style="historyMenuStyle" @pointerdown.stop>
          <button v-if="!historyMenuConversation.archived_at" type="button" role="menuitem" :disabled="conversationOperationPending" @click.stop="archiveConversation(historyMenuConversation.id)"><Archive :size="14" />归档</button>
          <template v-else>
            <button type="button" role="menuitem" :disabled="conversationOperationPending" @click.stop="restoreConversation(historyMenuConversation.id)"><ArchiveRestore :size="14" />恢复</button>
            <button type="button" class="is-danger" role="menuitem" :disabled="conversationOperationPending" @click.stop="deleteConversation(historyMenuConversation.id)"><Trash2 :size="14" />永久删除</button>
          </template>
        </div>
      </Teleport>

      <section ref="chatPanel" class="ai-chat-panel">
        <div ref="messageList" class="ai-message-list" aria-live="polite" role="log">
          <div v-if="!messages.length && !conversationSwitching" class="ai-chat-empty"><h1>有什么可以帮你的？</h1></div>
          <article v-for="message in messages" :key="message.id" :data-message-id="message.id" class="ai-message" :class="`ai-message--${message.role}`">
            <div class="ai-message__avatar"><component :is="message.role === 'assistant' ? Bot : message.role === 'user' ? FileText : Sparkles" :size="16" /></div>
            <div class="ai-message__body">
              <span class="ai-message__label">{{ message.role === 'assistant' ? 'AI 学习助手' : message.role === 'user' ? '你' : '提示' }}</span>
              <MarkdownRenderer v-if="message.role === 'assistant'" :content="message.content" />
              <p v-else>{{ message.content }}</p>
              <small v-if="message.scope" class="ai-message__scope">{{ message.scope }}</small>
              <div v-if="message.sources?.length" class="ai-message__sources"><span>本次参考</span><button v-for="source in message.sources" :key="`${source.source_type}-${source.id}`" type="button" @click="openSource(source)">{{ source.source_type === 'error' ? '错题' : '资料' }} · {{ source.title }}</button></div>
              <details v-if="message.audit?.length" class="ai-message__audit"><summary>本次操作 · {{ message.audit.length }}</summary><ul><li v-for="(audit, index) in message.audit" :key="`${audit.tool}-${audit.created_at || index}`"><span>{{ audit.tool }}</span><small>{{ audit.outcome === 'succeeded' ? '完成' : audit.outcome === 'rejected' ? '已拒绝' : audit.outcome === 'cancelled' ? '已取消' : '未完成' }}</small></li></ul></details>
              <div v-if="message.incomplete" class="ai-message__continuation"><span>回答达到长度上限，可能尚未完成。</span><button type="button" :disabled="sending || configured !== true" @click="continueGeneration(message)">继续生成</button></div>
              <button v-if="message.setup" type="button" class="ai-message__setup" @click="openSettings">去配置 DeepSeek</button>
              <small v-if="message.model">{{ message.model }}</small>
            </div>
          </article>
          <article v-if="waitingForWriteApproval" class="ai-message ai-message--assistant ai-message--approval"><div class="ai-message__avatar"><FileText :size="16" /></div><div class="ai-message__body"><span class="ai-message__label">需要你的确认</span><p>AI 准备{{ activeApproval.tool === 'create_note' ? '创建' : '更新' }}资料：<strong>{{ activeApproval.path }}</strong></p><div class="ai-write-approval__content"><section v-if="activeApproval.original_content"><small>当前内容</small><pre>{{ activeApproval.original_content }}</pre></section><section><small>拟写入内容</small><pre>{{ activeApproval.content }}</pre></section></div><div class="ai-write-approval__actions"><button type="button" :disabled="activeApprovalResolving" @click="resolveWriteApproval(false)">拒绝</button><button type="button" :disabled="activeApprovalResolving" @click="resolveWriteApproval(true)">{{ activeApprovalResolving ? '处理中…' : '确认写入' }}</button></div></div></article>
          <article v-else-if="sending" class="ai-message ai-message--assistant"><div class="ai-message__avatar"><Bot :size="16" /></div><div class="ai-message__body ai-message__thinking"><LoaderCircle :size="17" />正在准备最终答案…</div></article>
        </div>

        <form class="ai-composer" @submit.prevent="send">
          <div class="ai-composer__entry">
            <div class="ai-scope-popover">
              <button type="button" class="ai-composer__scope-button" aria-label="选择 AI 资料范围" :aria-expanded="scopeMenuOpen" :class="{ 'is-active': hasScopedContext || chatOnly }" @click="scopeMenuOpen = !scopeMenuOpen"><Folder :size="19" /><i v-if="hasScopedContext || chatOnly"></i></button>
              <section v-if="scopeMenuOpen" class="ai-scope-popover__card" aria-label="AI 资料范围">
                <header><strong>资料范围</strong><button v-if="hasScopedContext || chatOnly" type="button" aria-label="清除资料范围" @click="clearScopedContext"><X :size="14" /></button></header>
                <div class="ai-scope-mode" role="radiogroup" aria-label="资料使用方式">
                  <button type="button" role="radio" aria-label="使用纯聊天" :aria-checked="chatOnly" :class="{ 'is-selected': chatOnly }" @click="chooseChatFirstScope"><MessageSquare :size="14" /><span>纯聊天<small>默认不主动查资料</small></span></button>
                  <button type="button" role="radio" aria-label="使用资料库" :aria-checked="!chatOnly" :class="{ 'is-selected': !chatOnly }" @click="chooseLibraryScopeMode"><Folder :size="14" /><span>使用资料库<small>可选择整个资料库或路径</small></span></button>
                </div>
                <template v-if="!chatOnly">
                  <div class="ai-path-combobox"><button type="button" class="ai-path-combobox__trigger" aria-label="资料路径" aria-haspopup="listbox" :aria-expanded="folderMenuOpen" :disabled="folderOptionsLoading" @click="folderMenuOpen = !folderMenuOpen"><Folder :size="14" /><span>{{ selectedFolderLabel }}</span><ChevronDown :size="15" :class="{ 'is-open': folderMenuOpen }" /></button><div v-if="folderMenuOpen" class="ai-path-combobox__menu" role="listbox" aria-label="资料路径"><button type="button" role="option" :aria-selected="folderSelection === null" @click="chooseFolder(null)"><span>整个资料库</span><small>不限定路径</small></button><button v-for="folder in folderOptions" :key="folder.id" type="button" role="option" :aria-selected="readPositiveID(folderSelection) === folder.id" @click="chooseFolder(folder.id)"><span>{{ folder.path }}</span></button></div></div>
                  <footer><small>仅使用可读文本</small><button type="button" class="ai-scope-popover__apply" :disabled="folderOptionsLoading" @click="applyFolderScope">完成</button></footer>
                  <div v-if="selectedItems.length" class="ai-context-picker__items"><span>已转发资料</span><button v-for="item in selectedItems" :key="item.id" type="button" @click="removeSelectedItem(item.id)"><FileText :size="14" />{{ item.name }}<X :size="13" /></button></div>
                </template>
              </section>
            </div>
            <textarea v-model="composer" rows="1" maxlength="2000" placeholder="输入你的问题…" :disabled="sending || conversationSwitching || configured !== true || harnessReady !== true" @keydown="keydown" />
            <button v-if="sending" type="button" class="ai-composer__send ai-composer__cancel" aria-label="取消本次生成" title="取消本次生成" @click="cancelCurrentTurn"><X :size="18" /></button>
            <button v-else type="submit" class="ai-composer__send" :disabled="!canSend" aria-label="发送消息"><SendHorizontal :size="18" /></button>
          </div>
          <small class="ai-composer__context-status">{{ chatOnly ? '纯聊天 · 默认不主动查资料；你明确要求时仍可使用资料库' : 'Harness Agent · 可按资料范围检索、读取，并以版本保护创建或更新笔记' }}</small>
          <div v-if="configured === false" class="ai-composer__setup">尚未连接 DeepSeek <button type="button" @click="openSettings">去配置</button></div>
          <div v-else-if="harnessReady === false" class="ai-composer__setup">Harness 未就绪：{{ harnessReason || '请检查本地运行环境' }}</div>
        </form>
      </section>
    </section>
  </div>
</template>
