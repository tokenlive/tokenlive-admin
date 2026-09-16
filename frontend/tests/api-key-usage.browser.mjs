import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { tmpdir } from 'node:os'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const baseURL = process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'
const outputDir = process.env.TEST_SCREENSHOT_DIR || path.join(tmpdir(), 'tokenlive-api-key-usage')
await mkdir(outputDir, { recursive: true })
const browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' })
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
const page = await context.newPage()
page.setDefaultTimeout(15000)
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
let mode = 'ready'
const requests = []
const metrics = (request_count, total_tokens, total_cost, token_share) => ({
    request_count,
    total_tokens,
    total_cost,
    token_share,
    success_count: request_count,
    input_tokens: total_tokens * 0.8,
    output_tokens: total_tokens * 0.2,
    cached_tokens: total_tokens * 0.1,
    cache_creation_tokens: 0,
    success_rate: 100,
})
const payload = () => ({
    state: 'ready',
    data_source: 'clickhouse',
    window: { start: '2026-09-16T00:00:00+08:00', end: '2026-09-16T12:00:00+08:00', timezone: 'UTC+8' },
    generated_at: '2026-09-16T12:00:00+08:00',
    summary: { ...metrics(40, 100000, '3.72654', 100), key_count: 3 },
    items: [
        {
            ...metrics(28, 70000, '3.12654', 70),
            row_id: 'row-a',
            key_name: 'Production Agent',
            key_display: 'tl_l****7f2a',
            source: 'admin_user',
            owner: { kind: 'user', id: 'u', name: '平台应用组' },
            key_status: 'enabled',
            metadata_status: 'ready',
        },
        {
            ...metrics(8, 20000, '0.4', 20),
            row_id: 'row-b',
            key_name: '研发知识库',
            key_display: 'tl_l****91bc',
            source: 'portal_workspace',
            owner: { kind: 'workspace', id: 'ws-research', name: '' },
            key_status: 'disabled',
            metadata_status: 'ready',
        },
        {
            ...metrics(3, 5000, '0.125', 5),
            row_id: 'row-c',
            key_name: 'Legacy batch jobs',
            key_display: 'sk-****5e10',
            source: 'tenant',
            owner: { kind: 'tenant', id: 'legacy', name: 'Legacy team' },
            key_status: 'deleted',
            metadata_status: 'ready',
        },
    ],
    unattributed: metrics(1, 5000, '0.075', 5),
    warnings: ['unattributed_usage'],
})
await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url())
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
        'access-control-allow-methods': 'GET, OPTIONS',
    }
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    if (url.pathname.endsWith('/current/user')) {
        return route.fulfill({
            headers,
            json: {
                success: true,
                data: {
                    id: 'fixture-user',
                    name: 'Fixture user',
                    is_root: new URL(page.url()).searchParams.get('root') !== '0',
                },
            },
        })
    }
    if (url.pathname.endsWith('/dashboard/api-key-ranking')) {
        requests.push(Object.fromEntries(url.searchParams))
        if (mode === 'error' || mode === 'forbidden') {
            return route.fulfill({
                status: mode === 'error' ? 503 : 403,
                headers,
                json: { success: false, error: { detail: 'fixture' } },
            })
        }
        const data = payload()
        if (mode === 'disabled') data.state = 'disabled'
        if (mode === 'empty') {
            data.items = []
            data.summary = { ...metrics(0, 0, '0', null), key_count: 0 }
            data.unattributed = metrics(0, 0, '0', null)
            data.warnings = []
        }
        return route.fulfill({ headers, json: { success: true, data } })
    }
    return route.fulfill({ headers, json: { success: true, data: [], total: 0 } })
})
const navigate = (query = '') => page.goto(`${baseURL}/tests/fixtures/api-key-usage.html${query}`)
const resume = () =>
    page.evaluate(() => {
        for (const value of ['hidden', 'visible']) {
            Object.defineProperty(document, 'visibilityState', { configurable: true, value })
            document.dispatchEvent(new Event('visibilitychange'))
        }
    })
