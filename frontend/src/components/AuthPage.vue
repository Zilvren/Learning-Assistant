<script setup>
import { computed, ref } from "vue"
import { useAuth } from "../store/auth.js"

const emit = defineEmits(["signed-in"])

const auth = useAuth()
const mode = ref("login")
const account = ref("")
const username = ref("")
const email = ref("")
const password = ref("")
const confirmPassword = ref("")
const busy = ref(false)
const error = ref("")

const isRegister = computed(() => mode.value === "register")
const title = computed(() => isRegister.value ? "创建账号" : "欢迎回来")
const subtitle = computed(() => isRegister.value ? "注册后会自动进入你的错题空间" : "登录后继续整理和复习错题")

function switchMode(next) {
  mode.value = next
  error.value = ""
}

async function submit() {
  error.value = ""
  if (busy.value) return
  if (isRegister.value && password.value !== confirmPassword.value) {
    error.value = "两次输入的密码不一致"
    return
  }
  busy.value = true
  try {
    if (isRegister.value) {
      await auth.register(username.value, email.value, password.value)
    } else {
      await auth.login(account.value, password.value)
    }
    emit("signed-in")
  } catch (e) {
    error.value = e.message || "登录失败"
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="auth-shell">
    <section class="auth-panel">
      <div class="auth-copy">
        <div class="auth-mark">错题追踪器</div>
        <h1>{{ title }}</h1>
        <p>{{ subtitle }}</p>
      </div>

      <form class="auth-form" @submit.prevent="submit">
        <div class="auth-tabs">
          <button type="button" :class="{ active: !isRegister }" @click="switchMode('login')">登录</button>
          <button type="button" :class="{ active: isRegister }" @click="switchMode('register')">注册</button>
        </div>

        <label v-if="isRegister" class="form-label">用户名</label>
        <input v-if="isRegister" v-model.trim="username" class="form-input" autocomplete="username" placeholder="3-32 个字符" />

        <label v-if="isRegister" class="form-label">邮箱（可选）</label>
        <input v-if="isRegister" v-model.trim="email" class="form-input" autocomplete="email" placeholder="you@example.com" />

        <label class="form-label">{{ isRegister ? '密码' : '用户名或邮箱' }}</label>
        <input v-if="!isRegister" v-model.trim="account" class="form-input" autocomplete="username" placeholder="输入用户名或邮箱" />
        <input v-else v-model="password" class="form-input" type="password" autocomplete="new-password" placeholder="至少 8 位" />

        <label v-if="!isRegister" class="form-label">密码</label>
        <input v-if="!isRegister" v-model="password" class="form-input" type="password" autocomplete="current-password" placeholder="输入密码" />

        <label v-if="isRegister" class="form-label">确认密码</label>
        <input v-if="isRegister" v-model="confirmPassword" class="form-input" type="password" autocomplete="new-password" placeholder="再次输入密码" />

        <div v-if="error" class="auth-error">{{ error }}</div>

        <button class="btn btn-primary auth-submit" :disabled="busy">
          {{ busy ? '处理中...' : (isRegister ? '注册并进入' : '登录') }}
        </button>
      </form>
    </section>
  </main>
</template>
