<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router"
import { FileDown, Plus } from "lucide-vue-next"
import { api } from "../api/index.js"
import { isDue, useErrorLibrary } from "../composables/useErrorLibrary.js"
import { useSubjects } from "../store/subjects.js"
import { useToast } from "../store/toast.js"
import { exportPdfReport } from "../utils/pdfExport.js"
import ErrorDetailPane from "./errors/ErrorDetailPane.vue"
import ErrorEditorDialog from "./errors/ErrorEditorDialog.vue"
import ErrorFilters from "./errors/ErrorFilters.vue"
import ErrorListPane from "./errors/ErrorListPane.vue"
import ExportDialog from "./errors/ExportDialog.vue"
import BaseButton from "./ui/BaseButton.vue"
import ConfirmDialog from "./ui/ConfirmDialog.vue"
import PageHeader from "./ui/PageHeader.vue"

const route = useRoute()
const router = useRouter()
const library = useErrorLibrary()
const subjects = useSubjects()
const toast = useToast()
const currentSubject = ref("全部")
const keyword = ref("")
const searchMode = ref("全部")
const selectedCache = ref(null)
const editorOpen = ref(false)
const editorMode = ref("add")
const editorDirty = ref(false)
const editorBusy = ref(false)
const deleteOpen = ref(false)
const deleteBusy = ref(false)
const discardOpen = ref(false)
const pendingRoute = ref("")
const exportOpen = ref(false)
const exportStyle = ref(localStorage.getItem("pdfExportStyle") || "detailed")
const exportBusy = ref(false)
const reviewBusy = ref(false)
const relations = ref([])
const relationLibraryId = ref("")
const today = new Date().toISOString().slice(0, 10)

const hasRequestedId = computed(() => route.params.id !== undefined && route.params.id !== "")
const requestedId = computed(() => {
  const value = Array.isArray(route.params.id) ? route.params.id[0] : route.params.id
  return value === undefined ? "" : String(value)
})
const selectedId = computed(() => {
  const id = Number(route.params.id)
  return Number.isFinite(id) && id > 0 ? id : null
})
const selectedError = computed(() => library.errors.value.find((item) => item.id === selectedId.value) || (selectedCache.value?.id === selectedId.value ? selectedCache.value : null))
const detailNotFound = computed(() => hasRequestedId.value && !library.loading.value && !selectedError.value)
const pendingReviews = computed(() => library.errors.value.filter((item) => isDue(item, today)).length)
const pageDescription = computed(() => {
  if (hasRequestedId.value) return selectedError.value
    ? `${selectedError.value.subject || "未分类"} · ${reviewLabelForHeader(selectedError.value)}`
    : "查看完整题面、订正与复习记录。"
  return `共 ${library.errors.value.length} 道错题，其中 ${pendingReviews.value} 道已到复习时间。`
})

// reviewLabelForHeader 协调当前组件的状态和交互。
function reviewLabelForHeader(item) {
  return item.next_review ? `下次复习 ${item.next_review}` : `已复习 ${item.review_count || 0} 轮`
}

// syncSelected 协调当前组件的状态和交互。
function syncSelected() {
  const found = library.errors.value.find((item) => item.id === selectedId.value)
  if (found) selectedCache.value = found
  if (!selectedId.value) selectedCache.value = null
}

// refresh 协调当前组件的状态和交互。
async function refresh() {
  try {
    await library.refresh({ subject: currentSubject.value, keyword: keyword.value, mode: searchMode.value })
    syncSelected()
		await loadRelations()
  } catch (error) {
    toast.error(error.message || "错题加载失败")
  }
}

async function loadRelations() {
  relations.value = []
  if (!selectedId.value) return
  try {
    const result = await api.getLearningRelations("error", selectedId.value)
    if (selectedId.value) relations.value = result.items || []
  } catch { /* missing relations must never block an error record */ }
}

