# 首页演示

- 主图：哑光三角形，持续自动旋转，可拖动；无播放、线框、旋转及重置按钮。系统“减少动态效果”偏好关闭自动旋转。
- 文本展示实际提示词与 Nano Banana → Tripo H3.1 的生成流程，图像展示对应原画。
- 三维：角色与天使双翼两套真实 Tripo H3.1 资产；右侧选择器联动文本、原画和模型。角色支持骨骼；场景只显示实际可用的工具。模型、骨骼、材质与下载使用图标及无障碍名称。
- 骨骼与材质信息直接读取 GLB：65 个关节、1 个网格、1 个材质、1 张颜色贴图；不添加虚构的骨骼或贴图。资产来源见 [示例记录](../frontend/src/assets/demos/README.md)。
- Three.js 按需加载；离屏与后台停止绘制，卸载释放 GPU 资源。预览失败仍可下载优化后的 GLB。打开首页不会调用供应商生成任务。

## 验证

组件测试覆盖首页导航与复制、自动旋转与动态偏好、模型信息显示、加载失败及卸载清理。执行：

```sh
cd frontend
node node_modules/vue-tsc/bin/vue-tsc.js -b
node node_modules/vitest/vitest.mjs run src/components/sup3api/__tests__/LandingMesh.spec.ts src/components/sup3api/__tests__/CharacterDemo.spec.ts src/views/__tests__/Sup3APIHomeView.spec.ts
```

本次保留既有 Dashboard/Usage、客户菜单、API 密钥、分组及供应商账号逻辑，不修改后端 API。默认 CSP 与部署示例增加 Meshopt 所需的 `wasm-unsafe-eval`，以及内嵌贴图所需的 `connect-src blob:`；JavaScript `unsafe-eval` 仍不允许。已有自定义 CSP 的部署需同步这两项。

## 传输与缓存

角色 188 KiB、场景 138 KiB，分别较供应商文件减少 65% / 73%。每个模型不足一万三角形，色彩图为 1024px。资源经 Vite 内容哈希输出，复用现有一年不可变缓存；只加载选中且进入视口的三维 Demo，不预加载全部模型。原画共约几十 KiB，Three.js 为懒加载独立共享包。大流量部署可让 CDN 缓存 `/assets/`；缓存头不等于已经配置了 CDN。

自定义前端删除分组脚注、拖动提示、重复格式状态、文件编码说明及文档重复品牌标题。保留表单校验、协议约束、费用行为和错误反馈。
