# Admin Session Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent transient frontend failures from destroying persistent login credentials while preserving automatic refresh, startup recovery, and independent multi-device sessions.

**Architecture:** Extract framework-independent session decisions and refresh coordination into a small module that can be exercised with Node's native test runner. The Pinia user store owns local invalidation and active logout, while the HTTP interceptor and router guard consume those explicit operations without treating general request failures as authentication failures.

**Tech Stack:** Vue 3, Pinia, JavaScript ES modules, xy-http, Node.js native test runner/assertions, Prettier, ESLint, Vite.

## Global Constraints

- Only an explicit refresh-token 401 or a user-initiated logout may remove persistent credentials.
- Network errors, timeouts, 5xx responses, and application initialization failures must preserve credentials.
- A refresh token without an access token must be eligible for startup recovery.
- Authentication invalidation must not call the protected logout endpoint.
- User-initiated logout remains best-effort server revocation for the current device only.
- Do not add a frontend test framework or change token storage away from localStorage.
- Run Prettier on every modified frontend `.js` and `.vue` file.

---

### Task 1: Pure Session Decisions and Refresh Coordinator

**Files:**
- Create: `frontend/src/utils/session.js`
- Create: `frontend/tests/session.test.mjs`

**Interfaces:**
- Produces: `getStartupAction({ hasAccessToken, hasRefreshToken }) -> 'initialize' | 'refresh' | 'login'`.
- Produces: `isAuthenticationFailure(error) -> boolean`, true only for HTTP status 401.
- Produces: `createRefreshCoordinator(refresh) -> { run() }`, where concurrent `run()` calls share one in-flight Promise and a rejected refresh rejects every caller.

- [ ] **Step 1: Write the failing decision tests**

```javascript
import test from 'node:test'
import assert from 'node:assert/strict'
import { createRefreshCoordinator, getStartupAction, isAuthenticationFailure } from '../src/utils/session.js'

test('starts with refresh when only a refresh token remains', () => {
    assert.equal(getStartupAction({ hasAccessToken: false, hasRefreshToken: true }), 'refresh')
})

test('does not classify transient refresh failures as authentication failures', () => {
    assert.equal(isAuthenticationFailure({ response: { status: 500 } }), false)
    assert.equal(isAuthenticationFailure(new Error('Network Error')), false)
    assert.equal(isAuthenticationFailure({ response: { status: 401 } }), true)
})

test('shares one refresh across concurrent callers', async () => {
    let calls = 0
    const coordinator = createRefreshCoordinator(async () => {
        calls += 1
        await Promise.resolve()
        return 'new-token'
    })
    assert.deepEqual(await Promise.all([coordinator.run(), coordinator.run()]), ['new-token', 'new-token'])
    assert.equal(calls, 1)
})

test('rejects every caller when the shared refresh fails', async () => {
    const failure = new Error('temporary failure')
    const coordinator = createRefreshCoordinator(async () => {
        throw failure
    })
    const results = await Promise.allSettled([coordinator.run(), coordinator.run()])
    assert.deepEqual(results.map(({ status }) => status), ['rejected', 'rejected'])
    assert.equal(results[0].reason, failure)
    assert.equal(results[1].reason, failure)
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd frontend && node --test tests/session.test.mjs`

Expected: FAIL with `ERR_MODULE_NOT_FOUND` for `src/utils/session.js`.

- [ ] **Step 3: Implement the minimal pure module**

```javascript
export function getStartupAction({ hasAccessToken, hasRefreshToken }) {
    if (hasAccessToken) return 'initialize'
    if (hasRefreshToken) return 'refresh'
    return 'login'
}

export function isAuthenticationFailure(error) {
    return error?.response?.status === 401
}

export function createRefreshCoordinator(refresh) {
    let inFlight = null
    return {
        run() {
            if (!inFlight) {
                inFlight = Promise.resolve()
                    .then(refresh)
                    .finally(() => {
                        inFlight = null
                    })
            }
            return inFlight
        },
    }
}
```

- [ ] **Step 4: Run the test and verify GREEN**

Run: `cd frontend && node --test tests/session.test.mjs`

Expected: 4 tests pass, 0 fail.

- [ ] **Step 5: Commit the independently tested utility**

```bash
git add frontend/src/utils/session.js frontend/tests/session.test.mjs
git commit -m "test: define admin session recovery decisions"
```

### Task 2: User Store Session Lifecycle

**Files:**
- Modify: `frontend/src/store/modules/user.js`
- Modify: `frontend/src/config/storage.js`
- Extend test: `frontend/tests/session.test.mjs`

**Interfaces:**
- Consumes: `isAuthenticationFailure(error)` and a module-scoped refresh coordinator.
- Produces: `invalidateLocalSession()` which performs no network request and clears user/app/router/multi-tab state.
- Produces: `refreshAccessToken()` which resolves `true` on success, throws on transient failure without deleting credentials, and invalidates locally then resolves `false` on refresh 401.
- Produces: `logout()` which attempts server revocation once, then always calls `invalidateLocalSession()`.

- [ ] **Step 1: Extend tests for explicit credential-removal policy**

Add pure policy assertions to `frontend/tests/session.test.mjs`:

```javascript
import { getRefreshFailureAction } from '../src/utils/session.js'

test('only a refresh 401 invalidates local credentials', () => {
    assert.equal(getRefreshFailureAction({ response: { status: 401 } }), 'invalidate')
    assert.equal(getRefreshFailureAction({ response: { status: 500 } }), 'preserve')
    assert.equal(getRefreshFailureAction(new Error('Network Error')), 'preserve')
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd frontend && node --test tests/session.test.mjs`

