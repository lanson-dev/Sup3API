# 首页演示

- 主图：三棱锥框架与悬浮核心，使用柔和 PBR 材质，持续旋转且可拖动。四个顶点代表文本、图像、三维与更多能力；悬停或键盘聚焦时显示提示，并产生核心向节点流动的光点。触屏可点选，Escape 或点击画布收起。系统“减少动态效果”偏好关闭自动运动。
- 文本展示实际提示词与 Nano Banana → Tripo H3.1 的生成流程，图像展示对应原画。
- 三维：角色与天使双翼两套真实 Tripo H3.1 资产；右侧选择器联动文本、原画和模型。角色支持骨骼；场景只显示实际可用的工具。模型、骨骼、材质与下载使用图标及无障碍名称。
- 骨骼与材质信息直接读取 GLB：角色 65 个关节；两套模型均有颜色、Normal、Metallic/Roughness 三张贴图。材质面板拆分真实通道，支持点击放大；使用原生 dialog 支持 Escape 关闭与焦点恢复。资产来源见 [示例记录](../frontend/src/assets/demos/README.md)。
- 三维展台使用柔和环境光、低强度环境反射和静态柔影；放大模型构图，避免强反射。
- Three.js 按需加载；离屏与后台停止绘制，卸载释放 GPU 资源。预览失败仍可下载优化后的 GLB。打开首页不会调用供应商生成任务。

## 验证

组件测试覆盖首页导航与复制、节点提示与关闭、自动旋转与动态偏好、模型信息显示、加载失败及卸载清理。执行：

```sh
cd frontend
node node_modules/vue-tsc/bin/vue-tsc.js -b
node node_modules/vitest/vitest.mjs run src/components/sup3api/__tests__/LandingMesh.spec.ts src/components/sup3api/__tests__/CharacterDemo.spec.ts src/views/__tests__/Sup3APIHomeView.spec.ts
```

本次保留既有 Dashboard/Usage、客户菜单、API 密钥、分组及供应商账号逻辑，不修改后端 API。默认 CSP 与部署示例增加 Meshopt 所需的 `wasm-unsafe-eval`，以及内嵌贴图所需的 `connect-src blob:`；JavaScript `unsafe-eval` 仍不允许。已有自定义 CSP 的部署需同步这两项。

## 传输与缓存

主图直接由 Three.js 构建几何与 PBR 材质，不下载额外模型或贴图；复用下方展台的 Three.js 包。离屏、后台和卸载时停止渲染或释放资源。

角色 546 KiB、场景 486 KiB，较原始 PBR 文件均缩小约 86%。每个模型不足一万三角形，颜色 1024px，Normal 与 MR 数据贴图 512px。资源经 Vite 内容哈希输出，复用现有一年不可变缓存；只加载选中且进入视口的三维 Demo，不预加载全部模型。原画共约几十 KiB，Three.js 为懒加载独立共享包。大流量部署可让 CDN 缓存 `/assets/`；缓存头不等于已经配置了 CDN。

自定义前端删除分组脚注、拖动提示、重复格式状态、文件编码说明及文档重复品牌标题。保留表单校验、协议约束、费用行为和错误反馈。
