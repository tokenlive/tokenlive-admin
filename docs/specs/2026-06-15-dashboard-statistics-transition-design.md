# Dashboard Statistics Transition Design

## Background

The dashboard currently mixes business statistics and operational metrics:

- The overview has durable natural-day totals in Redis, but today's cost is temporarily
  overwritten with a Prometheus query.
- The model ranking reads request counts, tokens, cost, success rate, latency, and TTFT
  entirely from Prometheus.
- Prometheus counter resets and scrape gaps can make cost and token totals incomplete.
- Redis only contains global natural-day totals and does not contain sufficient per-model,
  arbitrary-range statistics.

ClickHouse is the intended long-term source for accurate business analytics. Building a
temporary per-model Redis aggregation system would duplicate that future work.

## Goals

- Restore today's overview cost to the existing Redis natural-day total.
- Keep the current model ranking functional without expanding the Redis statistics model.
- Clearly communicate that ranking request, token, success-rate, and cost values are
  Prometheus estimates and are not suitable for reconciliation.
- Keep latency and TTFT metrics presented as normal monitoring metrics.
- Preserve the current API contract and time-range controls.

## Non-Goals

- Adding ClickHouse integration.
- Adding new Redis keys or changing joylive-agent.
- Making the model ranking cost sum equal the overview cost.
- Guaranteeing exact historical business statistics.
- Splitting the ranking into multiple frontend API calls.

## Transitional Architecture

### Overview

`GET /api/v1/dashboard/overview` reads today's cost from:

```text
aigw:status:daily:cost:{YYYY-MM-DD}
```

The existing Prometheus fallback/override for today's cost is removed. If Redis is
unavailable or the key is absent, the value remains zero rather than silently presenting
an incomplete Prometheus value as the natural-day total.

### Model Ranking

`GET /api/v1/dashboard/model-ranking` remains Prometheus-backed for all ranking fields.
This preserves the existing arbitrary time ranges and sorting behavior.

The frontend displays one visible note above the ranking table explaining:

- Request counts, success rate, tokens, and cost are Prometheus estimates.
- Estimated totals may differ from the overview's natural-day totals.
- Latency and TTFT are monitoring metrics.

The cost column title is changed to indicate that it is estimated. Other business fields
remain covered by the table-level note to avoid excessive visual noise.

## User Experience

The overview's "today cost" card represents the durable Redis natural-day total.

The ranking displays a compact informational alert between the card controls and table.
Chinese copy:

> 请求数、成功率、Token 和费用来自 Prometheus 估算，可能与今日累计数据存在差异；延迟与 TTFT 为监控指标。

English copy:

> Requests, success rate, tokens, and cost are Prometheus estimates and may differ from today's cumulative totals. Latency and TTFT are monitoring metrics.

The ranking cost column is labeled `费用（估算）` / `Cost (estimated)`.

## Long-Term Migration

ClickHouse will become the source for business statistics:

- request count and status
- token usage
- cost
- model/provider/tenant dimensions
- arbitrary time-range ranking and reconciliation

Prometheus remains the source for live operational metrics such as QPS, latency
histograms, TTFT histograms, and alerting.

When ClickHouse integration is available, the existing ranking API can merge ClickHouse
business fields with Prometheus performance fields without changing the frontend response
shape.

## Error Handling

- Redis errors in the overview retain the existing zero-value behavior.
- Prometheus errors in the ranking retain the existing empty/zero behavior.
- The UI note is always visible because it describes data semantics, not a transient
  service error.

## Testing

- Add a backend regression test proving that Redis daily cost is not overwritten by a
  different Prometheus value.
- Keep existing model-ranking API tests because its backend behavior does not change.
- Format modified Vue and locale JavaScript files with Prettier.
- Run backend tests, Go vet/build, frontend production build, and `git diff --check`.
