<template>
  <Sup3APIShell console wide>
    <div class="ag-page-intro"><div class="ag-eyebrow">YOUR WORKSPACE</div><h1>{{ title }}</h1><p class="ag-lead">{{ subtitle }}</p></div>
    <p v-if="error" class="ag-notice ag-error" role="alert">{{ error }} <button class="ag-button ag-button-small ag-button-secondary" @click="load">重试</button></p>
    <p v-if="loading" class="ag-muted" role="status">正在读取账户数据…</p>
    <template v-else-if="route.path==='/dashboard'">
      <div class="ag-stats"><div class="ag-stat"><span>LLM 可用余额 · USD</span><strong>{{ auth.user?.balance == null ? '—' : '$'+auth.user.balance.toFixed(2) }}</strong></div><div class="ag-stat"><span>活跃 API 密钥</span><strong>{{ stats?.active_api_keys ?? '—' }}</strong></div><div class="ag-stat"><span>今日 LLM 请求</span><strong>{{ stats?.today_requests ?? '—' }}</strong></div></div>
      <div class="ag-grid"><section class="ag-panel"><div class="ag-eyebrow">START BUILDING</div><h2 style="margin-top:20px">你的下一次 API 调用。</h2><p class="ag-lead">在接入页选择 GPT、Claude、Tripo 或 Meshy，生成与你的输入匹配的代码。</p><div class="ag-actions" style="margin-top:25px"><RouterLink to="/connect" class="ag-button">配置 API 接入 ↗</RouterLink><RouterLink to="/keys" class="ag-link ag-small">管理密钥</RouterLink></div></section><section class="ag-panel"><div class="ag-eyebrow">FROM INPUT TO ASSET</div><h2 style="margin-top:20px">文本 · 图像 · 三维</h2><p class="ag-lead">用统一任务创建模型，按角色读取几何、材质、骨骼与动画。完整的阶段结果随任务返回。</p><RouterLink to="/docs/results" class="ag-link" style="display:inline-block;margin-top:25px">了解资产交付 →</RouterLink></section></div>
      <p class="ag-help" style="margin-top:20px">此处余额与调用统计为 LLM 账户数据。3D 使用独立的供应商 credits 与开通权限，请在资产任务的 cost 中查看用量。</p>
    </template>
    <template v-else>
      <div class="ag-actions" style="justify-content:space-between"><span class="ag-muted ag-small">LLM 调用记录 · {{ total }} 条</span><RouterLink to="/docs/lifecycle" class="ag-link ag-small">3D 任务与用量查询 →</RouterLink></div>
      <div v-if="usage.length" class="ag-table-wrap"><table class="ag-table"><thead><tr><th>时间</th><th>模型</th><th>输入 / 输出 Token</th><th>实际费用</th><th>耗时</th></tr></thead><tbody><tr v-for="row in usage" :key="row.id"><td>{{ new Date(row.created_at).toLocaleString() }}</td><td class="ag-mono">{{ row.model }}</td><td>{{ row.input_tokens }} / {{ row.output_tokens }}</td><td>${{ row.actual_cost.toFixed(6) }}</td><td>{{ row.duration_ms == null ? '—' : (row.duration_ms/1000).toFixed(2)+' s' }}</td></tr></tbody></table></div><div v-else-if="!error" class="ag-empty" style="margin-top:22px">暂无 LLM 调用记录。3D 用量请读取各任务的 cost 与 steps。</div>
    </template>
    <div v-if="route.path!=='/dashboard' && total>20" class="ag-actions" style="justify-content:flex-end;margin-top:25px"><button class="ag-button ag-button-secondary ag-button-small" :disabled="page<=1||loading" @click="page--;load()">上一页</button><span class="ag-small">{{ page }} / {{ Math.ceil(total/20) }}</span><button class="ag-button ag-button-secondary ag-button-small" :disabled="page*20>=total||loading" @click="page++;load()">下一页</button></div>
  </Sup3APIShell>
</template>
<script setup lang="ts">
import { computed, ref, watch, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import Sup3APIShell from '@/components/sup3api/Sup3APIShell.vue'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import type { UsageLog } from '@/types'
const auth=useAuthStore(),route=useRoute()
const loading=ref(false),error=ref(''),page=ref(1),total=ref(0)
const stats=ref<UserDashboardStats|null>(null),usage=ref<UsageLog[]>([])
const title=computed(()=>route.path==='/usage'?'调用记录':'开始构建你的想象。')
const subtitle=computed(()=>route.path==='/usage'?'查看模型请求、Token 用量与实际费用。':'管理接入、查看用量，让生成能力成为应用的一部分。')
function errorText(e:unknown){return (e as {message?:string})?.message || '读取失败，请稍后重试。'}
let epoch=0
async function load(){const current=++epoch;loading.value=true;error.value='';try{
  if(route.path==='/dashboard'){const data=await usageAPI.getDashboardStats();if(current===epoch)stats.value=data}
  else {const data=await usageAPI.list(page.value,20);if(current===epoch){usage.value=data.items;total.value=data.total}}
}catch(e){if(current===epoch)error.value=errorText(e)}finally{if(current===epoch)loading.value=false}}
watch(()=>route.path,()=>{page.value=1;usage.value=[];total.value=0;void load()},{immediate:true})
onBeforeUnmount(()=>{epoch++})
</script>
