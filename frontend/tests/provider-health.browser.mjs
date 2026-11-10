import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
let empty = false
let providerTrafficQuery
const window = { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T11:00:00+08:00', timezone: 'Asia/Taipei' }
const metric = (value, overrides = {}) => ({
    state: 'ok',
    value,
    source: 'memory',
    scope: 'successful_provider_requests',
    window,
    coverage: window,
    ...overrides,
})
await page.route('**/api/**', async (route) => {
    const request = route.request()
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
    }
    if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    let data = []
    if (new URL(request.url()).pathname.endsWith('/dashboard/snapshot')) {
        const query = request.postDataJSON()
        if (query.providers)
            data = {
                providers: {
                    state: 'partial',
                    window,
                    coverage: {},
                    data: [
                        {
                            provider_id: 'p-1',
                            provider_code: 'p-code',
                            provider_name: 'Provider',
                            calls: metric({ total: 4, success: 0, failure: 4 }, { scope: 'upstream_call' }),
                            success_rate: metric(empty ? null : 0, {
                                state: empty ? 'no_samples' : 'ok',
                                reason: empty ? 'zero_denominator' : '',
                                scope: 'upstream_call',
                            }),
                            avg_latency_ms: metric(null, {
                                state: 'error',
                                reason: 'source_unavailable',
                                source: 'prometheus',
                                scope: 'final_provider_requests',
                            }),
                            avg_ttft_ms: metric(null, { state: 'no_samples', reason: 'no_source_samples' }),
                            otps: metric(0),
                            total_tokens: metric(0),
                            total_cost: metric(0),
                            endpoint_count: metric(3, { scope: 'current_inventory', source: 'catalog' }),
                        },
                    ],
                },
            }
        if (query.traffic) {
            providerTrafficQuery = query.traffic
            data.traffic = { state: 'ok', window, coverage: window, data: { series: [] } }
        }
    }
    return route.fulfill({ headers, json: { success: true, data, total: 0 } })
})
try {
    await page.goto(`${process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'}/tests/fixtures/provider-health.html`)
    await page.waitForFunction(() => document.querySelectorAll('.kpi-value')[7]?.textContent === '3')
    assert.deepEqual(await page.locator('.kpi-value').allTextContents(), [
        '4',
        '0.0%',
        '—',
        '—',
        '0 t/s',
        '0',
        '¥0.0000',
        '3',
    ])
    assert.equal(providerTrafficQuery.provider, 'p-1')
    await page.locator('.kpi-value').nth(2).hover()
    const tooltip = page.locator('.ant-tooltip-inner').last()
    await tooltip.waitFor()
    assert.match(await tooltip.innerText(), /此指标读取失败/)
    assert.match(await tooltip.innerText(), /最终供应商归属的请求/)
    assert.match(await tooltip.innerText(), /Prometheus/)
    empty = true
    await page.reload()
    await page.waitForFunction(() => document.querySelectorAll('.kpi-value')[7]?.textContent === '3')
    assert.equal(await page.locator('.kpi-value').nth(1).textContent(), '—')
    await page.locator('.kpi-value').nth(1).hover()
    await page.locator('.ant-tooltip-inner').last().waitFor()
    assert.match(await page.locator('.ant-tooltip-inner').last().innerText(), /成功率未知/)
    assert.deepEqual(errors, [])
    console.log('Provider health browser checks passed')
} finally {
    await browser.close()
}
