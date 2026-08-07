<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useRoute, useRouter } from "vue-router"
import { ArchiveRestore, ChevronDown, ChevronRight, File, FileText, Folder, Grid2X2, List, MoreVertical, Plus, Search, Tags, Trash2, Upload } from "lucide-vue-next"
import { api } from "../../api/index.js"
import { useToast } from "../../store/toast.js"
import { rememberLibraryPath } from "../../utils/libraryPath.js"
import MarkdownRenderer from "../MarkdownRenderer.vue"
import BaseDialog from "../ui/BaseDialog.vue"

const props = defineProps({ folderId: [String, Number], trash: Boolean })
const route = useRoute(); const router = useRouter(); const toast = useToast()
const items = ref([]); const loading = ref(false); const query = ref(""); const kind = ref("all"); const tag = ref(String(route.query.tag || "")); const tags = ref([]); const view = ref(localStorage.getItem("libraryView") || "grid")
const currentFolder = ref(null); const breadcrumbs = ref([]); const selected = ref(new Set()); const menuId = ref(null); const fileInput = ref(null)
const dialog = ref(""); const formName = ref(""); const formTags = ref(""); const formReview = ref(false); const editing = ref(null); const busy = ref(false); const deleting = ref(false)
const notePreviews = ref({}); const previewLoading = ref(new Set()); const previewErrors = ref(new Set())
const activeFilter = ref("")
const folderId = computed(() => props.trash ? null : Number(props.folderId || route.params.folderId || 0) || null)
const title = computed(() => props.trash ? "回收站" : currentFolder.value?.name || "我的资料库")
const kindOptions = [{ value:"all", label:"全部类型" }, { value:"folder", label:"文件夹" }, { value:"note", label:"笔记" }, { value:"file", label:"文件" }]
const tagOptions = computed(() => [{ value:"", label:"全部标签" }, ...tags.value.map((value) => ({ value, label:`# ${value}` }))])
const kindLabel = computed(() => kindOptions.find((option) => option.value === kind.value)?.label || "全部类型")
const tagLabel = computed(() => tagOptions.value.find((option) => option.value === tag.value)?.label || "全部标签")

function itemIcon(item) { return item.kind === "folder" ? Folder : item.kind === "note" ? FileText : File }
function formatSize(size) { if (!size) return "—"; if (size < 1024) return `${size} B`; if (size < 1048576) return `${(size/1024).toFixed(1)} KB`; return `${(size/1048576).toFixed(1)} MB` }
function formatDate(value) { return value ? new Date(value).toLocaleString("zh-CN", { month:"2-digit", day:"2-digit", hour:"2-digit", minute:"2-digit" }) : "" }

async function loadNotePreview(item) {
  if (item.kind !== "note" || view.value !== "grid" || Object.hasOwn(notePreviews.value, item.id) || previewLoading.value.has(item.id)) return
  const loadingSet = new Set(previewLoading.value); loadingSet.add(item.id); previewLoading.value = loadingSet
  try {
    const result = await api.getLibraryContent(item.id)
    notePreviews.value = { ...notePreviews.value, [item.id]: result.content || "" }
  } catch {
    const errors = new Set(previewErrors.value); errors.add(item.id); previewErrors.value = errors
  } finally {
    const loading = new Set(previewLoading.value); loading.delete(item.id); previewLoading.value = loading
  }
}

async function load() {
  loading.value = true
  try {
    const result = await api.getLibraryItems({ parentId: folderId.value, kind: kind.value, query: query.value, tag: tag.value, trashed: props.trash })
    items.value = result.items || []
    tags.value = (await api.getLibraryTags()).tags || []
    currentFolder.value = folderId.value ? await api.getLibraryItem(folderId.value) : null
    await loadBreadcrumbs()
  } catch (error) { toast.error(error.message || "资料库加载失败") }
  finally { loading.value = false }
}

async function loadBreadcrumbs() {
  const trail = []; let cursor = currentFolder.value; let guard = 0
  while (cursor && guard++ < 30) { trail.unshift(cursor); cursor = cursor.parent_id ? await api.getLibraryItem(cursor.parent_id).catch(() => null) : null }
  breadcrumbs.value = trail
}

function openItem(item) {
  menuId.value = null
  if (item.kind === "folder") return router.push({ name:"library", params:{ folderId:item.id } })
  return router.push({ name:"library-item", params:{ itemId:item.id } })
}

