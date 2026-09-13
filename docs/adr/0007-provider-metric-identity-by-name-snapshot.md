# Provider 维度指标与事件按名称快照归属

Status: accepted

网关上报的指标（`RequestMetric.Provider`）与治理事件（`event_log.provider_name`）均以字符串标识供应商，不携带 `provider_id`。由于 `Provider.Name` 可被自由修改且无唯一约束，供应商改名后历史指标与事件将无法关联到当前记录，同名供应商的数据也会被静默合并。

我们接受这一缺陷并按现状交付供应商监控，而非将网关上报协议变更设为前置依赖：Gateway 的发布节奏独立，把一个详情页需求阻塞在另一个仓库的排期上代价过高。Admin 侧同步新增 `event_log.provider_id` 列并具备接收能力，字段为空时自动回退到名称匹配，使 Gateway 改造完成后无需再改 Admin 查询逻辑。历史数据不做回填——按当前 name→id 映射回填会给改过名的记录写入错误归属，而这正是待解决的问题本身，"看起来完整但部分错误"的数据比留空更危险。

未确认事项（已解决，见 ADR-0009）：Gateway 与 Admin 已统一将 Provider 的 `code` 作为运行时指标聚合与 Redis 键（`aigw:status:provider:{code}:{minute}:[s|f]`）的标准唯一键，Admin 在同步端点配置时穿透 `provider_code`，查询端按 `code` 为主、`name` 兜底兼容。
