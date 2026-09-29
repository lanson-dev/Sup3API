# Sup3API 站点测试指南

Sup3API 是可独立部署的站点，包含首页、API 接入、文档、客户控制台和管理员后台。应用通过 HTTP API 调用，无需依赖网页实现。

## 1. 打开站点

当前开发机已启动的完整服务：`http://127.0.0.1:18763/home`。这个地址只在运行服务的电脑上可用；尚未部署公共域名。新 Docker 部署默认端口是 `8080`。

| 地址 | 验收内容 |
| --- | --- |
| `/home` | 三维形态旋转、暂停、重置及线框切换；能力与代码标签键盘切换、复制反馈、手机菜单、FAQ；[首页验收记录](HOMEPAGE.md) |
| `/connect` | 切换六种协议，确认端点、模型、输入框与生成代码同步变化 |
| `/docs` | 搜索、目录跳转、代码复制、选中文本可读；Schema 下载 |
| `/login`、`/keys` | 使用已有账号登录，创建测试网关密钥，确认密钥默认隐藏 |
| `/dashboard`、`/usage` | 显示当前账号的实际数据；当前用量视图沿用 LLM 账本，不是统一 3D 账单 |
| `/admin/dashboard` | 管理员配置上游账号、模型分组和站点参数 |

手机宽度（约 390px）检查导航可点击、表单能填写、代码区域内部滚动。`/docs/errors` 中的错误 JSON 是文档示例，不表示当前调用失败。

## 2. 无生成费用的接口检查

仓库根目录运行，Python 3 标准库即可：

```powershell
py -3 tools/smoke-sup3api.py --base-url http://127.0.0.1:18763
```

检查服务健康、页面入口、Logo 与 OpenAPI Schema。它验证服务和资源加载；页面点击和渲染效果仍按上表人工检查。

从 `/keys` 获取 **Sup3API 网关密钥**，不是 Tripo/Meshy 供应商密钥。PowerShell 7 可以隐藏输入：

```powershell
$env:SUP3API_API_KEY = Read-Host 'Sup3API gateway key' -MaskInput
py -3 tools/smoke-sup3api.py --base-url http://127.0.0.1:18763 --assets
py -3 tools/smoke-sup3api.py --base-url http://127.0.0.1:18763 --models
Remove-Item Env:SUP3API_API_KEY
```

`--assets` 检查未认证访问被拒绝、当前 Key 的资产能力、已配置供应商的文字生成估价、无效原生字段返回 400。`--models` 查询当前 Key 可用模型。这些操作不会提交推理或生成任务，也不能证明供应商真实余额充足。没有开通 LLM 时可只测试 `--assets`，反之亦然。

也可以直接在 `/connect` 输入网关密钥，点击「查询可用模型」「查询资产能力」「校验与估价」。密钥仅存页面内存；示例代码读取 `SUP3API_API_KEY` 环境变量。

## 3. 手工生成验收（会产生实际费用）

先确认报价和供应商余额，再由你执行 `/connect` 生成的代码。站点诊断按钮不代替这一步。

1. **文本**：选择已开通的模型，发送简短提示词。检查完整 JSON、usage；需要 SSE 时按文档启用 stream 并使用流式客户端读取。
2. **图像**：选择实际可用的图像模型生成一张图片。验证返回的 URL/Base64；编辑接口按文档发送 multipart。
3. **3D**：先用 Tripo 或 Meshy 的 `text_to_3d` 估价，再调用 `POST /v1/assets/jobs`。每个新任务使用独立的 `Idempotency-Key`；同一次提交的网络重试复用该值。
4. 记录响应 `id`，每 3–5 秒查询 `GET /v1/assets/jobs/{id}`。只有 `status=succeeded` 且 `delivery_status=ready` 才开始下载。
5. 使用创建任务的同一 Key 下载 `artifacts[]`，验证大小和 SHA-256，并在 Blender/Godot 中打开模型。查看 `components` 与 `steps[].provider_result`；没有骨骼的模型不应被当作已绑定。

三维计费使用供应商原生 credits，与 LLM 的 USD 余额独立。Tripo Studio 订阅不自动等于 API 余额。文件交付失败时先排查交付状态，不重新创建付费任务。

## 4. 从源码启动自己的测试站

需要 Docker 和 Docker Compose。在仓库根目录构建本项目镜像：

```sh
docker build -t sup3api:local .
```

