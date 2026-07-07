<script setup>
import { ref } from "vue"
import Sidebar from "./components/Sidebar.vue"
import HomePage from "./components/HomePage.vue"
import ErrorList from "./components/ErrorList.vue"
import AuthPage from "./components/AuthPage.vue"
import { useAuth } from "./store/auth.js"

const activePage = ref("home")
const snackbar = ref(null)
const auth = useAuth()
auth.init()

function showSnack(text, color = "#10b981") {
  snackbar.value = { text, color }
  setTimeout(() => (snackbar.value = null), 3000)
}

const pages = { home: HomePage, list: ErrorList }
</script>

<template>
  <div v-if="!auth.ready.value" class="app-loading">正在启动...</div>
  <AuthPage v-else-if="auth.enabled.value && !auth.user.value" @signed-in="showSnack('登录成功')" />
  <div v-else class="app-layout">
    <Sidebar :active="activePage" @navigate="activePage = $event" @signed-out="showSnack('已退出登录', '#64748b')" />
    <main class="main-content">
      <Transition name="page" mode="out-in">
        <component :is="pages[activePage]" :key="activePage" @snack="showSnack" />
      </Transition>
    </main>
    <div v-if="snackbar" class="snackbar" :style="{ background: snackbar.color }">
      {{ snackbar.text }}
    </div>
  </div>
</template>
