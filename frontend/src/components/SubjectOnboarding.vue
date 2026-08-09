<script setup>
import { ref } from "vue"
import { BookOpenCheck, Plus } from "lucide-vue-next"
import { useSubjects } from "../store/subjects.js"
import { useToast } from "../store/toast.js"
import BaseButton from "./ui/BaseButton.vue"
import BaseDialog from "./ui/BaseDialog.vue"

defineProps({ open: Boolean })
const emit = defineEmits(["created"])
const name = ref("")
const busy = ref(false)
const subjects = useSubjects()
const toast = useToast()

// createSubject 协调当前组件的状态和交互。
async function createSubject() {
  const value = name.value.trim()
  if (!value || busy.value) return
  busy.value = true
  try {
    await subjects.add(value)
    name.value = ""
    toast.success(`科目“${value}”已创建`)
    emit("created")
  } catch (error) {
    toast.error(error.message || "科目创建失败")
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <BaseDialog :open="open" :show-close="false" :close-on-backdrop="false" size="sm">
    <div class="onboarding-mark"><BookOpenCheck :size="28" /></div>
    <span class="page-eyebrow">创建第一个科目</span>
    <h2 class="onboarding-title">先为错题安一个归处</h2>
    <p class="onboarding-copy">创建科目后即可录入、检索并按艾宾浩斯计划复习。名称之后仍可在设置中心维护。</p>
    <form class="onboarding-form" @submit.prevent="createSubject">
      <label class="field-label" for="first-subject">科目名称</label>
      <input id="first-subject" v-model="name" class="field-control" placeholder="例如：高等数学" autocomplete="off" autofocus />
      <BaseButton type="submit" variant="primary" :busy="busy" :disabled="!name.trim()">
        <template #icon><Plus :size="17" /></template>
        创建并开始整理
      </BaseButton>
    </form>
  </BaseDialog>
</template>
