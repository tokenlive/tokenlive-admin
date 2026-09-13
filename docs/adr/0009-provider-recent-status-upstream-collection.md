# 供应商列表最近使用状态采用网关上游调用切片采集

Status: accepted

## Context

在 AI 网关管理端（`tokenlive-admin`）中，模型列表页与端点列表页均提供了「最近状态」列（通过 10 个 10 分钟切片方块展示过去 100 分钟的成功与失败情况）。运营与运维人员在「供应商管理」列表页面需要快速获知各上游供应商维度的实时健康状况与请求成功/失败趋势。

针对供应商维度状态采集，存在以下设计权衡与背景：
1. **统计口径选择**：网关支持重试与故障转移。若使用业务请求（Request）口径，供应商 A 发生故障但转移给 B 成功时，A 的失败会被完全掩盖；
2. **聚合标识一致性**：ADR-0007 曾指出网关历史指标以 `provider_name` 上报，而 `Provider.Name` 可被自由修改且无唯一约束，缺乏稳定性；
3. **数据源设计**：若仅在 Admin 端聚合 Provider 名下所有 Endpoint 的状态，会导致单页 Redis MGet Key 数量爆炸，且端点删除或改绑后历史状态丢失。

## Decision

1. **采用上游调用（Upstream Call）口径**：
   严格延续 ADR-0008 确立的原则，以网关打向上游的每一次尝试（Attempt）为统计单元。故障转移时，A 失败转移到 B 成功，A 记 1 次失败，B 记 1 次成功，客观真实暴露上游供应商的劣化。

2. **端到端以 Provider Code 为主标识**：
   - Admin 向 Redis 同步端点配置（`ResolvedEndpoint`）时，穿透写入 `provider_code: ep.Provider.Code`。
   - 网关在 `core.Endpoint` 与 `core.AttemptRecord` 中透传 `ProviderCode`。
   - Redis 状态键采用唯一且不可变的 `code` 作为命名空间：
     - 成功：`aigw:status:provider:{provider_code}:{minute}:s` (TTL: 2h)
     - 失败：`aigw:status:provider:{provider_code}:{minute}:f` (TTL: 2h)
   - 查询端优先按 `code` 读取，若未命中且 `name != code` 则尝试按 `name` 回退兼容。

3. **网关原生异步写入 + Admin 批量 MGet 填充**：
   - 网关在 `status_collector` Filter 中利用现有 pipeline 批量写入 Provider 状态键，属于 `BestEffort` 级别，不阻塞业务请求。
   - Admin 在 `Provider.Query` 列表接口中，仅为当前分页展示的 Provider 批量拉取 100 分钟数据并累加为 10 个切片点（`StatusPoints`）。未配置 Redis 时，平滑回退至 Admin 内置的 `metrics.GlobalStore`。

4. **UI 对齐与指标降噪**：
   - 前端复用 `EndpointStatusStrip` 组件，放置于 `enabled`（启用状态）与 `url` 之间。
   - 跨不同模型的供应商平均 TTFT 与 OTPS 缺乏可比性，因此显式设置 `:show-perf="false"`，仅聚焦请求成功/失败计数。

## Consequences

- 供应商列表、模型列表、端点列表具备一致的 100 分钟切片状态观测体验。
- 供应商改名不会再导致实时状态历史断流或丢弃。
- 无数据库额外列或持久化开销，数据完全由 Redis 2 小时 TTL 滑动窗口保障。
