import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { access, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'
import { promisify } from 'node:util'
import test from 'node:test'
import { compileStyleAsync, parse } from '@vue/compiler-sfc'

const run = promisify(execFile)
const sourceDir = fileURLToPath(new URL('../src/', import.meta.url))
const chromeCandidates = [
    process.env.CHROME_BIN,
    '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
    '/usr/bin/google-chrome',
].filter(Boolean)
let chrome
for (const candidate of chromeCandidates) {
    try {
        await access(candidate)
        chrome = candidate
        break
    } catch {
        // Try the next installed Chromium browser.
    }
}

test(
    'provider monitor can scroll to the last policy event and pagination',
    { skip: !chrome && 'Set CHROME_BIN to run the browser layout regression' },
    async () => {
        const filename = path.join(sourceDir, 'views/resource/ProviderDetail.vue')
        const { descriptor } = parse(await readFile(filename, 'utf8'), { filename })
        const style = await compileStyleAsync({
            filename,
            id: 'data-v-scroll-test',
            source: descriptor.styles[0].content.replaceAll('@/', `${sourceDir}/`),
            scoped: true,
            preprocessLang: 'less',
        })
        assert.deepEqual(style.errors, [])

        // Keep the fixed-height card/tab hierarchy; only the monitoring data is synthetic.
        const fixture = `<!doctype html><meta charset="utf-8">
        <style>
            ${style.code}
            * { box-sizing: border-box; }
            body { margin: 0; }
            .info-card { height: 120px; }
            .detail-tabs { height: 48px; }
            .ant-card-body { padding: 24px; }
            .charts { height: 320px; }
            .event { height: 48px; }
            .pagination { height: 40px; }
        </style>
        <div class="app-page" style="height: 720px">
            <div class="provider-detail">
                <div class="info-card"></div>
                <div class="detail-card"><div class="ant-card-body">
                    <div class="detail-tabs"></div>
                    <div class="tab-content tab-content--scroll">
                        <div class="provider-monitor">
                            <div class="charts"></div>
                            <div class="events-panel">
                                ${Array.from({ length: 20 }, (_, index) => `<div class="event">Event ${index + 1}</div>`).join('')}
                                <div class="pagination">Pagination</div>
                            </div>
                        </div>
                    </div>
                </div></div>
            </div>
        </div>
        <script>
            document.querySelectorAll('div').forEach(element => element.setAttribute('data-v-scroll-test', ''));
            const pane = document.querySelector('.tab-content');
            const results = [720, 480].map(height => {
                document.querySelector('.app-page').style.height = height + 'px';
                pane.scrollTop = pane.scrollHeight;
                const bounds = pane.getBoundingClientRect();
                const visible = selector => {
                    const rect = document.querySelector(selector).getBoundingClientRect();
                    return rect.top >= bounds.top && rect.bottom <= bounds.bottom + 1;
                };
                return {
                    height,
                    overflowY: getComputedStyle(pane).overflowY,
                    scrollTop: pane.scrollTop,
                    lastEventVisible: visible('.event:nth-child(20)'),
                    paginationVisible: visible('.pagination')
                };
            });
            const output = document.createElement('pre');
            output.id = 'result';
            output.textContent = JSON.stringify(results);
            document.body.append(output);
        </script>`
        const directory = await mkdtemp(path.join(tmpdir(), 'provider-monitor-scroll-'))
        try {
            const htmlPath = path.join(directory, 'fixture.html')
            await writeFile(htmlPath, fixture)
            const { stdout } = await run(
                chrome,
                [
                    '--headless',
                    '--disable-gpu',
                    '--no-sandbox',
                    '--no-first-run',
                    `--user-data-dir=${path.join(directory, 'profile')}`,
                    '--dump-dom',
                    pathToFileURL(htmlPath).href,
                ],
                { timeout: 30000, maxBuffer: 1024 * 1024 }
            )
            const results = JSON.parse(stdout.match(/<pre id="result">(.*?)<\/pre>/s)?.[1] || 'null')
            assert.ok(results, 'Browser must return layout measurements')
            for (const result of results) {
                assert.equal(result.overflowY, 'auto', `${result.height}px: monitor must allow vertical scrolling`)
                assert.ok(result.scrollTop > 0, `${result.height}px: monitor must actually scroll`)
                assert.ok(result.lastEventVisible, `${result.height}px: last policy event must be reachable`)
                assert.ok(result.paginationVisible, `${result.height}px: pagination must be reachable`)
            }
        } finally {
            await rm(directory, { recursive: true, force: true })
        }
    }
)
