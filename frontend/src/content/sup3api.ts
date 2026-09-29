export type Protocol = 'responses' | 'chat' | 'claude' | 'images' | 'tripo' | 'meshy'
export const protocols: { id: Protocol; label: string; endpoint: string; model: string; output: string }[] = [
  { id: 'responses', label: 'GPT · Responses', endpoint: '/v1/responses', model: 'gpt-5.6-sol', output: 'output[] · usage · 工具调用 · SSE' },
  { id: 'chat', label: 'GPT · Chat', endpoint: '/v1/chat/completions', model: 'gpt-5.6-sol', output: 'choices[] · tool_calls · usage · SSE' },
  { id: 'claude', label: 'Claude', endpoint: '/v1/messages', model: 'claude-opus-5-5', output: 'content[] · stop_reason · usage · SSE' },
  { id: 'images', label: 'GPT · Images', endpoint: '/v1/images/generations', model: 'gpt-image-2.5-sunburst', output: 'data[] · b64_json / url · usage（如上游提供）' },
  { id: 'tripo', label: 'Tripo · 3D', endpoint: '/v1/assets/jobs', model: 'v3.1-20260211', output: 'job · artifacts[] · components · steps[].provider_result' },
  { id: 'meshy', label: 'Meshy · 3D', endpoint: '/v1/assets/jobs', model: 'meshy-7.1', output: 'job · artifacts[] · components · steps[].provider_result' },
]
export const operations = [
  ['text_to_3d', '文字生成 3D'], ['image_to_3d', '图像生成 3D'], ['multi_image_to_3d', '多视图生成 3D'],
  ['retexture', '模型重纹理'], ['rig', '自动绑定骨骼'], ['animate', '生成动画'],
]
export const assetModels: Record<string, Record<string, string[]>> = {
  tripo: { text_to_3d: ['v3.1-20260211', 'v3.0-20250812', 'v2.5-20250123'], image_to_3d: ['v3.1-20260211', 'v3.0-20250812', 'v2.5-20250123'], multi_image_to_3d: ['v3.1-20260211', 'v3.0-20250812', 'v2.5-20250123'], retexture: ['v3.5-20260815'], rig: ['v1.0-20240301', 'v2.5-20260210'], animate: [] },
  meshy: { text_to_3d: ['meshy-7.1', 'meshy-6', 'meshy-6-lite', 'meshy-t2'], image_to_3d: ['meshy-7.1', 'meshy-6', 'meshy-6-lite', 'meshy-t2'], multi_image_to_3d: ['meshy-7.1', 'meshy-6', 'meshy-6-lite'], retexture: ['meshy-7', 'meshy-6', 'meshy-6-lite'], rig: [], animate: [] },
}
export type RequestBody = Record<string, unknown>
export interface BuildInput { protocol: Protocol; model: string; prompt: string; image: string; operation: string; images: string[]; source: string; texture: boolean; pbr: boolean; animation: string; native: boolean }
export function buildRequest(v: BuildInput): RequestBody {
  if (v.protocol === 'responses') return { model: v.model, input: [{ role: 'user', content: [...(v.image ? [{ type: 'input_image', image_url: v.image }] : []), { type: 'input_text', text: v.prompt }] }], stream: false }
  if (v.protocol === 'chat') return { model: v.model, messages: [{ role: 'user', content: [...(v.image ? [{ type: 'image_url', image_url: { url: v.image } }] : []), { type: 'text', text: v.prompt }] }], stream: false }
  if (v.protocol === 'claude') {
    const match = v.image.match(/^data:(image\/[\w.+-]+);base64,(.+)$/s)
    const source = match ? { type: 'base64', media_type: match[1], data: match[2] } : { type: 'url', url: v.image }
    return { model: v.model, max_tokens: 1024, messages: [{ role: 'user', content: [...(v.image ? [{ type: 'image', source }] : []), { type: 'text', text: v.prompt }] }], stream: false }
  }
  if (v.protocol === 'images') return { model: v.model, prompt: v.prompt, n: 1, size: '1024x1024' }
  const gen = v.operation.endsWith('to_3d')
  const inputs: RequestBody = {}, parameters: RequestBody = {}
  if (v.operation === 'text_to_3d' || v.operation === 'retexture') inputs.prompt = v.prompt
  if (v.operation === 'image_to_3d') inputs.images = [v.image]
  if (v.operation === 'multi_image_to_3d') inputs.images = v.images
  if (!gen) inputs[v.source.startsWith('job_') ? 'job_id' : 'model_url'] = v.source
  if (gen || v.operation === 'retexture') { parameters.texture = v.texture; parameters.pbr = v.texture && v.pbr }
  if (v.operation === 'animate') parameters.animations = v.animation.split(',').map(s => s.trim()).filter(Boolean)
  if (!v.native) return { provider: v.protocol, operation: v.operation, ...(v.model ? { model: v.model } : {}), inputs, parameters }
  const payload: RequestBody = {}
  if (v.model) payload[v.protocol === 'meshy' ? 'ai_model' : 'model'] = v.model
  if (v.protocol === 'tripo') {
    if (v.operation === 'text_to_3d') payload.prompt = v.prompt
    if (v.operation === 'image_to_3d') payload.input = v.image
    if (v.operation === 'multi_image_to_3d') payload.inputs = v.images
    if (!gen) payload.input = v.source
    if (gen) payload.texture = v.texture
    if (gen || v.operation === 'retexture') payload.pbr = v.texture && v.pbr
    if (v.operation === 'retexture') payload.texture_prompt = { text: v.prompt }
    if (v.operation === 'animate') payload.animations = parameters.animations
  } else {
    if (v.operation === 'text_to_3d') { payload.mode = 'preview'; payload.prompt = v.prompt }
    if (v.operation === 'image_to_3d') payload.image_url = v.image
    if (v.operation === 'multi_image_to_3d') payload.image_urls = v.images
    if (!gen) payload[v.operation === 'animate' ? 'rig_task_id' : v.source.startsWith('job_') ? 'input_task_id' : 'model_url'] = v.source
    if (gen && v.operation !== 'text_to_3d') payload.should_texture = v.texture
    if ((gen && v.operation !== 'text_to_3d') || v.operation === 'retexture') payload.enable_pbr = v.texture && v.pbr
    if (v.operation === 'retexture') payload.text_style_prompt = v.prompt
    if (v.operation === 'animate') payload.action_ids = String(v.animation).split(',').filter(s => s.trim()).map(Number)
  }
  return { provider: v.protocol, operation: v.operation, input_format: v.protocol, payload }
}

