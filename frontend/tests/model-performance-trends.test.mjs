import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

const dashboardApiUrl = new URL('../src/apis/modules/dashboard.js', import.meta.url)
const monitorUrl = new URL('../src/views/resource/ModelMonitorTab.vue', import.meta.url)
const zhLocaleUrl = new URL('../src/locales/lang/zh-CN/pages.js', import.meta.url)
const enLocaleUrl = new URL('../src/locales/lang/en-US/pages.js', import.meta.url)

test('dashboard API exposes model performance trends with caller parameters', async () => {
    const source = await readFile(dashboardApiUrl, 'utf8')

    assert.match(
        source,
        /export const getModelPerformanceTrends = \(params\) =>\s*request\.basic\.get\('\/api\/v1\/dashboard\/model-performance-trends', params\)/
    )
})

test('model monitor loads and renders independent TTFT and OTPS trend charts', async () => {
    const source = await readFile(monitorUrl, 'utf8')
    const trafficBlock = source.slice(source.indexOf('const trafficChartOptions'), source.indexOf('const endpointChartOptions'))
    const ttftBlock = source.slice(source.indexOf('const ttftChartOptions'), source.indexOf('const otpsChartOptions'))
    const otpsBlock = source.slice(source.indexOf('const otpsChartOptions'), source.indexOf('function endpointSeriesLabel'))

    assert.match(
        source,
        /\.getModelPerformanceTrends\(\{\s*model,\s*time_range: range,\s*end_time: endTime\s*\}\)/
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
    const [zhSource, enSource] = await Promise.all([
        readFile(zhLocaleUrl, 'utf8'),
        readFile(enLocaleUrl, 'utf8'),
    ])

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
