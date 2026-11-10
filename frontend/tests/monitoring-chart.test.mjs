import assert from 'node:assert/strict'
import test from 'node:test'
import { monitoringChart } from '../src/utils/monitoring-chart.js'

const labels = {
    requests: 'Requests',
    success: 'Success',
    failure: 'Failure',
    rate: 'Request success rate',
}
const rate = (success, failure) =>
    success.map((value, index) => {
        const failed = failure[index]
        if (value == null || failed == null) return null
        const total = value + failed
        return total > 0 ? Number(((value / total) * 100).toFixed(1)) : null
    })

test('a reading with no plottable points is empty and has no hint', () => {
    for (const state of ['ok', 'partial', 'unavailable', undefined]) {
        const chart = monitoringChart({
            state,
            times: ['10:00', '10:01'],
            series: [{ success: [null, null], failure: [null, null] }],
            dark: false,
            labels,
            rate,
        })
        assert.equal(chart.mode, 'empty')
        assert.equal(chart.hint, false)
        assert.equal(chart.options, null)
    }
})

test('a failed reading with no points keeps the hint and does not pretend the window is empty', () => {
    const chart = monitoringChart({
        state: 'error',
        times: [],
        series: [],
        dark: true,
        labels,
        rate,
    })
    assert.equal(chart.mode, 'hint')
    assert.equal(chart.hint, true)
    assert.equal(chart.options, null)
})

test('zero is plottable, missing buckets stay disconnected, and the rate label stays caller-owned', () => {
    const chart = monitoringChart({
        state: 'partial',
        times: ['10:00', '10:01'],
        series: [{ success: [0, null], failure: [1, null] }],
        dark: false,
        labels,
        rate,
    })
    assert.equal(chart.mode, 'chart')
    assert.equal(chart.hint, true)
    assert.equal(chart.options.series[0].connectNulls, false)
    assert.equal(chart.options.series[0].data[0], 0)
    assert.equal(chart.options.series[2].name, 'Request success rate')
    assert.deepEqual(chart.options.series[2].data, [0, null])
    assert.deepEqual(chart.options.series[0].areaStyle.color.colorStops, [
        { offset: 0, color: 'rgba(35, 199, 183, 0.32)' },
        { offset: 1, color: 'rgba(35, 199, 183, 0.02)' },
    ])
})

test('more than one series plots each total without inventing a success rate', () => {
    const chart = monitoringChart({
        state: 'ok',
        times: ['10:00'],
        series: [
            { name: 'endpoint-a', values: [2] },
            { name: 'endpoint-b', values: [null] },
        ],
        dark: true,
        labels,
    })
    assert.equal(chart.mode, 'chart')
    assert.equal(chart.hint, false)
    assert.equal(chart.options.yAxis.length, 1)
    assert.deepEqual(
        chart.options.series.map((series) => series.name),
        ['endpoint-a', 'endpoint-b']
    )
    assert.equal(chart.options.series.every((series) => series.connectNulls === false), true)
})

test('a single value series keeps its own formatter and does not require a rate', () => {
    const chart = monitoringChart({
        state: 'partial',
        times: ['10:00', '10:01'],
        series: [{ name: 'TTFT', values: [1500, null], color: '#8b5cf6' }],
        dark: false,
        labels: { requests: 'ms' },
        format: (value) => (value == null ? '—' : `${value}ms`),
        axisFormat: (value) => (Number(value) >= 1000 ? `${(Number(value) / 1000).toFixed(1)}s` : `${value}ms`),
    })
    assert.equal(chart.mode, 'chart')
    assert.equal(chart.hint, true)
    assert.equal(chart.options.series.length, 1)
    assert.equal(chart.options.tooltip.valueFormatter(1500), '1500ms')
    assert.equal(chart.options.tooltip.valueFormatter(null), '—')
    assert.equal(chart.options.yAxis.axisLabel.formatter(1500), '1.5s')
    assert.equal(chart.options.series[0].connectNulls, false)
})