async function linkLibrary() {
  const libraryID = Number(relationLibraryId.value)
  if (!Number.isInteger(libraryID) || libraryID <= 0) return toast.warning("请输入要关联的笔记 ID")
  try {
    await api.createLearningRelation({ from_type: "error", from_id: selectedId.value, to_type: "library", to_id: libraryID })
    relationLibraryId.value = ""
    await loadRelations()
    toast.success("已关联笔记")
  } catch (error) { toast.error(error.message || "关联失败") }
}

async function removeRelation(id) {
  try {
    await api.deleteLearningRelation(id)
    relations.value = relations.value.filter((relation) => relation.id !== id)
    toast.success("已移除关联")
  } catch (error) { toast.error(error.message || "移除失败") }
}

// selectError 协调当前组件的状态和交互。
async function selectError(item) {
  selectedCache.value = item
  await router.push({ name: "errors", params: { id: item.id } })
}

// setTagSearch 协调当前组件的状态和交互。
async function setTagSearch(mode, tag) {
  searchMode.value = mode
  keyword.value = tag
  if (hasRequestedId.value) await router.push({ name: "errors" })
  await refresh()
}

// openAdd 协调当前组件的状态和交互。
function openAdd() {
  if (!subjects.subjectRef.value.length) {
    toast.warning("请先在设置中心创建科目")
    return router.push({ name: "settings", query: { section: "subjects" } })
  }
  editorMode.value = "add"
  editorOpen.value = true
}

// openEdit 协调当前组件的状态和交互。
function openEdit() {
  if (!selectedError.value) return toast.warning("请先选择一道错题")
  editorMode.value = "edit"
  editorOpen.value = true
}

// finishCloseEditor 协调当前组件的状态和交互。
function finishCloseEditor() {
  editorDirty.value = false
  editorOpen.value = false
}

// requestCloseEditor 协调当前组件的状态和交互。
function requestCloseEditor() {
  if (editorDirty.value) discardOpen.value = true
  else finishCloseEditor()
}

// saveEditor 协调当前组件的状态和交互。
async function saveEditor(payload) {
  if (editorBusy.value) return
  editorBusy.value = true
  try {
    if (editorMode.value === "edit") {
      await library.update(selectedId.value, payload)
      toast.success(`错题 #${selectedId.value} 已修订`)
      finishCloseEditor()
      await refresh()
    } else {
      const created = await library.create(payload)
      toast.success("新错题已收入错题本")
      finishCloseEditor()
      await refresh()
      const item = library.errors.value.find((entry) => entry.id === created.id)
      if (item) await selectError(item)
    }
  } catch (error) {
    toast.error(error.message || "保存失败")
  } finally {
    editorBusy.value = false
  }
}

// markReviewed 协调当前组件的状态和交互。
async function markReviewed() {
  if (!selectedId.value || reviewBusy.value) return
  reviewBusy.value = true
  try {
    const result = await library.review(selectedId.value)
    toast.success(result.next_review ? `已完成本轮复习，下次 ${result.next_review}` : `错题 #${selectedId.value} 已复习`)
    await refresh()
  } catch (error) {
    toast.error(error.message || "复习状态更新失败")
  } finally {
    reviewBusy.value = false
  }
}

// deleteError 协调当前组件的状态和交互。
async function deleteError() {
  if (!selectedId.value || deleteBusy.value) return
  deleteBusy.value = true
  const id = selectedId.value
  try {
    await library.remove(id)
    toast.success(`错题 #${id} 已删除`)
    deleteOpen.value = false
    selectedCache.value = null
    await router.replace({ name: "errors" })
    await refresh()
  } catch (error) {
    toast.error(error.message || "删除失败")
  } finally {
    deleteBusy.value = false
  }
}

// exportPdf 协调当前组件的状态和交互。
async function exportPdf() {
  if (exportBusy.value) return
  exportBusy.value = true
  try {
    localStorage.setItem("pdfExportStyle", exportStyle.value)
    const all = await api.getErrors()
    await exportPdfReport(all.errors, { style: exportStyle.value })
    exportOpen.value = false
    toast.success("PDF 错题册已生成")
  } catch (error) {
    toast.error(error.message || "导出失败")
  } finally {
    exportBusy.value = false
  }
}

