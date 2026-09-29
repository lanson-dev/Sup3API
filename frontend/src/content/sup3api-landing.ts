import { curlExample, nativeOperations } from './sup3api'

export const landingCapabilities = [
  {
    id: 'text',
    label: '文本与视觉',
    eyebrow: 'WORDS BECOME WORLDS',
    title: '每个世界，始于一句话。',
    description: '从创意构思到图像理解，让 GPT 与 Claude 成为应用的思考伙伴。',
    features: ['文本与图像输入', '流式响应与工具调用', '保留完整内容与用量'],
    link: '/docs/responses',
    linkText: '探索文本 API',
  },
  {
    id: 'image',
    label: '图像生成',
    eyebrow: 'MAKE THE INVISIBLE VISIBLE',
    title: '让脑海中的画面，浮现。',
    description:
      '用文字描绘全新画面，或从参考图像继续创作。接入模型的图像生成与编辑能力。',
    features: ['文字生成图像', '参考图编辑', '按模型返回图像结果'],
    link: '/docs/images',
    linkText: '探索图像 API',
  },
  {
    id: 'mesh',
    label: '三维资产',
    eyebrow: 'IDEAS TAKE SHAPE',
    title: '从平面灵感，到立体可能。',
    description:
      '通过 Tripo、Meshy 生成模型与材质，让三维资产进入你的应用和工作流。',
    features: [
      '文字、单图与多视图生成',
      '重纹理、骨骼与动画',
      '任务状态与完整资产交付',
    ],
    link: '/docs/assets',
    linkText: '探索三维 API',
  },
]
const nativeTripo = nativeOperations.tripo.find(
  (item) => item.id === 'text_to_3d',
)!
const nativeMeshy = nativeOperations.meshy.find(
  (item) => item.id === 'text_to_3d',
)!
export const landingExamples = [
  {
    id: 'unified',
    label: '统一 API',
    endpoint: '/v1/assets/jobs',
    body: {
      provider: 'meshy',
      operation: 'text_to_3d',
      inputs: { prompt: 'A floating garden in the clouds' },
    },
  },
  {
    id: 'tripo',
    label: 'Tripo 原生',
    endpoint: `/providers/tripo${nativeTripo.path}`,
    body: { ...nativeTripo.example, prompt: 'A floating garden in the clouds' },
  },
  {
    id: 'meshy',
    label: 'Meshy 原生',
    endpoint: `/providers/meshy${nativeMeshy.path}`,
    body: { ...nativeMeshy.example, prompt: 'A floating garden in the clouds' },
  },
].map((example) => ({
  ...example,
  code: curlExample(example.endpoint, example.body),
}))
export const landingSteps = [
  {
    number: '01',
    title: '创建密钥',
    description: '登录控制台，为应用选择已开通的分组。',
    to: '/keys',
  },
  {
    number: '02',
    title: '配置请求',
    description: '选择模型与输入，复制适合你的调用示例。',
    to: '/connect',
  },
  {
    number: '03',
    title: '接收结果',
    description: '处理流式响应，或获取三维任务与资产。',
    to: '/docs/quickstart',
  },
]
export const landingFaqs = [
  {
    question: '已有的 OpenAI、Claude 工作流可以接入吗？',
    answer:
      '支持对应的 API 协议。将服务地址和应用密钥替换为 Sup3API 的配置，并选择已开通的模型。具体接口与支持范围见文档。',
    to: '/docs/compatibility',
    link: '查看兼容说明',
  },
  {
    question: 'Tripo 和 Meshy 必须使用统一格式吗？',
    answer:
      '可以选择统一资产 API，也可以使用已支持的 Tripo、Meshy 原生接口，保留原生请求和返回结构。原生接口的具体支持范围见迁移文档。',
    to: '/docs/compatibility',
    link: '查看迁移文档',
  },
  {
    question: '生成的三维模型如何获取？',
    answer:
      '三维生成是异步任务。创建后查询任务状态，完成后读取模型、材质等资产；统一 API 的资产通过带认证的下载链接获取。',
    to: '/docs/lifecycle',
    link: '了解任务生命周期',
  },
  {
    question: '接入前可以先测试吗？',
    answer:
      '在 API 接入页填入应用密钥，查询可用能力。三维任务可先进行校验与估价，再由你的应用提交生成请求。',
    to: '/connect',
    link: '打开接入工作台',
  },
]
