import assert from 'node:assert/strict'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const baseURL = process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'

const browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' })
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } })
const page = await context.newPage()
page.setDefaultTimeout(15000)

const errors = []
page.on('pageerror', (error) => errors.push(error.message))

let trendsMode = 'ready'
let rankingMode = 'ready'
let rankingLimit = 10
let userDelayMs = 150
let trendsDelayMs = 200
let rankingDelayMs = 350

const wait = (ms) => new Promise((resolve) => setTimeout(resolve, ms))

const trendsPayload = () => {
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
    return {
        state: 'ready',
        data_source: 'clickhouse',
        bucket_seconds: 900,
        generated_at: '2026-11-17T12:00:00+08:00',
        window: { start: '2026-11-17T00:00:00+08:00', end: '2026-11-17T12:00:00+08:00', timezone: 'UTC+8' },
        summary: metrics(12800),
        buckets: Array.from({ length: 24 }, (_, i) => ({
            start: `2026-11-17T${String(i).padStart(2, '0')}:00:00+08:00`,
            ...metrics(100 + i * 10),
        })),
    }
}

const rankingPayload = (count) => {
    const items = Array.from({ length: count }, (_, idx) => ({
        row_id: `row-${idx + 1}`,
        key_name: `Production Key ${idx + 1}`,
        key_display: `tl_l****${String(1000 + idx)}`,
        source: 'admin_user',
        owner: { kind: 'user', id: `u-${idx}`, name: `Team ${idx + 1}` },
        key_status: idx % 5 === 0 ? 'disabled' : 'enabled',
        metadata_status: 'ready',
        request_count: 1000 - idx * 50,
        success_rate: 99.5,
        total_tokens: 50000 - idx * 2000,
        input_tokens: 40000 - idx * 1600,
        output_tokens: 10000 - idx * 400,
        cached_tokens: 5000,
        cache_creation_tokens: 0,
        total_cost: '1.2500',
        token_share: 10,
    }))
    return {
        state: 'ready',
        data_source: 'clickhouse',
        window: { start: '2026-11-17T00:00:00+08:00', end: '2026-11-17T12:00:00+08:00', timezone: 'UTC+8' },
        generated_at: '2026-11-17T12:00:00+08:00',
        summary: { request_count: 5000, total_tokens: 250000, total_cost: '10.5', key_count: count },
        items,
        unattributed: { request_count: 10, total_tokens: 1000, total_cost: '0.01', token_share: 0.4 },
        warnings: ['unattributed_usage'],
    }
}

await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url())
    const headers = {
        'access-control-allow-origin': '*',
        'access-control-allow-headers': 'Authorization, Content-Type',
        'access-control-allow-methods': 'GET, OPTIONS',
    }
    if (route.request().method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
    if (url.pathname.endsWith('/current/user')) {
        await wait(userDelayMs)
        return route.fulfill({
            headers,
            json: { success: true, data: { id: 'fixture-root', name: 'Root', is_root: true } },
        })
    }
    if (url.pathname.endsWith('/dashboard/usage-trends')) {
        await wait(trendsDelayMs)
        if (trendsMode === 'error') return route.fulfill({ status: 503, headers, json: { success: false } })
        if (trendsMode === 'disabled')
            return route.fulfill({ headers, json: { success: true, data: { state: 'disabled' } } })
        return route.fulfill({ headers, json: { success: true, data: trendsPayload() } })
    }
    if (url.pathname.endsWith('/dashboard/api-key-ranking')) {
        await wait(rankingDelayMs)
        if (rankingMode === 'error') return route.fulfill({ status: 503, headers, json: { success: false } })
        if (rankingMode === 'disabled')
            return route.fulfill({ headers, json: { success: true, data: { state: 'disabled' } } })
        const limit = Number(url.searchParams.get('limit') || rankingLimit)
        return route.fulfill({ headers, json: { success: true, data: rankingPayload(limit) } })
    }
    return route.fulfill({ headers, json: { success: true, data: [], total: 0 } })
})

