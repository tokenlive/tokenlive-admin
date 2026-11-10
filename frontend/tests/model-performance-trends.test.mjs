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

test('model monitor loads and renders independent TTFT and OTPS trend charts', async () => {
    const source = await readFile(monitorUrl, 'utf8')
    const trafficBlock = source.slice(
        source.indexOf('const trafficChartOptions'),
        source.indexOf('const endpointChartOptions')
    )
    const ttftBlock = source.slice(source.indexOf('const ttftChartOptions'), source.indexOf('const otpsChartOptions'))
    const otpsBlock = source.slice(
        source.indexOf('const otpsChartOptions'),
        source.indexOf('function endpointSeriesLabel')
    )

    assert.match(source, /pages\.model\.detail\.monitor\.ttftTrend/)
    assert.match(source, /pages\.model\.detail\.monitor\.otpsTrend/)
    assert.match(source, /const hasTTFTTrend = computed\(\(\) => hasMetricData\(performance\.value\.avg_ttft_ms\)\)/)
    assert.match(source, /const hasOTPSTrend = computed\(\(\) => hasMetricData\(performance\.value\.otps\)\)/)
    assert.match(source, /data: performance\.value\.avg_ttft_ms \|\| \[\]/)
    assert.match(source, /data: performance\.value\.otps \|\| \[\]/)
    assert.doesNotMatch(trafficBlock, /valueFormatter/)
    assert.match(ttftBlock, /valueFormatter: \(value\) => formatLatency\(value\)/)
    assert.match(otpsBlock, /valueFormatter: \(value\) => formatOtps\(value\)/)
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
