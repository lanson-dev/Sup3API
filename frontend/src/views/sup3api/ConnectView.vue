<template>
  <Sup3APIShell wide>
    <div class="ag-page-intro"><div class="ag-eyebrow">BUILD WITH SUP3API</div><h1>把生成能力，接入你的应用。</h1><p class="ag-lead">选择协议和输入，生成可运行的请求示例。文本、图像与三维模型，保留各自完整的输出。</p></div>
    <div class="ag-tabs" role="tablist" aria-label="API 协议"><button v-for="p in protocols" :key="p.id" role="tab" :aria-selected="protocol===p.id" @click="protocol=p.id">{{ p.label }}</button></div>
    <div class="ag-grid">
      <section class="ag-panel" aria-label="请求配置">
        <div class="ag-endpoint"><span class="ag-method">POST</span><span>{{ selected.endpoint }}</span></div>
        <div style="margin-top:22px" class="ag-fields">
          <label v-if="isAsset" class="ag-field"><span>操作</span><select v-model="operation" class="ag-select"><option v-for="op in operations" :key="op[0]" :value="op[0]">{{ op[1] }}</option></select></label>
          <label v-if="!isAsset || models.length" class="ag-field"><span>模型 ID</span><select v-if="isAsset" v-model="model" class="ag-select"><option v-for="m in models" :key="m">{{ m }}</option></select><input v-else v-model="model" class="ag-input" list="ag-live-models" /><datalist id="ag-live-models"><option v-for="m in liveModels" :key="m" :value="m" /></datalist></label>
        </div>
        <label v-if="needsPrompt" class="ag-field"><span>{{ operation==='retexture' && isAsset ? '纹理描述' : '提示词' }}</span><textarea v-model="prompt" class="ag-textarea" /></label>
        <label v-if="needsImage" class="ag-field"><span>{{ isAsset ? '输入图像' : '参考图像（可选）' }}</span><input v-model="image" class="ag-input" placeholder="https://…/reference.png" /><small class="ag-help">{{ protocol==='tripo' ? 'Tripo 使用公开 HTTPS 图片链接。' : '支持图片 URL 或 Base64 data URI。' }}</small></label>
        <label v-if="needsImage && protocol!=='tripo'" class="ag-field"><span>或从本地读取 PNG / JPEG</span><input type="file" accept="image/png,image/jpeg" @change="readImage" /><small class="ag-help">仅在浏览器中编码，读取文件不会发送生成请求。</small></label>
        <div v-if="isAsset && operation==='multi_image_to_3d'"><label v-for="(view,i) in viewNames" :key="view" class="ag-field"><span>{{ protocol==='tripo' ? view : `视角 ${i+1}${i===0 ? '（必填）' : '（可选）'}` }}</span><input v-model="views[i]" class="ag-input" placeholder="https://…/view.png" /></label></div>
        <label v-if="isAsset && !isGeneration" class="ag-field"><span>{{ operation==='animate' ? '成功绑定任务的 job_id' : '来源模型 URL 或 job_id' }}</span><input v-model="source" class="ag-input" :placeholder="operation==='animate' ? 'job_…' : 'https://…/model.glb 或 job_…'" /></label>
        <label v-if="isAsset && operation==='animate'" class="ag-field"><span>动作名称 / ID（逗号分隔）</span><input v-model="animation" class="ag-input" /></label>
        <div v-if="isAsset && (isGeneration || operation==='retexture') && !(native && protocol==='meshy' && operation==='text_to_3d')" class="ag-actions"><label v-if="operation!=='retexture'" class="ag-check"><input v-model="texture" type="checkbox" />生成纹理</label><label class="ag-check"><input v-model="pbr" type="checkbox" :disabled="!texture" />PBR 材质</label></div>
        <label v-if="isAsset" class="ag-check"><input v-model="native" type="checkbox" />使用供应商原生字段封套</label>
        <p v-if="native && protocol==='meshy' && operation==='text_to_3d'" class="ag-notice">Meshy 原生 preview 只生成无纹理预览。需要自动精修，请关闭原生封套并开启纹理。</p>
        <p v-if="formError" class="ag-notice ag-error" role="alert">{{ formError }}</p>
        <p class="ag-help">参考模型 ID 不代表此 Key 已开通。可在下方查询当前密钥的模型或能力。</p>
        <hr class="ag-divider" />
        <div class="ag-eyebrow">CONNECTION CHECK</div>
        <label class="ag-field" style="margin-top:18px"><span>Sup3API API Key（仅用于本站诊断）</span><input v-model="apiKey" type="password" class="ag-input" autocomplete="off" placeholder="输入你的 Sup3API API Key" /></label>
        <div class="ag-actions"><button class="ag-button ag-button-secondary" :disabled="busy || !apiKey.trim()" @click="diagnose(false)">{{ isAsset ? '查询资产能力' : '查询可用模型' }}</button><button v-if="isAsset" class="ag-button" :disabled="busy || !apiKey.trim() || !!formError" @click="diagnose(true)">校验与估价</button></div>
        <p class="ag-help">以上操作不创建生成任务。Key 仅驻留页面内存；请求发送至本站同源 API。</p>
        <p v-if="diagnosticError" class="ag-notice ag-error" role="alert">{{ diagnosticError }}</p><CodeBlock v-if="diagnostic" :code="diagnostic" label="实际接口返回" />
      </section>
      <section aria-label="接入代码与结果">
        <div class="ag-panel"><div class="ag-eyebrow">YOUR API ENDPOINT</div><p class="ag-mono" style="overflow-wrap:anywhere;margin:16px 0">{{ origin }}{{ selected.endpoint }}</p><div class="ag-actions"><RouterLink class="ag-link ag-small" to="/keys">创建密钥 ↗</RouterLink><RouterLink class="ag-link ag-small" :to="'/docs/'+docId">阅读接口文档 ↗</RouterLink></div></div>
        <p class="ag-help" style="margin-top:18px">环境变量 SUP3API_BASE_URL={{ origin }}；SUP3API_API_KEY=你的服务端密钥。3D 创建另需 JOB_REQUEST_ID。</p>
        <div class="ag-tabs" style="margin:20px 0 0" role="tablist" aria-label="示例语言"><button v-for="lang in languages" :key="lang" role="tab" :aria-selected="language===lang" @click="language=lang">{{ lang }}</button></div>
        <CodeBlock :code="snippet" :label="language" />
        <div class="ag-panel"><div class="ag-eyebrow">RESPONSE CONTRACT</div><h3 style="margin-top:16px">{{ selected.output }}</h3><p class="ag-help">{{ isAsset ? '202 返回任务 ID。轮询到 succeeded 且 delivery_status=ready 后，使用同一 Key 下载全部 artifacts。每个阶段的原生结果在 steps[].provider_result。' : '保留返回的全部内容块、工具调用、结束原因与用量。模型及上游路线决定可用的输入模态和具体输出。' }}</p><RouterLink :to="'/docs/'+(isAsset ? 'results' : docId)" class="ag-link ag-small" style="display:inline-block;margin-top:15px">返回字段与示例 →</RouterLink></div>
      </section>
    </div>
  </Sup3APIShell>
