# Remove Invalid Dashboard Trend Groups Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the unsupported Tenant and Endpoint choices from the Admin home-page traffic trend selector without changing the dashboard API.

**Architecture:** Keep the change entirely in the frontend. A dependency-free Node regression check inspects the rendered selector declarations in the Vue SFC, then the Vue template and now-unused locale entries are reduced to Global, Model, and Provider.

**Tech Stack:** Vue 3, JavaScript, Node.js, Prettier, Vite

## Global Constraints

- Keep `GET /api/v1/dashboard/trends` and its `group_by` compatibility unchanged.
- Keep Global, By Model, and By Provider unchanged.
- Run Prettier on every modified `.vue` and `.js` source file.
- Do not add frontend dependencies.

---

### Task 1: Remove Unsupported Trend Groups

**Files:**
- Create: `frontend/scripts/check-dashboard-trend-groups.mjs`
- Modify: `frontend/src/views/home/index.vue:220-240`
- Modify: `frontend/src/locales/lang/en-US/pages.js:120-124`
- Modify: `frontend/src/locales/lang/zh-CN/pages.js:117-121`
- Test: `frontend/scripts/check-dashboard-trend-groups.mjs`

**Interfaces:**
- Consumes: Home-page trend group option values declared as `<a-select-option value="...">`.
- Produces: A selector exposing only `""`, `model`, and `provider`; a check command with exit code 0 only for that contract.

- [x] **Step 1: Write the failing regression check**

Create a Node script that reads `src/views/home/index.vue`, extracts the second selector bound to `trendsGroupBy`, and asserts its literal option values equal `['', 'model', 'provider']`. It must report the actual values and exit non-zero on mismatch.

- [x] **Step 2: Run the check to verify it fails**

Run: `node scripts/check-dashboard-trend-groups.mjs`

Expected: FAIL because the actual values still include `tenant` and `endpoint`.

- [x] **Step 3: Remove the two options and locale entries**

Delete the `tenant` and `endpoint` `<a-select-option>` blocks from `src/views/home/index.vue`. Delete `pages.dashboard.trends.group.tenant` and `pages.dashboard.trends.group.endpoint` from both locale files. Do not modify API calls or backend code.

- [x] **Step 4: Format modified frontend source files**

Run:

```bash
npx prettier --config .prettierrc --write src/views/home/index.vue src/locales/lang/en-US/pages.js src/locales/lang/zh-CN/pages.js
```

- [x] **Step 5: Verify the regression check passes**

Run: `node scripts/check-dashboard-trend-groups.mjs`

Expected: PASS and report `global, model, provider`.

- [x] **Step 6: Verify the production frontend build and whitespace**

Run:

```bash
npm run build:prod
git diff --check
```

Expected: both commands exit 0.

- [x] **Step 7: Review the final diff**

Confirm the diff contains only the regression check, two removed Vue options, two removed entries per locale file, and this implementation plan.
