import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const url = (source) => `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
const snapshot = await readFile(new URL('../src/utils/monitoring-snapshot.js', import.meta.url), 'utf8')
const source = await readFile(new URL('../src/utils/detail-monitoring.js', import.meta.url), 'utf8')
const { createDetailMonitoring } = await import(
    url(source.replace("'./monitoring-snapshot.js'", JSON.stringify(url(snapshot))))
)
const window = { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T11:00:00+08:00', timezone: 'Asia/Taipei' }
const buckets = [
    { start: window.start, end: '2026-11-10T10:30:00+08:00' },
    { start: '2026-11-10T10:30:00+08:00', end: window.end },
]
const target = (patch = {}) => ({
    id: 'id-a',
    code: 'code-a',
    name: 'Provider A',
    time_range: '1h',
    start_time: window.start,
    end_time: window.end,
    timezone: window.timezone,
    ...patch,
})
const reading = (data, patch = {}) => ({ state: 'ok', window, buckets, coverage: window, data, ...patch })
const traffic = (label = 'code-a', total = [null, 0]) => ({
    series: [{ label, success: total, failure: total, total }],
})
const deferred = () => {
    let resolve, reject
    const promise = new Promise((yes, no) => {
        resolve = yes
        reject = no
    })
    return { promise, resolve, reject }
}
function fixture(kind = 'model', overrides = {}) {
    const calls = { snapshots: [], endpoints: [], breakers: [], events: [] }
    const api = {
        dashboard: {
            getSnapshot(query) {
                calls.snapshots.push(structuredClone(query))
                if (overrides.snapshot) return overrides.snapshot(query)
                const code = query.models?.model || query.providers?.provider || query.traffic.model || 'code-a'
                return Promise.resolve({
                    success: true,
                    data: {
                        models: reading([
                            {
                                model_code: code,
                                requests: { state: 'ok', value: { total: 0, success: 0, failure: 0 } },
                            },
                        ]),
                        providers: reading([{ provider_code: code }]),
                        traffic: reading(traffic(code)),
                        performance: reading({ avg_ttft_ms: [null, 0], otps: [0, null] }),
                    },
                })
            },
            getCircuitBreakers() {
                calls.breakers.push({})
                return overrides.breakers ? overrides.breakers() : Promise.resolve({ data: [] })
            },
        },
        endpoint: {
            getEndpointList(params) {
                calls.endpoints.push(structuredClone(params))
                return overrides.endpoints ? overrides.endpoints(params) : Promise.resolve({ data: [], total: 0 })
            },
            getEndpointsByProviderId(id) {
                calls.endpoints.push(id)
                return overrides.endpoints ? overrides.endpoints(id) : Promise.resolve({ data: [] })
            },
        },
        ops: {
            getEvents(params, options) {
                calls.events.push({ ...structuredClone(params), options })
                return overrides.events
                    ? overrides.events(params)
                    : Promise.resolve({
                          data: [{ id: `${params.model_code || params.provider_ref}-${params.current}` }],
                          total: 80,
                      })
            },
        },
    }
    const changes = []
    const session = createDetailMonitoring({ kind, api, onChange: (state) => changes.push(state) })
    return { session, calls, changes }
}

test('model combines compatible readings and keeps endpoint traffic separate without losing zero or null', async () => {
    const { session, calls } = fixture()
    await session.refresh(target())
    assert.equal(calls.snapshots.length, 2)
    assert.deepEqual(
        calls.snapshots.find((q) => q.models),
        {
            at: window.end,
            timezone: 'Asia/Taipei',
            traffic: { range: '1h', start: window.start, model: 'code-a' },
            models: { range: '1h', start: window.start, model: 'code-a', limit: 1 },
            performance: { range: '1h', start: window.start, model: 'code-a' },
        }
    )
    assert.equal(calls.snapshots.filter((q) => q.traffic.group_by === 'endpoint').length, 1)
    const state = session.getState()
    assert.equal(state.ranking.requests.value.total, 0)
    assert.deepEqual(state.traffic.series[0].total, [null, 0])
    assert.deepEqual(state.performance.avg_ttft_ms, [null, 0])
    assert.deepEqual(state.performance.otps, [0, null])
    assert.deepEqual(state.traffic.times, ['10:30', '11:00'])
    assert.equal(state.loading, false)
    assert.equal(state.eventsLoading, false)
})

test('event paging only reads events within the committed window and refresh resets the page', async () => {
    const { session, calls } = fixture('provider')
    const selection = target()
    await session.refresh(selection)
    const before = [calls.snapshots.length, calls.endpoints.length, calls.breakers.length]
    selection.end_time = '2030-01-01T00:00:00Z'
    selection.code = 'mutated'
    await session.pageEvents({ current: 2, pageSize: 50 })
    assert.deepEqual([calls.snapshots.length, calls.endpoints.length, calls.breakers.length], before)
    assert.deepEqual(calls.events[1], {
        start_time: window.start,
        end_time: window.end,
        current: 2,
        pageSize: 50,
        provider_ref: 'code-a',
        provider_name_fallback: 'Provider A',
        options: { enableAbortController: false },
    })
    assert.equal(session.getState().eventTotal, 80)
    assert.deepEqual(session.getState().eventPage, { current: 2, pageSize: 50 })
    await session.refresh(target({ end_time: '2026-11-10T11:01:00+08:00' }))
    assert.deepEqual(session.getState().eventPage, { current: 1, pageSize: 50 })
})

test('older target results cannot overwrite the newer target in any section', async () => {
    const oldMain = deferred(),
        oldTraffic = deferred(),
        oldDirectory = deferred(),
        oldBreakers = deferred(),
        oldEvents = deferred()
    let breakerCalls = 0
    const { session } = fixture('model', {
        snapshot(q) {
            if (q.traffic.model === 'code-a') return q.models ? oldMain.promise : oldTraffic.promise
            return Promise.resolve({
                data: {
                    models: reading([{ model_code: 'code-b' }]),
                    traffic: reading(traffic('code-b', [0, 2])),
                    performance: reading({ avg_ttft_ms: [1, 2], otps: [3, 4] }),
                },
            })
        },
        endpoints: (p) =>
            p.model_id === 'id-a' ? oldDirectory.promise : Promise.resolve({ data: [{ id: 'ep-b' }], total: 1 }),
        breakers: () => (breakerCalls++ === 0 ? oldBreakers.promise : Promise.resolve({ data: [{ id: 'breaker-b' }] })),
        events: (p) =>
            p.model_code === 'code-a' ? oldEvents.promise : Promise.resolve({ data: [{ id: 'event-b' }], total: 1 }),
    })
    const first = session.refresh(target())
    await session.refresh(target({ id: 'id-b', code: 'code-b' }))
    oldMain.resolve({
        data: {
            models: reading([{ model_code: 'code-a' }]),
            traffic: reading(traffic()),
            performance: reading({ avg_ttft_ms: [99], otps: [99] }),
        },
    })
    oldTraffic.resolve({ data: { traffic: reading(traffic('ep-a')) } })
    oldDirectory.resolve({ data: [{ id: 'ep-a' }], total: 1 })
    oldBreakers.resolve({ data: [{ id: 'breaker-a' }] })
    oldEvents.resolve({ data: [{ id: 'event-a' }], total: 1 })
    await first
    const state = session.getState()
    assert.equal(state.ranking.model_code, 'code-b')
    assert.equal(state.traffic.series[0].label, 'code-b')
    assert.deepEqual(state.endpoints, [{ id: 'ep-b' }])
    assert.deepEqual(state.breakers, [{ id: 'breaker-b' }])
    assert.deepEqual(state.events, [{ id: 'event-b' }])
    assert.equal(state.loading, false)
})

test('an older event page cannot overwrite a later page or its loading state', async () => {
    const pageTwo = deferred()
    const { session } = fixture('model', {
        events: (p) =>
            p.current === 2 ? pageTwo.promise : Promise.resolve({ data: [{ id: `page-${p.current}` }], total: 80 }),
    })
    await session.refresh(target())
    const second = session.pageEvents({ current: 2 })
    await session.pageEvents({ current: 3 })
    pageTwo.resolve({ data: [{ id: 'page-2' }], total: 1 })
    await second
    assert.deepEqual(session.getState().events, [{ id: 'page-3' }])
    assert.equal(session.getState().eventTotal, 80)
    assert.equal(session.getState().eventPage.current, 3)
    assert.equal(session.getState().eventsLoading, false)
})

test('stop invalidates pending work and prevents all later publications', async () => {
    const response = deferred()
    const { session, changes } = fixture('provider', { snapshot: () => response.promise })
    const pending = session.refresh(target())
    session.stop()
    const stopped = session.getState(),
        count = changes.length
    response.resolve({ data: { providers: reading([{ provider_code: 'old' }]), traffic: reading(traffic('old')) } })
    await pending
    assert.equal(session.getState(), stopped)
    assert.equal(changes.length, count)
    assert.equal(stopped.loading, false)
    assert.equal(stopped.eventsLoading, false)
    assert.equal(stopped.ranking, null)
    assert.deepEqual(stopped.events, [])
})

test('the same pending refresh coalesces its reads', async () => {
    const response = deferred()
    const { session, calls } = fixture('provider', { snapshot: () => response.promise })
    const first = session.refresh(target()),
        second = session.refresh(target())
    assert.equal(first, second)
    assert.equal(calls.snapshots.length, 1)
    assert.equal(calls.endpoints.length, 1)
    assert.equal(calls.breakers.length, 1)
    assert.equal(calls.events.length, 1)
    response.resolve({ data: { providers: reading([]), traffic: reading(traffic()) } })
    await first
})

test('provider fan-out deduplicates models and retains owned endpoint data when one source fails', async () => {
    const { session, calls } = fixture('provider', {
        endpoints: () =>
            Promise.resolve({
                data: [
                    { id: 'own-1', model: { model_code: 'm1' } },
                    { id: 'own-2', model: { model_code: 'm1' } },
                    { id: 'own-3', model: { model_code: 'm2' } },
                ],
            }),
        snapshot(q) {
            if (q.providers)
                return Promise.resolve({
                    data: { providers: reading([{ provider_code: 'code-a' }]), traffic: reading(traffic()) },
                })
            if (q.traffic.model === 'm2') return Promise.reject(new Error('one model unavailable'))
            return Promise.resolve({
                data: {
                    traffic: reading({
                        series: [
                            ...traffic('own-1').series,
                            ...traffic('own-2', [0, 3]).series,
                            ...traffic('foreign', [100, 100]).series,
                        ],
                    }),
                },
            })
        },
    })
    await session.refresh(target())
    assert.deepEqual(
        calls.snapshots.filter((q) => q.traffic.group_by === 'endpoint').map((q) => q.traffic.model),
        ['m1', 'm2']
    )
    const state = session.getState()
    assert.equal(state.readings.endpointTraffic.state, 'partial')
    assert.deepEqual(
        state.endpointTraffic.series.map((row) => row.label),
        ['own-1', 'own-2']
    )
    assert.deepEqual(state.endpointTraffic.series[0].total, [null, 0])
    assert.deepEqual(state.endpointTraffic.times, ['10:30', '11:00'])
    assert.equal(state.readings.ranking.state, 'ok')
})

test('provider endpoint series with incompatible bucket geometry are not merged', async () => {
    const { session } = fixture('provider', {
        endpoints: () =>
            Promise.resolve({
                data: [
                    { id: 'one', model: { model_code: 'm1' } },
                    { id: 'two', model: { model_code: 'm2' } },
                ],
            }),
        snapshot(q) {
            if (q.providers) return Promise.resolve({ data: { providers: reading([]), traffic: reading(traffic()) } })
            return Promise.resolve({
                data: {
                    traffic: reading(
                        traffic(q.traffic.model === 'm1' ? 'one' : 'two'),
                        q.traffic.model === 'm1'
                            ? {}
                            : { buckets: [{ ...buckets[0], end: '2026-11-10T10:31:00+08:00' }, buckets[1]] }
                    ),
                },
            })
        },
    })
    await session.refresh(target())
    assert.equal(session.getState().readings.endpointTraffic.state, 'partial')
    assert.deepEqual(
        session.getState().endpointTraffic.series.map((row) => row.label),
        ['one']
    )
})

test('directory, runtime and events errors differ from successful empty lists while other results survive', async () => {
    const { session } = fixture('provider', {
        endpoints: () => Promise.reject(new Error('catalog down')),
        breakers: () => Promise.resolve({ success: false, data: [] }),
        events: () => Promise.reject(new Error('events down')),
    })
    await session.refresh(target())
    const state = session.getState()
    for (const key of ['endpoints', 'breakers', 'events', 'endpointTraffic'])
        assert.equal(state.readings[key].state, 'error')
    assert.equal(state.readings.ranking.state, 'ok')
    assert.ok(state.ranking)
    assert.equal(state.eventsLoading, false)
    const healthy = fixture('provider').session
    await healthy.refresh(target())
    for (const key of ['endpoints', 'breakers', 'endpointTraffic'])
        assert.equal(healthy.getState().readings[key].state, 'ok')
})

test('model inventory exhausts pagination and preserves disabled endpoints', async () => {
    const firstPage = Array.from({ length: 100 }, (_, i) => ({ id: `ep-${i}`, enabled: 1 }))
    const disabled = { id: 'disabled', enabled: 0 }
    const { session, calls } = fixture('model', {
        endpoints: (p) => Promise.resolve({ data: p.current === 1 ? firstPage : [disabled], total: 101 }),
    })
    await session.refresh(target())
    assert.deepEqual(
        calls.endpoints.map((p) => p.current),
        [1, 2]
    )
    assert.equal(session.getState().endpoints.length, 101)
    assert.deepEqual(session.getState().endpoints.at(-1), disabled)
})

test('a failed selected reading retains healthy sibling readings from the same snapshot', async () => {
    const { session } = fixture('model', {
        snapshot: (q) =>
            Promise.resolve({
                data: {
                    models: reading([{ model_code: 'code-a' }]),
                    traffic: q.models ? { state: 'error', error: 'source_unavailable' } : reading(traffic('ep-a')),
                    performance: reading({ avg_ttft_ms: [0, null], otps: [null, 0] }),
                },
            }),
    })
    await session.refresh(target())
    assert.equal(session.getState().readings.traffic.state, 'error')
    assert.deepEqual(session.getState().traffic.series, [])
    assert.equal(session.getState().ranking.model_code, 'code-a')
    assert.deepEqual(session.getState().performance.avg_ttft_ms, [0, null])
    assert.equal(session.getState().readings.endpointTraffic.state, 'ok')
})

test('an invalid reading stays local and always finishes the refresh', async () => {
    const { session } = fixture('model', {
        snapshot: (query) =>
            Promise.resolve({
                data: query.models
                    ? {
                          models: reading([{ requests: { state: 'ok', value: { total: 0, success: 0, failure: 0 } } }]),
                          traffic: { state: 'ok', buckets, data: traffic() },
                          performance: reading({ avg_ttft_ms: [0, null], otps: [null, 0] }),
                      }
                    : { traffic: reading(traffic('endpoint')) },
            }),
    })
    await session.refresh(target())
    const state = session.getState()
    assert.equal(state.readings.traffic.state, 'error')
    assert.equal(state.ranking.requests.value.total, 0)
    assert.deepEqual(state.performance.avg_ttft_ms, [0, null])
    assert.equal(state.loading, false)
    assert.equal(state.eventsLoading, false)
})

test('provider endpoint samples that are unavailable remain distinct from read errors', async () => {
    const { session } = fixture('provider', {
        endpoints: () => Promise.resolve({ data: [{ id: 'ep-a', model: { model_code: 'm-a' } }] }),
        snapshot: (query) =>
            Promise.resolve({
                data: query.providers
                    ? {
                          providers: reading([{ provider_code: 'code-a' }]),
                          traffic: reading(traffic()),
                      }
                    : { traffic: { state: 'unavailable', coverage: { reason: 'outside_retention' } } },
            }),
    })
    await session.refresh(target())
    assert.equal(session.getState().readings.endpointTraffic.state, 'unavailable')
    assert.equal(session.getState().readings.traffic.state, 'ok')
})
