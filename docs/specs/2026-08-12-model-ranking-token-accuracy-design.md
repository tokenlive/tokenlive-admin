# 模型排行榜 Token 统计准确性设计

## 背景

首页“今日总 Tokens 消耗”按 `input_tokens + output_tokens` 展示。模型性能排行榜当前从 Prometheus 聚合 `tokens_total` 的所有 `type`，其中 `cached` 和 `cache_creation` 已经包含在 `input` 中，因此缓存使用量会被重复计入模型 Token 总量。

## 目标

- 保证排行榜中每个模型的 `total_tokens` 等于该时间范围内的输入 Token 与输出 Token 之和。
- 现有历史 Prometheus 数据无需迁移即可按正确口径查询。

## 非目标

- 不要求排行榜可见行之和等于全局“今日总 Tokens 消耗”卡片。
- 不改变 Top 10、启用模型过滤、请求数排序、时间范围选择或卡片统计逻辑。
- 不修改网关的 Token 指标上报，以保留 `cached` 和 `cache_creation` 分类用于明细分析。

## 设计

将模型排行榜 Token PromQL 从聚合全部类型：

```promql
sum by (model) (increase(<tokens_total>[<range>]))
```

调整为只聚合输入和输出：

```promql
sum by (model) (increase(<tokens_total>{type=~"input|output"}[<range>]))
```

`cached` 与 `cache_creation` 是输入 Token 的子集，不再重复累加。其他排行榜指标和数据处理流程保持不变。

## 测试

新增回归测试，验证排行榜 Token 查询：

- 包含 `type=~"input|output"` 标签过滤条件。
- 继续使用传入的时间范围和配置生成的指标名称。
- 不改变模型维度聚合。

然后运行 dashboard API 包测试，确认现有行为未受影响。