function openCreate(type) { dialog.value = type; formName.value = type === "folder" ? "新建文件夹" : "未命名笔记"; formTags.value = ""; formReview.value = false; editing.value = null }
function openRename(item) { dialog.value = "rename"; formName.value = item.name; formTags.value = (item.tags || []).join(", "); formReview.value = Boolean(item.review_enabled); editing.value = item; menuId.value = null }

async function submitDialog() {
  if (!formName.value.trim()) return
  busy.value = true
  try {
    const noteTags = formTags.value.split(/[,，]/).map(value => value.trim()).filter(Boolean)
    if (dialog.value === "rename") await api.updateLibraryItem(editing.value.id, { name:formName.value.trim(), tags:noteTags, conflict:"keep_both" })
    else await api.createLibraryItem({ parent_id:folderId.value, kind:dialog.value === "folder" ? "folder" : "note", name:formName.value.trim(), tags:noteTags, review_enabled:formReview.value, mime_type:dialog.value === "folder" ? "" : "text/markdown; charset=utf-8" })
    dialog.value = ""; toast.success("已保存"); await load()
  } catch (error) { toast.error(error.message || "保存失败") }
  finally { busy.value = false }
}

async function trashItem(item) { menuId.value=null; try { await api.trashLibraryItem(item.id); toast.success("已移入回收站"); await load() } catch(e){ toast.error(e.message) } }
async function restoreItem(item) { try { await api.restoreLibraryItem(item.id); toast.success("已恢复"); await load() } catch(e){ toast.error(e.message) } }
async function purgeItem(item) { if (deleting.value || !confirm(`永久删除“${item.name}”？此操作无法撤销。`)) return; deleting.value=true; try { await api.purgeLibraryItem(item.id); toast.success("已永久删除"); await load() } catch(e){ toast.error(e.message) } finally { deleting.value=false } }
async function duplicate(item) { menuId.value=null; try { await api.duplicateLibraryItem(item.id, folderId.value); toast.success("已创建副本"); await load() } catch(e){ toast.error(e.message) } }
async function togglePin(item) { menuId.value=null; await api.updateLibraryItem(item.id,{ pinned:!item.pinned }); await load() }
async function toggleReview(item) { menuId.value=null; try { await api.updateLibraryItem(item.id,{ review_enabled:!item.review_enabled }); toast.success(item.review_enabled ? "已移出复习计划" : "已加入复习计划"); await load() } catch(e){ toast.error(e.message || "更新复习计划失败") } }

async function upload(event) {
  const files = [...(event.target.files || [])]; if (!files.length) return
  busy.value = true
  for (const file of files) { try { await api.uploadLibraryFile(file, folderId.value); toast.success(`${file.name} 已上传`) } catch(e){ toast.error(`${file.name}：${e.message}`) } }
  event.target.value=""; busy.value=false; await load()
}

async function dropOnFolder(event, target) {
  const id = Number(event.dataTransfer.getData("text/library-item")); if (!id || target.kind !== "folder") return
  try { await api.updateLibraryItem(id,{ parent_id:target.id, conflict:"keep_both" }); toast.success(`已移动到 ${target.name}`); await load() } catch(e){ toast.error(e.message) }
}
function toggleSelected(id) { const next=new Set(selected.value); next.has(id)?next.delete(id):next.add(id); selected.value=next }
function openOrSelect(item) { if (selected.value.size) return toggleSelected(item.id); openItem(item) }
function toggleSelectAll() { selected.value = selected.value.size === items.value.length ? new Set() : new Set(items.value.map((item) => item.id)) }
async function batch(action) { const ids=[...selected.value]; if(!ids.length||deleting.value)return; const isPermanentDelete=props.trash&&action==='purge'; if(isPermanentDelete)deleting.value=true; try{await api.batchLibraryItems(action,ids);selected.value=new Set();toast.success(`已处理 ${ids.length} 项`);await load()}catch(e){toast.error(e.message)}finally{if(isPermanentDelete)deleting.value=false} }
function setView(next) { view.value=next; localStorage.setItem("libraryView",next) }
function toggleFilter(name) { activeFilter.value = activeFilter.value === name ? "" : name }
function selectFilter(name, value) { if (name === "kind") kind.value = value; else tag.value = value; activeFilter.value = "" }
function closeFilter(event) { if (!event.target.closest(".library-filter")) activeFilter.value = "" }
let searchTimer
watch(()=>route.query.tag,(value)=>{tag.value=String(value||"")})
watch([query,kind,tag,folderId,()=>props.trash],()=>{clearTimeout(searchTimer);searchTimer=setTimeout(load,250)})
watch([folderId,()=>props.trash],([nextFolderId,isTrash])=>{if(!isTrash)rememberLibraryPath(nextFolderId)},{immediate:true})
onMounted(() => { load(); document.addEventListener("pointerdown", closeFilter) })
onBeforeUnmount(() => document.removeEventListener("pointerdown", closeFilter))
</script>

