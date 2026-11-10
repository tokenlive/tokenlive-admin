const prefix = 'pages.provider.detail.measurement'
const usableStates = new Set(['ok', 'partial'])

// A value is displayable only when its own observation is usable. A sibling
// measurement's status cannot turn a missing value into zero or hide a valid one.
export function providerMetricValue(metric) {
    return metric && usableStates.has(metric.state) ? (metric.value ?? null) : null
}

export function providerMetricDisplay(metric, t, format = String) {
    const value = providerMetricValue(metric)
    const state = metric?.state || 'unavailable'
    const hint = [t(`${prefix}.state.${state}`)]
    for (const field of ['source', 'scope']) {
        if (metric?.[field]) {
            const key = `${prefix}.${field}.${metric[field]}`
            const label = t(key)
            hint.push(label === key ? t(`${prefix}.${field}.unknown`) : label)
        }
    }
    if (metric?.reason) {
        const key = `${prefix}.reason.${metric.reason}`
        const reason = t(key)
        if (reason !== key) hint.push(reason)
    }
    if (metric?.identity === 'provider_name') hint.push(t(`${prefix}.legacyName`))
    const coverage = metric?.coverage
    if (coverage?.start && coverage?.end) {
        hint.push(t(`${prefix}.coverage`, { start: coverage.start, end: coverage.end }))
    }
    return { value: value == null ? '—' : format(value), hint: hint.join(' · ') }
}

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
        ...providerMetricDisplay(health?.[field], t, format),
    }))
}
