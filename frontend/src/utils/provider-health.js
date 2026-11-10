import { metricDisplay } from './metric-observation.js'

export function providerHealthCards(health, t, formatTokens) {
    const count = (value) => Number(value).toLocaleString('en-US')
    const latency = (value) => `${(value / 1000).toFixed(2)}s`
    const definitions = [
        ['upstreamCalls', 'calls', (value) => count(value.total)],
        ['successRate', 'success_rate', (value) => `${Number(value).toFixed(1)}%`],
        ['latency', 'avg_latency_ms', latency],
        ['ttft', 'avg_ttft_ms', latency],
        ['otps', 'otps', (value) => `${Number(value).toLocaleString('en-US', { maximumFractionDigits: 2 })} t/s`],
        ['tokens', 'total_tokens', formatTokens],
        ['cost', 'total_cost', (value) => `¥${Number(value).toFixed(4)}`],
        ['endpoints', 'endpoint_count', count],
    ]
    return definitions.map(([key, field, format]) => ({
        key,
        label: t(`pages.provider.detail.monitor.kpi.${key}`),
        ...metricDisplay(health?.[field], t, format),
    }))
}
