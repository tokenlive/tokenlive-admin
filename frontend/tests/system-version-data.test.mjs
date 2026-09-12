import test from 'node:test'
import assert from 'node:assert/strict'
import { registerHooks } from 'node:module'
import { setImmediate } from 'node:timers/promises'
import { createRenderer, isRef } from 'vue'
import { AxiosError } from 'axios'

// Keep the real API module, XYHttp, Axios, and Vue lifecycle. Replace only browser
// UI/store dependencies and Axios's external transport; no backend is contacted.
const stub = (source) => `data:text/javascript,${encodeURIComponent(source)}`
const dependencies = new Map([
    ['ant-design-vue', stub('export const message = { error() {} }')],
    ['@/config', stub('export const config = () => ""')],
    [
        '@/store',
        stub('export const useUserStore = () => ({ isLogin: true, token: "test-session", hasRefreshToken: false })'),
    ],
])
registerHooks({
    resolve(specifier, context, nextResolve) {
        if (dependencies.has(specifier)) {
            return { url: dependencies.get(specifier), shortCircuit: true }
        }
        if (specifier.startsWith('@/')) {
            return nextResolve(new URL(`../src/${specifier.slice(2)}.js`, import.meta.url).href, context)
        }
        // XYHttp publishes extensionless ESM imports for bundlers.
        if (context.parentURL?.includes('/node_modules/xy-http/') && specifier.startsWith('.')) {
            return nextResolve(`${specifier}.js`, context)
        }
        return nextResolve(specifier, context)
    },
})
const { default: request } = await import('../src/utils/request.js')
const { useSystemVersion } = await import('../src/composables/useSystemVersion.js')

const summary = {
    identity: { edition: 'professional', install_channel: 'release', build: { version: 'v1.2.3', kind: 'release' } },
    gateway: { status: 'ready', scope: 'shared', groups: [{ version: 'v1.2.3', build_kind: 'release', count: 1 }] },
    can_manage_updates: true,
}
const updates = {
    enabled: true,
    components: [
        {
            component: 'admin',
            current: 'v1.2.3',
            latest: 'v1.3.0',
            state: 'available',
            source: {
                status: 'ready',
                stale: false,
                candidate: { version: 'v1.3.0', release_url: 'https://github.com/tokenlive/tokenlive-admin/releases' },
                last_attempt: '2026-09-12T00:00:00Z',
                last_success: '2026-09-12T00:00:00Z',
            },
        },
    ],
    retry_after_seconds: 0,
}
const ok = (data) => ({ status: 200, data: { success: true, data } })
const rejected = (status, id, data) => ({
    status,
    headers: status === 429 ? { 'retry-after': String(data.retry_after_seconds) } : {},
    data: { success: false, data, error: { id, code: status, detail: `Failed: ${id}`, status: String(status) } },
})
const deferred = () => {
    let resolve
    const promise = new Promise((done) => (resolve = done))
    return { promise, resolve }
}
const settle = () => setImmediate()

function events() {
    const target = new EventTarget()
    const listeners = new Map()
    const add = target.addEventListener.bind(target)
    const remove = target.removeEventListener.bind(target)
    target.addEventListener = (name, listener) => {
        listeners.set(name, listener)
        add(name, listener)
    }
    target.removeEventListener = (name, listener) => {
        if (listeners.get(name) === listener) listeners.delete(name)
        remove(name, listener)
    }
    return { target, listeners }
}