export function curlExample(endpoint: string, body: RequestBody, claude = false): string {
  const headers = ['  -H "Authorization: Bearer $SUP3API_API_KEY"', '  -H "Content-Type: application/json"']
  if (claude) headers.push('  -H "anthropic-version: 2023-06-01"')
  if (endpoint === '/v1/assets/jobs') headers.push('  -H "Idempotency-Key: $JOB_REQUEST_ID"')
  return `curl "$SUP3API_BASE_URL${endpoint}" \\\n${headers.join(' \\\n')} \\\n  --data-binary @- <<'SUP3API_JSON'\n${JSON.stringify(body, null, 2)}\nSUP3API_JSON`
}
export function sdkExample(endpoint: string, body: RequestBody, language: 'javascript' | 'python'): string {
  const is3d = endpoint === '/v1/assets/jobs', claude = endpoint === '/v1/messages'
  if (language === 'javascript') return `// Node.js 18+; keep your API key on the server.\nconst response = await fetch(process.env.SUP3API_BASE_URL + ${JSON.stringify(endpoint)}, {\n  method: "POST",\n  headers: {\n    "Authorization": "Bearer " + process.env.SUP3API_API_KEY,\n    "Content-Type": "application/json",${claude ? '\n    "anthropic-version": "2023-06-01",' : ''}${is3d ? '\n    "Idempotency-Key": process.env.JOB_REQUEST_ID,' : ''}\n  },\n  body: JSON.stringify(${JSON.stringify(body, null, 2)})\n});\nconst result = await response.json();\nif (!response.ok) throw new Error(JSON.stringify(result));\nconsole.log(result); // Preserve the full response.${is3d ? '\n// Poll /v1/assets/jobs/{id} until execution and delivery complete.' : ''}`
  return `# pip install requests\nimport os, json, requests\n\nresponse = requests.post(\n    os.environ["SUP3API_BASE_URL"] + ${JSON.stringify(endpoint)},\n    headers={\n        "Authorization": "Bearer " + os.environ["SUP3API_API_KEY"],${claude ? '\n        "anthropic-version": "2023-06-01",' : ''}${is3d ? '\n        "Idempotency-Key": os.environ["JOB_REQUEST_ID"],' : ''}\n    },\n    json=json.loads(${JSON.stringify(JSON.stringify(body))}),\n    timeout=180,\n)\nresponse.raise_for_status()\nresult = response.json()\nprint(result)${is3d ? '\n# Poll /v1/assets/jobs/{id}; download every artifact with auth.' : ''}`
}