Expected: FAIL because `getRefreshFailureAction` is not exported.

- [ ] **Step 3: Add the policy and wire the user store**

Add to `frontend/src/utils/session.js`:

```javascript
export function getRefreshFailureAction(error) {
    return isAuthenticationFailure(error) ? 'invalidate' : 'preserve'
}
```

Update `user.js` so that:

```javascript
async refreshAccessToken() {
    if (!this.refreshToken) return false
    try {
        const result = await refreshCoordinator.run(() =>
            apis.user.refreshToken({ refresh_token: this.refreshToken })
        )
        // Store the returned access and refresh tokens, then return true.
    } catch (error) {
        if (getRefreshFailureAction(error) === 'invalidate') {
            this.invalidateLocalSession()
            return false
        }
        throw error
    }
}
```

Use one module-level `createRefreshCoordinator` instance whose callback reads the current store refresh token. Implement `invalidateLocalSession()` to clear token, refresh token, user info, and reset dependent stores without calling an API. Make `logout()` call the API at most once and use `finally` to invoke local invalidation.

Remove `refreshExpiresAt` from store state and storage writes, and remove `refreshExpiresAt` from `frontend/src/config/storage.js`; `expires_at` remains the access-token expiry and is not persisted under a refresh-token name.

- [ ] **Step 4: Run focused tests and formatting**

Run:

```bash
cd frontend
node --test tests/session.test.mjs
npx prettier --config .prettierrc --write src/utils/session.js src/store/modules/user.js src/config/storage.js tests/session.test.mjs
```

Expected: all session tests pass and Prettier exits 0.

- [ ] **Step 5: Commit the store lifecycle change**

```bash
git add frontend/src/utils/session.js frontend/tests/session.test.mjs frontend/src/store/modules/user.js frontend/src/config/storage.js
git commit -m "fix: preserve admin credentials on transient failures"
```

### Task 3: Router Startup Recovery and HTTP Failure Handling

**Files:**
- Modify: `frontend/src/core/permission.js`
- Modify: `frontend/src/utils/request.js`
- Modify: `frontend/src/views/login/index.vue`
- Extend test: `frontend/tests/session.test.mjs`

**Interfaces:**
- Consumes: `getStartupAction` and `userStore.invalidateLocalSession()`.
- Router behavior: refresh-only startup calls `refreshAccessToken()` before `appStore.init()`; initialization rejection preserves credentials.
- HTTP behavior: refresh endpoint 401 invalidates locally; transient refresh failure rejects all waiters and preserves credentials; no-refresh-token 401 invalidates locally without server logout.

- [ ] **Step 1: Add a queue settlement regression test**

Expose `createRequestQueue()` from `session.js` and test both paths:

```javascript
test('settles every queued request on refresh failure', async () => {
    const queue = createRequestQueue()
    const first = queue.wait()
    const second = queue.wait()
    const failure = new Error('refresh failed')
    queue.reject(failure)
    const results = await Promise.allSettled([first, second])
    assert.deepEqual(results.map(({ status }) => status), ['rejected', 'rejected'])
})
```

- [ ] **Step 2: Run the test and verify RED**

Run: `cd frontend && node --test tests/session.test.mjs`

Expected: FAIL because `createRequestQueue` is not exported.

- [ ] **Step 3: Implement queue settlement and update integrations**

Implement a queue with `wait()`, `resolve(value)`, and `reject(error)`, clearing subscribers after either settlement.

In `request.js`, replace callback-only queue behavior. Never call `userStore.logout()` from an interceptor. Use `invalidateLocalSession()` only for an explicit refresh 401 or a 401 without a refresh token; on transient refresh errors, reject queued requests and leave credentials intact.

In `permission.js`, derive startup action from both tokens. When action is `refresh`, await `refreshAccessToken()` before application initialization. If refresh returns false, navigate to login. If refresh or initialization throws a transient error, preserve credentials and navigate to login with the redirect, without invoking logout.

In `login/index.vue`, remove the mounted calls to `userStore.clearTokens()` and `storage.local.removeItem(...)`, and remove the now-unused storage import.

- [ ] **Step 4: Run focused tests, formatting, lint, and build**

Run:

```bash
cd frontend
node --test tests/session.test.mjs tests/portal-workspace.test.mjs
npx prettier --config .prettierrc --write src/utils/session.js src/store/modules/user.js src/config/storage.js src/core/permission.js src/utils/request.js src/views/login/index.vue tests/session.test.mjs
npx eslint --ext .js,.vue --ignore-path .eslintignore --no-fix src/utils/session.js src/store/modules/user.js src/config/storage.js src/core/permission.js src/utils/request.js src/views/login/index.vue
npm run build:prod
```

Expected: all Node tests pass, Prettier exits 0, ESLint reports no errors, and Vite production build exits 0.

- [ ] **Step 5: Run backend authentication regression tests**

Run from repository root:

```bash
go test ./pkg/jwtx ./internal/mods/rbac/biz
```

Expected: both packages report `ok`.

- [ ] **Step 6: Review the final diff against the design and commit**

Run:

```bash
git diff --check
git diff --stat
git status --short
```

Confirm no unrelated files changed, then:

```bash
git add frontend/src/utils/session.js frontend/tests/session.test.mjs frontend/src/store/modules/user.js frontend/src/config/storage.js frontend/src/core/permission.js frontend/src/utils/request.js frontend/src/views/login/index.vue
git commit -m "fix: recover admin sessions without forced logout"
```
