import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
const source = await readFile(new URL('../src/utils/monitoring-snapshot.js', import.meta.url), 'utf8')
const { monitoringData, monitoringResponse, monitoringSnapshot } = await import(
    `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
)

test('snapshot display preserves absolute bucket end offsets and missing data', () => {
    const data = { series: [{ label: 'm', total: [null, 0, 1] }] }
    const reading = {
        state: 'partial',
        window: { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T10:12:00+08:00' },
        buckets: [
            { end: '2026-11-10T10:05:00+08:00' },
            { end: '2026-11-10T10:10:00+08:00' },
            { end: '2026-11-10T10:12:00+08:00' },
        ],
        data,
    }
    assert.deepEqual(monitoringData(reading), { ...data, times: ['10:05', '10:10', '10:12'] })
    assert.equal(monitoringData({ ...reading, state: 'unavailable' }), null)
    assert.equal(monitoringData({ ...reading, state: 'error' }), null)
    assert.equal(monitoringResponse({ data: { traffic: reading } }, 'traffic').monitoring, reading)
})

test('partial snapshot preserves successful measurements and clears failed readings', () => {
    const raw = {
        cutoff: '2026-11-10T10:37:42+08:00',
        overview: { state: 'partial', data: { qps: null, daily_requests: 0 } },
        traffic: { state: 'error' },
        models: { state: 'ok', data: [] },
    }
    const result = monitoringSnapshot(raw)
    assert.deepEqual(result.overview, { qps: null, daily_requests: 0 })
    assert.equal(result.trends, null)
    assert.deepEqual(result.model_ranking, [])
    assert.equal(result.monitoring.traffic.state, 'error')
})

test('missing and zero denominators never draw a 100 percent success rate', async () => {
    const { monitoringSuccessRates } = await import(
        `data:text/javascript;base64,${Buffer.from(source).toString('base64')}`
    )
    assert.deepEqual(monitoringSuccessRates([null, 0, 1, 0, 3], [null, 0, 0, 2, 1]), [null, null, 100, 0, 75])
})

test('detail status retains endpoint errors ahead of partial or healthy results', async () => {
    const { monitoringStatus } = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)
    const error = { state: 'error' }
    assert.equal(monitoringStatus([{ state: 'partial' }, null, error, { state: 'ok' }]), error)
    assert.equal(monitoringStatus([{ state: 'ok' }]), null)
})