首次部署把 `deploy/.env.example` 复制为 `deploy/.env`，已有文件则直接编辑，不覆盖旧值。设置 `POSTGRES_PASSWORD`、`ADMIN_EMAIL`、`ADMIN_PASSWORD`、`JWT_SECRET` 和 `TOTP_ENCRYPTION_KEY`（密钥格式参考模板）。本机测试将 `BIND_HOST=127.0.0.1`，设置空闲的 `SERVER_PORT`，默认 `8080`。

启用三维模块时，额外在该 `.env` 文件中填写：

```dotenv
SUP3_ALLOWED_USER_IDS=1
```

允许名单改成你实际开通的用户 ID；`1` 只是示例。启动后在「账号管理 → 创建账号 → Tripo / Meshy」填写上游 API Key，至少添加一个供应商账号并绑定对应供应商分组（或综合分组）。应用调用使用 `/keys` 创建并绑定同一分组的 Sup3API 密钥。在 `deploy` 目录执行：

```sh
docker compose -f docker-compose.local.yml -f sup3.compose.yml up -d
docker compose -f docker-compose.local.yml -f sup3.compose.yml logs --tail 80 sub2api
```

然后访问 `http://127.0.0.1:8080/home`（按你设置的端口调整），用 `.env` 中的管理员账号登录，配置上游和模型分组。Compose 服务名 `sub2api` 沿用上游，运行的镜像必须是刚构建的 `sup3api:local`。不要使用上游二进制自动更新覆盖自定义站点。部署方式参考 [资产模块说明](../README.SUP3.md)。

## 5. 开发检查

```sh
python tools/sync-sup3api-docs.py --check
cd frontend
corepack pnpm@9.15.9 install --frozen-lockfile
node node_modules/vue-tsc/bin/vue-tsc.js -b
node node_modules/vitest/vitest.mjs run src/content/__tests__/sup3api.spec.ts src/views/sup3api/__tests__/portal.spec.ts src/router/__tests__/title.spec.ts src/views/__tests__/Sup3APIHomeView.spec.ts src/components/sup3api/__tests__/LandingMesh.spec.ts
node node_modules/vite/bin/vite.js build
cd ../backend
go test ./internal/sup3
go build -tags embed -o sup3api ./cmd/server
```

Windows 可把 `python` 换成 `py -3`。本机已有工具链还可使用 `.local/go/bin/go.exe`；这些本地依赖与凭据不随 Git 提交。

## 常见结果

| 结果 | 检查方向 |
| --- | --- |
| 连接失败 | 服务进程、端口、Docker 日志；本机预览与 Docker 默认端口不同 |
| 401 / 403 | 网关 Key、密钥状态、分组、3D 用户允许名单；`INSUFFICIENT_BALANCE` 表示平台账户余额不足，模型列表查询也可能被阻止 |
| 404 / 503 或返回 HTML | 是否运行本项目 embed 构建、资产模块是否开启、API 路径是否正确 |
| providers[].available=false | 对应供应商服务端 API Key 未配置 |
| 模型列表为空 | 当前 Key 的分组、上游账号与模型配置；不代表网页损坏 |
| 报价成功但生成失败 | 上游余额、账号权限、服务故障；报价不进行真实生成 |
| 仍看到旧站名或旧图标 | 确认进程使用新构建，刷新浏览器；构建前先同步首页与 Schema |

## 兼容性验收

在 `/connect` 选三维能力 → Meshy 原生 API → Refine，填写此 Key 在原生入口创建的 preview ID；代码应直接调用 `/providers/meshy/openapi/v2/text-to-3d`，不包含统一封套。切换 Tripo 后操作与 URL 应同步变化。编辑 JSON 或切换选项不会提交生成。

统一模式可选择多个输出格式和必需组件。Meshy 选择 FBX 后切换 Tripo，保留原要求并显示不支持；点击「改用通用 GLB 格式」恢复。通过报价验证扩展参数与模型。组件缺失检查在交付后完成，可能已产生供应商费用。

`--assets` 另检查新输出契约报价、原生未认证/越权/不支持端点；不会 POST 原生生成。后端 `SUP3_TEST_DSN` 指向可创建临时 schema 的 PostgreSQL，运行 `go test ./internal/sup3` 可执行真实持久化 + 模拟上游测试。未设置时数据库测试跳过，不能据此声称幂等持久化已验证。

接入方式下拉仅有三个互斥选项：统一 API、Tripo 原生 API、Meshy 原生 API。统一模式的组件要求和扩展 JSON 收入默认折叠的「高级选项」，输出格式仍可直接选择。
