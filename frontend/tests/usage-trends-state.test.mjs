import test from 'node:test'
import assert from 'node:assert/strict'
import { createUsageTrendsController } from '../src/utils/usage-trends-state.js'
import { buildUsageTrendsChart, formatTrendValue } from '../src/utils/usage-trends-chart.js'

const ready = (requests = 1) => ({
    state: 'ready',
    summary: { requests },
    buckets: [],
    window: { start: '2026-11-09T00:00:00+08:00', end: '2026-11-09T12:00:00+08:00' },
})
const tick = () => new Promise((resolve) => setImmediate(resolve))
const controller = (load, options = {}) =>
    createUsageTrendsController({
        load,
        setTimeoutFn: () => 1,
        clearTimeoutFn: () => {},
        ...options,
    })
const activate = (c) => {
    c.setActive(true)
    c.setVisible(true)
    c.setAuthorized(true)
    return c.refresh()
}

test('late old-window responses cannot replace the current snapshot', async () => {
    const pending = []
    const c = controller((query, { signal }) => new Promise((resolve) => pending.push({ query, signal, resolve })))
    activate(c)
    assert.equal(pending[0].query.time_range, 'today')
    c.setTimeRange('7d')
    assert.equal(pending[0].signal.aborted, true)
    assert.equal(c.getState().data, null)
    pending[1].resolve(ready(7))
    await tick()
    pending[0].resolve(ready(1))
    await tick()
    assert.equal(c.getState().data.summary.requests, 7)
    assert.equal(c.getState().timeRange, '7d')
    c.dispose()
})

test('polls at 30 seconds only when authorized, active and visible', async () => {
    let calls = 0
    const timers = new Map()
    let id = 0
    const c = controller(
        async () => {
            calls++
            return ready()
        },
        {
            setTimeoutFn(fn, ms) {
                assert.equal(ms, 30000)
                timers.set(++id, fn)
                return id
            },
            clearTimeoutFn(timer) {
                timers.delete(timer)
            },
        }
    )
    c.setActive(true)
    c.setVisible(true)
    await c.refresh()
    assert.equal(calls, 0)
    await activate(c)
    assert.equal(calls, 1)
    assert.equal(timers.size, 1)
    const [timer, fn] = [...timers.entries()][0]
    timers.delete(timer)
    fn()
    await tick()
    assert.equal(calls, 2)
    c.setVisible(false)
    assert.equal(timers.size, 0)
    await c.refresh()
    assert.equal(calls, 2)
    c.setVisible(true)
    await tick()
    assert.equal(calls, 3)
    c.setActive(false)
    assert.equal(timers.size, 0)
    c.dispose()
})

test('disabled, empty, error and revoked authorization stay distinct', async () => {
    let response = ready(0)
    let fail = false
    const c = controller(async () => {
        if (fail) throw { response: { status: fail } }
        return response
    })
    await activate(c)
    assert.equal(c.getState().phase, 'empty')
    response = { state: 'disabled' }
    await c.refresh()
    assert.equal(c.getState().phase, 'disabled')
    assert.equal(c.getState().data, null)
    fail = 503
    await c.refresh()
    assert.equal(c.getState().phase, 'error')
    assert.equal(c.getState().data, null)
    fail = false
    response = ready(9)
    await c.refresh()
    assert.equal(c.getState().phase, 'ready')
    c.setAuthorized(false)
    assert.equal(c.getState().data, null)
    c.setAuthorized(true)
    await c.refresh()
    fail = 403
    await c.refresh()
    assert.equal(c.getState().phase, 'forbidden')
    assert.equal(c.getState().data, null)
    c.dispose()
})

test('chart preserves zero latency and null samples and uses independent failure-rate axis', () => {
    const data = {
        buckets: [
            {
                start: '2026-11-09T00:00:00+08:00',
                avg_latency_ms: 0,
                avg_ttft_ms: null,
                failure: 0,
                failure_rate: null,
            },
            {
                start: '2026-11-09T00:01:00+08:00',
                avg_latency_ms: null,
                avg_ttft_ms: 20,
                failure: 1,
                failure_rate: 100,
            },
        ],
    }
    const options = { data, t: (v) => v, timeRange: 'today', dark: false, locale: 'en-US' }
    const latency = buildUsageTrendsChart({ ...options, category: 'latency' })
    assert.deepEqual(latency.series[0].data, [0, null])
    assert.deepEqual(latency.series[1].data, [null, 20])
    assert.equal(latency.series[0].connectNulls, false)
    const errors = buildUsageTrendsChart({ ...options, category: 'errors' })
    assert.equal(errors.yAxis.length, 2)
    assert.equal(errors.series[1].yAxisIndex, 1)
    assert.deepEqual(errors.series[1].data, [null, 100])
    assert.equal(formatTrendValue(null), '—')
    assert.equal(formatTrendValue(0, 'ms'), '0 ms')
    assert.deepEqual(latency.xAxis.data, ['00:00', '00:01'])
})
