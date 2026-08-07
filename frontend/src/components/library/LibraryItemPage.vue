<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { onBeforeRouteLeave, useRouter } from "vue-router"
import { ArrowLeft, Download, Eye, ListTree, Pencil, ScanText } from "lucide-vue-next"
import { api } from "../../api/index.js"
import { useToast } from "../../store/toast.js"
import { libraryPath, rememberLibraryPath } from "../../utils/libraryPath.js"
import MarkdownEditor from "../MarkdownEditor.vue"
import MarkdownRenderer from "../MarkdownRenderer.vue"
import { extractOutline } from "../../utils/markdown.js"

const props=defineProps({itemId:[String,Number]});const router=useRouter();const toast=useToast();const item=ref(null);const content=ref("");const version=ref(0);const saving=ref(false);const dirty=ref(false);const mode=ref("preview");const previewPane=ref(null);const tagInput=ref("");let timer
const ocrInput=ref(null);const isNote=computed(()=>item.value?.kind==="note");const contentUrl=computed(()=>`/api/library/items/${props.itemId}/content`)
const outline=computed(()=>extractOutline(content.value))
function jumpToHeading(entry){const target=previewPane.value?.querySelector(`[data-outline-index="${entry.index}"]`);if(!target)return;previewPane.value.scrollTo({top:Math.max(0,target.offsetTop-18),behavior:"smooth"})}
async function load(){try{item.value=await api.getLibraryItem(props.itemId);rememberLibraryPath(item.value.parent_id);tagInput.value=(item.value.tags||[]).join(", ");if(isNote.value){const data=await api.getLibraryContent(props.itemId);content.value=data.content;version.value=item.value.current_version||data.version}}catch(e){toast.error(e.message||"资料加载失败")}}
function backToLibrary(){router.push(libraryPath(item.value?.parent_id))}
async function save(force=false){if(!isNote.value||!dirty.value)return;saving.value=true;try{const result=await api.saveLibraryContent(props.itemId,{content:content.value,base_version:version.value,checkpoint:false,force});version.value=result.current_version;item.value=result;dirty.value=false}catch(e){if(e.status===409&&confirm("这篇笔记已在其他位置更新。是否用当前内容覆盖？"))return save(true);toast.error(e.message||"保存失败")}finally{saving.value=false}}
watch(content,()=>{if(!item.value)return;dirty.value=true;clearTimeout(timer);timer=setTimeout(()=>save(false),800)})
async function ocr(event){const file=event.target.files?.[0];if(!file)return;try{const result=await api.ocrImage(file);content.value += `${content.value?'\n\n':''}${result.markdown||result.text||''}`;toast.success("OCR 内容已插入笔记")}catch(e){toast.error(e.message||"OCR 失败")}finally{event.target.value=''}}
async function saveMeta(changes={}){try{const tags=tagInput.value.split(/[,，]/).map(value=>value.trim()).filter(Boolean);item.value=await api.updateLibraryItem(props.itemId,{tags,...changes});tagInput.value=(item.value.tags||[]).join(", ");toast.success("笔记属性已更新")}catch(e){toast.error(e.message||"更新失败")}}
onMounted(load);onBeforeUnmount(()=>clearTimeout(timer));onBeforeRouteLeave(async()=>{if(dirty.value)await save();return true})
</script>
<template><div class="library-item-page page-stage">
  <header class="item-editor-head"><button class="item-back" @click="backToLibrary"><ArrowLeft :size="18"/>返回资料库</button><div><h1>{{ item?.name||'加载中…' }}</h1><p v-if="isNote">{{ saving?'正在保存…':dirty?'等待自动保存':'已保存' }}</p><div v-if="isNote" class="item-meta-editor"><input v-model="tagInput" aria-label="笔记标签" placeholder="添加标签，用逗号分隔" @change="saveMeta()"/><label><input type="checkbox" :checked="item?.review_enabled" @change="saveMeta({review_enabled:$event.target.checked})"/>加入复习计划</label></div></div><div class="item-head-actions"><button v-if="isNote" class="lib-btn" @click="ocrInput?.click()"><ScanText :size="16"/>OCR 插入</button><button v-if="isNote" class="lib-btn" @click="mode=mode==='preview'?'edit':'preview'"><component :is="mode==='preview'?Pencil:Eye" :size="16"/>{{mode==='preview'?'开始编辑':'专注预览'}}</button><a v-else class="lib-btn lib-btn--primary" :href="contentUrl" download><Download :size="16"/>下载</a></div></header>
  <input ref="ocrInput" hidden type="file" accept="image/*,.pdf" @change="ocr"/>
  <main v-if="item" class="item-editor-layout" :class="{'is-note':isNote,'is-preview-mode':isNote&&mode==='preview'}">
    <section v-if="isNote" class="note-workspace" :class="{'is-preview-mode':mode==='preview'}">
      <aside v-if="mode==='preview'" class="note-outline"><h2><ListTree :size="16"/>本文大纲</h2><nav v-if="outline.length" aria-label="笔记大纲"><button v-for="entry in outline" :key="entry.id" :style="{paddingLeft:`${8+(entry.level-1)*10}px`}" @click="jumpToHeading(entry)">{{entry.text}}</button></nav><p v-else>添加 Markdown 标题后，大纲会显示在这里。</p></aside>
      <div v-else class="note-edit"><MarkdownEditor v-model="content" :fill="true" label="笔记正文"/></div>
      <div ref="previewPane" class="note-preview"><MarkdownRenderer :content="content"/></div>
    </section>
    <section v-else class="file-preview"><img v-if="item.mime_type?.startsWith('image/')" :src="contentUrl" :alt="item.name"/><iframe v-else-if="item.mime_type==='application/pdf'" :src="contentUrl" :title="item.name"></iframe><div v-else><Download :size="42"/><h2>{{item.name}}</h2><p>此文件类型暂不支持在线预览，可以下载后使用本地应用打开。</p><a class="lib-btn lib-btn--primary" :href="contentUrl" download>下载文件</a></div></section>
  </main>
</div></template>
