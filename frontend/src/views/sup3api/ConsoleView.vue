<template>
  <Sup3APIShell console wide>
    <div class="ag-page-intro"><div class="ag-eyebrow">YOUR WORKSPACE</div><h1>{{ title }}</h1><p class="ag-lead">{{ subtitle }}</p></div>
    <p v-if="error" class="ag-notice ag-error" role="alert">{{ error }} <button class="ag-button ag-button-small ag-button-secondary" @click="load">重试</button></p>
    <p v-if="message" class="ag-notice" role="status">{{ message }}</p>
    <p v-if="loading" class="ag-muted" role="status">正在读取账户数据…</p>
    <template v-else-if="route.path==='/dashboard'">
      <div class="ag-stats"><div class="ag-stat"><span>LLM 可用余额 · USD</span><strong>{{ auth.user?.balance == null ? '—' : '$'+auth.user.balance.toFixed(2) }}</strong></div><div class="ag-stat"><span>活跃 API 密钥</span><strong>{{ stats?.active_api_keys ?? '—' }}</strong></div><div class="ag-stat"><span>今日 LLM 请求</span><strong>{{ stats?.today_requests ?? '—' }}</strong></div></div>
      <div class="ag-grid"><section class="ag-panel"><div class="ag-eyebrow">START BUILDING</div><h2 style="margin-top:20px">你的下一次 API 调用。</h2><p class="ag-lead">在接入页选择 GPT、Claude、Tripo 或 Meshy，生成与你的输入匹配的代码。</p><div class="ag-actions" style="margin-top:25px"><RouterLink to="/connect" class="ag-button">配置 API 接入 ↗</RouterLink><RouterLink to="/keys" class="ag-link ag-small">管理密钥</RouterLink></div></section><section class="ag-panel"><div class="ag-eyebrow">FROM INPUT TO ASSET</div><h2 style="margin-top:20px">文本 · 图像 · 三维</h2><p class="ag-lead">用统一任务创建模型，按角色读取几何、材质、骨骼与动画。完整的阶段结果随任务返回。</p><RouterLink to="/docs/results" class="ag-link" style="display:inline-block;margin-top:25px">了解资产交付 →</RouterLink></section></div>
      <p class="ag-help" style="margin-top:20px">此处余额与调用统计为 LLM 账户数据。3D 使用独立的供应商 credits 与开通权限，请在资产任务的 cost 中查看用量。</p>
    </template>
    <template v-else-if="route.path==='/keys'">
      <div class="ag-actions" style="justify-content:space-between;margin-bottom:25px"><span class="ag-muted ag-small">{{ total }} 个密钥 · 密钥内容默认隐藏</span><button class="ag-button" @click="showCreate=!showCreate">{{ showCreate ? '收起表单' : '+ 创建密钥' }}</button></div>
      <form v-if="showCreate" class="ag-panel" style="margin-bottom:25px" @submit.prevent="createKey"><h3>为你的应用创建密钥</h3><div class="ag-fields"><label class="ag-field"><span>名称</span><input v-model="keyName" class="ag-input" required maxlength="100" placeholder="例如：game-asset-service" /></label><label class="ag-field"><span>模型分组</span><select v-model="groupID" class="ag-select"><option :value="null">未分组（仅已开通的资产 API）</option><option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }} · {{ group.platform }}</option></select></label></div><div class="ag-fields"><label class="ag-field"><span>LLM USD 限额（0 = 不限）</span><input v-model.number="quota" class="ag-input" type="number" min="0" step="0.01" /></label><label class="ag-field"><span>有效天数（留空 = 不设到期日）</span><input v-model.number="expires" class="ag-input" type="number" min="1" max="3650" placeholder="30" /></label></div><p class="ag-help">分组决定 LLM 路线和模型权限；未分组密钥不能调用要求分组的 LLM 接口。USD 限额不覆盖 3D 供应商 credits。</p><button class="ag-button" :disabled="saving || !keyName.trim()" style="margin-top:18px">创建</button></form>
      <div v-if="createdKey" class="ag-notice"><strong>密钥已创建。</strong><p class="ag-help">请复制到服务端环境变量中；不要写入客户端代码。</p><div class="ag-actions" style="margin-top:12px"><code>{{ revealCreated ? createdKey : mask(createdKey) }}</code><button class="ag-button ag-button-secondary ag-button-small" @click="revealCreated=!revealCreated">{{ revealCreated ? '隐藏' : '显示' }}</button><button class="ag-button ag-button-small" @click="copyKey(createdKey)">复制密钥</button><button class="ag-button ag-button-secondary ag-button-small" @click="createdKey='';revealCreated=false">关闭</button></div></div>
      <div v-if="keys.length" class="ag-card-list"><article v-for="key in keys" :key="key.id" class="ag-key-row"><div><h3>{{ key.name }}</h3><code>{{ mask(key.key) }}</code><div class="ag-key-meta"><span>{{ key.group?.name || '未分组' }}</span><span>LLM 用量 ${{ key.quota_used.toFixed(2) }} / {{ key.quota>0 ? '$'+key.quota : '不限' }}</span><span>{{ key.expires_at ? '到期 '+new Date(key.expires_at).toLocaleDateString() : '未设置到期日' }}</span></div></div><div class="ag-actions"><span :class="['ag-badge',{'ag-badge-muted':key.status!=='active'}]">{{ key.status }}</span><button class="ag-button ag-button-secondary ag-button-small" @click="copyKey(key.key)">复制</button><button v-if="key.status==='active'||key.status==='inactive'" class="ag-button ag-button-secondary ag-button-small" :disabled="saving" @click="toggle(key)">{{ key.status==='active' ? '停用' : '启用' }}</button></div></article></div>
      <div v-else-if="!error" class="ag-empty">还没有 API 密钥。为应用创建第一个密钥后，即可开始接入。</div>
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
import { keysAPI } from '@/api/keys'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import { userGroupsAPI } from '@/api/groups'
import type { ApiKey, Group, UsageLog } from '@/types'
const auth=useAuthStore(),route=useRoute()
const loading=ref(false),saving=ref(false),error=ref(''),message=ref(''),page=ref(1),total=ref(0)
const stats=ref<UserDashboardStats|null>(null),keys=ref<ApiKey[]>([]),usage=ref<UsageLog[]>([]),groups=ref<Group[]>([])
const showCreate=ref(false),keyName=ref(''),groupID=ref<number|null>(null),quota=ref(0),expires=ref<number|string>(''),createdKey=ref(''),revealCreated=ref(false)
const title=computed(()=>route.path==='/keys'?'API 密钥':route.path==='/usage'?'调用记录':'开始构建你的想象。')
const subtitle=computed(()=>route.path==='/keys'?'为不同应用创建独立密钥，管理访问分组、用量与有效期。':route.path==='/usage'?'查看模型请求、Token 用量与实际费用。':'管理接入、查看用量，让生成能力成为应用的一部分。')
function errorText(e:unknown){return (e as {message?:string})?.message || '读取失败，请稍后重试。'}
let epoch=0
async function load(){const current=++epoch;loading.value=true;error.value='';try{
  if(route.path==='/dashboard'){const data=await usageAPI.getDashboardStats();if(current===epoch)stats.value=data}
  else if(route.path==='/keys'){const [data,g]=await Promise.all([keysAPI.list(page.value,20),userGroupsAPI.getAvailable()]);if(current===epoch){keys.value=data.items;total.value=data.total;groups.value=g}}
  else {const data=await usageAPI.list(page.value,20);if(current===epoch){usage.value=data.items;total.value=data.total}}
}catch(e){if(current===epoch)error.value=errorText(e)}finally{if(current===epoch)loading.value=false}}
function mask(key:string){return key.length>12 ? key.slice(0,7)+'••••••••••••'+key.slice(-4) : '••••••••••••'}
async function copyKey(key:string){try{await navigator.clipboard.writeText(key);message.value='密钥已复制。'}catch{message.value='复制不可用，请在安全环境中重试。'}}
async function createKey(){if(saving.value)return;saving.value=true;error.value='';try{const result=await keysAPI.create(keyName.value.trim(),groupID.value,undefined,undefined,undefined,quota.value,expires.value?Number(expires.value):undefined);createdKey.value=result.key;revealCreated.value=false;showCreate.value=false;keyName.value='';page.value=1;await load()}catch(e){error.value=errorText(e)}finally{saving.value=false}}
async function toggle(key:ApiKey){saving.value=true;try{await keysAPI.toggleStatus(key.id,key.status==='active'?'inactive':'active');await load()}catch(e){error.value=errorText(e)}finally{saving.value=false}}
watch(()=>route.path,()=>{page.value=1;createdKey.value='';message.value='';keys.value=[];usage.value=[];total.value=0;void load()},{immediate:true})
onBeforeUnmount(()=>{epoch++;createdKey.value='';keys.value=[]})
</script>