try {
    // 1. 首屏加载阶段（含 verify=1 鉴权延迟与接口错峰延迟）：验证骨架屏 -> 真实数据高度恒定
    await page.goto(`${baseURL}/tests/fixtures/api-key-usage.html?full=1&verify=1`, { waitUntil: 'domcontentloaded' })
    await page.locator('.usage-trends').waitFor()
    await page.locator('.api-key-usage-placeholder, .api-key-usage-panel').first().waitFor()

    const initialTrendsHeight = (await page.locator('.usage-trends').boundingBox()).height
    const initialRankingBox = await page
        .locator('.api-key-usage-placeholder, .api-key-usage-panel')
        .first()
        .boundingBox()

    await page.locator('.trend-summary').waitFor()
    await page.getByText('Production Key 1', { exact: true }).waitFor()
    await page.locator('.usage-trends canvas').waitFor()

    const readyTrendsHeight = (await page.locator('.usage-trends').boundingBox()).height
    const readyRankingBox = await page.locator('.api-key-usage-panel').boundingBox()

    assert.ok(
        Math.abs(readyTrendsHeight - initialTrendsHeight) <= 4,
        `UsageTrends height shifted: initial=${initialTrendsHeight}, ready=${readyTrendsHeight}`
    )
    assert.ok(
        Math.abs(readyRankingBox.height - initialRankingBox.height) <= 4,
        `APIKeyUsage height shifted: initial=${initialRankingBox.height}, ready=${readyRankingBox.height}`
    )

    // 2. 验证 APIKeyUsageRanking 内部滚动且切换 Top 20 时外层卡片高度不变
    const scrollCheck = await page.locator('.api-key-usage-panel .ant-table-content').evaluate((el) => ({
        scrollHeight: el.scrollHeight,
        clientHeight: el.clientHeight,
    }))
    assert.ok(scrollCheck.scrollHeight > scrollCheck.clientHeight, '10 rows should scroll inside fixed table body')

    await page.locator('.api-key-usage-panel [aria-label="排行条数"]').click()
    await page.locator('.ant-select-dropdown:visible').getByText('Top 20', { exact: true }).click()
    // 切换请求途中仍保留旧表格，不应退回骨架屏
    assert.equal(await page.locator('.usage-skeleton').count(), 0)
    await page.getByText('Production Key 20', { exact: true }).waitFor()
    const top20RankingHeight = (await page.locator('.api-key-usage-panel').boundingBox()).height
    assert.ok(
        Math.abs(top20RankingHeight - readyRankingBox.height) <= 2,
        `APIKeyUsage height changed on Top 20: ready=${readyRankingBox.height}, top20=${top20RankingHeight}`
    )

    // 3. 验证 UsageTrends 切换时间范围和指标分类时高度稳定，且 SWR 不清空图表
    for (const label of ['用量', '延迟', '错误', '请求']) {
        await page.locator('.usage-trends .ant-segmented-item').filter({ hasText: label }).click()
        const h = (await page.locator('.usage-trends').boundingBox()).height
        assert.ok(
            Math.abs(h - readyTrendsHeight) <= 4,
            `UsageTrends height shifted on ${label}: ${h} vs ${readyTrendsHeight}`
        )
    }
    await page.locator('.usage-trends .ant-select').click()
    await page.locator('.ant-select-dropdown:visible').getByText('最近 7 天', { exact: true }).click()
    assert.equal(await page.locator('.trend-skeleton').count(), 0, 'SWR should not revert to skeleton on range switch')
    await page.waitForTimeout(300)
    const afterSwitchTrendsHeight = (await page.locator('.usage-trends').boundingBox()).height
    assert.ok(Math.abs(afterSwitchTrendsHeight - readyTrendsHeight) <= 4)

    // 3b. 验证移动端 (390x844) 横向滚动与表格容器存在性
    await page.setViewportSize({ width: 390, height: 844 })
    await page.waitForTimeout(100)
    const mobileScroll = await page
        .locator('.api-key-usage-panel .ant-table-content')
        .evaluate((el) => el.scrollWidth > el.clientWidth)
    assert.equal(mobileScroll, true, 'mobile table should preserve horizontal scroll on .ant-table-content')
    assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth + 1))
    await page.setViewportSize({ width: 1440, height: 1000 })

    // 4. 验证记忆 disabled 状态：当服务端返回 disabled 后，刷新页面不再闪现骨架占位
    trendsMode = 'disabled'
    rankingMode = 'disabled'
    await page.locator('.usage-trends .ant-select').click()
    await page.locator('.ant-select-dropdown:visible').getByText('最近 1 小时', { exact: true }).click()
    await page.locator('.usage-trends').waitFor({ state: 'detached' })

    await page.locator('.api-key-usage-panel [aria-label="统计时间范围"]').click()
    await page.locator('.ant-select-dropdown:visible').getByText('最近 1 小时', { exact: true }).click()
    await page.locator('.api-key-usage-panel').waitFor({ state: 'detached' })

    await page.goto(`${baseURL}/tests/fixtures/api-key-usage.html?full=1&verify=1`, { waitUntil: 'domcontentloaded' })
    await page.waitForTimeout(100)
    assert.equal(
        await page.locator('.usage-trends').count(),
        0,
        'remembered disabled usage-trends should not flash skeleton'
    )
    assert.equal(
        await page.locator('.api-key-usage-placeholder, .api-key-usage-panel').count(),
        0,
        'remembered disabled api-key-ranking should not flash placeholder'
    )

    assert.deepEqual(errors, [])
    console.log(
        JSON.stringify({
            passed: true,
            initialTrendsHeight,
            readyTrendsHeight,
            initialRankingHeight: initialRankingBox.height,
            readyRankingHeight: readyRankingBox.height,
            top20RankingHeight,
        })
    )
} finally {
    await browser.close()
}
