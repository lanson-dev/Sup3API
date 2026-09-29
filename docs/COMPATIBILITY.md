# Sup3API 兼容契约

统一 API、原生 API 与应用业务的边界如下：

| 入口 | 输入与返回 | 适用场景 |
| --- | --- | --- |
| `/v1/assets/jobs` | 统一 Request → Job、artifacts、components | 新应用跨供应商切换 |
| `/providers/meshy` | Meshy JSON → Meshy JSON/SSE | 迁移已有官方工作流 |
| `/providers/tripo/v3` | Tripo V3 JSON → Tripo JSON | 迁移已有 V3 工作流 |
| `/v1/assets/jobs` + `input_format` / `payload` | 已支持字段映射 → 统一 Job | 保留旧版接入兼容 |

API 层负责认证、任务归属、协议、幂等和交付。应用负责编辑器、场景、内容决策及流程编排。Sup3API 不承诺不同模型生成同样的视觉结果。

## 统一请求

```json
{
  "provider": "meshy",
  "operation": "text_to_3d",
  "inputs": {"prompt": "A wooden crate"},
  "output": {
    "formats": ["glb"],
    "required_components": ["geometry", "materials", "textures"]
  }
}
```

此共同参数可改为 `provider=tripo`。不填写模型时使用各自默认版本。扩展放在 `extensions.meshy` 或 `extensions.tripo`；只接受选中供应商的命名空间和经过验证的字段。旧 `provider_options` 继续可用，同名冲突拒绝。

`GET /v1/assets/capabilities` 的 `operation_details` 提供模型、直接输出格式及 `extension_fields` 类型/枚举。`POST /v1/assets/quotes` 验证具体组合，无生成费用。`output.formats` 与旧 `parameters.formats` 冲突时报错。

Tripo 普通生成输出 GLB，quad 输出 FBX；Meshy 生成/重纹理支持 GLB/FBX/OBJ/STL/USDZ/3MF。Meshy 绑定/动画暂不能指定输出格式。组件验证依赖可解析 GLB；骨骼/权重要求 rig 或 animate，动画要求 animate。

交付后不满足要求：`delivery_status=failed`、`error.code=output_requirements_unmet`，`output_validation.missing` 给出 `format:...` 或 `component:...`。文件保留。供应商可能已经收费；重试交付不生成新模型，也不跳过输出检查。没有自动转换；需要其他格式可显式调用原生 convert。

## 原生路由

Meshy 将原根地址改为 `https://YOUR_HOST/providers/meshy`，保留 `/openapi/...`。Tripo 将 V3 根地址改为 `https://YOUR_HOST/providers/tripo/v3`，不要重复添加版本。Authorization Bearer 使用网关 Key。请求体不加封套。

Meshy 创建：`POST /openapi/v2/text-to-3d`，及 `/openapi/v1/` 下的 `image-to-3d`、`multi-image-to-3d`、`retexture`、`rigging`、`animations`、`convert`。
每个创建路径支持 `GET /{task_id}`、`DELETE /{task_id}`、`GET /{task_id}/stream`。DELETE 和流内容的业务语义由上游决定。

Tripo 创建：`POST /generation/text-to-model`、`/generation/image-to-model`、`/generation/multiview-to-model`、`/models/texture`、`/animations/rig`、`/animations/retarget`、`/models/convert`；查询 `GET /tasks/{task_id}`。

原生字段、默认值和响应体保留，不经过统一适配。JSON 会规范化，重复键按最后值解析后再检查和转发。Meshy preview 与 refine 独立，不自动追加收费步骤。Tripo convert 的 GLTF 不能视为 GLB。

建议创建时提供 8–128 字符 `Idempotency-Key`。同一作用域中，相同路径与规范化请求重复调用，返回已存 HTTP 状态和响应体；不保证重放上游动态响应头。参数不同返回 409。提交仍在进行或结果不确定时返回 `submission_unknown`，由运营者凭 `X-Sup3-Request-ID` 核对，不应换新 ID 重试。未提供幂等 Key 的每次 POST 是新调用。

## 所有权、存储和限制

任务与前置任务引用绑定账号 + 网关 Key + 供应商凭据指纹。更换网关或上游 Key 后，旧任务不能直接访问，需运营者迁移。不能引用供应商账号中其他用户或历史任务；统一 `job_` 与原生任务 ID 不可混用。

原生入口不支持任务列表、上传/file_token、webhook/callback、账号余额、历史导入、Tripo V2、未列出端点和查询参数。图片/模型优先使用公开 HTTPS；Meshy 原生 data URI 仍受上游限制。请求/非流响应最大 16 MiB。

原生任务不在统一 Job 列表，不进入持久资产存储或统一交付校验，直接使用上游下载 URL。所有 3D 请求由运营者供应商 credits 支付，使用资产用户允许名单，当前不扣 LLM 钱包、不形成统一账单。Tripo Studio 会员不自动等于 API 额度。

上游状态/JSON 错误原样返回；网关错误使用 `error.code`，原生入口错误标记 `X-Sup3-Error-Origin: gateway`，认证层采用站点认证错误结构。

## 验证范围

模拟上游覆盖保留新字段、preview/refine、SSE、DELETE 错误、任务隔离及持久幂等。PostgreSQL 集成测试使用临时 schema。真实站点仅测试免费报价/能力及拒绝路径；尚未进行付费生成和全部官方 SDK 的端到端认证。测试步骤见 [TESTING.md](TESTING.md)。