function mountVersion(t, respond = (config) => ok(config.url.endsWith('/version') ? summary : updates)) {
    t.mock.timers.enable({ apis: ['setInterval', 'Date'], now: Date.parse('2026-09-12T00:00:00Z') })
    const previous = {
        window: globalThis.window,
        document: globalThis.document,
        appInfo: globalThis.__APP_INFO__,
        adapter: request.basic.store.defaults.adapter,
    }
    const windowEvents = events()
    const documentEvents = events()
    globalThis.window = windowEvents.target
    globalThis.document = documentEvents.target
    globalThis.document.visibilityState = 'visible'
    globalThis.__APP_INFO__ = { version: 'frontend-build-only' }
    const calls = []
    request.basic.store.defaults.adapter = async (config) => {
        calls.push({ method: config.method, url: config.url, authorization: config.headers.Authorization })
        const response = { ...(await respond(config)), config }
        if (response.status >= 400) {
            throw new AxiosError('Request failed', 'ERR_BAD_RESPONSE', config, null, response)
        }
        return response
    }
    const renderer = createRenderer({
        createComment: () => ({}),
        insert() {},
        remove() {},
        parentNode: () => null,
        nextSibling: () => null,
    })
    let state
    const app = renderer.createApp({
        setup() {
            state = useSystemVersion()
            return () => null
        },
    })
    app.mount({})
    let unmounted = false
    const unmount = () => {
        if (!unmounted) app.unmount()
        unmounted = true
    }
    t.after(() => {
        unmount()
        for (const [name, value] of Object.entries({
            window: previous.window,
            document: previous.document,
            __APP_INFO__: previous.appInfo,
        })) {
            if (value === undefined) delete globalThis[name]
            else globalThis[name] = value
        }
        request.basic.store.defaults.adapter = previous.adapter
    })
    return {
        state,
        calls,
        unmount,
        windowEvents,
        documentEvents,
        visibility(value) {
            documentEvents.target.visibilityState = value
            documentEvents.target.dispatchEvent(new Event('visibilitychange'))
        },
        focus() {
            windowEvents.target.dispatchEvent(new Event('focus'))
        },
    }
}

test('mount loads the summary before cached updates through the authenticated request wrapper', async (t) => {
    const { state, calls } = mountVersion(t)
    for (const name of ['summary', 'updates', 'fallbackVersion', 'checking', 'error', 'retryAfterSeconds']) {
        assert.equal(isRef(state[name]), true, name)
    }
    await state.load()
    assert.deepEqual(calls, [
        { method: 'get', url: '/api/v1/current/version', authorization: 'test-session' },
        { method: 'get', url: '/api/v1/system/updates', authorization: 'test-session' },
    ])
    assert.deepEqual(state.summary.value, summary)
    assert.deepEqual(state.updates.value, updates)
    assert.equal(state.fallbackVersion.value, '')
    assert.equal(state.error.value, null)
})

test('capability must be exactly true; user names and cached-looking data grant no permission', async (t) => {
    const { state, calls } = mountVersion(t, () =>
        ok({ ...summary, can_manage_updates: 'true', username: 'root', updates })
    )
    await state.load()
    await state.check()
    assert.equal(calls.length, 1)
    assert.equal(state.updates.value, null)
})

test('summary failures expose only build fallback, never a synthesized runtime identity', async (t) => {
    const { state, calls } = mountVersion(t, () => {
        throw new Error('offline')
    })
    await state.load()
    assert.equal(calls.length, 1)
    assert.equal(state.summary.value, null)
    assert.equal(state.updates.value, null)
    assert.equal(state.fallbackVersion.value, 'frontend-build-only')
    assert.equal(state.error.value.message, 'offline')
})

test('loads coalesce and load is also the explicit About-open cache refresh', async (t) => {
    const pending = deferred()
    let first = true
    const { state, calls } = mountVersion(t, (config) => {
        if (first) {
            first = false
            return pending.promise
        }
        return ok(config.url.endsWith('/version') ? summary : updates)
    })
    const a = state.load()
    const b = state.load()
    assert.equal(a, b)
    await settle()
    assert.equal(calls.length, 1)
    pending.resolve(ok(summary))
    await a
    await state.load()
    assert.equal(calls.length, 4)
    assert.equal(
        calls.every(({ method }) => method === 'get'),
        true
    )
})

