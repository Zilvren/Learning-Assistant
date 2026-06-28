<script setup>
import { ref } from "vue"
import Sidebar from "./components/Sidebar.vue"
import HomePage from "./components/HomePage.vue"
import ErrorList from "./components/ErrorList.vue"

const activePage = ref("home")
const snackbar = ref(null)

function showSnack(text, color = "#10b981") {
  snackbar.value = { text, color }
  setTimeout(() => (snackbar.value = null), 3000)
}

const pages = { home: HomePage, list: ErrorList }
</script>

<template>
  <div class="app-layout">
    <Sidebar :active="activePage" @navigate="activePage = $event" />
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
