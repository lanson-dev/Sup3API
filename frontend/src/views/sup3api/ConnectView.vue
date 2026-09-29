<template>
  <Sup3APIShell wide>
    <div class="ag-page-intro"><div class="ag-eyebrow">BUILD WITH SUP3API</div><h1>把生成能力，接入你的应用。</h1><p class="ag-lead">选择协议和输入，生成可运行的请求示例。文本、图像与三维模型，保留各自完整的输出。</p></div>
    <div class="ag-tabs" role="tablist" aria-label="API 协议"><button v-for="p in protocols" :key="p.id" role="tab" :aria-selected="protocol===p.id" @click="protocol=p.id">{{ p.label }}</button></div>
    <div class="ag-grid">
      <section class="ag-panel" aria-label="请求配置">
        <div class="ag-endpoint"><span class="ag-method">POST</span><span>{{ endpoint }}</span></div>
        <label v-if="isAsset" class="ag-field" style="margin-top:22px"><span>接入方式</span><select v-model="accessMode" class="ag-select"><option value="unified">统一 API · 跨供应商请求与交付</option><option value="native-tripo">Tripo 原生 API · 兼容 V3 工作流</option><option value="native-meshy">Meshy 原生 API · 兼容已有工作流</option><option value="fields">原生字段封套 · 返回统一任务（旧版）</option></select></label>
        <template v-if="isNativeAPI">
          <label class="ag-field"><span>原生操作</span><select v-model="nativeOperation" class="ag-select"><option v-for="op in nativeChoices" :key="op.id" :value="op.id">{{ op.label }}</option></select></label>
          <label class="ag-field"><span>供应商原生请求 JSON</span><textarea v-model="nativeJSON" class="ag-textarea ag-mono" style="min-height:270px" spellcheck="false" /></label>
          <p class="ag-notice">保留供应商字段、默认值与原生响应。前置任务必须使用同一网关密钥通过此入口创建；不会自动执行精修或转换。原生文件上传、任务列表、历史任务导入与 Webhook 暂不支持。</p>
          <RouterLink class="ag-link ag-small" to="/docs/compatibility">迁移与兼容范围 ↗</RouterLink>
        </template>
        <template v-else>
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
        <template v-if="isAsset && mode==='unified'">
          <div v-if="isGeneration || operation==='retexture'" class="ag-field"><span>交付格式（可多选）</span><div class="ag-actions"><label v-for="format in formatChoices" :key="format" class="ag-check"><input v-model="formats" type="checkbox" :value="format" />{{ format.toUpperCase() }}</label></div><small class="ag-help">已选：{{ formats.join(", ") || "无" }}。切换供应商保留要求；不支持的格式会在提交前报错。当前不自动增加转换任务。</small><button v-if="formats.some(f=>!formatChoices.includes(f))" class="ag-button ag-button-secondary" @click="formats=['glb']">改用通用 GLB 格式</button></div>
          <div v-if="isGeneration || operation==='retexture'" class="ag-field"><span>必须包含的组件（交付后验证）</span><div class="ag-actions"><label v-for="item in componentChoices" :key="item[0]" class="ag-check"><input v-model="requiredComponents" type="checkbox" :value="item[0]" />{{ item[1] }}</label></div></div>
          <label class="ag-field"><span>供应商扩展参数（JSON，可选）</span><textarea v-model="extensionsJSON" class="ag-textarea ag-mono" spellcheck="false" placeholder="{}" /><small class="ag-help">自动放入 extensions.{{ protocol }}。切换供应商时请检查扩展；报价会校验参数与模型的兼容性。</small></label>
        </template>
        </template>
        <p v-if="native && protocol==='meshy' && operation==='text_to_3d'" class="ag-notice">Meshy 原生 preview 只生成无纹理预览。需要自动精修，请关闭原生封套并开启纹理。</p>
        <p v-if="formError" class="ag-notice ag-error" role="alert">{{ formError }}</p>
        <p class="ag-help">参考模型 ID 不代表此 Key 已开通。可在下方查询当前密钥的模型或能力。</p>
        <hr class="ag-divider" />
        <div class="ag-eyebrow">CONNECTION CHECK</div>
        <label class="ag-field" style="margin-top:18px"><span>Sup3API API Key（仅用于本站诊断）</span><input v-model="apiKey" type="password" class="ag-input" autocomplete="off" placeholder="输入你的 Sup3API API Key" /></label>
        <div class="ag-actions"><button class="ag-button ag-button-secondary" :disabled="busy || !apiKey.trim()" @click="diagnose(false)">{{ isAsset ? '查询资产能力' : '查询可用模型' }}</button><button v-if="isAsset && !isNativeAPI" class="ag-button" :disabled="busy || !apiKey.trim() || !!formError" @click="diagnose(true)">校验与估价</button></div>
        <p class="ag-help">以上操作不创建生成任务。Key 仅驻留页面内存；请求发送至本站同源 API。</p>
        <p v-if="diagnosticError" class="ag-notice ag-error" role="alert">{{ diagnosticError }}</p><CodeBlock v-if="diagnostic" :code="diagnostic" label="实际接口返回" />
      </section>
      <section aria-label="接入代码与结果">
        <div class="ag-panel"><div class="ag-eyebrow">YOUR API ENDPOINT</div><p class="ag-mono" style="overflow-wrap:anywhere;margin:16px 0">{{ origin }}{{ endpoint }}</p><div class="ag-actions"><RouterLink class="ag-link ag-small" to="/keys">创建密钥 ↗</RouterLink><RouterLink class="ag-link ag-small" :to="'/docs/'+docId">阅读接口文档 ↗</RouterLink></div></div>
        <p class="ag-help" style="margin-top:18px">环境变量 SUP3API_BASE_URL={{ origin }}；SUP3API_API_KEY=你的服务端密钥。统一 3D 创建必需 JOB_REQUEST_ID，原生创建建议提供以防重复提交。</p>
        <div class="ag-tabs" style="margin:20px 0 0" role="tablist" aria-label="示例语言"><button v-for="lang in languages" :key="lang" role="tab" :aria-selected="language===lang" @click="language=lang">{{ lang }}</button></div>
        <CodeBlock :code="snippet" :label="language" />
        <div class="ag-panel"><div class="ag-eyebrow">RESPONSE CONTRACT</div><h3 style="margin-top:16px">{{ isNativeAPI ? '供应商原生 JSON · 状态 · 资产 URL' : selected.output }}</h3><p class="ag-help">{{ isNativeAPI ? '创建返回供应商任务 ID；使用同一原生入口查询，保留官方状态码和响应字段。资产下载使用供应商返回的 URL，受其有效期限制。' : isAsset ? '202 返回任务 ID。轮询到 succeeded 且 delivery_status=ready 后，使用同一 Key 下载全部 artifacts。每个阶段的原生结果在 steps[].provider_result。' : '保留返回的全部内容块、工具调用、结束原因与用量。模型及上游路线决定可用的输入模态和具体输出。' }}</p><RouterLink :to="'/docs/'+(isNativeAPI ? 'compatibility' : isAsset ? 'results' : docId)" class="ag-link ag-small" style="display:inline-block;margin-top:15px">返回字段与示例 →</RouterLink></div>
      </section>
    </div>
  </Sup3APIShell>
