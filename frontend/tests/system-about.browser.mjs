// Mock only HTTP; exercise the real layout, API client and composable.
// PLAYWRIGHT_MODULE=/path/to/playwright node tests/system-about.browser.mjs
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const baseURL = process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'
const outputDir = process.env.TEST_SCREENSHOT_DIR || '/tmp/tokenlive-system-about'
await mkdir(outputDir, { recursive: true })
const browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' })
const summaryFixture = () => ({
    identity: { edition: 'professional', install_channel: 'release', build: { version: 'v1.2.3', kind: 'release' } },
    gateway: {
        status: 'observed',
        scope: 'shared',
        groups: [
            { version: 'v1.2.3', build_kind: 'release', count: 2 },
            { version: 'dev-local-long-build-20260912-abcdef0123456789', build_kind: 'dev', count: 1 },
        ],
    },
    can_manage_updates: true,
})
const sourceFixture = (component, version = 'v1.3.0') => ({
    status: 'ready',
    stale: false,
    last_attempt: '2026-09-12T00:00:00Z',
    last_success: '2026-09-12T00:00:00Z',
    candidate: { version, release_url: `https://github.com/tokenlive/tokenlive-${component}/releases/tag/${version}` },
})
const updatesFixture = () => ({
    enabled: true,
    components: [
        { component: 'admin', current: 'v1.2.3', latest: 'v1.3.0', state: 'available', source: sourceFixture('admin') },
        {
            component: 'gateway',
            current: 'v1.2.3',
            latest: 'v1.3.0',
            count: 2,
            state: 'available',
            source: sourceFixture('gateway'),
        },
        {
            component: 'gateway',
            current: 'dev-local-long-build-20260912-abcdef0123456789',
            latest: 'v1.3.0',
            count: 1,
            state: 'uncomparable',
            source: sourceFixture('gateway'),
        },
    ],
    retry_after_seconds: 0,
})
const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
    permissions: ['clipboard-read', 'clipboard-write'],
})
const page = await context.newPage()
const errors = []
page.on('pageerror', (error) => errors.push(error.message))
let summary = summaryFixture()
let updates = updatesFixture()
let summaryFails = false
let checkStatus = 200
let checkCalls = 0
let summaryCalls = 0
let updateCalls = 0
let obsoleteCalls = 0
await page.route('**/api/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    if (path === '/api/v1/current/version') {
        summaryCalls++
        if (summaryFails) return route.fulfill({ status: 503, json: { success: false, error: { detail: 'offline' } } })
        return route.fulfill({ json: { success: true, data: summary } })
    }
    if (path === '/api/v1/system/updates') {
        updateCalls++
        return route.fulfill({ json: { success: true, data: updates } })
    }
    if (path === '/api/v1/system/updates/check') {
        assert.equal(route.request().method(), 'POST')
        checkCalls++
        updates =
            checkStatus === 409
                ? {
                      ...updates,
                      enabled: false,
                      components: updates.components.map((item) => ({ ...item, state: 'disabled' })),
                  }
                : { ...updates, retry_after_seconds: 60 }
        return route.fulfill({
            status: checkStatus,
            headers: checkStatus === 429 ? { 'Retry-After': '60' } : {},
            json: {
                success: checkStatus === 200,
                data: updates,
                ...(checkStatus !== 200 && {
                    error: {
                        id: checkStatus === 409 ? 'update_check_disabled' : 'update_check_cooldown',
                        code: checkStatus,
                        detail: checkStatus === 409 ? 'Fixture disabled' : 'Fixture cooldown',
                    },
                }),
            },
        })
    }
    if (path.endsWith('/pub/version')) obsoleteCalls++
    return route.fulfill({ json: { success: true, data: [] } })
})