export interface DocSection { title: string; text?: string[]; headers?: string[]; rows?: string[][]; code?: string; label?: string; endpoint?: string; note?: string; links?: { label: string; href: string }[] }
export interface DocPage { id: string; group: string; title: string; intro: string; sections: DocSection[] }
const sample = (protocol: Protocol, patch: Partial<BuildInput> = {}) => buildRequest({ protocol, model: protocols.find(p => p.id === protocol)!.model, prompt: 'A stylized wooden treasure chest, game-ready', image: '', operation: 'text_to_3d', images: [], source: 'job_replace_with_your_job_id', texture: true, pbr: true, animation: protocol === 'tripo' ? 'preset:biped:walk' : '0', native: false, ...patch })
const imageURL = 'https://example.com/reference.png'
export const docs: DocPage[] = [
  { id: 'overview', group: '开始', title: '一个入口，多种生成能力。', intro: 'Sup3API 为应用提供文本、图像与三维资产 API。保留适合每种模型的协议、参数和结果结构，让应用只负责自己的产品逻辑。', sections: [
    { title: '选择你的接口', headers: ['能力', '接口', '执行与返回'], rows: [['文本 / 视觉 / 工具','POST /v1/responses','OpenAI Responses JSON 或 SSE'],['聊天兼容','POST /v1/chat/completions','Chat Completions JSON 或 SSE'],['Claude / 视觉 / 工具','POST /v1/messages','Anthropic Messages JSON 或 SSE'],['图像生成 / 编辑','POST /v1/images/generations · /v1/images/edits','Images 数据数组或模型支持的图像事件'],['Tripo / Meshy 三维资产','POST /v1/assets/jobs','202 异步任务 → 模型、PBR、骨骼、动画文件']] },
    { title: '模型决定能力，协议保留结果', text: ['Sup3API 不把所有结果转换成一个 text 字段。GPT 的 output[]、Claude 的 content[]、图像的 data[]、3D 的 artifacts[] 都有各自的用途。工具调用、结束原因、用量和任务阶段也需要由调用方处理。','示例中的模型 ID 是参考值。实际可用模型由密钥所属分组、运营者配置、上游权限与模型能力共同决定；请查询模型与能力接口。不会承诺任意 GPT 或 Claude 模型接收 GLB 文件。'], note: '3D 模型输入用于重纹理、绑定等资产操作。需要 GPT/Claude 理解模型时，应由应用生成渲染图后调用视觉接口。' },
    { title: '第一次调用', links: [{label:'快速开始 →',href:'/docs/quickstart'},{label:'打开 API 接入配置器 →',href:'/connect'},{label:'下载 3D OpenAPI Schema',href:'/docs/assets.openapi.json'}] },
  ] },
  { id: 'quickstart', group: '开始', title: '快速开始', intro: '创建密钥，选择接口，在你的服务端发出第一次请求。接入配置器可以为实际参数生成 cURL、JavaScript 和 Python 示例。', sections: [
    { title: '1. 创建 Sup3API 密钥', text: ['登录控制台，在「API 密钥」中创建密钥并选择可用分组。GPT/Claude 的路由和权限受分组控制。3D 需要运营者单独开通资产额度访问。上游供应商密钥由服务端管理。'], links: [{label:'管理 API 密钥 ↗',href:'/keys'}] },
    { title: '2. 设置服务地址', text: ['SUP3API_BASE_URL 填写本站根地址，不带 /v1。OpenAI SDK 的 base_url 则需要在根地址后加 /v1；Anthropic SDK 的 base_url 使用根地址。不要同时重复拼接 /v1。'], code: 'export SUP3API_BASE_URL="https://YOUR_SUP3API_HOST"\nexport SUP3API_API_KEY="YOUR_SUP3API_KEY"\n# New logical job: create a new ID. Retrying the same job: reuse it.\nexport JOB_REQUEST_ID="asset-example-001"', label:'Shell 环境变量' },
    { title: '3. 请求文本或图像理解', endpoint:'POST /v1/responses', code:curlExample('/v1/responses',sample('responses')),label:'cURL' },
    { title: '4. 创建三维任务', endpoint:'POST /v1/assets/jobs',code:curlExample('/v1/assets/jobs',sample('tripo')),label:'cURL',note:'创建任务会消耗供应商额度。先调用 /v1/assets/quotes 校验参数并获取估价；报价本身不创建生成任务。' },
    { title: '5. 轮询并下载', code:'curl "$SUP3API_BASE_URL/v1/assets/jobs/$JOB_ID" \\\n  -H "Authorization: Bearer $SUP3API_API_KEY"\n\n# After status=succeeded AND delivery_status=ready:\ncurl "$SUP3API_BASE_URL$ARTIFACT_URL" \\\n  -H "Authorization: Bearer $SUP3API_API_KEY" \\\n  -o model.glb',label:'cURL', text:['建议每 3–5 秒轮询。分别判断执行状态与交付状态。不要因为下载失败而重新提交生成任务。'] },
  ] },
  { id:'authentication',group:'开始',title:'认证与访问',intro:'所有模型接口使用 Sup3API API Key。网页登录会话用于控制台，不能代替模型接口的密钥。',sections:[
    {title:'请求头',headers:['请求头','用途'],rows:[['Authorization: Bearer <key>','全部模型与资产接口'],['x-api-key: <key>','Claude 客户端也可以使用此认证头'],['anthropic-version: 2023-06-01','Claude Messages 版本头'],['Content-Type: application/json','JSON 接口；图片编辑的 multipart 由客户端生成边界'],['Idempotency-Key: <8–128 characters>','3D 创建任务必填。同一密钥下，相同 ID 与参数复用同一个任务。']]},
    {title:'权限边界',text:['模型列表只展示当前密钥分组允许的 LLM 模型。访问 3D 接口还要求账号列入资产模块允许名单。失效或被撤销的密钥不能继续访问其任务。','3D 任务与下载绑定创建它的账号和 API Key。同一账号下另一个 Key 也不能读取该任务。跨供应商处理应导出模型，并提供公开 HTTPS model_url。','密钥只应保存在你的服务端环境变量。接入页的诊断密钥仅保存在当前页面内存，发送到本站同源接口，不写入网址、示例代码或浏览器存储。']},
    {title:'错误结构',code:JSON.stringify({error:{code:'asset_access_denied',message:'user is not provisioned for asset provider credits',retryable:false}},null,2),label:'3D 错误示例'},
  ]},
  {id:'models',group:'开始',title:'模型与能力',intro:'先发现可用能力，再决定输入与输出。本站不把所有供应商的模型视为可互换的同一种资源。',sections:[
    {title:'语言与图像模型',endpoint:'GET /v1/models',code:'curl "$SUP3API_BASE_URL/v1/models" \\\n  -H "Authorization: Bearer $SUP3API_API_KEY"',label:'cURL',text:['返回当前 Key 允许访问的模型列表。列表不是所有模态、工具、文件操作的能力保证；还需结合模型官方说明和上游账号类型。本站未配置上游时，会返回相应不可用错误。']},
    {title:'三维模型能力',endpoint:'GET /v1/assets/capabilities',headers:['字段','含义'],rows:[['providers[].available','运营者是否配置了此供应商的 API 凭证，不代表已验证余额'],['models_by_operation','每种操作可选模型版本；rig 与生成模型不是同一套版本'],['operations','支持的生成、重纹理、绑定与动画操作'],['input_formats','sup3api 统一格式，以及 tripo 或 meshy 字段适配格式'],['billing / multiplier','provider_native_credits / 1；不是 LLM 的 USD 余额']]},
    {title:'三维操作选择',headers:['操作','输入','输出'],rows:operations.map(([op,name])=>[`${op} · ${name}`,op==='text_to_3d'?'prompt':op==='image_to_3d'?'images[1]':op==='multi_image_to_3d'?'images[1–4]':op==='animate'?'rig job_id + animations':'job_id 或 model_url',op==='rig'?'绑定模型 / 骨骼 / 权重':op==='animate'?'动画模型 / 动画轨道':'模型 / 材质 / 贴图（取决于参数）'])},
  ]},
  {id:'responses',group:'文本与图像',title:'GPT · Responses',intro:'使用 OpenAI Responses 请求结构，将文本与图像内容块发送给支持视觉的模型。完整返回 output 数组及用量，不只返回最终文本。',sections:[
    {title:'文本 + 图像输入',endpoint:'POST /v1/responses',code:JSON.stringify(sample('responses',{image:imageURL}),null,2),text:['input 支持字符串或消息数组。图像使用 input_image 内容块，image_url 可以是图片 URL 或模型支持的 data URI。此网关没有通用 Files 上传接口，不能把其他账号的 file_id 当成本平台文件使用。']},
    {title:'读取完整返回',headers:['字段','客户端处理'],rows:[['id / model / status','保留请求与模型标识'],['output[]','按 type 区分 message、工具调用、推理等模型输出'],['output[].content[]','读取 output_text 等内容块；不要假设 output[0] 一定是最终文本'],['usage','保留实际返回的输入、输出与其他 token 明细'],['stream=true','解析 text/event-stream；保留事件 type，处理 completed / failed / incomplete']]},
    {title:'JavaScript',code:sdkExample('/v1/responses',sample('responses'),'javascript'),label:'Node.js'},
    {title:'Chat Completions 兼容',endpoint:'POST /v1/chat/completions',code:JSON.stringify(sample('chat',{image:imageURL}),null,2),text:['聊天协议使用 messages[].content[].image_url.url；与 Responses 的 input_image 不能混用。返回 choices[].message / delta、finish_reason、tool_calls 与 usage。跨协议桥接时按目标协议返回，供应商特有能力可能受上游路线限制。']},
  ]},
  {id:'claude',group:'文本与图像',title:'Claude · Messages',intro:'保留 Anthropic Messages 结构。文本、图像、工具调用和停止原因各自保留，不与 OpenAI 的字段混用。',sections:[
    {title:'请求格式',endpoint:'POST /v1/messages',code:curlExample('/v1/messages',sample('claude',{image:imageURL}),true),label:'cURL',text:['max_tokens 必填。系统提示词使用顶层 system，图片放在 type=image 的 source 中。source 支持上游可用的 url 或 base64 形式；不要把 Chat Completions 的 image_url 直接放进 Claude content。']},
    {title:'Base64 图像块',code:JSON.stringify({type:'image',source:{type:'base64',media_type:'image/png',data:'BASE64_BYTES_WITHOUT_DATA_URI_PREFIX'}},null,2)},
    {title:'返回与流式事件',headers:['字段','含义'],rows:[['content[]','text、tool_use、thinking 等实际模型返回的内容块'],['stop_reason / stop_sequence','结束、工具调用、长度限制等原因'],['usage','输入、输出及缓存用量（如上游提供）'],['stream=true','message_start、content_block_*、message_delta、message_stop；还需处理 error 事件']]},
    {title:'工具调用由应用完成',text:['当返回 tool_use 时，应用执行工具后，把 tool_result 放进后续 user 消息。Sup3API 负责协议、鉴权和转发；不会替应用执行编辑器命令或游戏逻辑。']},
  ]},
  {id:'images',group:'文本与图像',title:'图像生成与编辑',intro:'GPT Image 接收文字生成图像，也可通过编辑接口接收图片与提示词。图像模型与用于视觉理解的语言模型需要分别选择。',sections:[
    {title:'文生图',endpoint:'POST /v1/images/generations',code:curlExample('/v1/images/generations',sample('images')),label:'cURL'},
    {title:'图像编辑',endpoint:'POST /v1/images/edits',code:'curl "$SUP3API_BASE_URL/v1/images/edits" \\\n  -H "Authorization: Bearer $SUP3API_API_KEY" \\\n  -F "model=gpt-image-2.5-sunburst" \\\n  -F "image[]=@reference.png" \\\n  -F "prompt=Turn this into a clean game asset concept"',label:'multipart/form-data',text:['编辑接口支持的文件格式、图片数量、尺寸、quality、background、output_format 与 mask 取决于模型和上游账号路线。由 HTTP 客户端生成 multipart Content-Type 与 boundary，不要手动设置成 application/json。']},
    {title:'保留每一张图片',code:JSON.stringify({created:1790630400,data:[{b64_json:'BASE64_IMAGE_BYTES'}],usage:{input_tokens:100,output_tokens:900,total_tokens:1000}},null,2),label:'示意响应（用量仅示意）',text:['遍历 data[]，解码 b64_json 或下载返回的 url。不要只保留第一张图。GPT Image 通常返回 Base64，部分兼容路线返回 URL；以实际响应为准。usage 不是每种路线都提供。','使用 Responses 的 image_generation 工具时，图片属于 output[] 中的 image_generation_call，其 result 是图片数据，不是 Images API 的 data[]。']},
  ]},
  {id:'assets',group:'三维资产',title:'统一 3D 任务',intro:'通过同一任务契约调用 Tripo 与 Meshy。生成、交付、用量和资产清单分别表达，避免重复生成与丢失结果。',sections:[
    {title:'创建任务',endpoint:'POST /v1/assets/jobs',code:JSON.stringify(sample('meshy'),null,2)},
    {title:'请求字段',headers:['字段','约束'],rows:[['provider','tripo 或 meshy'],['operation','text_to_3d / image_to_3d / multi_image_to_3d / retexture / rig / animate'],['model','使用 capabilities.models_by_operation 中的版本；省略会选择适配器默认版本'],['inputs.prompt','Tripo 文生模型 1–1024 字符；Meshy 1–800 字符'],['inputs.images','Tripo：HTTPS；Meshy：HTTPS 或 PNG/JPEG Base64 data URI'],['inputs.model_url / job_id','重纹理与绑定二选一；动画必须引用同 Key、同供应商的成功 rig 任务'],['parameters','texture、pbr、target_faces 或 max_faces、topology、pose、texture_resolution、animations、formats'],['provider_options','按供应商与操作校验的特有选项；未知字段报错，不静默忽略']]},
    {title:'图生模型',code:JSON.stringify(sample('meshy',{operation:'image_to_3d',image:imageURL}),null,2),text:['请求 JSON 总体最大 16 MiB。Meshy 内联 PNG/JPEG 解码后每张最大 10 MiB，每边最大 16384 像素；仍须符合上游实际限制。Tripo 请使用公开 HTTPS 图片 URL。','3D 文件当前通过公开 HTTPS model_url 输入，支持的格式由对应操作决定。没有通用二进制模型上传接口，不接受 data:model/*。']},
    {title:'多视图与模型处理',headers:['场景','规则'],rows:[['Tripo 多视图','四个固定槽位 [front,left,back,right]；front 必填，至少另一个视角非空，其余用空字符串'],['Meshy 多视图','1–4 张图片；按模型能力选择视角'],['重纹理','输入成功 job_id 或公开 model_url；inputs.prompt 是纹理描述'],['骨骼绑定','Meshy 要求适合人形绑定的有纹理模型；Tripo rig 模型版本单独选择'],['动画','Tripo v1 使用 preset:biped:*；v2.5 使用相应版本动作名；Meshy 使用动作 ID 字符串，如 ["0","1"]']]},
    {title:'格式与供应商参数',text:['Meshy 可通过 parameters.formats 选择 glb、fbx、obj、stl、usdz、3mf；默认请求 glb 与 fbx。需要便携组件解析时保留 glb。Tripo 输出格式取决于操作，quad 模式可能返回 FBX。','Tripo 使用 max_faces（上限），Meshy 使用 target_faces（目标）；不可混用。Meshy 的 texture_resolution 为 2k / 4k / 8k，Tripo 用 provider_options.texture_quality。'],links:[{label:'查看完整参数 Schema ↗',href:'/docs/assets.openapi.json'}]},
  ]},
  {id:'native',group:'三维资产',title:'Tripo / Meshy 字段接入',intro:'已有供应商请求体可以放入显式的原生字段封套。Sup3API 将受支持的字段转换为统一任务，然后执行同样的校验、鉴权和交付流程。',sections:[
    {title:'Tripo V3 字段',code:JSON.stringify(sample('tripo',{native:true,operation:'image_to_3d',image:imageURL}),null,2),text:['这是 Tripo V3 字段适配器，不是旧版 V2 task API 的完整路径代理。原生 input 支持公开 HTTPS 资源或本平台 job_id（用于模型处理），不接收其他账号的 file_token / task_id。']},
    {title:'Meshy 字段',code:JSON.stringify(sample('meshy',{native:true,operation:'image_to_3d',image:imageURL}),null,2)},
    {title:'字段映射',headers:['原生字段','Sup3API 字段'],rows:[['Tripo model / Meshy ai_model','model'],['Tripo input / inputs · Meshy image_url / image_urls','inputs.images'],['Tripo face_limit · Meshy target_polycount','parameters.max_faces · parameters.target_faces'],['Tripo texture / pbr · Meshy should_texture / enable_pbr','parameters.texture / pbr'],['Meshy target_formats','parameters.formats'],['Meshy model_url / input_task_id','inputs.model_url / job_id'],['Tripo texture_prompt.text · Meshy text_style_prompt','inputs.prompt（retexture）'],['Meshy rig_task_id / action_ids','inputs.job_id / parameters.animations']]},
    {title:'兼容范围',text:['封套要求 provider 与 input_format 一致；operation 必填。payload 不能与顶层 model、inputs、parameters、provider_options 混用。供应商专有开关仅支持已校验的子集，不支持的字段返回 400。','Meshy 原生 mode=preview 只执行无纹理预览，不额外精修收费。要自动 preview + refine，使用统一请求并设置 texture=true。原生 mode=refine 暂不支持，不能直接引用供应商账号中任意已有任务。','input_task_id / rig_task_id 接收 Sup3API job_ ID，仍检查当前账号与 Key 的所有权。返回始终是 Sup3API Job；原生查询内容放在 steps[].provider_result，原生 POST 返回形状不在兼容承诺内。'],note:'原生字段适配不等于所有供应商端点、模型和参数均已代理。通过 quotes 先验证参数；不要从官方文档复制未支持的字段后假定它们会生效。'},
  ]},
  {id:'results',group:'三维资产',title:'完整结果与资产下载',intro:'同时保存统一资产清单和每个上游阶段的任务结果。应用可以按角色导入文件，也可以读取供应商特有字段。',sections:[
    {title:'任务返回',code:JSON.stringify({id:'job_...',schema_version:'sup3.asset.v1',request:{provider:'meshy',operation:'text_to_3d',model:'meshy-7.1'},status:'succeeded',progress:100,delivery_status:'ready',steps:[{name:'preview',upstream_id:'...',status:'succeeded',provider_result:{status:'SUCCEEDED',model_urls:{glb:'https://provider-cdn.example/preview.glb'}}},{name:'refine',upstream_id:'...',status:'succeeded',provider_result:{status:'SUCCEEDED',model_urls:{glb:'https://provider-cdn.example/textured.glb'},texture_urls:[]}}],artifacts:[{id:'a1',role:'model',format:'glb',media_type:'model/gltf-binary',url:'/v1/assets/jobs/job_.../artifacts/a1',size:123456,sha256:'...'}],components:{geometry:{status:'available',artifact_ids:['a1_geometry']},skeleton:{status:'absent',reason:'no skin in source model'}},cost:{provider:'meshy',credits:30,unit:'credits',kind:'reported',multiplier:1}},null,2),label:'结构示意，省略部分字段'},
    {title:'按角色保留所有文件',headers:['角色 / 组件','内容'],rows:[['model','供应商实际提供的原始 GLB、FBX、OBJ 等模型'],['geometry','从 GLB 提取的几何衍生文件'],['materials / textures','PBR 清单与贴图；保留材质及纹理关联'],['skeleton / skin_weights','实际骨骼层级、逆绑定矩阵、顶点关节索引与权重'],['animations','动画轨道与供应商动画文件'],['preview','模型预览图（如提供）']]},
    {title:'保留差异，不编造缺失项',text:['components 的 status 为 available、absent、unsupported 或 pending。无骨骼模型不会被伪装成已绑定；Draco/meshopt 压缩、FBX 组件提取目前不支持。原始文件仍保留。','steps[].provider_result 保存每次查询的供应商任务内容，包括未知扩展字段与原生 URL。多阶段任务分别保存 preview/refine 结果。原生 URL 可能过期；长期使用 artifacts[].url 的认证下载，并校验 size 与 sha256。旧任务在重新查询前可能没有 provider_result。','不同模型与操作的输出格式、材质和动画数量不同。遍历全部 artifacts，而不是假设唯一 model_url 或第一张贴图。derived_from 指明衍生文件来源。']},
  ]},
  {id:'lifecycle',group:'三维资产',title:'状态、重试与费用',intro:'生成成功与文件交付成功是两个状态。下载问题应重试交付，不能重复付费生成。',sections:[
    {title:'执行状态',headers:['status','处理方式'],rows:[['queued / submitting / running','继续轮询；submitting 已记录付费提交意图'],['succeeded','检查 delivery_status 是否 ready'],['failed / canceled','停止轮询并检查 error'],['submission_unknown','提交结果不确定。由运营者核对上游，不自动重复付费提交']]},
    {title:'交付与控制接口',headers:['接口','用途'],rows:[['POST /v1/assets/quotes','验证并估价；不消耗生成额度'],['GET /v1/assets/jobs','当前账号及 Key 的最近 100 个任务'],['GET /v1/assets/jobs/{id}','查询执行、阶段、资产和用量'],['POST /v1/assets/jobs/{id}/retry-delivery','重新交付生成成功但下载失败的任务'],['POST /v1/assets/jobs/{id}/refresh-artifacts','重新查询已成功任务的文件 URL 并提取'],['POST /v1/assets/jobs/{id}/cancel','本地排队任务可取消；Meshy 仅上游待处理阶段可取消；Tripo 适配器不提供上游取消']]},
    {title:'幂等性',text:['Idempotency-Key 为 8–128 字符。相同 Key、相同请求 ID 与相同规范化参数返回原任务（200），首次创建返回 202；请求 ID 相同但参数不同返回 409。网络中断时先用相同 ID 重试，不能随意生成新 ID。']},
    {title:'费用边界',text:['3D quote 是供应商原生 credits 估算，multiplier=1。cost.kind=reported 才表示所有阶段已取得消耗报告；pending / partially_reported 不是最终账单。','3D 额度由运营者的供应商 API 账号支付，与 LLM USD 钱包独立，目前通过账号允许名单控制。网页中的 LLM 余额和 Key USD 限额不覆盖 3D 额度。Tripo Studio 订阅不自动转换为 API 余额。']},
  ]},
  {id:'errors',group:'参考',title:'错误处理',intro:'协议不同，错误封套也可能不同。检查 HTTP 状态码，再读取 error；流式请求还需检查流中的错误事件。',sections:[
    {title:'常见状态码',headers:['HTTP','含义','处理'],rows:[['400 / 422','字段、模型、模态或参数不支持','修改请求，不自动原样重试'],['401 / 403','认证、分组或资产额度权限不足','检查 Key 状态与账号开通情况'],['402','上游额度不足','由运营者补充供应商额度'],['404','接口、任务或资产不存在 / 无权访问','确认路径及创建任务所用 Key'],['409','幂等冲突、任务忙或不可取消','保持幂等参数，按具体 error.code 处理'],['429','速率限制','使用带抖动的指数退避'],['502 / 503','上游错误、未配置或交付不可用','区分生成与下载；不盲目重试付费 POST']]},
    {title:'三维错误字段',code:JSON.stringify({error:{code:'invalid_request',message:'unsupported native payload field: unknown',retryable:false}},null,2),text:['retryable=true 不代表可以创建另一个付费任务。使用相同的幂等 ID 查询原提交状态。错误消息不会返回供应商 API Key。']},
  ]},
  {id:'testing',group:'参考',title:'测试与验收',intro:'先验证连接、权限与参数，再执行实际生成。接入页的诊断操作不会创建生成任务。',sections:[
    {title:'1. 在浏览器中检查',text:['打开 API 接入页，切换 GPT、Claude、Images、Tripo 与 Meshy，检查端点与请求示例同步变化。复制代码前填写实际可用的模型与输入。','登录控制台，在 API 密钥页创建网关密钥。在接入页输入此密钥后查询模型或资产能力，三维任务先执行「校验与估价」。'],links:[{label:'打开接入页 ↗',href:'/connect'},{label:'管理密钥 ↗',href:'/keys'}]},
    {title:'2. 运行无生成费用的检查',text:['在源码仓库根目录运行以下命令。SUP3API_BASE_URL 使用本站根地址；加入 --assets 前需在当前 shell 设置 SUP3API_API_KEY。','公共检查验证健康状态、页面入口、Logo 和 Schema。--assets 验证认证、已配置供应商的报价和无效字段处理；--models 单独验证模型列表。不会提交生成请求，也不会验证上游真实余额。'],code:'python tools/smoke-sup3api.py --base-url "$SUP3API_BASE_URL"\npython tools/smoke-sup3api.py --base-url "$SUP3API_BASE_URL" --assets\npython tools/smoke-sup3api.py --base-url "$SUP3API_BASE_URL" --models',label:'Shell'},
    {title:'3. 验证真实结果',text:['执行生成示例会产生实际费用。三维任务使用新的 Idempotency-Key，保存返回 id，并查询到执行成功且交付就绪；下载所有所需资产，在你的目标应用中检查模型、材质与动画。','网页显示的错误示例不是实时故障；实际故障请按响应 HTTP 状态码与 error 字段排查。'],links:[{label:'本地启动与完整测试指南 ↗',href:'https://github.com/lanson-dev/Sup3API/blob/main/docs/TESTING.md'},{label:'任务生命周期 ↗',href:'/docs/lifecycle'}]},
  ]},
  {id:'sources',group:'参考',title:'参考资料与开源',intro:'接口说明依据官方文档与本项目实际路由编写。官方能力描述不等于当前部署已经开通全部能力。',sections:[
    {title:'官方文档 · 核对于 2026-09-29',links:[{label:'OpenAI · Images and vision',href:'https://developers.openai.com/api/docs/guides/images-vision'},{label:'OpenAI · Image generation',href:'https://developers.openai.com/api/docs/guides/image-generation'},{label:'Claude · Messages API',href:'https://platform.claude.com/docs/en/api/messages/create'},{label:'Claude · Vision',href:'https://platform.claude.com/docs/en/build-with-claude/vision'},{label:'Tripo · Image to model / V3',href:'https://developers.tripo3d.ai/en/docs/generation-image-to-model/standard'},{label:'Tripo · Text to model / V3',href:'https://developers.tripo3d.ai/en/docs/generation-text-to-model/standard'},{label:'Meshy · Image to 3D',href:'https://docs.meshy.ai/api/image-to-3d'},{label:'Meshy · Text to 3D',href:'https://docs.meshy.ai/api/text-to-3d'},{label:'Sub2API · 上游文档',href:'https://github.com/Wei-Shaw/sub2api/blob/main/README_CN.md'}]},
    {title:'项目边界与许可',text:['Sup3API 客户网站、接入页和文档独立设计。管理员后台和 LLM 网关基于 Sub2API；3D 适配、异步任务和组件提取属于独立的资产 API 模块。','遵循项目现有 LGPL-3.0 许可，保留上游版权与许可说明。应用、游戏编辑器、场景管理与资产导入流程不属于网关。'],links:[{label:'Sup3API 源代码与 README',href:'https://github.com/lanson-dev/Sup3API'},{label:'LGPL-3.0 License',href:'https://github.com/lanson-dev/Sup3API/blob/main/LICENSE'},{label:'3D OpenAPI JSON',href:'/docs/assets.openapi.json'}]},
  ]},
]
