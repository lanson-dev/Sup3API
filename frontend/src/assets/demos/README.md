# 首页示例资产

2026-09-30 在用户授权的 Tripo Studio 账户生成。

- `mannequin-concept.webp`：Nano Banana 原画，864 × 1184；任务 `3a4b8237-d992-4368-acc8-0dd3118e9124`。
- `mannequin-rigged.glb`：原画经 Tripo H3.1 转为模型，任务 `2b0150cc-de30-404f-9a79-61401fbd27ef`；Mixamo 人形绑定结果 `fb6b1710-8c0b-4d27-aa61-cf5bc2c7fa0a`。65 个关节、1 个网格、1 个材质、内嵌 1024px JPEG 颜色贴图；粗糙度 0.9、金属度 0。Meshopt 压缩，浏览器使用 Three.js 自带解码器。
- 本套生成消耗 1 次免费图片额度、25 模型积分、20 绑定积分。此前探索者版本未用于首页。

设计方向：接近游戏引擎默认人形的原创中性灰模；无五官、无服装、无装饰，哑光浅灰材质，A 姿势。非 Epic 官方资产。页面展示静态生成结果，不代表每次打开首页都会调用供应商。

原画提示词：

> A neutral humanoid game-engine test mannequin, in the restrained spirit of a default Unreal Engine gray mannequin, original design. Featureless smooth oval head: absolutely no face, eyes, nose, mouth, hair or mask. Gender-neutral simplified adult proportions, softly abstract anatomical volumes rather than muscles, solid matte pale warm-gray polymer body, subtle graphite articulation seams at shoulders, elbows, wrists, hips and knees. Entire body is the mannequin itself: absolutely NO clothing, cape, hood, jacket, trousers, boots, armor, accessories, weapons, emblems, decorative panels, glowing lines or mechanical greebles. Clean continuous sculptural form, quiet and anonymous, a slightly mysterious presence through the blank face and soft shadow only. Symmetrical A-pose, arms 35 degrees away from torso, five separated fingers on each hand, feet apart. Full body visible head to toe, front view, centered. Premium realistic 3D render with soft diffuse studio lighting, high roughness and low reflections, off-white background, no pedestal, no text, no watermark. Clear separate limbs suitable for biped skeletal rigging.

## 场景摆件

- `angel-wings.webp`：Nano Banana，任务 `ef510a7a-1356-44fd-9bd2-531530619032`。
- `angel-wings.glb`：Tripo H3.1，任务 `7add4b32-7286-413e-b298-9ff26e9fa6e1`；导出 `5274dff5-b8d0-41fb-94ca-a19cd9157cde`。9,494 个三角形，无骨骼。消耗 1 次免费图片额度和 25 模型积分。
- 两套资产的实际完整提示词与首页展示数据保存在 `src/content/sup3api-landing.ts`。

## 网页优化

| GLB | 原始文件 | 网页文件 | 三角形 |
| --- | ---: | ---: | ---: |
| 中性人形 | 548,808 B | 192,436 B | 9,474 |
| 天使双翼 | 519,944 B | 140,956 B | 9,494 |

保留 Meshopt 几何压缩，使用 `tools/optimize-demo-glb.py` 将 2048px 色彩图缩为 1024px JPEG（质量 82）；压缩网格与蒙皮数据逐字节不变。原始供应商结果仍可在 Tripo 任务中导出。网页下载按钮提供优化后的同一份 GLB。

Vite 构建生成带内容哈希的文件名，复用服务器 `public, max-age=31536000, immutable` 缓存。仅当选中三维标签且面板进入视口时加载当前模型。静止、离屏、后台停止绘制；切换示例释放上一个模型的 GPU 资源。
