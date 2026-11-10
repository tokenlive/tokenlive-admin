import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const monitorUrl = new URL('../src/views/resource/ModelMonitorTab.vue', import.meta.url)
const zhLocaleUrl = new URL('../src/locales/lang/zh-CN/pages.js', import.meta.url)
const enLocaleUrl = new URL('../src/locales/lang/en-US/pages.js', import.meta.url)

test('model performance query retains explicit start, cutoff and zone in snapshot selection', async () => {
    const source = await readFile(new URL('../src/utils/monitoring-snapshot.js', import.meta.url), 'utf8')
    const { monitoringQuery } = await import(`data:text/javascript;base64,${Buffer.from(source).toString('base64')}`)
    assert.deepEqual(
        monitoringQuery('performance', {
            model: 'm',
            time_range: '6h',
            start_time: '2026-11-10T10:00:00+08:00',
            end_time: '2026-11-10T10:37:42+08:00',
            timezone: 'Asia/Taipei',
        }),
        {
            timezone: 'Asia/Taipei',
            at: '2026-11-10T10:37:42+08:00',
            performance: { model: 'm', range: '6h', start: '2026-11-10T10:00:00+08:00' },
        }
    )
})

test('model monitor renders TTFT and OTPS through the shared chart presentation', async () => {
    const source = await readFile(monitorUrl, 'utf8')
    const ttft = source.slice(source.indexOf('const ttftChart'), source.indexOf('const otpsChart'))
    const otps = source.slice(source.indexOf('const otpsChart'), source.indexOf('function loadAll'))

    assert.match(source, /pages\.model\.detail\.monitor\.ttftTrend/)
    assert.match(source, /pages\.model\.detail\.monitor\.otpsTrend/)
    assert.match(source, /ttftChart\.mode === 'empty'/)
    assert.match(source, /otpsChart\.mode === 'chart'/)
    assert.match(ttft, /values: performance\.value\.avg_ttft_ms/)
    assert.match(ttft, /format: formatLatency/)
    assert.match(otps, /values: performance\.value\.otps/)
    assert.match(otps, /format: formatOtps/)
    assert.doesNotMatch(source, /hasMetricData|trafficChartOptions|ttftChartOptions/)
})

test('model performance chart copy exists in both locales', async () => {
    const [zhSource, enSource] = await Promise.all([readFile(zhLocaleUrl, 'utf8'), readFile(enLocaleUrl, 'utf8')])

    for (const source of [zhSource, enSource]) {
        assert.match(source, /'pages\.model\.detail\.monitor\.ttftTrend':/)
        assert.match(source, /'pages\.model\.detail\.monitor\.otpsTrend':/)
        assert.match(source, /'pages\.model\.detail\.monitor\.performance\.empty':/)
    }
})

test('model monitor preserves the selected local time-zone offset', async () => {
    const source = await readFile(monitorUrl, 'utf8')

    assert.match(source, /const endTime = end\.format\(\)/)
    assert.doesNotMatch(source, /const endTime = end\.toISOString\(\)/)
})
