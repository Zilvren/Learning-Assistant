<script setup>
import { computed, ref } from "vue"
import { useRouter, useRoute } from "vue-router"
import { ArrowRight, BadgeCheck, BookOpenCheck, CircleAlert, MailCheck } from "lucide-vue-next"
import { useAuth } from "../store/auth.js"
import BaseButton from "./ui/BaseButton.vue"

const router = useRouter()
const route = useRoute()
const auth = useAuth()
const state = ref("ready")
const error = ref("")
const busy = ref(false)
const isSuccess = computed(() => state.value === "success")

const token = typeof route.query.token === "string" ? route.query.token : ""

async function verify() {
  if (!token || busy.value) return
  busy.value = true
  state.value = "checking"
  error.value = ""
  try {
    await auth.verifyEmail(token)
    state.value = "success"
  } catch (requestError) {
    state.value = "error"
    error.value = requestError.message || "验证链接无效或已过期。"
  } finally {
    busy.value = false
  }
}

if (!token) {
  state.value = "error"
  error.value = "验证链接不完整，请重新打开邮件中的链接。"
}

function continueToApp() {
  router.replace({ name: "home" })
}

function backToLogin() {
  router.replace({ name: "login" })
}
</script>

<template>
  <main class="auth-page verification-page">
    <section class="auth-manifesto">
      <div class="auth-brand"><span><BookOpenCheck :size="22" /></span><strong>学习空间</strong><small>账号验证</small></div>
      <div class="auth-manifesto__copy">
        <span class="page-eyebrow">安全 · 私密 · 专注</span>
        <h1>每一份资料，<br />都只属于<span>你。</span></h1>
        <p>确认邮箱后，你的学习资料库才会正式开启。我们只用它来保护账号和发送必要的安全通知。</p>
      </div>
      <ul>
        <li><BadgeCheck :size="16" />一次验证，长期有效</li>
        <li><BadgeCheck :size="16" />验证链接 24 小时内有效</li>
      </ul>
      <div class="auth-folio-number">VERIFY<br /><strong>02</strong></div>
    </section>

    <section class="auth-form-panel">
      <div class="auth-form-wrap verification-card" :class="{ 'verification-card--success': isSuccess }" aria-live="polite">
        <div v-if="state === 'ready'" class="verification-card__status">
          <span class="verification-card__orb"><MailCheck :size="29" /></span>
          <span class="page-eyebrow">最后一步</span>
          <h2>确认验证邮箱</h2>
          <p>确认后会激活账号，并安全登录到你的学习空间。</p>
          <BaseButton variant="primary" size="lg" :busy="busy" class="auth-submit" @click="verify">验证并进入<template #icon><ArrowRight :size="17" /></template></BaseButton>
        </div>
        <div v-else-if="state === 'checking'" class="verification-card__status">
          <span class="verification-card__orb verification-card__orb--loading"><MailCheck :size="29" /></span>
          <span class="page-eyebrow">正在确认</span>
          <h2>验证你的邮箱</h2>
          <p>请稍候，我们正在安全地确认这封验证链接。</p>
        </div>
        <div v-else-if="isSuccess" class="verification-card__status">
          <span class="verification-card__orb"><BadgeCheck :size="31" /></span>
          <span class="page-eyebrow">验证完成</span>
          <h2>学习空间已准备好</h2>
          <p>你的邮箱已验证，并已为你安全登录。现在可以开始整理第一份学习资料。</p>
          <BaseButton variant="primary" size="lg" class="auth-submit" @click="continueToApp">进入学习空间<template #icon><ArrowRight :size="17" /></template></BaseButton>
        </div>
        <div v-else class="verification-card__status">
          <span class="verification-card__orb verification-card__orb--error"><CircleAlert :size="30" /></span>
          <span class="page-eyebrow">无法完成验证</span>
          <h2>这个链接不可用</h2>
          <p>{{ error }}</p>
          <BaseButton variant="default" size="lg" class="auth-submit" @click="backToLogin">返回登录</BaseButton>
        </div>
      </div>
    </section>
  </main>
</template>
