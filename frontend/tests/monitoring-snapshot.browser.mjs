import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { tmpdir } from 'node:os'
import path from 'node:path'
const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
let mode = 'partial'
let releaseOld
let oldStarted
const started = new Promise((resolve) => {
    oldStarted = resolve
})
const snapshot = (query) => {
    const range = query.traffic?.range || '1h'
    const window = { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T10:10:00+08:00', timezone: 'Asia/Taipei' }
    const buckets = [
        { start: window.start, end: '2026-11-10T10:05:00+08:00' },
        { start: '2026-11-10T10:05:00+08:00', end: window.end },
    ]
    const requests = range === '6h' ? 601 : range === '24h' ? 2401 : 0
    return {
        cutoff: window.end,
        overview: {
            state: mode === 'error' ? 'error' : 'partial',
            window,
            coverage: {},
            data:
                mode === 'error'
                    ? undefined
                    : {
                          qps: null,
                          daily_requests: requests,
                          daily_prompt_tokens: null,
                          daily_completion_tokens: null,
                          daily_cost: null,
                          avg_latency_ms: null,
                          avg_ttft_ms: null,
                          active_circuit_breakers: [],
                      },
        },
        traffic: {
            state: mode === 'error' ? 'error' : 'partial',
            window,
            buckets,
            coverage: {},
            data:
                mode === 'error'
                    ? undefined
                    : { series: [{ label: 'global', success: [null, 2], failure: [null, 1], total: [null, 3] }] },
        },
        models: { state: 'error', window, coverage: {}, error: 'source_unavailable' },
    }
}
await page.route('**/api/**', async (route) => {
    const request = route.request()
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
    }
    if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    const url = new URL(request.url())
    let data = []
    if (url.pathname.endsWith('/dashboard/snapshot')) {
        const query = request.postDataJSON()
        if (mode === 'race' && query.traffic?.range === '6h') {
            oldStarted()
            await new Promise((resolve) => {
                releaseOld = resolve
            })
        }
        data = snapshot(query)
    } else if (url.pathname.endsWith('/dashboard/usage-trends')) data = { state: 'disabled' }
    return route.fulfill({ headers, json: { success: true, data, total: 0 } })
})
const open = async () => {
    await page.goto(`${process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'}/tests/fixtures/usage-trends.html?full=1`)
    await page.locator('.telemetry-card').first().waitFor()
    await page.waitForFunction(() => document.querySelectorAll('.monitoring-status').length === 3)
}
try {
    await open()
    assert.equal(await page.locator('.usage-trends').count(), 0)
    assert.match(await page.locator('.telemetry-card').first().innerText(), /0/)
    assert.equal(await page.locator('.telemetry-card').nth(1).locator('.animated-number').count(), 0)
    assert.match(await page.locator('.telemetry-card').nth(1).innerText(), /—/)
    assert.match(await page.locator('.monitoring-status').first().innerText(), /不完整/)
    mode = 'race'
    const select = page.locator('.ant-select').first()
    await select.click()
    await page
        .locator('.ant-select-item-option')
        .filter({ hasText: /6.*小时/ })
        .click()
    await started
    await select.click()
    await page
        .locator('.ant-select-item-option')
        .filter({ hasText: /24.*小时/ })
        .click()
    await page.waitForFunction(
        () => document.querySelector('.telemetry-card .animated-number')?.textContent === '2,401'
    )
    releaseOld()
    // Await the old response and the number animation; it must not replace the new selection.
    await page.waitForTimeout(1000)
    assert.equal(await page.locator('.telemetry-card').first().locator('.animated-number').textContent(), '2,401')
    mode = 'error'
    await open()
    assert.equal(await page.locator('.telemetry-card .animated-number').count(), 0)
    await page.screenshot({ path: path.join(tmpdir(), 'tokenlive-monitoring-snapshot.png'), animations: 'disabled' })
    assert.deepEqual(errors, [])
    console.log('Monitoring snapshot browser checks passed')
} finally {
    await browser.close()
}