test('manual checks coalesce and do not invent a cooldown when the server returns none', async (t) => {
    const pending = deferred()
    const { state, calls } = mountVersion(t, (config) =>
        config.method === 'post' ? pending.promise : ok(config.url.endsWith('/version') ? summary : updates)
    )
    await state.load()
    const a = state.check()
    const b = state.check()
    assert.equal(a, b)
    assert.equal(state.checking.value, true)
    await settle()
    assert.equal(calls.filter(({ method }) => method === 'post').length, 1)
    assert.equal(calls.at(-1).url, '/api/v1/system/updates/check')
    pending.resolve(ok({ ...updates, components: [] }))
    await a
    assert.equal(state.checking.value, false)
    assert.deepEqual(state.updates.value.components, [])
    assert.equal(state.retryAfterSeconds.value, 0)
})

test('R15 HTTP 409 is a failed attempt with a usable disabled snapshot', async (t) => {
    const disabled = { ...updates, enabled: false, components: [{ ...updates.components[0], state: 'disabled' }] }
    const { state, calls } = mountVersion(t, (config) =>
        config.method === 'post'
            ? rejected(409, 'update_check_disabled', disabled)
            : ok(config.url.endsWith('/version') ? summary : updates)
    )
    await state.load()
    await state.check()
    assert.deepEqual(state.updates.value, disabled)
    assert.equal(state.error.value.response.status, 409)
    assert.equal(state.error.value.response.data.error.id, 'update_check_disabled')
    assert.equal(state.checking.value, false)
    await state.check()
    assert.equal(calls.filter(({ method }) => method === 'post').length, 1)
})

test('R15 HTTP 429 keeps the failed envelope and counts down the server remaining seconds', async (t) => {
    const cooldown = { ...updates, retry_after_seconds: 3 }
    let posts = 0
    const { state, calls } = mountVersion(t, (config) => {
        if (config.method === 'post') {
            posts++
            return posts === 1 ? rejected(429, 'update_check_cooldown', cooldown) : ok(updates)
        }
        return ok(config.url.endsWith('/version') ? summary : updates)
    })
    await state.load()
    await state.check()
    assert.deepEqual(state.updates.value, cooldown)
    assert.equal(state.error.value.response.data.error.id, 'update_check_cooldown')
    assert.equal(state.error.value.response.headers['retry-after'], '3')
    assert.equal(state.retryAfterSeconds.value, 3)
    t.mock.timers.tick(1000)
    assert.equal(state.retryAfterSeconds.value, 2)
    await state.check()
    assert.equal(calls.filter(({ method }) => method === 'post').length, 1)
    t.mock.timers.tick(2000)
    assert.equal(state.retryAfterSeconds.value, 0)
    await state.check()
    assert.equal(calls.filter(({ method }) => method === 'post').length, 2)
    assert.equal(state.error.value, null)
})

test('visible polling and focus read caches only; hidden and unmounted instances stop polling', async (t) => {
    const harness = mountVersion(t)
    await harness.state.load()
    t.mock.timers.tick(30_000)
    await settle()
    assert.equal(harness.calls.length, 4)
    harness.focus()
    await settle()
    assert.equal(harness.calls.length, 6)
    harness.visibility('hidden')
    t.mock.timers.tick(90_000)
    harness.focus()
    await settle()
    assert.equal(harness.calls.length, 6)
    harness.visibility('visible')
    await settle()
    assert.equal(harness.calls.length, 8)
    harness.unmount()
    assert.equal(harness.windowEvents.listeners.size, 0)
    assert.equal(harness.documentEvents.listeners.size, 0)
    t.mock.timers.tick(90_000)
    harness.focus()
    await harness.state.load()
    await harness.state.check()
    assert.equal(harness.calls.length, 8)
    assert.equal(
        harness.calls.every(({ method }) => method === 'get'),
        true
    )
})

test('cooldown uses elapsed time after resuming a hidden page', async (t) => {
    const { state, visibility } = mountVersion(t, (config) =>
        ok(config.url.endsWith('/version') ? summary : { ...updates, retry_after_seconds: 60 })
    )
    await state.load()
    assert.equal(state.retryAfterSeconds.value, 60)
    visibility('hidden')
    t.mock.timers.tick(61_000)
    visibility('visible')
    assert.equal(state.retryAfterSeconds.value, 0)
    await state.load()
})