// confirmDiscard 协调当前组件的状态和交互。
function confirmDiscard() {
  discardOpen.value = false
  finishCloseEditor()
  if (pendingRoute.value) {
    const target = pendingRoute.value
    pendingRoute.value = ""
    router.push(target)
  }
}

// onBeforeUnload 协调当前组件的状态和交互。
function onBeforeUnload(event) {
  if (!editorOpen.value || !editorDirty.value) return
  event.preventDefault()
  event.returnValue = ""
}

watch(() => route.params.id, () => { syncSelected(); void loadRelations() })
onBeforeRouteLeave((to) => {
  if (!editorOpen.value || !editorDirty.value) return true
  pendingRoute.value = to.fullPath
  discardOpen.value = true
  return false
})
onMounted(async () => {
  await subjects.load()
  await refresh()
  window.addEventListener("beforeunload", onBeforeUnload)
})
onBeforeUnmount(() => window.removeEventListener("beforeunload", onBeforeUnload))
</script>

<template>
  <div class="error-library page-stage" :class="{ 'error-library--detail': hasRequestedId }">
    <PageHeader eyebrow="学习资料库" :title="hasRequestedId ? '错题详情' : '错题库'" :description="pageDescription">
      <template #actions>
        <BaseButton @click="exportOpen = true"><template #icon><FileDown :size="16" /></template>编排导出</BaseButton>
        <BaseButton variant="primary" @click="openAdd"><template #icon><Plus :size="17" /></template>录入错题</BaseButton>
      </template>
    </PageHeader>

    <ErrorFilters
      v-if="!hasRequestedId"
      :subjects="subjects.subjectRef.value" :subject="currentSubject" :keyword="keyword" :mode="searchMode" :loading="library.loading.value"
      @update:subject="currentSubject = $event" @update:keyword="keyword = $event" @update:mode="searchMode = $event" @search="refresh"
    />

    <section v-if="!hasRequestedId" class="error-workspace error-workspace--cards">
      <ErrorListPane :items="library.errors.value" :selected-id="selectedId" :loading="library.loading.value" :today="today" @select="selectError" @tag="setTagSearch" />
    </section>

    <section v-else class="error-detail-stage">
      <ErrorDetailPane
        :item="selectedError"
        :today="today"
        :reviewing="reviewBusy"
        :not-found="detailNotFound"
        :requested-id="requestedId"
		:relations="relations"
		:relation-library-id="relationLibraryId"
        @back="router.push({ name: 'errors' })"
        @edit="openEdit"
        @delete="deleteOpen = true"
        @review="markReviewed"
        @tag="setTagSearch"
		@link-library="linkLibrary"
		@unlink-relation="removeRelation"
		@open-library="(id) => router.push({ name: 'library-item', params: { itemId: id } })"
		@update:relation-library-id="relationLibraryId = $event"
      />
    </section>

    <ErrorEditorDialog
      :open="editorOpen" :mode="editorMode" :record="editorMode === 'edit' ? selectedError : null"
      :subjects="subjects.subjectRef.value" :busy="editorBusy" @request-close="requestCloseEditor" @dirty-change="editorDirty = $event" @save="saveEditor"
    />
    <ExportDialog v-model="exportStyle" :open="exportOpen" :busy="exportBusy" @close="exportOpen = false" @export="exportPdf" />
    <ConfirmDialog :open="deleteOpen" title="删除这则错题？" message="删除后将无法恢复，相关复习进度也会一并移除。" confirm-text="确认删除" danger :busy="deleteBusy" @close="deleteOpen = false" @confirm="deleteError" />
    <ConfirmDialog :open="discardOpen" title="舍弃未保存的修改？" message="编辑器中仍有未保存内容，离开后这些修改会丢失。" confirm-text="舍弃修改" danger @close="discardOpen = false; pendingRoute = ''" @confirm="confirmDiscard" />
  </div>
</template>
