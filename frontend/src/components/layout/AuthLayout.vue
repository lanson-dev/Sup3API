<template>
  <Sup3APIShell>
    <div class="ag-auth">
      <section class="ag-auth-story"><div class="ag-eyebrow">ONE CONNECTION. MORE POSSIBILITIES.</div><h1>连接想象。<br><span>开始构建。</span></h1><p class="ag-lead">从文字、图像到三维资产，<br>通过 Sup3API API 接入你的应用。</p><RouterLink to="/docs/quickstart" class="ag-link" style="display:inline-block;margin-top:28px">阅读快速开始 ↗</RouterLink></section>
      <div><div class="ag-auth-form"><slot /></div><div class="ag-auth-footer"><slot name="footer" /></div></div>
    </div>
  </Sup3APIShell>
</template>
<script setup lang="ts">
import { onMounted } from 'vue'
import Sup3APIShell from '@/components/sup3api/Sup3APIShell.vue'
import { useAppStore } from '@/stores/app'
const app = useAppStore()
onMounted(() => app.fetchPublicSettings())
</script>
