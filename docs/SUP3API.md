# Sup3API 客户门户与 API 契约

客户页面使用 Sup3API；`/admin/*` 保留 Sub2API 管理界面。API 层负责认证、协议适配、模型路由、异步任务、用量与资产交付。游戏编辑器、场景和应用业务不进入网关。

## 页面与维护入口

| 页面 | 实现 |
| --- | --- |
| `/home` | `frontend/src/content/sup3api-home.html`，由 `Sup3APIHomeView.vue` 加载并同步独立 HTML |
| `/connect` | `frontend/src/views/sup3api/ConnectView.vue` |
| `/docs/:section?` | `DocsView.vue`；内容与示例在 `frontend/src/content/sup3api.ts` |
| `/dashboard`、`/keys`、`/usage` | `ConsoleView.vue`；使用现有用户 API，独立客户界面 |
| 登录、注册、找回密码等 | 复用认证逻辑，使用 Sup3API `AuthLayout` |
| 其他用户页面 | Sup3API `AppLayout` 分支；管理员路由沿用原布局 |
| `/docs/assets.openapi.json` | 从 `docs/sup3/openapi.json` 同步的可下载 3D Schema |

全站图标使用 `frontend/public/sup3api-mark.svg` 的蓝色立方体标记。客户界面的站名与图标独立于管理员配置。原始上游图标保存在 `assets/upstream/sub2api-logo.svg`。

## 输入协议

- GPT Responses：`POST /v1/responses`，`input_text` / `input_image` 内容块。
- Chat Completions：`POST /v1/chat/completions`，`messages` 与 `image_url.url`。
- Claude：`POST /v1/messages`，`content`、`image.source`、`max_tokens`。
- GPT Images：`POST /v1/images/generations` 与 `/v1/images/edits`；编辑使用 multipart。
- 3D：`POST /v1/assets/jobs` 与不生成模型的 `/v1/assets/quotes`。

LLM/图像采用现有网关路由，实际支持取决于密钥分组、供应商账号类型与模型。没有因为增加前端而宣称所有官方端点都已代理。

统一 3D 请求保持 `provider / operation / model / inputs / parameters / provider_options`。新增供应商字段封套：

```json
{
  "provider": "meshy",
  "operation": "image_to_3d",
  "input_format": "meshy",
  "payload": {
    "ai_model": "meshy-7.1",
    "image_url": "https://example.com/reference.png",
    "should_texture": true,
    "enable_pbr": true,
    "target_formats": ["glb", "fbx", "obj"]
  }
}
```

这不是透明的原生端点代理。封套在 HTTP 边界规范化后，复用报价、校验、幂等性和所有权检查。未映射字段直接拒绝。Meshy 原生 `mode=preview` 只生成预览，不自动付费精修；统一请求的 `texture=true` 才运行 preview + refine。跨账号的供应商 task ID/file token 不接受；任务引用必须为当前 Key 拥有的 Sup3API job ID。

Meshy 图像新增 PNG/JPEG Base64 data URI 输入，解码后每张最多 10 MiB，最大边长 16384，JSON 总体最多 16 MiB。Tripo 仍要求公开 HTTPS 图像链接。3D 文件使用 `model_url`；没有通用二进制模型上传接口。格式与尺寸还受上游限制。

## 返回契约

GPT/Claude 保留各协议完整的结构化输出、工具调用、用量与流事件。客户端不能只读取一个文本字段。

3D 保留所有已发现的文件，在 `artifacts[]` 暴露带认证、大小、SHA-256 的下载；GLB 解析产生几何、材质、贴图、骨骼、权重、动画组件。组件可能为 absent/unsupported，不伪造缺失结果。

新增 `steps[].provider_result` 保存每阶段供应商完整任务查询 JSON，保留模型特有字段与原始输出。它可能含有会过期的供应商 CDN URL；耐久交付仍使用网关 artifact URL。既有终态任务需 `refresh-artifacts` 才会补充最新阶段的原始返回，历史 preview 原始返回不能凭空补回。

Meshy `parameters.formats` 可选 glb/fbx/obj/stl/usdz/3mf；默认 glb/fbx。无 GLB 时便携组件提取不可用。Tripo 输出取决于操作与模型。

## 测试与构建

```sh
python tools/sync-sup3api-docs.py
cd frontend
corepack pnpm@9.15.9 install --frozen-lockfile
node node_modules/vue-tsc/bin/vue-tsc.js -b
node node_modules/vitest/vitest.mjs run src/content/__tests__/sup3api.spec.ts
node node_modules/vite/bin/vite.js build
cd ../backend
go test ./internal/sup3
go build -tags embed -o sup3api ./cmd/server
```

`tools/sync-sup3api-docs.py --check` 检查站内下载与规范是否一致。完整前端构建由上游 Dockerfile 执行；避免使用上游二进制更新器覆盖此分支。

网页连接诊断仅调用本站同源 GET 或报价 POST；不会自动创建付费任务。API Key 不写入浏览器存储、URL 或代码示例。接入页生成的代码在用户自己的服务端执行。

官方资料链接、兼容范围与费用边界均列于站内 `/docs/sources`、`/docs/native` 和 `/docs/lifecycle`。

统一请求可省略 `input_format` 或填写 `sup3api`；早期 `agraphs` 值仍作为兼容别名接受。对外能力发现返回 `sup3api`。客户端示例环境变量使用 `SUP3API_BASE_URL` / `SUP3API_API_KEY`；服务端已有 `SUP3_*` 配置名保持不变。

[站点测试指南](TESTING.md) 包含浏览器验收、无生成费用的检查脚本和手工生成流程。
