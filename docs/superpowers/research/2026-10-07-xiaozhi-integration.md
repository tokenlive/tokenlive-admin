# 小智服务端接入 TokenLive：边界与可行性

核查日期：2026-10-07。只读研究，未部署、未修改业务代码，也未验证端到端兼容性。沿用本仓库 `docs/superpowers/research` 研究笔记惯例。

## 结论（架构建议，不是已实现能力）

如果要扩展设备语音对话业务，适合做**生态集成**：独立运行小智语音服务，LLM 请求经过 Gateway，Admin 统一治理配置。如果项目仍只定位为通用模型网关后台，则不宜直接吞并小智整套产品。

不建议把音频长连接、VAD/ASR/TTS 运行时塞入 Admin 的 Go 进程，也不建议一开始重写小智 Python/Java 全栈。最小验证可以只接 LLM，不引入其 Java 管理端；多用户、多设备配置完善后再考虑管理 API 适配。依据为下列事实 F1–F5 和本地边界 L1–L3。

## 上游事实

| 编号 | 2026-10-07 核查事实 | 第一方来源 |
| --- | --- | --- |
| F1 | 本报告假设“小智服务端”指 `xinnan-tech/xiaozhi-esp32-server`。设备端项目将其列为**第三方 Python 服务端**，同时列有 Java、两种 Go 实现，不能称其为唯一官方服务端。 | [设备端 README：相关开源项目](https://github.com/78/xiaozhi-esp32/blob/main/README_zh.md#相关开源项目) |
| F2 | 该实现使用 Python、Java、Vue；支持仅运行 server（配置文件、无需数据库）及全模块部署（多用户、多智能体、管理界面）。全模块文档要求 Java 管理端、MySQL、Redis，并分别启动 Python 语音服务及 manager-web。 | [README](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/README.md)、[全模块部署](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/docs/Deployment_all.md) |
| F3 | 上游提供 VAD、ASR、LLM、TTS、记忆、工具/MCP 等组件；设备协议是 JSON 控制消息加 Opus 二进制音频，并有 hello/listen/abort 等时序。不是仅有一个 Chat Completions HTTP 接口。 | [README 功能清单](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/README.md#功能清单-)、[设备 WebSocket 协议](https://github.com/78/xiaozhi-esp32/blob/main/docs/websocket_zh.md) |
| F4 | Python OpenAI provider 接收 `model_name`、`api_key`、`base_url`（回退 `url`），调用 `client.chat.completions.create`，使用流式输出并有 `tools` 路径。存在低改动接入 OpenAI 兼容网关的接口基础，但不等于端到端已兼容。 | [OpenAI provider 源码](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/core/providers/llm/openai/openai.py) |
| F5 | Python 通过可配置 `manager-api.url` 和 `secret` 访问管理端，使用服务间 Bearer 凭证。核心调用包括 `/config/server-base`、`/config/agent-models`，还涉及替换词、聊天上报/总结/标题、通讯录等；返回体要求 `code/data/msg`，设备不存在/未绑定有专用错误码。 | [ManageApiClient](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/config/manage_api_client.py)、[配置模板](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/config_from_api.yaml)、[ConfigController](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/manager-api/src/main/java/xiaozhi/modules/config/controller/ConfigController.java) |
| F6 | 示例 `config.yaml` 的 `server.auth.enabled` 为 `false`；WS 实现支持按配置启用鉴权、白名单免验 token。设备 token 使用 HMAC-SHA256，绑定 client-id/device-id 及时间戳。服务间 secret、设备 token 与 Gateway API Key 是不同凭证。 | [默认配置](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/config.yaml)、[WebSocketServer](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/core/websocket_server.py)、[AuthManager](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/xiaozhi-server/core/auth.py) |
| F7 | 设备和智能体实体均有 `userId`，设备还有 `agentId`、`macAddress`。这两个实体未声明 TokenLive 风格的 tenant/workspace 字段；不能直接认定多用户管理等价于 TokenLive 租户隔离。 | [AgentEntity](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/manager-api/src/main/java/xiaozhi/modules/agent/entity/AgentEntity.java)、[DeviceEntity](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/main/manager-api/src/main/java/xiaozhi/modules/device/entity/DeviceEntity.java) |
| F8 | 上游 README 明确警告功能未完善、未通过网络安全测评，并要求不要用于生产环境。仓库 LICENSE 标为 MIT；此处不判断依赖、模型、音色或服务条款的法律适用性。 | [README 警告](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/README.md#警告-️)、[LICENSE](https://github.com/xinnan-tech/xiaozhi-esp32-server/blob/main/LICENSE) |

上述 server 资料读取自 `main`；核查时 GitHub commits API 返回 HEAD `7a260ee14386ab4883666487f904b5621de03856`，提交时间为 `2026-10-07T00:03:25Z`。链接使用分支，后续内容可能变化；实施前应固定并复核版本。

## 本地事实

- L1：Admin 明确聚焦资源、租户授权、API Key、RBAC、观测、策略配置；实际请求执行在 Gateway。[README-zh.md](../../../README-zh.md)
- L2：当前模块注册为 RBAC、Resource、Space、Policy、Dashboard、Ops、SystemVersion，未注册设备/语音模块。[mods.go](../../../internal/mods/mods.go)
- L3：Gateway 注册 Chat Completions 等路由；检查的 LLM 路由表没有 ASR/TTS 音频接口。现有 WS engine 明确实现 Responses 双向流，不能据此认定支持小智 Opus/控制协议。[LLM 路由](/Users/chenzhiguo/Projects/tokenlive-gateway/internal/router/llm.go:51)、[WS engine](/Users/chenzhiguo/Projects/tokenlive-gateway/pkg/core/engine_websocket.go:34)
- L4：企业内部部署可由 Admin 承载终端消费者；公共平台模式的消费者归 Portal/Workspace，Admin 面向运营团队。[双模式用户体系 ADR](../../adr/0001-separate-admin-portal-user-systems.md)

## 推荐分工与实施门槛

1. **先做隔离 PoC。** 设备连接独立小智运行时；运行时负责音频、打断、MCP、ASR/TTS；LLM 配置指向 Gateway 的兼容入口。ASR/TTS 首期保留上游 provider，不宣称现有网关已经覆盖语音成本与全部音频协议。（依据 F2–F4、L1–L3）
2. **再整合控制台。** 需要时新增设备绑定、智能体/语音配置、模型授权及会话视图。可以先保留上游管理 API，通过适配层连接；如要用 Go 替代其 manager-api，需实现 F5 的完整已用契约及 OTA/绑定/鉴权流程，不是只改一个 URL，也不宜让双方直接共享并修改业务表。
3. **明确身份来源。** 由可信后端映射 `device → user/tenant/workspace → 受限 Gateway Key`；不要让终端自报 tenant 成为授权依据，不要全租户共享超级 Key。消费者页面依据 L4 决定放 Admin 还是 Portal。设备 token、运行时管理 secret、模型凭证独立管理。（依据 F6、F7、L4）
4. **明确投产阻力。** F8 是显式上游风险提示。公网试验前至少关闭不必要入口、启用设备鉴权、审核白名单、隔离管理接口；生产化还需安全审查、租户越权测试、限流/并发压测、密钥保护、录音与转写的访问和保留策略。
5. **不能只验“能回答”。** 验证 SSE 首字延迟、工具调用、语音打断/取消、断线重连、Gateway 故障回退、越权与额度，并建立会话关联与分项成本。OpenAI provider 的禁思考参数按 `base_url` 域名注入，换成网关域名后原自动逻辑可能不触发；其 `session_id` 参数也未直接作为请求关联 header 发出，需单独设计观测链路。（依据 F4 源码）

## 核查方式与未解决项

Web 搜索与打开工具没有返回可读结果；本笔记使用获准的只读 HTTPS 请求读取上述第一方 README、协议和源码，不以搜索摘要作证。尚未确定用户指的是哪一版服务端、计划面向内部还是公共用户、预计并发以及是否要求统一语音计费；这些选择影响后续方案，不影响“运行时独立、治理层集成”的初步建议。
