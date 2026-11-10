const palette = [
    '#2f8cff',
    '#23c7b7',
    '#ff4d5e',
    '#ffb020',
    '#8b5cf6',
    '#6ee7d8',
    '#f472b6',
    '#7bb8ff',
    '#a78bfa',
    '#58d68d',
]

function finite(value) {
    return value != null && Number.isFinite(Number(value))
}

function hasPoints(series) {
    return series.some((item) => (item.values || item.success || item.failure || []).some(finite))
}

function theme(dark) {
    const text = dark ? 'rgba(231, 236, 246, 0.68)' : 'rgba(31, 41, 55, 0.68)'
    return {
        text,
        tooltip: {
            backgroundColor: dark ? 'rgba(18, 24, 38, 0.96)' : 'rgba(255, 255, 255, 0.98)',
            borderColor: dark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.1)',
            textStyle: { color: dark ? '#e7ecf6' : '#1f2937' },
            extraCssText: 'box-shadow: 0 12px 32px rgba(15, 23, 42, 0.18); border-radius: 8px;',
        },
        splitLine: { lineStyle: { color: dark ? 'rgba(126, 145, 178, 0.11)' : 'rgba(15, 23, 42, 0.08)' } },
        axisLine: { lineStyle: { color: dark ? 'rgba(126, 145, 178, 0.18)' : 'rgba(15, 23, 42, 0.12)' } },
    }
}

function fill(color, strong, fade) {
    return {
        type: 'linear',
        x: 0,
        y: 0,
        x2: 0,
        y2: 1,
        colorStops: [
            { offset: 0, color: strong },
            { offset: 1, color: fade },
        ],
    }
}

function line(name, data, color, extra = {}) {
    return {
        name,
        type: 'line',
        smooth: true,
        showSymbol: false,
        connectNulls: false,
        itemStyle: { color },
        data,
        ...extra,
    }
}

// One presentation decision for every legacy monitoring line chart.
// Callers supply names, including which success rate they mean. Zero is data;
// null is a gap. A failed reading never becomes an empty window.
export function monitoringChart({
    state,
    times = [],
    series = [],
    dark = false,
    labels = {},
    rate,
    format,
    axisFormat,
}) {
    const rows = series || []
    const plottable = hasPoints(rows)
    if (state === 'error' && !plottable) return { mode: 'hint', hint: true, options: null }
    if (!plottable) return { mode: 'empty', hint: false, options: null }

    const tone = theme(dark)
    const successRate = rows.length === 1 && rows[0].success && rate
    const specs = successRate
        ? [
              line(labels.success, rows[0].success || [], '#23c7b7', {
                  areaStyle: { color: fill('#23c7b7', 'rgba(35, 199, 183, 0.32)', 'rgba(35, 199, 183, 0.02)') },
              }),
              line(labels.failure, rows[0].failure || [], '#ff4d5e', {
                  areaStyle: { color: fill('#ff4d5e', 'rgba(255, 77, 94, 0.3)', 'rgba(255, 77, 94, 0.02)') },
              }),
              line(labels.rate, rate(rows[0].success || [], rows[0].failure || []), '#8b5cf6', {
                  yAxisIndex: 1,
                  showSymbol: true,
                  symbolSize: 6,
                  lineStyle: { width: 2, type: 'dashed' },
              }),
          ]
        : rows.map((item, index) => {
              const color = item.color || palette[index % palette.length]
              return line(item.name, item.values || [], color, {
                  lineStyle: item.color ? { width: 2 } : undefined,
                  areaStyle: item.color ? { color: fill(color, `${color}47`, `${color}05`) } : undefined,
              })
          })
    const options = {
        tooltip: {
            ...tone.tooltip,
            trigger: 'axis',
            axisPointer: { type: 'cross' },
            ...(format ? { valueFormatter: format } : {}),
        },
        legend:
            specs.length > 1 || successRate
                ? { data: specs.map((item) => item.name), textStyle: { color: tone.text } }
                : undefined,
        grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
        xAxis: {
            type: 'category',
            boundaryGap: false,
            data: times,
            axisLabel: { color: tone.text },
            axisLine: tone.axisLine,
        },
        yAxis: successRate
            ? [
                  {
                      type: 'value',
                      name: labels.requests,
                      minInterval: 1,
                      axisLabel: { color: tone.text },
                      splitLine: tone.splitLine,
                  },
                  {
                      type: 'value',
                      name: labels.rate,
                      min: 0,
                      max: 100,
                      axisLabel: { formatter: '{value} %', color: tone.text },
                      splitLine: { show: false },
                  },
              ]
            : rows.length === 1 && rows[0].color
              ? {
                    type: 'value',
                    name: labels.requests,
                    axisLabel: { color: tone.text, ...(axisFormat ? { formatter: axisFormat } : {}) },
                    splitLine: tone.splitLine,
                }
              : [
                    {
                        type: 'value',
                        name: labels.requests,
                        minInterval: 1,
                        axisLabel: { color: tone.text },
                        splitLine: tone.splitLine,
                    },
                ],
        series: specs,
    }
    if (!options.legend) delete options.legend
    return { mode: 'chart', hint: state != null && state !== 'ok', options }
}