const waitText = async (text) => page.getByText(text, { exact: true }).first().waitFor()
try {
    await navigate()
    await waitText('Production Agent')
    const before = page.url()
    await page.getByText('Production Agent', { exact: true }).click()
    assert.equal(page.url(), before)
    assert.equal(await page.locator('.ant-drawer, .ant-modal').count(), 0)
    assert.equal(await page.locator('.ant-pagination').count(), 0)
    await page.locator('.usage-token-detail').first().focus()
    await page.locator('.ant-tooltip:visible').waitFor()
    assert.ok((await page.locator('.ant-tooltip:visible').innerText()).includes('缓存命中'))
    await page.locator('.usage-token-detail').first().blur()
    await page.screenshot({
        path: path.join(outputDir, 'api-key-usage-desktop.png'),
        fullPage: true,
        animations: 'disabled',
    })
    await page.locator('[aria-label="排序指标"]').click()
    await Promise.all([
        page.waitForResponse((response) => response.url().includes('sort_by=cost')),
        page.locator('.ant-select-dropdown:visible').getByText('费用降序', { exact: true }).click(),
    ])
    assert.equal(requests.at(-1).sort_by, 'cost')

    mode = 'error'
    await resume()
    await waitText('更新失败，当前显示上次成功结果')
    assert.equal(await page.getByText('Production Agent', { exact: true }).count(), 1)
    await page.locator('[aria-label="统计时间范围"]').click()
    await page.locator('.ant-select-dropdown:visible').getByText('最近 7 天', { exact: true }).click()
    await waitText('数据暂不可用')
    assert.equal(await page.getByText('Production Agent', { exact: true }).count(), 0)

    mode = 'empty'
    await navigate()
    await waitText('该时段暂无调用')
    mode = 'disabled'
    await navigate()
    await waitText('尚未启用 API Key 用量统计')
    mode = 'ready'
    await navigate('?root=0')
    const count = requests.length
    await page.locator('#usage-denied').waitFor()
    assert.equal(requests.length, count)
    assert.equal(await page.locator('.api-key-usage-panel').count(), 0)

    await navigate()
    await waitText('Production Agent')
    mode = 'forbidden'
    await resume()
    await waitText('用量查看权限未通过验证，请重新验证登录身份')
    assert.equal(await page.getByText('Production Agent', { exact: true }).count(), 0)

    mode = 'ready'
    await navigate('?locale=en-us&theme=dark')
    await waitText('Production Agent')
    await waitText('API Key Usage')
    await page.screenshot({
        path: path.join(outputDir, 'api-key-usage-dark-en.png'),
        fullPage: true,
        animations: 'disabled',
    })
    await page.setViewportSize({ width: 390, height: 844 })
    await navigate()
    await waitText('Production Agent')
    assert.ok(await page.locator('.ant-table-content').evaluate((el) => el.scrollWidth > el.clientWidth))
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1))
    await page.screenshot({
        path: path.join(outputDir, 'api-key-usage-mobile.png'),
        fullPage: true,
        animations: 'disabled',
    })

    await page.setViewportSize({ width: 1440, height: 1000 })
    await navigate('?full=1')
    await waitText('Production Agent')
    const placement = await page.evaluate(() => {
        const model = document.querySelector('.dashboard-ranking-panel')
        const keys = document.querySelector('.api-key-usage-panel')
        return !!model && !!keys && !!(model.compareDocumentPosition(keys) & Node.DOCUMENT_POSITION_FOLLOWING)
    })
    assert.equal(placement, true)
    await page.screenshot({
        path: path.join(outputDir, 'api-key-usage-home.png'),
        fullPage: true,
        animations: 'disabled',
    })
    await navigate('?full=1&verify=1')
    await waitText('Production Agent')
    assert.deepEqual(errors, [])
    console.log(JSON.stringify({ passed: true, scenarios: 12, requests: requests.length, screenshots: outputDir }))
} finally {
    await browser.close()
}