</template>
<script setup lang="ts">
import { ref, computed, watch, onBeforeUnmount } from 'vue'
import Sup3APIShell from '@/components/sup3api/Sup3APIShell.vue'
import CodeBlock from '@/components/sup3api/CodeBlock.vue'
import { protocols, operations, assetModels, buildRequest, curlExample, sdkExample, nativeOperations, type RequestBody, type Protocol } from '@/content/sup3api'
const protocol = ref<Protocol>('responses'), operation = ref('text_to_3d'), model = ref(protocols[0]!.model)
const prompt = ref('A stylized wooden treasure chest, game-ready'), image = ref(''), source = ref(''), views = ref(['','','',''])
const animation = ref('preset:biped:walk'), texture = ref(true), pbr = ref(true)
const mode = ref<'unified'|'native'|'fields'>('unified'), native = computed(()=>mode.value==='fields')
const accessMode=computed({get:()=>mode.value==='native' ? `native-${protocol.value}` : mode.value,set:(value:string)=>{if(value==='native-tripo' || value==='native-meshy'){protocol.value=value==='native-tripo'?'tripo':'meshy';mode.value='native'}else{mode.value=value==='fields'?'fields':'unified'}}})
const nativeOperation = ref('text_to_3d'), nativeJSON = ref('{}'), extensionsJSON=ref('{}')
const formats=ref<string[]>(['glb']),requiredComponents=ref<string[]>([])
const componentChoices=[['geometry','几何'],['materials','材质'],['textures','贴图']]
const formatChoices=computed(()=>protocol.value==='tripo'?['glb']:['glb','fbx','obj','stl','usdz','3mf'])
const apiKey = ref(''), busy = ref(false), diagnostic = ref(''), diagnosticError = ref(''), liveModels = ref<string[]>([]), fileError = ref('')
const languages = ['cURL', 'JavaScript', 'Python', 'JSON'] as const, language = ref<typeof languages[number]>('cURL')
const origin = window.location.origin, viewNames = ['正面 front（必填）', '左侧 left', '背面 back', '右侧 right']
const selected = computed(() => protocols.find(p => p.id === protocol.value)!)
const isAsset = computed(() => protocol.value==='tripo' || protocol.value==='meshy'), isGeneration = computed(() => operation.value.endsWith('to_3d'))
const isNativeAPI=computed(()=>isAsset.value && mode.value==='native')
const nativeChoices=computed(()=>nativeOperations[protocol.value] || [])
const nativeSelected=computed(()=>nativeChoices.value.find(o=>o.id===nativeOperation.value) || nativeChoices.value[0])
const endpoint=computed(()=>isNativeAPI.value ? '/providers/'+protocol.value+(nativeSelected.value?.path || '') : selected.value.endpoint)
watch([protocol,nativeOperation,mode],()=>{if(nativeChoices.value.length && !nativeChoices.value.some(o=>o.id===nativeOperation.value))nativeOperation.value=nativeChoices.value[0]!.id;nativeJSON.value=JSON.stringify(nativeSelected.value?.example || {},null,2)},{immediate:true})
const parseObject=(text:string):RequestBody=>{const v=JSON.parse(text);if(!v || typeof v!=='object' || Array.isArray(v))throw new Error('请输入 JSON 对象');return v}
const models = computed(() => assetModels[protocol.value]?.[operation.value] || [])
const needsPrompt = computed(() => !isAsset.value || ['text_to_3d','retexture'].includes(operation.value))
const needsImage = computed(() => (isAsset.value && operation.value==='image_to_3d') || ['responses','chat','claude'].includes(protocol.value))
const docId = computed(() => isNativeAPI.value ? 'compatibility' : isAsset.value ? 'assets' : protocol.value==='chat' ? 'responses' : protocol.value)
watch([protocol,operation], () => { model.value=isAsset.value ? models.value[0] || '' : selected.value.model; animation.value=protocol.value==='meshy' ? '0' : 'preset:biped:walk'; if(operation.value==='retexture')texture.value=true; diagnostic.value=''; diagnosticError.value=''; fileError.value='' })
const formError = computed(() => {
  if(isNativeAPI.value){try{parseObject(nativeJSON.value)}catch{return '原生请求必须是有效 JSON 对象。'}return ''}
  if(isAsset.value && mode.value==='unified'){
    try{parseObject(extensionsJSON.value)}catch{return '扩展参数必须是有效 JSON 对象。'}
    if((isGeneration.value || operation.value==='retexture') && (!formats.value.length || formats.value.some(f=>!formatChoices.value.includes(f))))return '请选择当前供应商支持的交付格式。'
    if((isGeneration.value || operation.value==='retexture') && requiredComponents.value.length && !formats.value.includes('glb'))return '组件验证需要 GLB 格式。'
    if((isGeneration.value || operation.value==='retexture') && !texture.value && requiredComponents.value.some(c=>c==='materials'||c==='textures'))return '必需材质或贴图时，请开启纹理生成。'
  }
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
const body = computed(() => {if(isNativeAPI.value){try{return parseObject(nativeJSON.value)}catch{return {}}}let extensions:RequestBody={};try{extensions=parseObject(extensionsJSON.value)}catch{/* Shown by formError. */}return buildRequest({protocol:protocol.value,model:model.value,prompt:prompt.value,image:image.value,operation:operation.value,images:protocol.value==='tripo' ? views.value : views.value.filter(Boolean),source:source.value,texture:texture.value,pbr:pbr.value,animation:animation.value,native:native.value,formats:isGeneration.value || operation.value==='retexture'?formats.value:[],requiredComponents:isGeneration.value || operation.value==='retexture'?requiredComponents.value:[],extensions})})
const snippet = computed(() => language.value==='JSON' ? JSON.stringify(body.value,null,2) : language.value==='cURL' ? curlExample(endpoint.value,body.value,protocol.value==='claude') : sdkExample(endpoint.value,body.value,language.value==='Python'?'python':'javascript'))
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
