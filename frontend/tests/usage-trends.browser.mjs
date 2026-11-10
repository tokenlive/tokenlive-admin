import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { tmpdir } from 'node:os'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'
const output = process.env.TEST_SCREENSHOT_DIR || path.join(tmpdir(), 'tokenlive-usage-trends')
await mkdir(output, { recursive: true })
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } })
const errors = []
const queries = []
page.on('pageerror', (error) => errors.push(error.message))
let mode = 'ready'
const metrics = (n) => ({
    requests: n,
    success: n * 0.9,
    failure: n * 0.1,
    input_tokens: n * 400,
    output_tokens: n * 60,
    cached_tokens: n * 320,
    cache_creation_tokens: n * 20,
    total_tokens: n * 460,
    failure_rate: n ? 10 : null,
    avg_latency_ms: n ? 1200 : null,
    avg_ttft_ms: n ? 140 : null,
})
const payload = () => ({
    state: 'ready',
    data_source: 'clickhouse',
    bucket_seconds: 900,
    generated_at: '2026-11-09T12:00:00+08:00',
    window: { start: '2026-11-09T00:00:00+08:00', end: '2026-11-09T12:00:00+08:00', timezone: 'Asia/Taipei' },
    summary: metrics(mode === 'empty' ? 0 : 12800),
    buckets: Array.from({ length: 48 }, (_, i) => ({
        start: `2026-11-09T${String(Math.floor(i / 4)).padStart(2, '0')}:${String((i % 4) * 15).padStart(2, '0')}:00+08:00`,
        ...metrics(mode === 'empty' || i < 18 ? 0 : Math.round((250 + Math.sin(i / 3) * 120 + (i % 5) * 35) / 10) * 10),
    })),
})
await page.route('**/api/**', async (route) => {
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
    }
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    const url = new URL(route.request().url())
    if (!url.pathname.endsWith('/dashboard/usage-trends')) {
        return route.fulfill({ headers, json: { success: true, data: [], total: 0 } })
    }
    queries.push(url.searchParams.get('time_range'))
    if (mode === 'error') return route.fulfill({ status: 503, headers, json: { success: false } })
    if (mode === 'forbidden') return route.fulfill({ status: 403, headers, json: { success: false } })
    return route.fulfill({
        headers,
        json: { success: true, data: mode === 'disabled' ? { state: 'disabled' } : payload() },
    })
})
const open = async (params = '') => {
    const response = page.waitForResponse((response) =>
        new URL(response.url()).pathname.endsWith('/dashboard/usage-trends')
    )
    await page.goto(`${base}/tests/fixtures/usage-trends.html${params}`)
    await response
    if (mode === 'disabled') await page.locator('.usage-trends').waitFor({ state: 'detached' })
    else await page.locator('.trend-summary, .trend-status').waitFor()
}
const panel = page.locator('.usage-trends')
const tab = (text) => panel.locator('.ant-segmented-item').filter({ hasText: text }).click()
try {
    await open()
    assert.equal(queries.at(-1), 'today')
    assert.equal(await panel.locator('.trend-metric').count(), 3)
    await panel.locator('canvas').waitFor()
    await page.screenshot({ path: path.join(output, 'requests-light.png') })
    const count = queries.length
    await tab('用量')
    assert.equal(await panel.locator('.trend-metric').count(), 4)
    assert.equal(queries.length, count, 'category changes must reuse the same snapshot')
    await page.screenshot({ path: path.join(output, 'tokens-light.png') })
    await tab('延迟')
    assert.equal(await panel.locator('.trend-metric').count(), 2)
    assert.match(await panel.locator('.trend-number').first().textContent(), /ms/)
    await tab('错误')
    assert.match(await panel.locator('.trend-number').last().textContent(), /10%/)
    await page.screenshot({ path: path.join(output, 'errors-light.png') })
    await panel.locator('.ant-select').click()
    await page
        .locator('.ant-select-item-option')
        .filter({ hasText: /1.*小时/ })
        .click()
    await page.waitForFunction(() => document.querySelector('.trend-summary') !== null)
    assert.equal(queries.at(-1), '1h')

    await open('?theme=dark&locale=en')
    await tab('Tokens')
    await page.screenshot({ path: path.join(output, 'tokens-dark.png') })
    await page.setViewportSize({ width: 390, height: 850 })
    await page.screenshot({ path: path.join(output, 'tokens-mobile.png') })
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), 'mobile must not overflow')

    mode = 'empty'
    await open()
    assert.equal((await panel.locator('.trend-number').allTextContents()).join(','), '0,0,0')
    await tab('延迟')
    assert.equal((await panel.locator('.trend-number').allTextContents()).join(','), '—,—')
    mode = 'disabled'
    await open()
    assert.equal(await panel.count(), 0, 'disabled ClickHouse must remove the entire card')
    mode = 'error'
    await open()
    assert.equal(await panel.locator('.trend-summary').count(), 0)
    mode = 'ready'
    await panel.getByRole('button', { name: /重\s*试/ }).click()
    await panel.locator('.trend-summary').waitFor()
    mode = 'forbidden'
    await open()
    assert.match(await panel.locator('.trend-status').textContent(), /权限/)
    mode = 'ready'
    await page.setViewportSize({ width: 1440, height: 1100 })
    await open('?full=1')
    assert.equal(await page.locator('.telemetry-card').count(), 6)
    assert.equal(
        await page.locator('.api-key-usage-panel').count(),
        0,
        'the non-Root overview must not expose the key ranking'
    )
    assert.ok((await panel.boundingBox()).y > (await page.locator('.telemetry-card').last().boundingBox()).y)
    await page.screenshot({ path: path.join(output, 'home-overview.png'), animations: 'disabled' })
    mode = 'disabled'
    await open('?full=1')
    assert.equal(await panel.count(), 0)
    assert.equal(await page.locator('.telemetry-card').count(), 6)
    assert.ok(await page.getByText('流量与成功率走势', { exact: true }).isVisible())
    assert.deepEqual(errors, [])
    console.log(`Browser checks passed; screenshots: ${output}`)
} finally {
    await browser.close()
}
