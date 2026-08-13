<script setup>
import { computed, nextTick, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { Bot, BookOpenText, CornerDownLeft, FileText, Lightbulb, LoaderCircle, RefreshCw, Settings2, Sparkles } from "lucide-vue-next"
import { api } from "../api/index.js"
import { useToast } from "../store/toast.js"
import MarkdownRenderer from "./MarkdownRenderer.vue"
import BaseButton from "./ui/BaseButton.vue"
import PageHeader from "./ui/PageHeader.vue"

const router = useRouter()
const toast = useToast()
const composer = ref("")
const messages = ref([])
const sending = ref(false)
const configured = ref(null)
const messageList = ref(null)

const quickPrompts = [
  "请分析我的资料库，给出本周最值得优先复习的内容。",
  "根据我的错题和笔记，列出 3 个薄弱知识点与行动建议。",
  "帮我根据当前资料和每日目标，制定一个 30 分钟学习计划。",
]
const canSend = computed(() => Boolean(composer.value.trim()) && !sending.value && configured.value === true)

function scrollToLatest() {
  nextTick(() => messageList.value?.scrollTo({ top: messageList.value.scrollHeight, behavior: "smooth" }))
}

function resetConversation() {
  messages.value = []
  composer.value = ""
}

function setQuickPrompt(value) {
  composer.value = value
}

function historyForRequest() {
  return messages.value
    .filter((message) => message.role === "user" || message.role === "assistant")
    .slice(-8)
    .map((message) => ({ role: message.role, content: message.content }))
}

async function send() {
  const content = composer.value.trim()
  if (!content || sending.value) return
  if (configured.value === false) return toast.warning("请先在设置中配置 DeepSeek API Key")
  if (configured.value !== true) return
  const history = historyForRequest()
  messages.value.push({ id: `user-${Date.now()}`, role: "user", content })
  composer.value = ""
  sending.value = true
  scrollToLatest()
  try {
    const result = await api.aiChat({ message: content, history })
    messages.value.push({ id: `assistant-${Date.now()}`, role: "assistant", content: result.answer, model: result.model, sources: result.sources || [] })
  } catch (error) {
    const needsSetup = error?.detail?.code === "deepseek_not_configured" || error?.code === "deepseek_not_configured"
    if (needsSetup) configured.value = false
    messages.value.push({ id: `error-${Date.now()}`, role: "error", content: error.message || "AI 暂时无法回答，请稍后重试。", setup: needsSetup })
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
  try {
    configured.value = Boolean((await api.getDeepSeekToken()).configured)
  } catch {
    configured.value = false
  }
})
</script>

<template>
  <div class="ai-chat-page page-stage">
    <PageHeader eyebrow="资料库智能分析" title="AI 学习助手" description="让 DeepSeek 基于可读笔记、Office 文本、错题与学习进度给出建议。">
      <template #actions><BaseButton @click="resetConversation"><template #icon><RefreshCw :size="16" /></template>新对话</BaseButton></template>
    </PageHeader>

    <section v-if="configured === false" class="ai-setup-callout">
      <div><Bot :size="22" /><div><strong>先连接 DeepSeek</strong><p>API Key 只保存在后端并加密存储；不会显示在聊天页面。</p></div></div>
      <BaseButton variant="primary" @click="openSettings"><template #icon><Settings2 :size="16" /></template>去配置</BaseButton>
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
              <div v-if="message.sources?.length" class="ai-message__sources"><span>本次参考</span><button v-for="source in message.sources" :key="`${source.source_type}-${source.id}`" type="button" @click="openSource(source)">{{ source.source_type === 'error' ? '错题' : '资料' }} · {{ source.title }}</button></div>
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
