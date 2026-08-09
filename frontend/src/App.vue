<script setup>
import { computed } from "vue"
import { RouterView, useRoute, useRouter } from "vue-router"
import AppToast from "./components/AppToast.vue"
import { useAuth } from "./store/auth.js"

const auth = useAuth()
const route = useRoute()
const router = useRouter()
const skipBootstrap = computed(() => Boolean(route.meta.skipAuth))

async function retryBootstrap() {
  const ok = await auth.init()
  if (ok && auth.enabled.value && !auth.user.value) {
    await router.replace({ name: "login", query: { redirect: router.currentRoute.value.fullPath } })
  }
}
</script>

<template>
  <main v-if="!skipBootstrap && !auth.ready.value" class="app-boot" aria-live="polite">
    <span class="app-boot__mark" aria-hidden="true">L</span>
    <h1>正在打开学习空间</h1>
    <p>正在确认本地服务与登录状态…</p>
  </main>
  <main v-else-if="!skipBootstrap && auth.initError.value" class="app-boot app-boot--error" role="alert">
    <span class="app-boot__mark" aria-hidden="true">!</span>
    <h1>暂时无法连接</h1>
    <p>{{ auth.initError.value }}</p>
    <button type="button" @click="retryBootstrap">重新连接</button>
  </main>
  <RouterView v-else />
  <AppToast />
</template>
