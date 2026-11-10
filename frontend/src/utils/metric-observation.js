const prefix = 'pages.monitoring.measurement'
const usableStates = new Set(['ok', 'partial'])

// A value is displayable only when its own observation is usable. A sibling
// measurement's status cannot turn a missing value into zero or hide a valid one.
export function metricValue(metric) {
    return metric && usableStates.has(metric.state) ? (metric.value ?? null) : null
}

export function metricDisplay(metric, t, format = String) {
    const value = metricValue(metric)
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

// Ant Design reverses the comparator for descending order, including nulls.
export function compareMetric(a, b, order = 'ascend', project = (value) => value) {
    const left = metricValue(a),
        right = metricValue(b)
    if (left == null || right == null) {
        const missing = left == null ? (right == null ? 0 : 1) : -1
        return order === 'descend' ? -missing : missing
    }
    return project(left) - project(right)
}
