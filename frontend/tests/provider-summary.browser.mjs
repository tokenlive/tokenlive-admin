import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
let mode = 'late-detail'
let releaseOld = () => {}
let oldStarted = false
let requestedProviders = []
const window = { start: '2026-11-10T10:00:00+08:00', end: '2026-11-10T11:00:00+08:00', timezone: 'Asia/Taipei' }
const reading = (provider) => ({
    state: 'ok',
    window,
    coverage: window,
    data: [
        {
            provider_id: provider,
            success_rate: {
                state: 'ok',
                value: provider === 'b' ? 80 : 10,
                source: 'memory',
                scope: 'upstream_call',
                window,
                coverage: window,
            },
        },
    ],
})
const deferred = () =>
    new Promise((resolve) => {
        const prior = releaseOld
        releaseOld = () => {
            prior()
            resolve()
        }
    })
await page.route('**/api/**', async (route) => {
    const request = route.request()
    const pathname = new URL(request.url()).pathname
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
    }
    if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    let data = []
    if (pathname.endsWith('/dashboard/snapshot')) {
        const provider = request.postDataJSON().providers.provider
        requestedProviders.push(provider)
        if (mode === 'late-health' && provider === 'code-a') {
            oldStarted = true
            await deferred()
        }
        data = { providers: reading(provider === 'code-b' ? 'b' : 'a') }
    } else if (pathname.endsWith('/dashboard/circuit-breakers')) {
        return route.fulfill({
            status: 503,
            headers,
            json: { success: false, error: { detail: 'status unavailable' } },
        })
    } else if (/\/providers\/[ab]\/quota$/.test(pathname)) {
        const id = pathname.endsWith('/a/quota') ? 'a' : 'b'
        if (mode === 'late-quota' && id === 'a') {
            oldStarted = true
            await deferred()
        }
        data = { provider: 'codex', plan: id === 'a' ? 'alpha' : 'beta', windows: [] }
    } else if (/\/providers\/[ab]\/endpoints$/.test(pathname)) {
        const id = pathname.endsWith('/a/endpoints') ? 'a' : 'b'
        if (mode !== 'unavailable' && id === 'a') await deferred()
        if (mode === 'unavailable') return route.fulfill({ status: 503, headers, json: { success: false } })
        data = Array.from({ length: id === 'a' ? 1 : 2 }, (_, i) => ({
            id: `${id}-${i}`,
            code: `${id}-${i}`,
            enabled: 1,
        }))
    } else if (/\/providers\/[ab]$/.test(pathname)) {
        const id = pathname.endsWith('/a') ? 'a' : 'b'
        if (mode === 'late-detail' && id === 'a') {
            oldStarted = true
            await deferred()
        }
        data = {
            id,
            code: `code-${id}`,
            name: `Provider ${id.toUpperCase()}`,
            protocol: 'openai',
            auth_type: mode === 'late-quota' ? 'oauth_token' : 'api_key',
            url: mode === 'late-quota' ? 'https://chatgpt.com' : 'https://example.test',
            enabled: 1,
            status_points: [{ success_count: 1, fail_count: 0 }],
        }
    }
    return route.fulfill({ headers, json: { success: true, data, total: Array.isArray(data) ? data.length : 0 } })
})
const summary = page.locator('.provider-pulse-rail__stats')
async function openFixture() {
    await page.goto(`${process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'}/tests/fixtures/provider-summary.html`)
}
async function waitOldStarted() {
    for (let i = 0; i < 100 && !oldStarted; i++) await page.waitForTimeout(25)
    assert.ok(oldStarted, 'the old provider request must be in flight')
}
async function switchToB() {
    await page.evaluate(() => window.providerSummaryFixture.open('b'))
    await page.waitForFunction(() =>
        document.querySelector('.provider-pulse-rail__stats')?.textContent.includes('80.0%')
    )
    assert.match(await summary.innerText(), /2\s*\/ 2/)
    assert.match(await summary.innerText(), /熔断端点\s*—/)
    assert.equal(await summary.locator('.is-healthy').count(), 0)
}
try {
    await openFixture()
    await waitOldStarted()
    await switchToB()
    releaseOld()
    await page.waitForTimeout(100)
    assert.match(await page.locator('.info-value').first().innerText(), /Provider B/)
    assert.ok(!requestedProviders.includes('code-a'), 'a discarded detail response must not start a health read')

    mode = 'late-health'
    oldStarted = false
    requestedProviders = []
    releaseOld = () => {}
    await page.reload()
    await waitOldStarted()
    await switchToB()
    releaseOld()
    await page.waitForTimeout(100)
    assert.match(await summary.innerText(), /80\.0%/)
    assert.match(await summary.innerText(), /2\s*\/ 2/)

    mode = 'late-quota'
    oldStarted = false
    releaseOld = () => {}
    await page.reload()
    await waitOldStarted()
    await switchToB()
    await page.waitForFunction(() => document.querySelector('.quota-meta-value')?.textContent === 'beta')
    releaseOld()
    await page.waitForTimeout(100)
    assert.equal(await page.locator('.quota-meta-value').first().innerText(), 'beta')

    mode = 'unavailable'
    releaseOld = () => {}
    await page.reload()
    await page.waitForFunction(() =>
        document.querySelector('.provider-pulse-rail__stats')?.textContent.includes('10.0%')
    )
    assert.match(await summary.innerText(), /—\s*\/ —/)
    assert.match(await summary.innerText(), /熔断端点\s*—/)
    assert.equal(await summary.locator('.is-healthy').count(), 0)
    assert.deepEqual(errors, [])
    console.log('Provider summary browser checks passed')
} finally {
    releaseOld()
    await browser.close()
}
