<script setup>
import { ArrowLeft, Check, Edit3, Link2, Trash2, X } from "lucide-vue-next"
import { hasContent, reviewLabel, subjectColor } from "../../composables/useErrorLibrary.js"
import MarkdownRenderer from "../MarkdownRenderer.vue"
import BaseButton from "../ui/BaseButton.vue"
import EmptyState from "../ui/EmptyState.vue"

defineProps({ item: Object, today: String, reviewing: Boolean, notFound: Boolean, requestedId: [Number, String], relations: Array, relationLibraryId: String })
defineEmits(["back", "edit", "delete", "review", "tag", "link-library", "unlink-relation", "open-library", "update:relationLibraryId"])
</script>

<template>
  <article class="error-detail" data-testid="formal-error-detail">
    <template v-if="item">
      <header class="error-detail__head">
        <button type="button" class="mobile-back" @click="$emit('back')"><ArrowLeft :size="17" />返回错题库</button>
        <div class="error-detail__identity">
          <div><span class="folio-id">#{{ item.id }}</span><span class="subject-dot" :style="{ '--subject-color': subjectColor(item.subject) }">{{ item.subject }}</span></div>
          <h2 data-testid="formal-error-detail-title">{{ item.title || `未命名错题 #${item.id}` }}</h2>
          <p>{{ reviewLabel(item, today) }}<template v-if="item.created"> · 录入于 {{ item.created.slice(0, 10) }}</template></p>
        </div>
        <div class="error-detail__actions">
          <BaseButton variant="success" :busy="reviewing" @click="$emit('review')"><template #icon><Check :size="16" /></template>标记复习</BaseButton>
          <BaseButton @click="$emit('edit')"><template #icon><Edit3 :size="16" /></template>编辑</BaseButton>
          <BaseButton variant="quiet-danger" aria-label="删除错题" @click="$emit('delete')"><template #icon><Trash2 :size="16" /></template>删除</BaseButton>
        </div>
      </header>
      <div class="error-detail__scroll">
        <section class="detail-manuscript"><h3><span>01</span>题目</h3><MarkdownRenderer :content="item.question" /></section>
        <section v-if="hasContent(item.wrong)" class="detail-manuscript answer-note answer-note--wrong"><h3><span>02</span>错解批注</h3><MarkdownRenderer :content="item.wrong" /></section>
        <section v-if="hasContent(item.correct)" class="detail-manuscript answer-note answer-note--correct"><h3><span>03</span>正解订正</h3><MarkdownRenderer :content="item.correct" /></section>
        <section v-if="hasContent(item.reason)" class="detail-manuscript"><h3><span>04</span>错因归纳</h3><MarkdownRenderer :content="item.reason" /></section>
        <div class="tag-row detail-tags">
          <button v-for="tag in item.tags || []" :key="tag" type="button" class="tag-pill" @click="$emit('tag', '题目标签', tag)">{{ tag }}</button>
          <button v-for="tag in item.reason_tags || []" :key="tag" type="button" class="tag-pill tag-pill--reason" @click="$emit('tag', '错因标签', tag)">{{ tag }}</button>
        </div>
        <section class="detail-manuscript error-relations">
          <h3><span>05</span><Link2 :size="15" />关联笔记</h3>
          <div class="error-relations__form"><input :value="relationLibraryId" inputmode="numeric" placeholder="资料库笔记 ID" @input="$emit('update:relationLibraryId', $event.target.value)" @keydown.enter="$emit('link-library')"/><button type="button" @click="$emit('link-library')">关联</button></div>
          <p v-if="!relations?.length">把相关笔记关联到这道题，复习时可以一键回看知识点。</p>
          <div v-else class="error-relations__list"><button v-for="relation in relations" :key="relation.id" type="button" class="error-relation" @click="$emit('open-library', relation.target_id)"><span>{{ relation.target_name || '关联笔记' }}</span><X :size="14" @click.stop="$emit('unlink-relation', relation.id)"/></button></div>
        </section>
      </div>
    </template>
    <div v-else-if="notFound" class="error-detail__state" data-testid="formal-error-detail-not-found">
      <EmptyState :title="`没有找到错题 #${requestedId}`" description="它可能已经被删除，或不在当前数据目录中。" />
      <BaseButton @click="$emit('back')"><template #icon><ArrowLeft :size="16" /></template>返回错题库</BaseButton>
    </div>
    <EmptyState v-else title="选择一则错题开始复盘" description="从错题卡片进入详情后，可查看完整题面、订正和错因。" />
  </article>
</template>
