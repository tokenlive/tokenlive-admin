export const trendSeries = {
    requests: [
        { field: 'requests', label: 'totalRequests', color: '#6885f8' },
        { field: 'success', color: '#56b987' },
        { field: 'failure', color: '#e76e73' },
    ],
    usage: [
        { field: 'input_tokens', color: '#6885f8' },
        { field: 'output_tokens', color: '#56b987' },
        { field: 'cached_tokens', color: '#808a9c', dashed: true },
        { field: 'cache_creation_tokens', color: '#ae8ae7' },
    ],
    latency: [
        { field: 'avg_latency_ms', color: '#6885f8', unit: 'ms' },
        { field: 'avg_ttft_ms', color: '#56b987', unit: 'ms' },
    ],
    errors: [
        { field: 'failure', color: '#e76e73' },
        { field: 'failure_rate', color: '#ae8ae7', unit: '%' },
    ],
}

export function formatTrendValue(value, unit = '', locale = 'en-US', compact = false) {
    if (value === null || value === undefined || !Number.isFinite(Number(value))) return '—'
    return (
        new Intl.NumberFormat(locale, {
            maximumFractionDigits: unit ? 2 : compact ? 1 : 0,
            ...(compact ? { notation: 'compact' } : {}),
        }).format(Number(value)) + (unit === '%' ? '%' : unit ? ` ${unit}` : '')
    )
}

// ISO strings already carry the server's offset; do not silently use browser time.
export function formatTrendTime(value, withDate = false) {
    if (!value) return '—'
    return withDate ? `${value.slice(5, 10)} ${value.slice(11, 16)}` : value.slice(11, 16)
}

export function buildUsageTrendsChart({ data, category, timeRange, t, dark, locale }) {
    const text = dark ? '#a3adc0' : '#64748b'
    const grid = dark ? 'rgba(126,145,178,0.15)' : 'rgba(15,23,42,0.07)'
    const specs = trendSeries[category]
    const buckets = data?.buckets || []
    const label = (field) => t(`pages.dashboard.usageTrends.${field}`)
    return {
        animationDurationUpdate: 200,
        grid: { left: 12, right: 12, top: 24, bottom: 12, containLabel: true },
        tooltip: {
            trigger: 'axis',
            confine: true,
            backgroundColor: dark ? '#171c2a' : '#ffffff',
            borderColor: grid,
            textStyle: { color: dark ? '#e7ecf6' : '#111827' },
        },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: buckets.map((bucket) => formatTrendTime(bucket.start, timeRange === '7d')),
            axisLine: { lineStyle: { color: grid } },
            axisTick: { show: false },
            axisLabel: { color: text, hideOverlap: true, margin: 14 },
        },
        yAxis: [
            {
                type: 'value',
                min: 0,
                minInterval: category === 'latency' ? 0 : 1,
                axisLabel: {
                    color: text,
                    formatter: (v) => formatTrendValue(v, category === 'latency' ? 'ms' : '', locale, true),
                },
                splitLine: { lineStyle: { color: grid, type: 'dashed' } },
            },
            ...(category === 'errors'
                ? [
                      {
                          type: 'value',
                          min: 0,
                          max: 100,
                          axisLabel: { color: '#ae8ae7', formatter: '{value}%' },
                          splitLine: { show: false },
                      },
                  ]
                : []),
        ],
        series: specs.map((spec, i) => ({
            name: label(spec.label || spec.field),
            type: 'line',
            smooth: 0.15,
            smoothMonotone: 'x',
            showSymbol: buckets.length === 1,
            symbolSize: 5,
            connectNulls: false,
            yAxisIndex: spec.unit === '%' ? 1 : 0,
            itemStyle: { color: spec.color },
            lineStyle: { width: 2, type: spec.dashed ? 'dashed' : 'solid' },
            areaStyle:
                i === 0
                    ? {
                          color: {
                              type: 'linear',
                              x: 0,
                              y: 0,
                              x2: 0,
                              y2: 1,
                              colorStops: [
                                  { offset: 0, color: `${spec.color}30` },
                                  { offset: 1, color: `${spec.color}02` },
                              ],
                          },
                      }
                    : undefined,
            tooltip: { valueFormatter: (value) => formatTrendValue(value, spec.unit, locale) },
            data: buckets.map((bucket) => bucket[spec.field] ?? null),
        })),
    }
}
