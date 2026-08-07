<script setup>
import { computed, ref } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ArrowRight, BookOpenCheck, CheckCircle2, LockKeyhole, Mail, UserRound } from "lucide-vue-next"
import { useAuth } from "../store/auth.js"
import { useToast } from "../store/toast.js"
import BaseButton from "./ui/BaseButton.vue"

const route = useRoute()
const router = useRouter()
const auth = useAuth()
const toast = useToast()
const mode = ref("login")
const account = ref("")
const username = ref("")
const email = ref("")
const password = ref("")
const confirmPassword = ref("")
const busy = ref(false)
const error = ref("")
const isRegister = computed(() => mode.value === "register")
const registrationEnabled = computed(() => auth.registrationEnabled.value)

function switchMode(next) {
  mode.value = next
  error.value = ""
  password.value = ""
  confirmPassword.value = ""
}

async function submit() {
  if (busy.value) return
  error.value = ""
  if (isRegister.value && password.value !== confirmPassword.value) return error.value = "两次输入的密码不一致"
  busy.value = true
  try {
    if (isRegister.value) await auth.register(username.value.trim(), email.value.trim(), password.value)
    else await auth.login(account.value.trim(), password.value)
    toast.success(isRegister.value ? "账号已创建，欢迎开始整理" : "欢迎回来")
    const redirect = typeof route.query.redirect === "string" && route.query.redirect.startsWith("/") ? route.query.redirect : "/"
    await router.replace(redirect)
  } catch (requestError) {
    error.value = requestError.message || "登录失败"
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="auth-page">
    <section class="auth-manifesto">
      <div class="auth-brand"><span><BookOpenCheck :size="22" /></span><strong>学习空间</strong><small>个人资料库</small></div>
      <div class="auth-manifesto__copy">
        <span class="page-eyebrow">整理 · 复习 · 掌握</span>
        <h1>把零散资料变成<br />真正属于你的<span>知识。</span></h1>
        <p>在一个清爽的学习空间里整理题目、订正答案、归纳错因，并按计划重新复习。</p>
      </div>
      <ul>
        <li><CheckCircle2 :size="16" />结构化记录题目与错因</li>
        <li><CheckCircle2 :size="16" />按复习曲线生成今日任务</li>
        <li><CheckCircle2 :size="16" />Markdown、公式与 OCR 一处完成</li>
      </ul>
      <div class="auth-folio-number">LEARN<br /><strong>01</strong></div>
    </section>

    <section class="auth-form-panel">
      <div class="auth-form-wrap">
        <span class="page-eyebrow">{{ isRegister ? '创建学习空间' : '欢迎回来' }}</span>
        <h2>{{ isRegister ? "创建个人学习空间" : "继续今天的学习" }}</h2>
        <p>{{ isRegister ? "注册后即可整理笔记、错题与学习文件。" : registrationEnabled ? "使用用户名或邮箱登录。" : "此学习空间仅限已有账户登录。" }}</p>

        <div v-if="registrationEnabled" class="auth-tabs" role="tablist">
          <button type="button" role="tab" :aria-selected="!isRegister" :class="{ active: !isRegister }" @click="switchMode('login')">登录</button>
          <button type="button" role="tab" :aria-selected="isRegister" :class="{ active: isRegister }" @click="switchMode('register')">注册</button>
        </div>

        <form @submit.prevent="submit">
          <label v-if="isRegister"><span class="field-label">用户名</span><span class="auth-input"><UserRound :size="17" /><input v-model="username" placeholder="3–32 个字符" autocomplete="username" required /></span></label>
          <label v-if="isRegister"><span class="field-label">邮箱 <small>可选</small></span><span class="auth-input"><Mail :size="17" /><input v-model="email" type="email" placeholder="you@example.com" autocomplete="email" /></span></label>
          <label v-if="!isRegister"><span class="field-label">用户名或邮箱</span><span class="auth-input"><UserRound :size="17" /><input v-model="account" placeholder="输入用户名或邮箱" autocomplete="username" required autofocus /></span></label>
          <label><span class="field-label">密码</span><span class="auth-input"><LockKeyhole :size="17" /><input v-model="password" type="password" :placeholder="isRegister ? '至少 8 位' : '输入密码'" :autocomplete="isRegister ? 'new-password' : 'current-password'" minlength="8" required /></span></label>
          <label v-if="isRegister"><span class="field-label">确认密码</span><span class="auth-input"><LockKeyhole :size="17" /><input v-model="confirmPassword" type="password" placeholder="再次输入密码" autocomplete="new-password" minlength="8" required /></span></label>
          <div v-if="error" class="form-alert" role="alert">{{ error }}</div>
          <BaseButton type="submit" variant="primary" size="lg" :busy="busy" class="auth-submit">{{ isRegister ? "注册并开始学习" : "进入学习空间" }}<template #icon><ArrowRight :size="17" /></template></BaseButton>
        </form>
      </div>
    </section>
  </main>
</template>