const sidebar = () => page.locator('.basic-side').getByRole('button', { name: /关于 TokenLive|About TokenLive/ })
const dialog = () => page.getByRole('dialog', { name: /关于 TokenLive|About TokenLive/ })
async function screenshot(name) {
    await page.waitForFunction(() => !document.querySelector('.ant-message-notice'))
    await page.waitForFunction(
        () =>
            !document.querySelector(
                '.ant-zoom-enter-active, .ant-zoom-leave-active, .ant-fade-enter-active, .ant-fade-leave-active'
            )
    )
    await page.evaluate(async () => {
        await new Promise(requestAnimationFrame)
        await Promise.all(
            document
                .getAnimations()
                .filter((animation) => animation.effect?.getComputedTiming().iterations !== Infinity)
                .map((animation) => animation.finished.catch(() => {}))
        )
    })
    await page.screenshot({ path: `${outputDir}/${name}.png` })
}
async function reload() {
    await page.goto(`${baseURL}/tests/fixtures/system-about.html`)
    await page.waitForLoadState('networkidle')
    await sidebar().waitFor()
}
async function openAbout() {
    const before = summaryCalls
    const response = page.waitForResponse('**/api/v1/current/version')
    await sidebar().click()
    await response
    await dialog().waitFor()
    await page.waitForLoadState('networkidle')
    assert.ok(summaryCalls > before, 'Opening About refreshes cached Summary')
}
async function closeAbout() {
    await page.waitForFunction(() => document.querySelector('.ant-modal-wrap')?.contains(document.activeElement))
    await page.keyboard.press('Escape')
    await dialog().waitFor({ state: 'hidden' })
}
async function resetScenario(nextSummary = summaryFixture(), nextUpdates = updatesFixture()) {
    summary = nextSummary
    updates = nextUpdates
    summaryFails = false
    checkStatus = 200
    checkCalls = summaryCalls = updateCalls = 0
    await page.setViewportSize({ width: 1440, height: 900 })
    await reload()
}
function groupRow(version) {
    return dialog()
        .locator('.system-about__gateway-group')
        .filter({ has: page.locator('.system-about__group-version', { hasText: version }) })
}
try {
    await reload()
    assert.equal(await dialog().count(), 0, 'No automatic dialog')
    assert.match(await sidebar().innerText(), /专业版.*Admin.*v1\.2\.3/)
    assert.match(await sidebar().getAttribute('aria-label'), /有可用更新/)
    assert.equal(summaryCalls, 1, 'One layout owner fetches one Summary')
    assert.equal(updateCalls, 1)
    assert.equal(obsoleteCalls, 0)
    assert.equal(checkCalls, 0, 'Mount only reads cache')

    const originalBox = await sidebar().boundingBox()
    const triggerBox = await page.locator('.ant-layout-sider-trigger').boundingBox()
    assert.ok(originalBox.y + originalBox.height <= triggerBox.y + 1)
    await page.locator('.basic-side__body').hover()
    await page.mouse.wheel(0, 1800)
    assert.ok(Math.abs((await sidebar().boundingBox()).y - originalBox.y) <= 1)
    await sidebar().focus()
    const openResponse = page.waitForResponse('**/api/v1/current/version')
    await page.keyboard.press('Enter')
    await openResponse
    await dialog().waitFor()
    assert.equal(checkCalls, 0, 'Keyboard-open does not check')
    assert.match(await dialog().innerText(), /专业版/)
    // R3: version, build kind and count must belong to the same actual group row.
    const releaseGroup = groupRow('v1.2.3')
    assert.equal(await releaseGroup.count(), 1)
    assert.equal(await releaseGroup.locator('.system-about__group-version').innerText(), 'v1.2.3')
    assert.equal(await releaseGroup.locator('.system-about__group-count').innerText(), '2 个节点')
    assert.match(await releaseGroup.innerText(), /正式构建/)
    const devGroup = groupRow('dev-local-long-build-20260912-abcdef0123456789')
    assert.equal(await devGroup.locator('.system-about__group-count').innerText(), '1 个节点')
    assert.match(await devGroup.innerText(), /开发构建/)
    assert.match(await dialog().innerText(), /最近 3 分钟内有效上报的节点/)
    assert.match(await dialog().innerText(), /不是完整节点清单或实时健康状态/)
    assert.match(await dialog().innerText(), /共享缓存/)
    assert.equal(await dialog().getByRole('button', { name: '检查更新', exact: true }).count(), 1)
    assert.match(await dialog().innerText(), /最近尝试|最近成功/)
    const releaseLink = dialog()
        .getByRole('link', { name: /发行说明/ })
        .first()
    assert.equal(
        await releaseLink.getAttribute('href'),
        'https://github.com/tokenlive/tokenlive-admin/releases/tag/v1.3.0'
    )
    assert.equal(await releaseLink.getAttribute('target'), '_blank')
    assert.equal(await releaseLink.getAttribute('rel'), 'noopener noreferrer')
    await screenshot('about-dark-mixed-versions')
    await dialog().getByRole('button', { name: '检查更新', exact: true }).scrollIntoViewIfNeeded()
    await screenshot('about-dark-update-details')
    await dialog().getByRole('button', { name: '复制版本信息', exact: true }).click()
    const copied = await page.evaluate(() => navigator.clipboard.readText())
    assert.match(copied, /TokenLive 专业版 · Admin v1\.2\.3/)
    assert.match(copied, /安装渠道：发行包/)
    assert.match(copied, /Gateway v1\.2\.3 \(正式构建\) × 2/)
    assert.doesNotMatch(copied, /v1\.3\.0|release_url|private-node/)
    await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined }))
    await dialog().getByRole('button', { name: '复制版本信息', exact: true }).click()
    await page.evaluate(() => delete navigator.clipboard)
    assert.equal(await page.evaluate(() => navigator.clipboard.readText()), copied)
    await page.evaluate(() =>
        Object.defineProperty(navigator, 'clipboard', {
            configurable: true,
            value: { writeText: () => Promise.reject(new Error('Clipboard denied')) },
        })
    )
    await dialog().getByRole('button', { name: '复制版本信息', exact: true }).click()
    await page.getByText('自动复制失败，请手动复制').waitFor()
    await page.evaluate(() => delete navigator.clipboard)
    await closeAbout()
    assert.equal(await sidebar().evaluate((element) => document.activeElement === element), true)

    await page.locator('.basic-side__trigger').click()
    await page.locator('.ant-layout-sider-collapsed').waitFor()
    assert.equal((await sidebar().innerText()).trim(), '')
    assert.match(await sidebar().getAttribute('aria-label'), /有可用更新/)
    await screenshot('collapsed-update-badge')
    await openAbout()
    await closeAbout()
    await page.locator('.basic-side__trigger').click()
    await page.evaluate(() => window.layoutTest.setConfig({ theme: 'light', sideTheme: 'light', headerTheme: 'light' }))
    await openAbout()
    await dialog()
        .locator('.ant-modal-body')
        .evaluate((element) => element.scrollTo(0, 0))
    await screenshot('about-light-mixed-versions')
    await closeAbout()

    for (const layout of ['leftRight', 'topBottom']) {
        for (const menuMode of ['side', 'mix', 'top']) {
            const before = summaryCalls
            await page.evaluate((config) => window.layoutTest.setConfig(config), { layout, menuMode })
            await page.waitForLoadState('networkidle')
            assert.equal(summaryCalls, before, 'Switching layout does not create another version owner')
            if (menuMode !== 'top') assert.match(await sidebar().innerText(), /专业版.*Admin.*v1\.2\.3/)
            assert.equal(await page.locator('.basic-header .ant-badge-dot').count(), 1)
            await page.locator('.basic-header .anticon-setting').click()
            await page
                .locator('.ant-drawer')
                .getByRole('button', { name: /关于 TokenLive/ })
                .click()
            await dialog().waitFor()
            await closeAbout()
            assert.equal(
                await page.evaluate(
                    () =>
                        document.activeElement?.matches('.basic-header button') &&
                        !!document.activeElement.querySelector('.anticon-setting')
                ),
                true
            )
        }
    }
    assert.equal(checkCalls, 0, 'All layout entries read caches only')
    await page.evaluate(() => {
        window.layoutTest.setLocale('en-us')
        window.layoutTest.setConfig({ layout: 'leftRight', menuMode: 'side' })
    })
    await page.setViewportSize({ width: 390, height: 844 })
    await openAbout()
    await dialog()
        .locator('.ant-modal-body')
        .evaluate((element) => element.scrollTo(0, 0))
    await screenshot('mobile-about')
    assert.match(await dialog().innerText(), /Professional|Gateway/)
    assert.match(await dialog().innerText(), /last 3 minutes/)
    assert.doesNotMatch(await dialog().innerText(), /app\.about\./)
    const modalBox = await page.locator('.ant-modal-content').boundingBox()
    assert.ok(modalBox.width >= 300 && modalBox.x >= 0 && modalBox.x + modalBox.width <= 391)
    assert.ok(modalBox.y >= 0 && modalBox.y + modalBox.height <= 844)

    // The fixture's root-like name is not capability.
    await resetScenario({ ...summaryFixture(), can_manage_updates: false })
    await openAbout()
    assert.equal(updateCalls, 0)
    assert.equal(checkCalls, 0)
    assert.equal(await page.locator('.ant-badge-dot').count(), 0)
    assert.equal(await groupRow('v1.2.3').locator('.system-about__group-count').innerText(), '2 个节点')
    assert.doesNotMatch(await dialog().innerText(), /检查更新|最近尝试|最近成功|发行说明|升级指引|v1\.3\.0/)

    for (const [state, label] of [
        ['stale', '历史结果已过期'],
        ['unavailable', '检查来源不可用'],
        ['disabled', '检查已关闭'],
    ]) {
        const snapshot = updatesFixture()
        snapshot.enabled = state !== 'disabled'
        snapshot.components = snapshot.components.map((item) => ({
            ...item,
            state,
            source: { ...item.source, stale: true, status: state === 'unavailable' ? 'unavailable' : 'ready' },
        }))
        await resetScenario(summaryFixture(), snapshot)
        await openAbout()
        assert.equal(await page.locator('.ant-badge-dot').count(), 0)
        assert.match(await dialog().innerText(), /历史结果.*不是当前可升级目标/)
        assert.equal(
            await dialog().locator('.system-about__component').first().getByText(label, { exact: true }).count(),
            1
        )
        assert.equal(await dialog().getByRole('link').count(), 0)
        assert.equal(await dialog().getByRole('button', { name: '复制升级指引' }).count(), 0)
        assert.equal(
            await dialog().getByRole('button', { name: '检查更新', exact: true }).isDisabled(),
            state === 'disabled'
        )
    }
    for (const [state, label] of [
        ['current', '与当前稳定发行一致'],
        ['ahead', '高于当前稳定发行'],
        ['uncomparable', '版本不可比较'],
        ['no_candidate', '暂无稳定候选版本'],
        ['unknown', '暂无法判断'],
        ['checking', '正在检查'],
    ]) {
        const snapshot = updatesFixture()
        snapshot.components = [{ ...snapshot.components[0], state }]
        await resetScenario(summaryFixture(), snapshot)
        await openAbout()
        assert.equal(await page.locator('.ant-badge-dot').count(), 0)
        assert.equal(await dialog().getByRole('button', { name: '复制升级指引' }).count(), 0)
        assert.doesNotMatch(await dialog().innerText(), /app\.about\./)
        assert.equal(await dialog().locator('.system-about__component').getByText(label, { exact: true }).count(), 1)
    }
    for (const stale of [undefined, 'false']) {
        const snapshot = updatesFixture()
        snapshot.components = [{ ...snapshot.components[0], source: { ...snapshot.components[0].source, stale } }]
        await resetScenario(summaryFixture(), snapshot)
        await openAbout()
        assert.equal(await page.locator('.ant-badge-dot').count(), 0)
        assert.equal(await dialog().getByRole('link').count(), 0)
        assert.doesNotMatch(await dialog().innerText(), /有可用更新/)
    }

    for (const status of [200, 409, 429]) {
        await resetScenario()
        checkStatus = status
        await openAbout()
        const button = dialog().getByRole('button', { name: '检查更新', exact: true })
        const response = page.waitForResponse('**/api/v1/system/updates/check')
        await button.evaluate((element) => {
            element.click()
            element.click()
            element.click()
        })
        await response
        await page.waitForFunction(() => document.querySelector('.system-about__check')?.disabled)
        assert.equal(checkCalls, 1, 'Repeated clicks do not bypass the shared server cooldown')
        assert.doesNotMatch(await page.locator('body').innerText(), /已是最新版本|全部.*最新|检查成功/)
        if (status === 409) assert.match(await dialog().innerText(), /联网更新检查已关闭/)
        else assert.match(await dialog().innerText(), /请在 \d+ 秒后重试/)
        if (status === 429) assert.match(await dialog().getByRole('alert').innerText(), /检查过于频繁/)
    }
    await resetScenario({ ...summaryFixture(), gateway: { status: 'unknown', scope: 'this_admin', groups: [] } })
    await openAbout()
    assert.match(await dialog().innerText(), /当前 Admin 实例/)
    assert.match(await dialog().innerText(), /暂无有效上报/)

    const standalone = {
        ...summaryFixture(),
        identity: { edition: 'standalone', install_channel: 'homebrew', build: { version: 'v2.0.0', kind: 'release' } },
    }
    const standaloneUpdates = {
        enabled: true,
        retry_after_seconds: 0,
        components: [
            {
                component: 'standalone',
                current: 'v2.0.0',
                latest: 'v2.1.0',
                state: 'available',
                source: sourceFixture('standalone', 'v2.1.0'),
            },
        ],
    }
    await resetScenario(standalone, standaloneUpdates)
    await openAbout()
    assert.match(await sidebar().innerText(), /单机版 v2\.0\.0/)
    assert.doesNotMatch(await dialog().innerText(), /Gateway|Admin v1\.2\.3/)
    assert.match(await dialog().innerText(), /brew update\nbrew upgrade tokenlive/)
    assert.match(await dialog().innerText(), /仅当.*Homebrew services.*手动.*重启.*请求/)
    await dialog().getByRole('button', { name: '复制升级指引' }).click()
    const guidance = await page.evaluate(() => navigator.clipboard.readText())
    assert.match(guidance, /brew update\nbrew upgrade tokenlive/)
    assert.match(guidance, /仅当.*Homebrew services/)
    assert.match(guidance, /brew services restart tokenlive/)
    await screenshot('standalone-homebrew')
    await resetScenario(
        { ...standalone, identity: { ...standalone.identity, install_channel: 'unknown' } },
        standaloneUpdates
    )
    await openAbout()
    assert.match(await dialog().innerText(), /安装渠道未知/)
    assert.doesNotMatch(await dialog().innerText(), /brew upgrade|brew services/)

    for (const url of [
        'javascript:alert(1)',
        'https://github.com.evil.test/tokenlive/tokenlive-admin/releases/tag/v1.3.0',
        'https://github.com/tokenlive/tokenlive-gateway/releases/tag/v1.3.0',
        'https://user@github.com/tokenlive/tokenlive-admin/releases/tag/v1.3.0',
    ]) {
        const snapshot = updatesFixture()
        snapshot.components = [
            {
                ...snapshot.components[0],
                source: {
                    ...snapshot.components[0].source,
                    candidate: { version: '<img src=x onerror=alert(1)>', release_url: url },
                },
            },
        ]
        await resetScenario(summaryFixture(), snapshot)
        await openAbout()
        assert.equal(await dialog().getByRole('link').count(), 0)
        assert.equal(await dialog().locator('img').count(), 1, 'Only the existing product logo is rendered')
    }
    await resetScenario()
    summaryFails = true
    await reload()
    await openAbout()
    assert.match(await sidebar().innerText(), /前端构建/)
    assert.match(await dialog().innerText(), /前端构建信息.*不代表运行中的服务版本/s)
    assert.doesNotMatch(await dialog().innerText(), /专业版|Admin v1\.2\.3|检查更新|v1\.3\.0/)
    assert.deepEqual(errors, [])
    console.log(
        'PASS: shared state, cache-only open, scoped counts, privileges, states, 409/429, cooldown, secure links, Homebrew, fallback, keyboard/copy, 6 layouts, dark/light/collapsed/mobile'
    )
    console.log(`Screenshots: ${outputDir}`)
} finally {
    await browser.close()
}
