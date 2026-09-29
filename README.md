<div align="center">
  <img src="frontend/public/sup3api-mark.svg" width="72" alt="Sup3API" />
  <h1>Sup3API</h1>
  <p>一个入口，连接多模态 AI。Text · Image · 3D APIs.</p>
</div>

Sup3API 是独立部署的多模态 AI API 站点，也是 AGraphs 生态中的 API 服务项目。它面向不同应用开放，连接 GPT、Claude、Tripo 与 Meshy；站点名称、Logo、客户门户和文档均使用 Sup3API 品牌。客户门户、API 接入页、站内文档采用独立的极简界面；管理员后台与 LLM 网关基于 [Sub2API](https://github.com/Wei-Shaw/sub2api)。

## 能力与边界

| 能力 | API | 输入与返回 |
| --- | --- | --- |
| GPT 文本、视觉与工具 | `/v1/responses`、`/v1/chat/completions` | 文本 / 图片内容块 → 完整协议 JSON 或 SSE |
| Claude 文本、视觉与工具 | `/v1/messages` | Messages、图片 source → content、工具调用、usage 或 SSE |
| 图像生成、编辑 | `/v1/images/generations`、`/v1/images/edits` | 文本 / multipart 图片 → data[] 图像与用量 |
| Tripo / Meshy 三维资产 | `/v1/assets/jobs` | 文字 / 图像 / 模型 URL / job_id → 异步任务与资产清单 |

模型、模态、工具与流式支持受上游路线、账号权限与分组配置约束。查询 `/v1/models` 与 `/v1/assets/capabilities` 确认可用资源。**不将全部供应商端点视为已兼容，也不宣称 GPT/Claude 可直接理解任意 GLB。**

## 客户门户

- `/home`：Sup3API 品牌首页，顶部提供 API 接入与文档。
- `/connect`：协议 / 模型 / 输入配置，生成 cURL、JavaScript、Python；同源模型查询与 3D 估价。
- `/docs`：快速开始、认证、GPT、Claude、Images、3D、原生字段映射、完整返回、重试、费用与错误说明。
- `/dashboard`、`/keys`、`/usage`：客户控制台、密钥管理和调用记录。
- `/admin/*`：保留成熟的 Sub2API 管理界面与运营能力。

图标统一使用 Sup3API 蓝色立方体标记。接入页的诊断 Key 仅存于当前页面内存，不进入示例代码、URL 或浏览器存储。

### 从网页开始接入

1. 管理员配置上游账号、模型与密钥分组；使用三维能力时，另外开启资产模块并配置供应商 API Key 和用户允许名单。
2. 客户登录后，在 `/keys` 创建 Sup3API 网关密钥。应用使用此密钥调用本站 API，供应商密钥由服务端管理。
3. 打开 `/connect`，选择 GPT Responses、Chat、Claude、Images、Tripo 或 Meshy，配置模型与输入。可复制 cURL、JavaScript、Python 或 JSON 示例。
4. 输入网关密钥，查询实际可用的模型或资产能力；三维请求可先「校验与估价」。网页诊断不会创建付费生成任务。
5. 在应用服务端执行生成请求；三维任务继续轮询状态并下载资产。字段说明见 `/docs`，机器可读规范见 `/docs/assets.openapi.json`。

代码示例支持复制；API 接入页与文档页使用高对比度蓝色文本选区，深色代码块选中后仍可阅读。

### API 层与应用层

API 层负责认证、协议与字段适配、模型路由、任务状态、用量记录和资产交付。应用通过 HTTP API 使用这些能力，负责编辑器、场景、项目和业务工作流。客户门户是 API 的使用入口，应用无需依赖门户页面或管理员界面。

## 如何测试站点

启动后打开 `/home`、`/connect` 和 `/docs`。先运行只读站点检查，再使用自己的网关密钥测试模型查询与三维估价：

```sh
python tools/smoke-sup3api.py --base-url http://127.0.0.1:8080
# Set SUP3API_API_KEY in the current shell before authenticated checks.
python tools/smoke-sup3api.py --base-url http://127.0.0.1:8080 --assets
python tools/smoke-sup3api.py --base-url http://127.0.0.1:8080 --models
```

脚本不会调用生成接口。页面交互、登录、真实生成验收和常见错误详见 [站点测试指南](docs/TESTING.md)。开发机已有本地服务时，以实际端口为准（当前预览为 `18763`）；新 Docker 部署默认 `8080`。

品牌文件：[独立图标](frontend/public/sup3api-mark.svg) · [横版 Logo](frontend/public/sup3api-wordmark.svg)。

## 三维 API 示例

```sh
export SUP3API_BASE_URL="https://YOUR_SUP3API_HOST"
export SUP3API_API_KEY="YOUR_GATEWAY_KEY"

curl "$SUP3API_BASE_URL/v1/assets/jobs" \
  -H "Authorization: Bearer $SUP3API_API_KEY" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: asset-example-001" \
  -d '{
    "provider":"meshy",
    "operation":"text_to_3d",
    "model":"meshy-7.1",
    "inputs":{"prompt":"A stylized wooden treasure chest"},
    "parameters":{"texture":true,"pbr":true,"formats":["glb","fbx"]}
  }'
```

先调用 `/v1/assets/quotes` 校验与估价，不创建生成任务。创建返回任务 ID，随后轮询 `GET /v1/assets/jobs/{id}`。`status=succeeded` 且 `delivery_status=ready` 后，使用同一 Key 下载所有 `artifacts[]`。

返回包含模型、PBR 材质、贴图、几何、骨骼、权重、动画（以实际模型为准）。`steps[].provider_result` 保留各阶段完整上游任务结果；原生 CDN 链接可能过期，使用认证 artifact 下载获得耐久文件。缺失组件显式标记 absent/unsupported。

也支持 `input_format=tripo/meshy` + `payload` 的原生字段封套，**仅限已校验的字段子集**。统一与原生字段不能混用。Meshy 原生 preview 不会自动付费精修。任务引用始终检查账号与 API Key 所有权。

图像：Tripo 使用公开 HTTPS URL；Meshy 还支持 PNG/JPEG Base64 data URI（每张解码后 ≤10 MiB，JSON 总体 ≤16 MiB）。模型使用公开 HTTPS `model_url` 或同供应商任务 ID；通用二进制上传尚未实现。

## 部署

依赖 PostgreSQL、Redis、Go 与 Node/pnpm。沿用 [上游部署说明](https://github.com/Wei-Shaw/sub2api/blob/main/README_CN.md)，从**本仓库源码构建**。3D 模块默认关闭：

```dotenv
SUP3_ENABLED=true
SUP3_ALLOWED_USER_IDS=1
SUP3_TRIPO_API_KEY=your-tripo-api-key
SUP3_MESHY_API_KEY=your-meshy-api-key
SUP3_DATA_DIR=/app/data/sup3-assets
```

```sh
docker build -t sup3api:local .
# Use the built image in your compose configuration.
# deploy/sup3.compose.yml contains the additive asset module environment.
```

供应商密钥只存服务端。3D 消耗运营者的原生 credits，与 LLM USD 钱包独立；当前按用户允许名单开放，不是已经完成的公开多租户 3D 计费系统。Tripo Studio 会员额度不默认等于 API 余额。

## 开发与验证

```sh
python tools/sync-sup3api-docs.py --check
cd frontend
corepack pnpm@9.15.9 install --frozen-lockfile
node node_modules/vue-tsc/bin/vue-tsc.js -b
node node_modules/vitest/vitest.mjs run src/content/__tests__/sup3api.spec.ts
node node_modules/vite/bin/vite.js build
cd ../backend
go test ./internal/sup3
go build -tags embed -o sup3api ./cmd/server
```

前端构建到 `backend/internal/web/dist`，`embed` 构建包含客户网站。更新时重新构建此分支，不使用会覆盖自定义模块的上游二进制更新器。

维护网页文档时，修改 `frontend/src/content/sup3api.ts`；首页源文件为 `frontend/src/content/sup3api-home.html`。修改首页或 `docs/sup3/openapi.json` 后，在仓库根目录执行 `python tools/sync-sup3api-docs.py`，同步公开 HTML 与 Schema，再构建前端。

## 文档与来源

- [客户门户实现与接口边界](docs/SUP3API.md)
- [3D OpenAPI Schema](docs/sup3/openapi.json)
- [资产模块部署、架构与验证](README.SUP3.md)
- [OpenAI 官方文档](https://developers.openai.com/api/docs/guides/images-vision)
- [Claude 官方文档](https://platform.claude.com/docs/en/api/messages/create)
- [Tripo V3 文档](https://developers.tripo3d.ai/en/docs/generation-image-to-model/standard)
- [Meshy API 文档](https://docs.meshy.ai/api/image-to-3d)

## 开源与归属

基于 Sub2API，沿用 [LGPL-3.0](LICENSE)，保留上游版权、许可与 [原始 README](docs/upstream/README.md)。Go module 路径保留上游名称以减少合并冲突。Sup3API 的独立客户界面不改变底层开源许可。原始上游 README 中的相对链接以其上游仓库为基准。
