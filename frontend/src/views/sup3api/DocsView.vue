<template>
  <Sup3APIShell wide>
    <div class="ag-doc-layout">
      <aside class="ag-doc-nav" aria-label="文档目录">
        <label class="ag-field"><span class="ag-eyebrow">DEVELOPER DOCS</span><input v-model="search" class="ag-input" type="search" placeholder="搜索文档…" aria-label="搜索文档" /></label>
        <div class="ag-doc-links"><template v-for="(item,index) in filtered" :key="item.id"><div v-if="index===0 || item.group!==filtered[index-1]?.group" class="ag-doc-group">{{ item.group }}</div><RouterLink :to="'/docs/'+item.id">{{ item.id==='overview' ? '概览' : item.title }}</RouterLink></template></div>
        <p v-if="!filtered.length" class="ag-help">没有匹配的文档。</p>
      </aside>
      <article class="ag-doc-article">
        <template v-if="page">
          <div class="ag-eyebrow">{{ page.group }} / SUP3API API</div><h1>{{ page.title }}</h1><p class="ag-lead">{{ page.intro }}</p>
          <section v-for="(section,index) in page.sections" :id="'section-'+index" :key="section.title">
            <h2>{{ section.title }}</h2>
            <div v-if="section.endpoint" class="ag-endpoint"><span class="ag-method">{{ section.endpoint.split(' ')[0] }}</span><span>{{ section.endpoint.split(' ').slice(1).join(' ') }}</span></div>
            <p v-for="text in section.text" :key="text">{{ text }}</p>
            <div v-if="section.rows" class="ag-table-wrap"><table class="ag-table"><thead><tr><th v-for="header in section.headers" :key="header" scope="col">{{ header }}</th></tr></thead><tbody><tr v-for="(row,i) in section.rows" :key="i"><td v-for="(cell,j) in row" :key="j">{{ cell }}</td></tr></tbody></table></div>
            <CodeBlock v-if="section.code" :code="section.code" :label="section.label" />
            <div v-if="section.note" class="ag-notice">{{ section.note }}</div>
            <ul v-if="section.links"><li v-for="link in section.links" :key="link.href"><a class="ag-link" :href="link.href" :target="link.href.startsWith('https:') ? '_blank' : undefined" rel="noopener noreferrer">{{ link.label }}</a></li></ul>
          </section>
          <nav class="ag-doc-footer" aria-label="文档翻页"><RouterLink v-if="previous" :to="'/docs/'+previous.id">← {{ previous.title }}</RouterLink><span v-else></span><RouterLink v-if="next" :to="'/docs/'+next.id">{{ next.title }} →</RouterLink></nav>
        </template>
        <template v-else><h1>文档未找到</h1><RouterLink class="ag-link" to="/docs/overview">返回文档概览</RouterLink></template>
      </article>
    </div>
  </Sup3APIShell>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import Sup3APIShell from '@/components/sup3api/Sup3APIShell.vue'
import CodeBlock from '@/components/sup3api/CodeBlock.vue'
import { docs } from '@/content/sup3api'
const route = useRoute(), search = ref('')
const index = computed(() => docs.findIndex(d => d.id === (route.params.section || 'overview')))
const page = computed(() => docs[index.value]), previous = computed(() => docs[index.value-1]), next = computed(() => docs[index.value+1])
const filtered = computed(() => docs.filter(d => JSON.stringify(d).toLowerCase().includes(search.value.trim().toLowerCase())))
</script>
