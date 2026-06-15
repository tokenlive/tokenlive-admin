# Dashboard Trend Bucketing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Limit dashboard trend responses to at most 120 aggregate time buckets.

**Architecture:** Introduce one helper that resolves a requested range into point count, bucket width, and Redis coverage. Use the bucket width for both Prometheus query step and `increase` window, and aggregate Redis minute counters into matching buckets.

**Tech Stack:** Go, Gin, Prometheus HTTP API, Redis, testify

---

### Task 1: Resolve Trend Bucket Configuration

**Files:**
- Modify: `internal/mods/dashboard/api/dashboard.api.go`
- Test: `test/dashboard_test.go`

- [ ] Add failing tests for the expected point count and step of every supported range.
- [ ] Add a `trendRangeConfig` helper that selects fixed buckets for `1h`, `6h`, `24h`, and `7d`, plus a dynamic bucket for `today`.
- [ ] Use the resolved configuration when building the response timeline and Prometheus queries.
- [ ] Run `go test ./test -run TestQueryTrends -count=1`.

### Task 2: Aggregate Redis Fallback Buckets

**Files:**
- Modify: `internal/mods/dashboard/api/dashboard.api.go`
- Test: `test/dashboard_test.go`

- [ ] Add a failing test proving minute Redis values are summed into larger chart buckets.
- [ ] Aggregate Redis minute values into the resolved chart buckets.
- [ ] Run `go test ./test -run TestQueryTrends -count=1`.

### Task 3: Full Verification

**Files:**
- Verify: `internal/mods/dashboard/api/dashboard.api.go`
- Verify: `test/dashboard_test.go`

- [ ] Run `gofmt` on changed Go files.
- [ ] Run `go test ./... -count=1`.
- [ ] Run `go vet ./...`.
- [ ] Run `go build ./...`.
- [ ] Run `git diff --check`.
