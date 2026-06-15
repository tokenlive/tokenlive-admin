# Dashboard Statistics Transition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Restore Redis as the source of today's overview cost and clearly label Prometheus-backed ranking business statistics as estimates.

**Architecture:** Keep the existing overview and model-ranking API contracts. Remove the Prometheus override from the overview so its natural-day cost remains Redis-backed, while retaining the Prometheus-backed ranking and documenting its estimated business fields in the frontend.

**Tech Stack:** Go, Gin, go-redis, Vue 3, Ant Design Vue, Vue i18n, Prettier

---

### Task 1: Protect Redis Daily Cost Semantics

**Files:**
- Modify: `internal/mods/dashboard/api/dashboard.api.go`
- Test: `internal/mods/dashboard/api/dashboard.api_test.go`

- [ ] **Step 1: Write a failing overview regression test**

Add a test that creates a Redis-backed dashboard, writes today's daily cost as `11.027258`,
configures a mock Prometheus response with a different cost, calls `QueryOverview`, and
asserts that `daily_cost` remains `11.027258`.

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test ./internal/mods/dashboard/api -run TestQueryOverviewUsesRedisDailyCost -count=1
```

Expected: FAIL because the current Prometheus override replaces the Redis daily cost.

- [ ] **Step 3: Remove the Prometheus daily-cost override**

Delete the following override from `QueryOverview`:

```go
if dailyCost, ok := a.getDailyCostFromPrometheus(time.Now()); ok {
	res.DailyCost = dailyCost
}
```

Remove `getDailyCostFromPrometheus` and its now-obsolete unit test if it has no remaining
callers. Keep the generic Prometheus single-value result helper if other code uses it.

- [ ] **Step 4: Run the focused test and verify it passes**

Run:

```bash
go test ./internal/mods/dashboard/api -run TestQueryOverviewUsesRedisDailyCost -count=1
```

Expected: PASS.

### Task 2: Label Ranking Business Statistics as Estimates

**Files:**
- Modify: `frontend/src/views/home/index.vue`
- Modify: `frontend/src/locales/lang/zh-CN/pages.js`
- Modify: `frontend/src/locales/lang/en-US/pages.js`

- [ ] **Step 1: Add localized data-semantics copy**

Add locale keys for:

```text
pages.dashboard.modelRanking.estimateNotice
pages.dashboard.modelRanking.columns.estimatedCost
```

Use the approved Chinese and English copy from the design document.

- [ ] **Step 2: Add a visible informational note**

Add a compact Ant Design Vue informational alert above the model-ranking table:

```vue
<a-alert
    :message="$t('pages.dashboard.modelRanking.estimateNotice')"
    type="info"
    show-icon />
```

Keep it visible during normal table rendering so users understand the data semantics.

- [ ] **Step 3: Mark the cost column as estimated**

Change the model-ranking cost column title to use
`pages.dashboard.modelRanking.columns.estimatedCost`. Do not change the API response or
sorting controls.

- [ ] **Step 4: Format all modified frontend files**

Run:

```bash
cd frontend
npx prettier --config .prettierrc --write src/views/home/index.vue src/locales/lang/zh-CN/pages.js src/locales/lang/en-US/pages.js
```

Expected: Prettier completes without errors.

### Task 3: Verify the Transitional Implementation

**Files:**
- Verify: `internal/mods/dashboard/api/dashboard.api.go`
- Verify: `internal/mods/dashboard/api/dashboard.api_test.go`
- Verify: `frontend/src/views/home/index.vue`
- Verify: `frontend/src/locales/lang/zh-CN/pages.js`
- Verify: `frontend/src/locales/lang/en-US/pages.js`

- [ ] **Step 1: Run backend tests**

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Run Go static verification**

Run:

```bash
go vet ./...
go build ./...
```

Expected: both commands succeed.

- [ ] **Step 3: Build the frontend**

Run:

```bash
cd frontend
npm run build:prod
```

Expected: build succeeds; existing non-fatal Rollup warnings may remain.

- [ ] **Step 4: Check the final diff**

Run:

```bash
git diff --check
git status --short
```

Expected: no whitespace errors, and unrelated existing worktree changes remain untouched.