</template>
<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import Sup3APIShell from '@/components/sup3api/Sup3APIShell.vue'
import CodeBlock from '@/components/sup3api/CodeBlock.vue'
import { protocols, operations, assetModels, buildRequest, curlExample, sdkExample, type Protocol } from '@/content/sup3api'
const protocol = ref<Protocol>('responses'), operation = ref('text_to_3d'), model = ref(protocols[0]!.model)
const prompt = ref('A stylized wooden treasure chest, game-ready'), image = ref(''), source = ref(''), views = ref(['','','',''])
const animation = ref('preset:biped:walk'), texture = ref(true), pbr = ref(true), native = ref(false)
const apiKey = ref(''), busy = ref(false), diagnostic = ref(''), diagnosticError = ref(''), liveModels = ref<string[]>([]), fileError = ref('')
const languages = ['cURL', 'JavaScript', 'Python', 'JSON'] as const, language = ref<typeof languages[number]>('cURL')
const origin = window.location.origin, viewNames = ['正面 front（必填）', '左侧 left', '背面 back', '右侧 right']
const selected = computed(() => protocols.find(p => p.id === protocol.value)!)
const isAsset = computed(() => protocol.value==='tripo' || protocol.value==='meshy'), isGeneration = computed(() => operation.value.endsWith('to_3d'))
const models = computed(() => assetModels[protocol.value]?.[operation.value] || [])
const needsPrompt = computed(() => !isAsset.value || ['text_to_3d','retexture'].includes(operation.value))
const needsImage = computed(() => (isAsset.value && operation.value==='image_to_3d') || ['responses','chat','claude'].includes(protocol.value))
const docId = computed(() => isAsset.value ? 'assets' : protocol.value==='chat' ? 'responses' : protocol.value)
watch([protocol,operation], () => { model.value=isAsset.value ? models.value[0] || '' : selected.value.model; animation.value=protocol.value==='meshy' ? '0' : 'preset:biped:walk'; if(operation.value==='retexture')texture.value=true; diagnostic.value=''; diagnosticError.value=''; fileError.value='' })
const formError = computed(() => {
  if(fileError.value) return fileError.value
  if(needsPrompt.value && !prompt.value.trim()) return '请输入提示词。'
  if(!isAsset.value && !model.value.trim()) return '请输入模型 ID。'
  if(isAsset.value && operation.value==='image_to_3d' && !image.value.trim()) return '请输入图片 URL 或选择图片。'
  if(isAsset.value && operation.value==='multi_image_to_3d' && (protocol.value==='tripo' ? !views.value[0] || views.value.filter(Boolean).length<2 : !views.value.some(Boolean))) return '请填写所需的多视图图片。Tripo 至少需要正面和另一个视角。'
  if(isAsset.value && !isGeneration.value && !source.value.trim()) return '请输入来源模型 URL 或任务 ID。'
  if(isAsset.value && operation.value==='animate' && (!source.value.startsWith('job_') || !animation.value.trim())) return '动画需要 Sup3API rig 任务 ID 和动作。'
  if(protocol.value==='meshy' && operation.value==='animate' && !animation.value.split(',').every(s=>/^\d+$/.test(s.trim()))) return 'Meshy 动作 ID 必须是非负整数。'
  return ''
})
const body = computed(() => buildRequest({protocol:protocol.value,model:model.value,prompt:prompt.value,image:image.value,operation:operation.value,images:protocol.value==='tripo' ? views.value : views.value.filter(Boolean),source:source.value,texture:texture.value,pbr:pbr.value,animation:animation.value,native:native.value}))
const snippet = computed(() => language.value==='JSON' ? JSON.stringify(body.value,null,2) : language.value==='cURL' ? curlExample(selected.value.endpoint,body.value,protocol.value==='claude') : sdkExample(selected.value.endpoint,body.value,language.value==='Python'?'python':'javascript'))
async function readImage(event: Event) {
  fileError.value=''
  const file=(event.target as HTMLInputElement).files?.[0]
  if(!file)return
  if(file.size>10*1024*1024 || !['image/png','image/jpeg'].includes(file.type)){fileError.value='请选择不超过 10 MiB 的 PNG 或 JPEG。';return}
  try { image.value=await new Promise<string>((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(String(reader.result));reader.onerror=reject;reader.readAsDataURL(file)}) } catch { fileError.value='图片读取失败，请重新选择。' }
}
let controller: AbortController | undefined
async function diagnose(quote: boolean) {
  if(busy.value)return
  busy.value=true; diagnostic.value='';diagnosticError.value='';controller=new AbortController()
  const timer=setTimeout(()=>controller?.abort(),20000)
  try {
    const response=await fetch(quote?'/v1/assets/quotes':isAsset.value?'/v1/assets/capabilities':'/v1/models',{method:quote?'POST':'GET',headers:{Authorization:`Bearer ${apiKey.value.trim()}`,...(quote?{'Content-Type':'application/json'}:{})},body:quote?JSON.stringify(body.value):undefined,signal:controller.signal,credentials:'omit'})
    const contentType=response.headers.get('content-type') || ''
    if(!contentType.includes('json'))throw new Error('接口未启用或服务未返回 JSON，请联系管理员确认配置。')
    const data=await response.json()
    if(!response.ok)throw new Error(`HTTP ${response.status}: ${data.error?.message || data.message || '请求失败'}`)
    diagnostic.value=JSON.stringify(data,null,2)
    if(!isAsset.value)liveModels.value=(data.data || []).map((m:{id:string})=>m.id)
  } catch(error) { diagnosticError.value=error instanceof Error ? error.message : '连接检查失败' } finally {clearTimeout(timer);busy.value=false}
}
onBeforeUnmount(()=>{controller?.abort();apiKey.value=''})
</script>
