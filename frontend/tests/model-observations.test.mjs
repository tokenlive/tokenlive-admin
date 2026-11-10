import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
const url = (source) => `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
const shared = await readFile(new URL('../src/utils/metric-observation.js', import.meta.url), 'utf8')
const model = await readFile(new URL('../src/utils/model-observations.js', import.meta.url), 'utf8')
const { metricValue, compareMetric } = await import(url(shared))
const { modelObservationCards, modelMetricDisplay } = await import(
    url(model.replace("'./metric-observation.js'", JSON.stringify(url(shared))))
)
const metric = (value, patch = {}) => ({ state: 'ok', value, source: 'memory', scope: 'model_requests', ...patch })
const t = (key) => `translated ${key}`

test('model cards preserve measured zero while independently explaining missing observations', () => {
    const row = {
        requests: metric({ total: 4, success: 0, failure: 4 }),
        success_rate: metric(0),
        avg_latency_ms: metric(0),
        avg_ttft_ms: metric(null, { state: 'no_samples' }),
        otps: metric(0),
        total_tokens: metric(0),
        total_cost: metric(null, { state: 'error', reason: 'source_unavailable' }),
        p95_latency_ms: metric(null, { state: 'unsupported', reason: 'no_histogram' }),
    }
    const cards = modelObservationCards(row, [], { state: 'error' }, t, String)
    assert.deepEqual(
        cards.map((c) => c.value),
        ['4', '0.0%', '0.00s', '—', '0 t/s', '0', '—', '—']
    )
    assert.match(cards[2].hint.join(' '), /no_histogram/)
    assert.match(cards[6].hint.join(' '), /source_unavailable/)
    assert.match(cards[0].hint.join(' '), /model_requests/)
})

test('model table preserves true zero for every numeric column and avoids formatting absent data', () => {
    const row = {
        requests: metric({ total: 0 }),
        success_rate: metric(0),
        avg_latency_ms: metric(0),
        avg_ttft_ms: metric(0),
        otps: metric(0),
        total_tokens: metric(0),
        total_cost: metric(0),
    }
    assert.deepEqual(
        ['request_count', 'success_rate', 'avg_latency', 'avg_ttft', 'otps', 'total_tokens', 'total_cost'].map(
            (key) => modelMetricDisplay(row, key, t, String).value
        ),
        ['0', '0.0%', '0.00s', '0.00s', '0 t/s', '0', '¥0.0000']
    )
    assert.equal(modelMetricDisplay({}, 'total_tokens', t, String).value, '—')
    assert.equal(metricValue(metric(123, { state: 'error' })), null)
})

test('Ant Design ascending and descending sorts both leave unavailable values last', () => {
    const rows = [
        metric(null, { state: 'no_samples' }),
        metric(0),
        metric(8),
        metric(1),
        metric(999, { state: 'error' }),
    ]
    for (const order of ['ascend', 'descend']) {
        const direction = order === 'descend' ? -1 : 1
        const sorted = [...rows].sort((a, b) => direction * compareMetric(a, b, order))
        assert.deepEqual(sorted.map(metricValue), order === 'ascend' ? [0, 1, 8, null, null] : [8, 1, 0, null, null])
    }
    const counts = [metric(null), metric({ total: 0 }), metric({ total: 8 })]
    assert.deepEqual(
        counts.sort((a, b) => compareMetric(a, b, 'ascend', (v) => v.total)).map((m) => metricValue(m)?.total),
        [0, 8, undefined]
    )
})
