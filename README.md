<img src="frontend/public/sup3api-mark.svg" width="64" alt="Sup3API" />

# Sup3API

文本、图像和 3D 模型 API 网关，接入 GPT、Claude、Tripo 和 Meshy。

## 接入

GPT / Claude 使用兼容协议。3D 接入三选一：

| 方式 | 入口 |
| --- | --- |
| 统一 API | `/v1/assets/jobs` |
| Tripo 原生 API | `/providers/tripo/v3` |
| Meshy 原生 API | `/providers/meshy` |

管理员在「账号管理」配置供应商 API Key、同步模型并绑定分组；应用使用同组的 Sup3API 密钥。支持模型白名单和别名映射。支持范围见 [兼容说明](docs/COMPATIBILITY.md)。

## 启动

需要 Docker Compose。首次部署将 `deploy/.env.example` 复制为 `deploy/.env`，填写数据库密码、管理员账号及密钥配置；3D 另需 `SUP3_ALLOWED_USER_IDS`，并在「账号管理」添加 Tripo/Meshy 上游账号。完整步骤见 [部署指南](docs/TESTING.md#4-从源码启动自己的测试站)。

```sh
docker build -t sup3api:local .
docker compose --env-file deploy/.env -f deploy/docker-compose.local.yml -f deploy/sup3.compose.yml up -d
```

默认访问 `http://localhost:8080`：`/connect` 生成调用示例，`/docs` 查阅 API 文档，`/admin` 管理上游。

## 文档

- [部署与测试](docs/TESTING.md)
- [API 兼容说明](docs/COMPATIBILITY.md)
- [3D OpenAPI](docs/sup3/openapi.json)
- [开发与架构](docs/SUP3API.md)

基于 [Sub2API](https://github.com/Wei-Shaw/sub2api)，使用 [LGPL-3.0](LICENSE)。保留[上游说明](docs/upstream/README.md)。
