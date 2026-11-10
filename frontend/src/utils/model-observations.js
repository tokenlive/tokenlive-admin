import { metricDisplay } from './metric-observation.js'

export function modelMetricDisplay(row, key, t, formatTokens) {
    const definitions = {
        request_count: ['requests', (value) => value.total.toLocaleString()],
        success_rate: ['success_rate', (value) => `${value.toFixed(1)}%`],
        avg_latency: ['avg_latency_ms', (value) => `${(value / 1000).toFixed(2)}s`],
        avg_ttft: ['avg_ttft_ms', (value) => `${(value / 1000).toFixed(2)}s`],
        otps: ['otps', (value) => `${value.toLocaleString('en-US', { maximumFractionDigits: 2 })} t/s`],
        total_tokens: ['total_tokens', formatTokens],
        total_cost: ['total_cost', (value) => `¥${value.toFixed(4)}`],
    }
    const [field, format] = definitions[key]
    const display = metricDisplay(row?.[field], t, format)
    if (key === 'avg_latency' || key === 'avg_ttft') {
        const suffix = key === 'avg_latency' ? 'latency_ms' : 'ttft_ms'
        display.hint +=
            '\n' +
            ['p50', 'p95', 'p99']
                .map((percentile) => {
                    const part = metricDisplay(row?.[`${percentile}_${suffix}`], t, format)
                    return `${percentile.toUpperCase()}: ${part.value} · ${part.hint}`
                })
                .join('\n')
    }
    return display
}

export function modelObservationCards(row, endpoints, endpointReading, t, formatTokens) {
    return [
        ['requests', 'request_count'],
        ['successRate', 'success_rate'],
        ['latency', 'avg_latency'],
        ['ttft', 'avg_ttft'],
        ['otps', 'otps'],
        ['tokens', 'total_tokens'],
        ['cost', 'total_cost'],
    ]
        .map(([key, field]) => {
            const display = modelMetricDisplay(row, field, t, formatTokens)
            return {
                key,
                label: t(`pages.model.detail.monitor.kpi.${key}`),
                value: display.value,
                hint: display.hint.split('\n'),
            }
        })
        .concat({
            key: 'endpoints',
            label: t('pages.model.detail.monitor.kpi.endpoints'),
            value: endpointReading?.state === 'ok' ? endpoints.length.toLocaleString() : '—',
            hint: endpointReading?.state === 'error' ? [t('pages.monitoring.measurement.state.error')] : undefined,
        })
}
