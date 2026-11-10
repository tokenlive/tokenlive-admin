import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
const source = await readFile(new URL('../src/utils/provider-health.js', import.meta.url), 'utf8')
const { providerMetricValue, providerMetricDisplay, providerHealthCards } = await import(
    `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)
const t = (key, values) => (values ? `translated ${key} ${JSON.stringify(values)}` : `translated ${key}`)
const metric = (value, changes = {}) => ({ state: 'ok', value, source: 'memory', scope: 'upstream_call', ...changes })

test('every provider card preserves genuine zero and endpoint inventory uses its own observation', () => {
    const health = {
        calls: metric({ total: 0, success: 0, failure: 0 }),
        success_rate: metric(0),
        avg_latency_ms: metric(0),
        avg_ttft_ms: metric(0),
        otps: metric(0),
        total_tokens: metric(0),
        total_cost: metric(0),
        endpoint_count: metric(0),
    }
    const cards = providerHealthCards(health, t, String)
    assert.deepEqual(
        cards.map((card) => card.value),
        ['0', '0.0%', '0.00s', '0.00s', '0 t/s', '0', '¥0.0000', '0']
    )
})

test('unavailable observations never format null or stale values as zero', () => {
    for (const state of ['no_samples', 'unsupported', 'unavailable', 'error']) {
        assert.equal(providerMetricValue(metric(123, { state })), null)
        const display = providerMetricDisplay(metric(null, { state }), t)
        assert.equal(display.value, '—')
        assert.match(display.hint, new RegExp(`state.${state}`))
    }
    assert.equal(providerMetricDisplay(undefined, t).value, '—')
})

test('one failed observation keeps independent successful cards and partial values visible', () => {
    const cards = providerHealthCards(
        {
            calls: metric({ total: 4, success: 0, failure: 4 }),
            success_rate: metric(0),
            avg_latency_ms: metric(null, { state: 'error' }),
            total_cost: metric(0, { state: 'partial' }),
        },
        t,
        String
    )
    assert.deepEqual(
        cards.map((card) => card.value),
        ['4', '0.0%', '—', '—', '—', '—', '¥0.0000', '—']
    )
})

test('tooltips retain source, request population, legacy identity and actual coverage', () => {
    const values = { start: '2026-11-10T10:10:00+08:00', end: '2026-11-10T11:00:00+08:00' }
    const display = providerMetricDisplay(
        metric(20, {
            state: 'partial',
            source: 'prometheus',
            scope: 'final_provider_requests',
            identity: 'provider_name',
            coverage: values,
        }),
        t
    )
    assert.match(display.hint, /source.prometheus/)
    assert.match(display.hint, /scope.final_provider_requests/)
    assert.match(display.hint, /legacyName/)
    assert.match(display.hint, /10:10:00/)
})
