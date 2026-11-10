// Wire-to-display adapter. Absolute bucket times stay authoritative; presentation
// labels never determine a query window or cache identity.
export function monitoringQuery(kind, params = {}) {
    const query = { timezone: params.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone }
    if (params.end_time) query.at = params.end_time
    const options = { range: params.time_range || '1h' }
    if (params.start_time) options.start = params.start_time
    if (kind === 'overview') query.overview = true
    else {
        if (params.model) options.model = params.model
        if (params.group_by) options.group_by = params.group_by
        if (params.provider) options.provider = params.provider
        if (params.sort_by) options.sort = params.sort_by
        if (params.limit !== undefined) options.limit = params.limit
        query[kind] = options
    }
    return query
}

export function monitoringData(reading) {
    if (!reading?.data || ['unavailable', 'error'].includes(reading.state)) return null
    const data = reading.data
    if (!reading.buckets?.length || Array.isArray(data)) return data
    const days = new Date(reading.window.end) - new Date(reading.window.start) > 86400000
    const times = reading.buckets.map(({ end }) =>
        days ? `${end.slice(5, 10)} ${end.slice(11, 16)}` : end.slice(11, 16)
    )
    return { ...data, times }
}

export function monitoringResponse(response, kind) {
    const reading = response.data?.[kind]
    return { ...response, data: monitoringData(reading), monitoring: reading }
}

export function monitoringSnapshot(payload) {
    return {
        overview: monitoringData(payload.overview),
        trends: monitoringData(payload.traffic),
        model_ranking: monitoringData(payload.models),
        monitoring: { overview: payload.overview, traffic: payload.traffic, models: payload.models },
    }
}

export function monitoringSuccessRates(success = [], failure = []) {
    return success.map((value, index) => {
        const failed = failure[index]
        if (value == null || failed == null) return null
        const total = value + failed
        return total > 0 ? Number(((value / total) * 100).toFixed(1)) : null
    })
}

export function monitoringStatus(readings) {
    const priority = { error: 3, unavailable: 2, partial: 1, ok: 0 }
    return readings
        .filter(Boolean)
        .reduce(
            (worst, reading) => ((priority[reading.state] || 0) > (priority[worst?.state] || 0) ? reading : worst),
            null
        )
}