test('capability revocation clears privileged data and ignores an already pending manual response', async (t) => {
    const pending = deferred()
    let allowed = true
    const { state, calls } = mountVersion(t, (config) => {
        if (config.method === 'post') return pending.promise
        return ok(config.url.endsWith('/version') ? { ...summary, can_manage_updates: allowed } : updates)
    })
    await state.load()
    const checking = state.check()
    await settle()
    allowed = false
    await state.load()
    assert.equal(state.summary.value.can_manage_updates, false)
    assert.equal(state.updates.value, null)
    assert.equal(state.retryAfterSeconds.value, 0)
    pending.resolve(ok({ ...updates, retry_after_seconds: 60 }))
    await checking
    await state.check()
    assert.equal(state.updates.value, null)
    assert.equal(state.retryAfterSeconds.value, 0)
    assert.equal(calls.filter(({ method }) => method === 'post').length, 1)
})

test('authorization errors clear capability and prevent an older pending summary from restoring it', async (t) => {
    const pending = deferred()
    let nextSummary = false
    const { state } = mountVersion(t, (config) => {
        if (config.method === 'post') return rejected(403, 'forbidden', null)
        if (config.url.endsWith('/version')) return nextSummary ? pending.promise : ok(summary)
        return ok(updates)
    })
    await state.load()
    nextSummary = true
    const loading = state.load()
    await state.check()
    assert.equal(state.summary.value.can_manage_updates, false)
    assert.equal(state.updates.value, null)
    pending.resolve(ok(summary))
    await loading
    assert.equal(state.summary.value.can_manage_updates, false)
    assert.equal(state.updates.value, null)
})

test('revocation also clears privileged snapshots retained inside a failed-check error', async (t) => {
    const pendingSummary = deferred()
    const pendingCheck = deferred()
    let holdSummary = false
    const { state } = mountVersion(t, (config) => {
        if (config.method === 'post') return pendingCheck.promise
        if (config.url.endsWith('/version')) return holdSummary ? pendingSummary.promise : ok(summary)
        return ok(updates)
    })
    await state.load()
    const checking = state.check()
    holdSummary = true
    const loading = state.load()
    pendingCheck.resolve(rejected(429, 'update_check_cooldown', { ...updates, retry_after_seconds: 60 }))
    await checking
    assert.equal(state.error.value.response.data.data.components[0].latest, 'v1.3.0')
    pendingSummary.resolve(ok({ ...summary, can_manage_updates: false }))
    await loading
    assert.equal(state.updates.value, null)
    assert.equal(state.retryAfterSeconds.value, 0)
    assert.equal(state.error.value, null)
})

test('a pending cached update read cannot overwrite a newer manual check snapshot', async (t) => {
    const pending = deferred()
    let hold = false
    const current = { ...updates, components: [{ ...updates.components[0], state: 'current' }] }
    const { state } = mountVersion(t, (config) => {
        if (config.method === 'post') return ok(current)
        if (config.url.endsWith('/version')) return ok(summary)
        return hold ? pending.promise : ok(updates)
    })
    await state.load()
    hold = true
    const loading = state.load()
    await settle()
    await state.check()
    pending.resolve(ok(updates))
    await loading
    assert.deepEqual(state.updates.value, current)
})

test('a summary outage after success removes old privileged data and exposes build fallback', async (t) => {
    let fail = false
    const { state } = mountVersion(t, (config) => {
        if (fail) throw new Error('offline')
        return ok(config.url.endsWith('/version') ? summary : updates)
    })
    await state.load()
    fail = true
    await state.load()
    assert.equal(state.summary.value, null)
    assert.equal(state.updates.value, null)
    assert.equal(state.fallbackVersion.value, 'frontend-build-only')
})

test('late responses do not repopulate an unmounted instance', async (t) => {
    const pending = deferred()
    const { state, calls, unmount } = mountVersion(t, () => pending.promise)
    const loading = state.load()
    await settle()
    unmount()
    pending.resolve(ok(summary))
    await loading
    assert.equal(state.summary.value, null)
    assert.equal(state.updates.value, null)
    assert.equal(calls.length, 1)
})