<template>
  <div class="library-page page-stage" @click.self="menuId=null">
    <header class="library-head">
      <nav class="library-crumbs" aria-label="当前位置">
        <RouterLink :to="{name:'library'}">资料库</RouterLink>
        <template v-for="crumb in breadcrumbs" :key="crumb.id"><ChevronRight :size="14"/><RouterLink :to="{name:'library',params:{folderId:crumb.id}}">{{ crumb.name }}</RouterLink></template>
      </nav>
      <div class="library-title-row"><div><h1>{{ title }}</h1><p>{{ props.trash ? '删除内容保留 30 天，可在这里恢复。' : '用文件夹和标签整理笔记与学习文件。' }}</p></div>
        <div v-if="!props.trash" class="library-actions"><button class="lib-btn" @click="fileInput?.click()"><Upload :size="17"/>上传</button><button class="lib-btn lib-btn--primary" @click="openCreate('note')"><Plus :size="17"/>笔记</button><button class="lib-btn" @click="openCreate('folder')"><Folder :size="17"/>文件夹</button></div>
      </div>
    </header>

    <section class="library-toolbar" aria-label="资料筛选">
      <label class="library-search"><Search :size="18"/><input v-model="query" placeholder="搜索名称、标签和笔记正文" aria-label="搜索资料"/></label>
      <div class="library-filter" :class="{open:activeFilter==='kind'}"><button type="button" class="library-filter__trigger" aria-haspopup="listbox" :aria-expanded="activeFilter==='kind'" @click="toggleFilter('kind')"><span>类型</span><strong>{{ kindLabel }}</strong><ChevronDown :size="15"/></button><div v-if="activeFilter==='kind'" class="library-filter__menu" role="listbox" aria-label="资料类型"><button v-for="option in kindOptions" :key="option.value" type="button" role="option" :aria-selected="kind===option.value" :class="{active:kind===option.value}" @click="selectFilter('kind',option.value)">{{ option.label }}</button></div></div>
      <div class="library-filter" :class="{open:activeFilter==='tag'}"><button type="button" class="library-filter__trigger" aria-haspopup="listbox" :aria-expanded="activeFilter==='tag'" @click="toggleFilter('tag')"><span>标签</span><strong>{{ tagLabel }}</strong><ChevronDown :size="15"/></button><div v-if="activeFilter==='tag'" class="library-filter__menu" role="listbox" aria-label="标签筛选"><button v-for="option in tagOptions" :key="option.value" type="button" role="option" :aria-selected="tag===option.value" :class="{active:tag===option.value}" @click="selectFilter('tag',option.value)">{{ option.label }}</button></div></div>
      <div class="library-view-toggle"><button :class="{active:view==='grid'}" aria-label="网格视图" @click="setView('grid')"><Grid2X2 :size="17"/></button><button :class="{active:view==='list'}" aria-label="列表视图" @click="setView('list')"><List :size="18"/></button></div>
    </section>

    <section v-if="selected.size" class="library-selection-bar" aria-live="polite" :aria-busy="deleting"><strong>已选择 {{selected.size}} 项</strong><button class="lib-btn" :disabled="deleting" @click="toggleSelectAll">{{selected.size===items.length?'取消全选':'全选'}}</button><button v-if="props.trash" class="lib-btn" :disabled="deleting" @click="batch('restore')"><ArchiveRestore :size="16"/>恢复</button><button class="lib-btn" :disabled="deleting" @click="batch(props.trash?'purge':'trash')"><Trash2 :size="16"/>{{props.trash?(deleting?'删除中…':'永久删除'):'移入回收站'}}</button><button class="lib-btn" :disabled="deleting" @click="selected=new Set()">取消选择</button></section>
    <div v-if="loading" class="library-loading" role="status" aria-live="polite">加载中</div>
    <section v-else-if="items.length" class="library-items" :class="`library-items--${view}`" aria-label="资料列表">
      <article v-for="item in items" :key="item.id" class="library-card" :class="{selected:selected.has(item.id),'is-previewable':item.kind==='note','has-menu':menuId===item.id}" draggable="true" tabindex="0" @contextmenu.prevent="menuId=item.id" @mouseenter="loadNotePreview(item)"
        @click="openOrSelect(item)" @dragstart="$event.dataTransfer.setData('text/library-item',String(item.id))" @dragover.prevent @drop="dropOnFolder($event,item)" @keydown.enter="openOrSelect(item)">
        <div class="library-card__utility"><input type="checkbox" :checked="selected.has(item.id)" :aria-label="`选择 ${item.name}`" @click.stop="toggleSelected(item.id)"/>
          <button class="library-more" :aria-label="`${item.name} 操作`" @click.stop="menuId=menuId===item.id?null:item.id"><MoreVertical :size="18"/></button></div>
        <div class="library-card__flip">
          <div class="library-card__face library-card__front">
            <div class="library-card__top"><span class="library-card__icon" :class="`is-${item.kind}`"><component :is="itemIcon(item)" :size="21"/></span></div>
            <div class="library-card__body"><strong>{{ item.name }}</strong><span>{{ item.kind==='folder'?'文件夹':item.kind==='note'?'Markdown 笔记':item.mime_type || '文件' }}</span></div>
            <footer><span>{{ formatDate(item.updated_at) }}</span><span v-if="item.tags?.length" class="library-card__footer-tags" :title="`标签：${item.tags.join('、')}`"><Tags :size="11" aria-hidden="true"/><span v-for="value in item.tags" :key="value">{{ value }}</span></span><span class="library-card__size">{{ item.kind==='folder'?'':formatSize(item.size) }}</span></footer>
          </div>
          <div v-if="item.kind==='note'" class="library-card__face library-card__back" role="button" tabindex="-1">
            <span class="library-card__preview-label">Markdown 预览</span>
            <div v-if="previewLoading.has(item.id)" class="library-card__preview-state">正在读取笔记…</div>
            <div v-else-if="previewErrors.has(item.id)" class="library-card__preview-state is-error">预览加载失败，点击打开笔记</div>
            <MarkdownRenderer v-else-if="notePreviews[item.id]" class="library-card__preview" :content="notePreviews[item.id]"/>
            <div v-else class="library-card__preview-state">这篇笔记还没有正文</div>
            <span class="library-card__preview-hint">点击打开完整笔记</span>
          </div>
        </div>
        <div v-if="menuId===item.id" class="library-menu" @click.stop>
          <button v-if="props.trash" :disabled="deleting" @click="restoreItem(item)"><ArchiveRestore :size="16"/>恢复</button><button v-if="props.trash" class="danger" :disabled="deleting" @click="purgeItem(item)"><Trash2 :size="16"/>{{deleting?'删除中…':'永久删除'}}</button>
          <template v-else><button @click="openRename(item)">重命名</button><button v-if="item.kind==='note'" @click="toggleReview(item)">{{ item.review_enabled?'移出复习计划':'加入复习计划' }}</button><button @click="togglePin(item)">{{ item.pinned?'取消置顶':'置顶' }}</button><button @click="duplicate(item)">创建副本</button><button class="danger" @click="trashItem(item)">移入回收站</button></template>
        </div>
      </article>
    </section>
    <section v-else class="library-empty"><span><Folder :size="34"/></span><h2>{{ props.trash?'回收站是空的':'这里还没有资料' }}</h2><p>{{ props.trash?'删除的内容会暂存在这里。':'创建文件夹、笔记，或上传一份学习资料。' }}</p><button v-if="!props.trash" class="lib-btn lib-btn--primary" @click="openCreate('note')"><Plus :size="17"/>创建第一篇笔记</button></section>

    <input ref="fileInput" hidden type="file" multiple accept=".md,.txt,.png,.jpg,.jpeg,.webp,.gif,.pdf,.docx,.xlsx,.pptx" @change="upload"/>
    <BaseDialog :open="!!dialog" :title="dialog==='rename'?'编辑资料':dialog==='folder'?'新建文件夹':'新建 Markdown 笔记'" size="sm" @close="dialog=''">
      <div class="library-dialog-form">
        <label class="library-form-field"><span>名称</span><input v-model="formName" autofocus @keydown.enter="submitDialog"/></label>
        <label v-if="dialog!=='folder'" class="library-form-field"><span>标签</span><input v-model="formTags" placeholder="用逗号分隔，例如：数学, 导数"/></label>
        <label v-if="dialog==='note'" class="library-review-toggle"><input v-model="formReview" type="checkbox"/><span><strong>加入复习计划</strong><small>创建后会进入今日复习队列</small></span></label>
      </div>
      <template #footer><button class="lib-btn" @click="dialog=''">取消</button><button class="lib-btn lib-btn--primary" :disabled="busy" @click="submitDialog">{{ busy?'保存中…':'保存' }}</button></template>
    </BaseDialog>
  </div>
</template>
