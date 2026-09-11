// Run against the dev server:
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
try {
    const context = await browser.newContext({
        viewport: { width: 1440, height: 900 },
        permissions: ['clipboard-read', 'clipboard-write'],
    })
    const page = await context.newPage()
    const screenshot = async (name) => {
        // Vue's transition classes precede the browser animation itself.
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
    const errors = []
    page.on('pageerror', (error) => errors.push(error.message))
    let versionRequests = 0
    let versionMode = 'success'
    await page.route('**/api/**', async (route) => {
        if (route.request().url().endsWith('/pub/version')) {
            versionRequests++
            if (versionMode === 'failure') return route.abort('failed')
            return route.fulfill({
                json: { success: true, data: { version: 'v9.8.7-test', app_name: 'TokenLive' } },
            })
        }
        return route.fulfill({ json: { success: true, data: [] } })
    })
    await page.goto(`${baseURL}/tests/fixtures/system-about.html`)
    await page.waitForLoadState('networkidle')
    await screenshot('initial')
    const versionButton = page.locator('.basic-side').getByRole('button', { name: /关于 TokenLive/ })
    await versionButton.waitFor({ timeout: 5000 })
    assert.match(await versionButton.innerText(), /v9\.8\.7-test/)
    assert.doesNotMatch(await page.locator('.basic-header').innerText(), /v9\.8\.7-test/)
    assert.equal(versionRequests, 1)

    // The footer remains above the collapse control while the menu scrolls.
    const originalBox = await versionButton.boundingBox()
    const triggerBox = await page.locator('.ant-layout-sider-trigger').boundingBox()
    assert.ok(originalBox.y + originalBox.height <= triggerBox.y + 1)
    await page.locator('.basic-side__body').hover()
    await page.mouse.wheel(0, 1800)
    assert.ok(Math.abs((await versionButton.boundingBox()).y - originalBox.y) <= 1)
    await versionButton.focus()
    await page.keyboard.press('Enter')
    const dialog = page.getByRole('dialog', { name: '关于 TokenLive' })
    await dialog.waitFor()
    assert.match(await dialog.innerText(), /v9\.8\.7-test/)
    await dialog.getByRole('button', { name: '复制版本信息' }).click()
    assert.equal(await page.evaluate(() => navigator.clipboard.readText()), 'TokenLive v9.8.7-test')
    await screenshot('about-dark')
    // Also exercise the HTTP clipboard fallback and its failure feedback.
    await page.evaluate(() => navigator.clipboard.writeText('before-fallback'))
    await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', { configurable: true, value: undefined }))
    await dialog.getByRole('button', { name: '复制版本信息' }).click()
    await page.evaluate(() => delete navigator.clipboard)
    assert.equal(await page.evaluate(() => navigator.clipboard.readText()), 'TokenLive v9.8.7-test')
    await page.evaluate(() =>
        Object.defineProperty(navigator, 'clipboard', {
            configurable: true,
            value: { writeText: () => Promise.reject(new Error('Clipboard denied')) },
        })
    )
    await dialog.getByRole('button', { name: '复制版本信息' }).click()
    await page.getByText('自动复制失败，请手动复制').waitFor()
    await page.evaluate(() => delete navigator.clipboard)
    await page.keyboard.press('Escape')
    await dialog.waitFor({ state: 'hidden' })

    await page.locator('.basic-side__trigger').click()
    await page.locator('.ant-layout-sider-collapsed').waitFor()
    assert.equal((await versionButton.innerText()).trim(), '')
    await versionButton.click()
    await dialog.waitFor()
    await page.waitForFunction(() => document.querySelector('.ant-modal-wrap')?.contains(document.activeElement))
    await page.keyboard.press('Escape')
    await dialog.waitFor({ state: 'hidden' })
    await screenshot('collapsed')

    await page.locator('.basic-side__trigger').click()
    await page.evaluate(() => window.layoutTest.setConfig({ theme: 'light', sideTheme: 'light', headerTheme: 'light' }))
    await screenshot('expanded-light')

    // Every layout keeps an About entry; switching layout does not refetch.
    for (const layout of ['leftRight', 'topBottom']) {
        for (const menuMode of ['side', 'mix', 'top']) {
            await page.evaluate((config) => window.layoutTest.setConfig(config), { layout, menuMode })
            if (menuMode !== 'top') {
                await versionButton.waitFor()
                assert.match(await versionButton.innerText(), /v9\.8\.7-test/)
            }
            await page.locator('.basic-header .anticon-setting').click()
            await page.getByRole('button', { name: '关于 TokenLive' }).last().click()
            await dialog.waitFor()
            assert.match(await dialog.innerText(), /v9\.8\.7-test/)
            await page.waitForFunction(() =>
                document.querySelector('.ant-modal-wrap')?.contains(document.activeElement)
            )
            await page.keyboard.press('Escape')
            await dialog.waitFor({ state: 'hidden' })
            assert.equal(
                await page.evaluate(
                    () =>
                        document.activeElement?.matches('.basic-header button') &&
                        !!document.activeElement.querySelector('.anticon-setting')
                ),
                true,
                'Closing About returns keyboard focus to the visible Settings trigger'
            )
        }
    }
    assert.equal(versionRequests, 1)

    await page.evaluate(() => {
        window.layoutTest.setLocale('en-us')
        window.layoutTest.setConfig({ layout: 'leftRight', menuMode: 'side' })
    })
    await page.setViewportSize({ width: 390, height: 844 })
    await page
        .locator('.basic-side')
        .getByRole('button', { name: /About TokenLive/ })
        .click()
    await page.getByRole('dialog', { name: 'About TokenLive' }).waitFor()
    // Ant moves focus into the dialog only after its enter transition completes.
    await page.waitForFunction(() => document.querySelector('.ant-modal-wrap')?.contains(document.activeElement))
    await screenshot('mobile-about')
    const modalBox = await page.locator('.ant-modal-content').boundingBox()
    assert.ok(
        modalBox.width >= 300 && modalBox.x >= 0 && modalBox.x + modalBox.width <= 391,
        `Mobile dialog is within the viewport: ${JSON.stringify(modalBox)}`
    )
    assert.ok(modalBox.y >= 0 && modalBox.y + modalBox.height <= 844)
    assert.equal(await page.locator('.ant-modal').evaluate((element) => getComputedStyle(element).opacity), '1')
    assert.equal(
        await page.getByRole('dialog', { name: 'About TokenLive' }).isVisible(),
        true,
        'The mobile About dialog remains open after transitions settle'
    )

    // A failed public-version request retains the build-time fallback.
    versionMode = 'failure'
    await page.setViewportSize({ width: 1440, height: 900 })
    await page.reload()
    await page.waitForLoadState('networkidle')
    await versionButton.waitFor()
    assert.match(await versionButton.innerText(), /\d/)
    assert.doesNotMatch(await versionButton.innerText(), /undefined|null/)
    assert.deepEqual(errors, [])
    console.log('PASS: footer placement, fixed position, keyboard, copy, collapse, 6 layouts, locale, mobile, fallback')
    console.log(`Screenshots: ${outputDir}`)
} finally {
    await browser.close()
}
