import assert from 'node:assert/strict'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const browser = await chromium.launch({ headless: true, channel: 'chrome' })
const base = process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'
const deferred = () => {
    let resolve
    const promise = new Promise((done) => (resolve = done))
    return { promise, resolve }
}
const flushRendering = (page) =>
    page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))))

try {
    for (const kind of ['model', 'provider']) {
        const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } })
        const errors = []
        const requests = []
        const delayed = new Map()
        let failLists = false
        page.on('pageerror', (error) => errors.push(error.message))
        const marker = (code) =>
            ({ 'p-code': 1, 'p-a': 11, 'p-b': 22, 'p-stop': 33, 'p-fail': 44, 'p-zero': 0 })[code] ?? 1
        const headers = {
            'access-control-allow-origin': '*',
            'access-control-allow-headers': 'Authorization, Content-Type',
        }
        const reading = (query, data) => {
            const start = query.traffic?.start || query.models?.start || query.providers?.start
            const window = { start, end: query.at, timezone: query.timezone }
            return {
                state: 'ok',
                source: 'memory',
                window,
                coverage: window,
                buckets: [{ start, end: query.at }],
                data,
            }
        }
        const snapshot = (query) => {
            const code = query.models?.model || query.providers?.provider || query.traffic.model
            const count = marker(code)
            const metric = (value, scope = 'successful_provider_requests') => ({
                value,
                state: 'ok',
                source: 'memory',
                scope,
                window: reading(query, null).window,
                coverage: reading(query, null).window,
            })
            const data = {
                cutoff: query.at,
                traffic: reading(query, {
                    series: [
                        {
                            label: query.traffic.group_by === 'endpoint' ? 'ep-1' : code,
                            success: [0],
                            failure: [count],
                            total: [count],
                        },
                    ],
                }),
            }
            if (query.models) {
                data.models = reading(query, [
                    {
                        model_code: code,
                        request_count: count,
                        success_rate: 0,
                        avg_latency_ms: 1000,
                        avg_ttft_ms: 100,
                        otps: 0,
                        total_tokens: 0,
                        total_cost: 0,
                    },
                ])
                data.performance = reading(query, { avg_ttft_ms: [0], otps: [0] })
            }
            if (query.providers) {
                data.providers = reading(query, [
                    {
                        provider_id: query.traffic.provider,
                        provider_code: code,
                        calls: metric({ total: count, success: 0, failure: count }, 'upstream_call'),
                        success_rate: metric(0, 'upstream_call'),
                        avg_latency_ms: metric(1000),
                        avg_ttft_ms: metric(100),
                        otps: metric(0),
                        total_tokens: metric(0),
                        total_cost: metric(0),
                        endpoint_count: metric(1, 'current_inventory'),
                    },
                ])
            }
            return data
        }
        await page.route('**/api/**', async (route) => {
            const request = route.request()
            if (request.method() === 'OPTIONS') return route.fulfill({ status: 204, headers })
            const url = new URL(request.url())
            const params = Object.fromEntries(url.searchParams)
            const query = request.method() === 'POST' ? request.postDataJSON() : undefined
            requests.push({ path: url.pathname, params, query })
            const send = (data, total = 0) => route.fulfill({ headers, json: { success: true, data, total } })
            if (url.pathname.endsWith('/dashboard/snapshot')) {
                const code = query.models?.model || query.providers?.provider
                const gate = delayed.get(code)
                if (gate) {
                    delayed.delete(code)
                    gate.started.resolve()
                    await gate.release.promise
                }
                return send(snapshot(query))
            }
            const isEvents = url.pathname.endsWith('/ops/events')
            if (failLists) return route.fulfill({ status: 503, headers, json: { success: false } })
            if (isEvents) {
                const current = Number(params.current)
                const code = params.model_code || params.provider_ref
                return send(
                    Array.from({ length: 20 }, (_, index) => ({
                        id: `${code}-${current}-${index}`,
                        event_time: '2026-11-10T10:00:00Z',
                        event_type: 'rate_limit',
                        model_code: code,
                        endpoint_code: 'Endpoint',
                        message: `Event ${current}-${index}`,
                    })),
                    45
                )
            }
            if (url.pathname.endsWith('/dashboard/circuit-breakers')) return send([])
            return send([{ id: 'ep-1', code: 'Endpoint', model: { model_code: 'shared-model' } }], 1)
        })
        const select = (patch) => page.evaluate((value) => window.detailMonitoringFixture.select(value), patch)
        const firstKPI = page.locator('.kpi-value').first()
        const expectCount = (value) =>
            page.waitForFunction((text) => document.querySelector('.kpi-value')?.textContent === text, String(value))
        const mainQueries = () => requests.filter(({ query }) => query?.models || query?.providers)
        try {
            await page.goto(`${base}/tests/fixtures/provider-health.html?kind=${kind}`)
            await expectCount(1)
            await page.locator('tr[data-row-key="p-code-1-0"]').waitFor()
            await page.waitForFunction(() => !document.querySelector('button.ant-btn-loading'))
            assert.equal(mainQueries().length, 1, `${kind}: primary readings share one request`)
            const initialQuery = mainQueries()[0].query
            assert.ok(initialQuery.traffic)
            if (kind === 'model') {
                assert.ok(initialQuery.models)
                assert.ok(initialQuery.performance)
            } else assert.ok(initialQuery.providers)
            assert.equal(
                requests.filter(({ query }) => query).length,
                2,
                `${kind}: endpoint traffic remains one separate request`
            )

            const initialEvents = requests.find(({ path }) => path.endsWith('/ops/events'))
            const initialRequests = requests.length
            const snapshotsBeforePaging = requests.filter(({ query }) => query).length
            await page.locator('.ant-pagination-item-2').click()
            await page.locator('tr[data-row-key="p-code-2-0"]').waitFor()
            const pageTwo = requests.filter(({ path }) => path.endsWith('/ops/events')).at(-1)
            assert.equal(requests.length, initialRequests + 1, `${kind}: event pagination only loads events`)
            assert.equal(requests.filter(({ query }) => query).length, snapshotsBeforePaging)
            assert.equal(pageTwo.params.start_time, initialEvents.params.start_time)
            assert.equal(pageTwo.params.end_time, initialEvents.params.end_time)
            assert.equal(pageTwo.params.current, '2')

            const old = { started: deferred(), release: deferred() }
            delayed.set('p-a', old)
            await select({ id: 'a', code: 'p-a', name: 'Provider A' })
            await old.started.promise
            await select({ id: 'b', code: 'p-b', name: 'Provider B' })
            await expectCount(22)
            const oldResponse = page.waitForResponse(
                (response) => response.request().postData()?.includes('p-a') === true
            )
            old.release.resolve()
            await (await oldResponse).finished()
            await flushRendering(page)
            assert.equal(
                await firstKPI.textContent(),
                '22',
                `${kind}: an old identity cannot overwrite the current target`
            )

            const stopped = { started: deferred(), release: deferred() }
            delayed.set('p-stop', stopped)
            await select({ id: 'stop', code: 'p-stop', name: 'Stopped provider' })
            await stopped.started.promise
            await select({ active: false })
            await expectCount('—')
            const stoppedResponse = page.waitForResponse(
                (response) => response.request().postData()?.includes('p-stop') === true
            )
            stopped.release.resolve()
            await (await stoppedResponse).finished()
            await flushRendering(page)
            assert.equal(await firstKPI.textContent(), '—', `${kind}: inactive tabs reject pending results`)
            await select({ active: true })
            await expectCount(33)

            await select({ id: 'zero', code: 'p-zero', name: 'Zero traffic' })
            await expectCount(0)
            await page.locator('.monitor-panel .x-chart').first().locator('canvas').first().waitFor()
            assert.equal(
                await page.locator('.monitor-panel').first().locator('.ant-empty').count(),
                0,
                `${kind}: observed zero traffic remains a chart`
            )
            assert.equal(await page.locator('.kpi-value').nth(5).textContent(), '0')
            assert.equal(await page.locator('.kpi-value').nth(6).textContent(), '¥0.0000')

            failLists = true
            await select({ id: 'fail', code: 'p-fail', name: 'Unavailable lists' })
            await expectCount(44)
            await page.locator('.events-panel .ant-alert-error').waitFor()
            await page.waitForFunction(() => !document.querySelector('button.ant-btn-loading'))
            assert.ok(
                (await page.locator('.monitoring-status.ant-alert-error').count()) >= 2,
                `${kind}: list failures stay visible`
            )
            assert.equal(
                await page.locator('.events-panel .ant-empty').count(),
                0,
                `${kind}: failed events are not presented as an empty result`
            )
            assert.ok(
                (await page.locator('.monitor-panel').first().locator('.x-chart').count()) > 0,
                `${kind}: healthy traffic survives other failures`
            )
            if (kind === 'model') assert.equal(await page.locator('.kpi-value').nth(7).textContent(), '—')
            assert.deepEqual(errors, [], `${kind}: no runtime page errors`)
            console.log(`${kind} detail monitoring browser checks passed`)
        } finally {
            for (const gate of delayed.values()) gate.release.resolve()
            await page.close()
        }
    }
} finally {
    await browser.close()
}
