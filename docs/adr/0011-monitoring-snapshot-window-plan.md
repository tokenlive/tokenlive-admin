# 旧监控采用统一快照与私有窗口计划

Status: accepted

## Context

旧监控的 HTTP、WebSocket、流量趋势、模型排行和供应商排行分别解释时间范围、获取当前时间和定义结果。历史 `today`、七天内存查询、分钟边界与缓存时区因此存在不一致。调用方还无法区分真实的零与已经过期的历史数据。

## Decision

- 以 `monitor.Reader.Read(ctx, Query)` 作为外部 interface。Query 选择 overview、traffic、performance、models、providers；返回具名、带类型的 Reading，HTTP 与 WebSocket 共用此 seam。
- 每次 Read 只捕获一次评估截点。内部 windowPlan 负责时区、起止、分桶、源查询 lookback 和缓存身份，不向调用方暴露执行计划。
- 分钟序列采用向内取整的半开窗口。排除尚未完成的分钟，保留最后一个不满完整步长的桶并返回真实起止。Prometheus 末桶使用真实宽度查询，不能拿完整步长代替。
- Reading 返回请求的实际窗口、桶、数据源和覆盖状态：`ok`、`partial`、`unavailable`、`error`。缺失序列值为 null；错误返回稳定错误码，不泄露底层连接信息。同一快照允许局部失败。
- 无效范围、时区、未来截点与排名参数在源读取之前拒绝。缺省时区使用服务端本地时区，缺省范围一小时。供应商超过七天的范围继续按 ADR-0008 限制到七天。
- Redis 分钟计数按两小时 TTL、内存按七天及本次进程实际启动时间确定可覆盖区间。Prometheus 使用额外的分钟级源样本存在性查询核实历史覆盖，不能仅依据 increase 的返回值声称完整；不完整桶置 null。
- Overview 保持实时 QPS、今日累计、近五分钟延迟的不同窗口。日累计键按网关／服务端本地自然日采集；所选时区午夜与本地午夜不一致时，日累计明确不可用，不重命名日累计键来冒充时区换算。Overview 不接受历史日期重建。
- 首页和监控详情通过同一前端 adapter 转换绝对桶时间，并保留 null 与状态。HTTP 旧响应不得覆盖较新筛选或已接管的 WebSocket 数据。

## Consequences

调用方只选择结果，不再推断数据源、分钟索引或过期历史是否等于零。测试通过 Read、HTTP 和 WebSocket interface 验证组合行为；源 adapter 的计数与 Request／Upstream Call 口径继续单独验证。

提供 `POST /api/v1/dashboard/snapshot`。原有单项 GET 入口及其兼容参数解析已在消费者迁移后移除，仓库内消费者统一使用 Snapshot。独立熔断器查询仍服务监控详情页。

缓存按实际窗口、时区、步长与筛选隔离，过期条目在写入时清理。快照只承诺共同评估截点，不承诺多个源的原子一致性。

ADR-0010 的 ClickHouse 使用趋势保持独立：它基于请求日志，未启用 ClickHouse 时隐藏整卡。本次没有合并日志趋势和旧 Prometheus／Redis／内存监控，也没有扩展供应商指标能力或重构详情页的查询编排。
