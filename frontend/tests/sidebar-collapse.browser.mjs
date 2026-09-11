// PLAYWRIGHT_MODULE=/path/to/playwright node tests/sidebar-collapse.browser.mjs
import assert from 'node:assert/strict'
import { createRequire } from 'node:module'
import { mkdir } from 'node:fs/promises'

const require = createRequire(import.meta.url)
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || 'playwright')
const outputDir = process.env.TEST_SCREENSHOT_DIR || '/tmp/tokenlive-sidebar-collapse'
await mkdir(outputDir, { recursive: true })
const browser = await chromium.launch({ headless: true, channel: process.env.PLAYWRIGHT_CHANNEL || 'chrome' })
try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: 2 })
    const errors = []
    page.on('pageerror', (error) => errors.push(error.message))
    await page.route('**/api/**', (route) =>
        route.fulfill({ json: { success: true, data: { version: 'v1.0.0', app_name: 'TokenLive' } } })
    )
    await page.goto(`${process.env.TEST_BASE_URL || 'http://127.0.0.1:9211'}/tests/fixtures/system-about.html`)
    await page.waitForLoadState('networkidle')
    const side = page.locator('.basic-side')
    const settle = async () => {
        await page.evaluate(async () => {
            await new Promise(requestAnimationFrame)
            await Promise.all(document.getAnimations().map((animation) => animation.finished.catch(() => {})))
        })
    }
    const groups = side.locator('.ant-menu-item-group-title .basic-menu__name')
    assert.deepEqual(await groups.allTextContents(), ['资源管理', '治理策略', '系统管理'])
    await side.locator('.basic-side__trigger').click()
    await page.locator('.ant-menu-inline-collapsed').waitFor()
    await settle()
    await side.screenshot({ path: `${outputDir}/collapsed-before.png` })

    const measurements = await side.evaluate((element) => {
        const measure = (node) => {
            const rect = node.getBoundingClientRect()
            const style = getComputedStyle(node)
            return {
                center: rect.x + rect.width / 2,
                width: rect.width,
                padding: style.padding,
                margin: style.margin,
                class: node.getAttribute('class'),
            }
        }
        return {
            side: measure(element),
            icon: measure(element.querySelector('.ant-menu-item .anticon')),
            item: measure(element.querySelector('.ant-menu-item')),
            version: measure(element.querySelector('.sidebar-version .anticon')),
            trigger: measure(element.querySelector('.basic-side__trigger .anticon')),
            group: measure(element.querySelector('.ant-menu-item-group-title')),
        }
    })
    console.log('Collapsed measurements:', JSON.stringify(measurements))
    const issues = []
    const check = (condition, message) => {
        if (!condition) issues.push(message)
    }
    for (const theme of ['dark', 'light']) {
        for (const width of [60, 72]) {
            await page.evaluate((config) => window.layoutTest.setConfig(config), {
                theme,
                sideTheme: theme,
                headerTheme: theme,
                sideCollapsedWidth: width,
            })
            await settle()
            const sideBox = await side.boundingBox()
            const center = sideBox.x + sideBox.width / 2
            const icons = side.locator(
                '.ant-menu-item .anticon, .ant-menu-submenu-title .anticon, .sidebar-version .anticon, .basic-side__trigger .anticon, .brand img'
            )
            for (const icon of await icons.all()) {
                const box = await icon.boundingBox()
                check(
                    box && Math.abs(box.x + box.width / 2 - center) <= 1,
                    `${theme}/${width}: icon must share the sidebar center (${center}), got ${box?.x + box?.width / 2}`
                )
            }
            for (const title of await groups.all()) {
                check(!(await title.isVisible()), `${theme}/${width}: collapsed group title must not show clipped text`)
            }
            const iconlessTitle = side.locator('[data-menu-id="menu-29"] .ant-menu-title-content')
            check(
                await iconlessTitle.evaluate((node) => node.getBoundingClientRect().width > 0),
                `${theme}/${width}: menu items without an icon must not become blank`
            )
            const separators = await side.locator('.ant-menu-item-group-title').evaluateAll((titles) =>
                titles.slice(1).map((title) => {
                    const style = getComputedStyle(title, '::after')
                    return { width: parseFloat(style.width), content: style.content }
                })
            )
            check(
                separators.length === 2 && separators.every((item) => item.width > 0 && item.content !== 'none'),
                `${theme}/${width}: collapsed groups keep a subtle visible separator`
            )
            const firstItem = side.locator('.ant-menu-item').first()
            await firstItem.hover()
            await page.getByRole('tooltip').filter({ hasText: '供应商' }).waitFor()
            await page.mouse.move(300, 300)
            await side.screenshot({ path: `${outputDir}/collapsed-${theme}-${width}.png` })
        }
    }
    await side.locator('.basic-side__trigger').click()
    await page.locator('.ant-menu-inline').waitFor()
    await settle()
    for (const title of await groups.all()) {
        check(await title.isVisible(), 'Expanded group titles remain visible')
        check(
            await title.evaluate((node) => node.scrollWidth <= node.clientWidth),
            'Expanded group text is not clipped'
        )
    }
    await side.screenshot({ path: `${outputDir}/expanded.png` })
    assert.deepEqual(errors, [])
    assert.deepEqual(issues, [])
    console.log('PASS: centered icons, collapsed group separators, tooltips, expanded titles; dark/light and 60/72px')
    console.log(`Screenshots: ${outputDir}`)
} finally {
    await browser.close()
}
