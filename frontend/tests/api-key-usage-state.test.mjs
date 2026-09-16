import test from 'node:test'
import assert from 'node:assert/strict'
import { createAPIKeyUsageController } from '../src/utils/api-key-usage-state.js'

const ready = (requests = 1) => ({
    state: 'ready',
    data_source: 'clickhouse',
    window: { start: '2026-09-16T00:00:00+08:00', end: '2026-09-16T12:00:00+08:00', timezone: 'UTC+8' },
    generated_at: '2026-09-16T12:00:00+08:00',
    summary: { request_count: requests, total_tokens: 100, total_cost: '0.3', key_count: 0 },
    items: [],
    unattributed: { request_count: requests, total_tokens: 100, total_cost: '0.3', token_share: 100 },
    warnings: ['unattributed_usage'],
})
const deferred = () => {
    let resolve
    const promise = new Promise((done) => {
        resolve = done
    })
    return { promise, resolve }
}
const controllerFor = (load) =>
    createAPIKeyUsageController({
        load,
        onChange: () => {},
        setTimeoutFn: () => 1,
        clearTimeoutFn: () => {},
    })
const activate = (controller) => {
    controller.setActive(true)
    controller.setVisible(true)
    controller.setAuthorized(true)
    return controller.refresh()
}

test('requires authorized, active and visible before loading and coalesces refresh', async () => {
    let calls = 0
    const controller = controllerFor(async () => {
        calls++
        return ready()
    })
    controller.setVisible(true)
    controller.setActive(true)
    await controller.refresh()
    assert.equal(calls, 0)
    controller.setAuthorized(true)
    await Promise.all([controller.refresh(), controller.refresh()])
    assert.equal(calls, 1)
    assert.equal(controller.getState().phase, 'ready')
    controller.dispose()
})

test('same-filter outage keeps a timestamped stale result; changed-filter outage does not', async () => {
    let failing = false
    const controller = controllerFor(async () => {
        if (failing) throw { response: { status: 503 } }
        return ready()
    })
    await activate(controller)
    failing = true
    await controller.refresh()
    assert.equal(controller.getState().stale, true)
    assert.equal(controller.getState().data.generated_at, '2026-09-16T12:00:00+08:00')
    controller.setQuery({ time_range: '7d' })
    await controller.refresh()
    assert.equal(controller.getState().phase, 'error')
    assert.equal(controller.getState().data, null)
    controller.dispose()
})

test('slow old response cannot overwrite a changed filter', async () => {
    const first = deferred()
    const second = deferred()
    const calls = []
    const controller = controllerFor((query, { signal }) => {
        calls.push({ query, signal })
        return calls.length === 1 ? first.promise : second.promise
    })
    const pendingFirst = activate(controller)
    controller.setQuery({ sort_by: 'cost' })
    assert.equal(calls[0].signal.aborted, true)
    second.resolve(ready(2))
    await controller.refresh()
    first.resolve(ready(1))
    await pendingFirst
    assert.equal(controller.getState().data.summary.request_count, 2)
    assert.equal(controller.getState().query.sort_by, 'cost')
    controller.dispose()
})

for (const failure of [
    { response: { status: 401 } },
    { response: { status: 403 } },
    { usageAuthenticationFailure: true },
]) {
    test(`authorization failure ${JSON.stringify(failure)} clears every cached result and stops polling`, async () => {
        let fail = false
        let calls = 0
        const controller = controllerFor(async () => {
            calls++
            if (fail) throw failure
            return ready()
        })
        await activate(controller)
        fail = true
        await controller.refresh()
        assert.equal(controller.getState().phase, 'forbidden')
        assert.equal(controller.getState().data, null)
        controller.setQuery({ time_range: '1h' })
        controller.setQuery({ time_range: 'today' })
        await controller.refresh()
        assert.equal(controller.getState().data, null)
        assert.equal(calls, 2)
        controller.dispose()
    })
}

test('revocation and disposal ignore even a transport that fails to honor cancellation', async () => {
    const pending = deferred()
    const controller = controllerFor(() => pending.promise)
    const loading = activate(controller)
    controller.setAuthorized(false)
    controller.dispose()
    pending.resolve(ready())
    await loading
    assert.equal(controller.getState().data, null)
})

test('disabled, empty, unattributed-only and malformed results remain distinct', async () => {
    for (const [response, phase] of [
        [{ state: 'disabled' }, 'disabled'],
        [ready(0), 'empty'],
        [ready(1), 'ready'],
        [{ state: 'ready' }, 'error'],
    ]) {
        const controller = controllerFor(async () => response)
        await activate(controller)
        assert.equal(controller.getState().phase, phase)
        controller.dispose()
    }
})

test('refreshes after 30 seconds, pauses hidden/inactive, and resumes without duplicate timers', async (t) => {
    t.mock.timers.enable({ apis: ['setTimeout'] })
    let calls = 0
    const controller = createAPIKeyUsageController({
        load: async () => {
            calls++
            return ready()
        },
        onChange: () => {},
    })
    await activate(controller)
    t.mock.timers.tick(29999)
    assert.equal(calls, 1)
    t.mock.timers.tick(1)
    await controller.refresh()
    assert.equal(calls, 2)
    controller.setVisible(false)
    t.mock.timers.tick(90000)
    assert.equal(calls, 2)
    controller.setVisible(true)
    await controller.refresh()
    assert.equal(calls, 3)
    controller.setActive(false)
    t.mock.timers.tick(60000)
    assert.equal(calls, 3)
    controller.setActive(true)
    controller.setActive(true)
    await controller.refresh()
    assert.equal(calls, 4)
    controller.dispose()
    t.mock.timers.tick(60000)
    assert.equal(calls, 4)
})
