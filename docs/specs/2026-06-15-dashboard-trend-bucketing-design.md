# Dashboard Trend Bucketing Design

## Goal

Keep every dashboard trend response at or below 120 points while preserving the
request totals represented by the selected time range.

## Bucket Configuration

| Time range | Bucket width | Maximum points |
| --- | --- | --- |
| `1h` | 1 minute | 60 |
| `6h` | 5 minutes | 72 |
| `24h` | 15 minutes | 96 |
| `7d` | 2 hours | 84 |
| `today` | Smallest of 1, 5, 10, or 15 minutes that produces at most 120 points | 120 |

## Prometheus Queries

The Prometheus range query step and the PromQL `increase` window use the same
bucket width. Each returned value therefore represents the total requests in
one complete chart bucket rather than a sampled minute.

The query start is calculated from the number of chart points:

```text
start = end - (point_count - 1) * bucket_width
```

This matches Prometheus range queries, whose start and end timestamps are both
inclusive, and keeps the latest bucket in the response.

## Redis Fallback

Redis stores one value per minute. For supported ranges, the fallback reads the
available minute values and sums them into the same chart buckets used by
Prometheus. Missing historical minutes remain zero.

## Verification

Tests verify that every configured range returns at most 120 points, uses the
expected Prometheus step and `increase` window, retains the latest bucket, and
correctly aggregates Redis minute values.
