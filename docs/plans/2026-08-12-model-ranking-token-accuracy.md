# Model Ranking Token Accuracy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make each model ranking row report input plus output Tokens without double-counting cached or cache-creation Tokens.

**Architecture:** Keep the gateway metric and dashboard data flow unchanged. Centralize the model Token PromQL in a small query builder, constrain its `type` label to `input|output`, and cover the exact query contract with a focused unit test.

**Tech Stack:** Go, PromQL, Testify, Go test

## Global Constraints

- Preserve the existing Top 10 limit, enabled-model filtering, request-count sorting, time-range behavior, and overview card logic.
- Do not change gateway metric emission or migrate historical Prometheus data.
- The model `total_tokens` value must equal input Tokens plus output Tokens for the selected range.

---

### Task 1: Filter model Token aggregation to input and output

**Files:**
- Modify: `internal/mods/dashboard/api/dashboard.api.go:1019-1021`
- Test: `internal/mods/dashboard/api/dashboard.api_test.go`

**Interfaces:**
- Consumes: `mTokensTotal string` and the resolved Prometheus range string.
- Produces: `buildModelTokensQuery(promRange string) string`, used by `getModelRanking`.

- [ ] **Step 1: Write the failing regression test**

Add this test to `internal/mods/dashboard/api/dashboard.api_test.go`:

```go
func TestBuildModelTokensQueryCountsInputAndOutputOnly(t *testing.T) {
	query := buildModelTokensQuery("24h")

	expected := `sum by (model) (increase(` + mTokensTotal + `{type=~"input|output"}[24h]))`
	assert.Equal(t, expected, query)
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run:

```bash
go test ./internal/mods/dashboard/api -run TestBuildModelTokensQueryCountsInputAndOutputOnly -count=1
```

Expected: FAIL to compile with `undefined: buildModelTokensQuery`, proving the regression test requires the new query contract.

- [ ] **Step 3: Add the minimal query builder**

Add this helper near the model-ranking implementation in `internal/mods/dashboard/api/dashboard.api.go`:

```go
func buildModelTokensQuery(promRange string) string {
	return fmt.Sprintf(
		`sum by (model) (increase(%s{type=~"input|output"}[%s]))`,
		mTokensTotal,
		promRange,
	)
}
```

Replace the existing Token query inside `getModelRanking`:

```go
tokensMap := a.queryPrometheusMultiValues(buildModelTokensQuery(promRange), "model")
```

Leave the cost query and all other ranking behavior unchanged.

- [ ] **Step 4: Run the focused test and verify GREEN**

Run:

```bash
go test ./internal/mods/dashboard/api -run TestBuildModelTokensQueryCountsInputAndOutputOnly -count=1
```

Expected: PASS.

- [ ] **Step 5: Run package regression tests**

Run:

```bash
go test ./internal/mods/dashboard/api/... -count=1
```

Expected: PASS with zero failures.

- [ ] **Step 6: Inspect the final diff**

Run:

```bash
git diff --check
git diff -- internal/mods/dashboard/api/dashboard.api.go internal/mods/dashboard/api/dashboard.api_test.go
```

Expected: no whitespace errors; the production diff changes only the Token PromQL construction and its call site.

- [ ] **Step 7: Commit the implementation**

```bash
git add internal/mods/dashboard/api/dashboard.api.go internal/mods/dashboard/api/dashboard.api_test.go
git commit -m "fix: correct model ranking token totals"
```
