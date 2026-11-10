import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1600, height: 1200 } })
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
const window = { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T11:00:00+08:00', timezone: 'Asia/Taipei' }
const metric = (value, patch = {}) => ({
    value,
    state: 'ok',
    source: 'memory',
    scope: 'model_requests',
    window,
    coverage: window,
    ...patch,
})
const row = (code, latency) => ({
    model_id: code,
    model_code: code,
    model_name: code,
    requests: metric({ total: 1, success: 0, failure: 1 }),
    success_rate: metric(0),
    avg_latency_ms: latency,
    avg_ttft_ms: metric(null, { state: 'no_samples' }),
    p95_latency_ms: metric(null, { state: 'unsupported', reason: 'no_histogram' }),
    otps: metric(0),
    total_tokens: metric(0),
    total_cost: metric(null, { state: 'error', reason: 'source_unavailable' }),
})
await page.route('**/api/**', async (route) => {
    const request = route.request()
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
    }
    if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    let data = []
    const path = new URL(request.url()).pathname
    if (path.endsWith('/dashboard/snapshot'))
        data = {
            models: {
                state: 'partial',
                window,
                coverage: {},
                data: [
                    row('unknown', metric(null, { state: 'no_samples' })),
                    row('zero', metric(0)),
                    row('slow', metric(2000)),
                ],
            },
            traffic: { state: 'ok', window, coverage: window, data: { series: [] } },
            overview: { state: 'unavailable', window, coverage: {} },
        }
    if (path.endsWith('/dashboard/usage-trends')) data = { state: 'disabled' }
    return route.fulfill({ headers, json: { success: true, data, total: 0 } })
})
try {
    await page.goto(`${process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'}/tests/fixtures/usage-trends.html?full=1`)
    const table = page.locator('.dashboard-ranking-table')
    await table.locator('tr[data-row-key="zero"]').waitFor()
    const zero = table.locator('tr[data-row-key="zero"]')
    assert.match(await zero.innerText(), /0\.0%/)
    assert.match(await zero.innerText(), /0\.00s/)
    assert.match(await zero.innerText(), /0 t\/s/)
    assert.match(await zero.innerText(), /—/)
    const latencyHeader = table.locator('th').filter({ hasText: '平均延迟' })
    await latencyHeader.click()
    assert.deepEqual(
        await table.locator('tbody tr[data-row-key]').evaluateAll((rows) => rows.map((r) => r.dataset.rowKey)),
        ['zero', 'slow', 'unknown']
    )
    await latencyHeader.click()
    assert.deepEqual(
        await table.locator('tbody tr[data-row-key]').evaluateAll((rows) => rows.map((r) => r.dataset.rowKey)),
        ['slow', 'zero', 'unknown']
    )
    await zero.locator('td').nth(3).locator('span').hover()
    await page.locator('.ant-tooltip-inner').filter({ hasText: '直方图' }).waitFor()
    assert.match(await page.locator('.ant-tooltip-inner').filter({ hasText: '直方图' }).innerText(), /直方图/)
    assert.match(await page.locator('.ant-tooltip-inner').filter({ hasText: '直方图' }).innerText(), /业务请求/)
    assert.deepEqual(errors, [])
    console.log('Model observations homepage browser checks passed')
} finally {
    await browser.close()
}
