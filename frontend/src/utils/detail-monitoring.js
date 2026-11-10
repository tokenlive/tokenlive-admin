import { monitoringData } from './monitoring-snapshot.js'

const emptyTraffic = () => ({ times: [], series: [] })
const emptyPerformance = () => ({ times: [], avg_ttft_ms: [], otps: [] })
const errorReading = (reason = 'source_unavailable') => ({ state: 'error', error: reason })
const defaults = {
    ranking: () => null,
    traffic: emptyTraffic,
    endpointTraffic: emptyTraffic,
    performance: emptyPerformance,
    endpoints: () => [],
    breakers: () => [],
    events: () => [],
}
const initialState = (pageSize = 20) => ({
    loading: false,
    eventsLoading: false,
    ...Object.fromEntries(Object.entries(defaults).map(([key, create]) => [key, create()])),
    eventTotal: 0,
    eventPage: { current: 1, pageSize },
    readings: {},
})

function checked(response) {
    if (!response || response.success === false) throw new Error('Reading failed')
    return response
}

// A session owns the target, fixed monitoring window and event-page revisions.
// Both detail views use this interface; transport, fan-out and stale-response
// checks stay here instead of being repeated by every view and chart.
export function createDetailMonitoring({ kind, api, onChange = () => {} }) {
    if (!['model', 'provider'].includes(kind)) throw new Error('Unknown monitoring target')
    let state = initialState()
    let generation = 0
    let eventGeneration = 0
    let selection = null
    let current = null
    const emit = (patch) => {
        state = { ...state, ...patch }
        onChange(state)
    }
    const owns = (revision) => revision === generation && selection !== null
    const section = (name, reading, data, revision) => {
        if (!owns(revision)) return
        emit({
            [name]: data ?? defaults[name](),
            readings: { ...state.readings, [name]: reading },
        })
    }
    const queryWindow = (target) => ({
        at: target.end_time,
        timezone: target.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
    })
    const windowOptions = (target) => ({ range: target.time_range, start: target.start_time })
    const readSnapshot = async (query, selections, revision) => {
        let payload
        try {
            payload = checked(await api.dashboard.getSnapshot(query)).data
        } catch {
            for (const [name] of selections) section(name, errorReading(), null, revision)
            return
        }
        for (const [name, key, project = (value) => value] of selections) {
            let reading = payload?.[key] || errorReading('missing_reading')
            let data = null
            try {
                data = monitoringData(reading)
                if (data != null) {
                    if (name === 'ranking' && !Array.isArray(data)) throw new Error('Invalid ranking')
                    data = project(data)
                }
            } catch {
                reading = errorReading('invalid_response')
                data = null
            }
            section(name, reading, data, revision)
        }
    }
    const readList = async (name, load, revision, source) => {
        try {
            const response = checked(await load())
            if (!Array.isArray(response.data)) throw new Error('Invalid list')
            section(name, { state: 'ok', source }, response.data, revision)
            return response.data
        } catch {
            section(name, errorReading(), null, revision)
            return null
        }
    }

    async function modelEndpoints(target, revision) {
        // The generic list includes disabled endpoints used by the inventory KPI.
        // Exhaust its pages; the routing-only endpoint omits disabled entries.
        const rows = []
        for (let page = 1; owns(revision); page++) {
            const response = checked(
                await api.endpoint.getEndpointList({ model_id: target.id, pageSize: 100, current: page })
            )
            if (!Array.isArray(response.data)) throw new Error('Invalid endpoint list')
            rows.push(...response.data)
            if (response.data.length < 100 || (Number.isFinite(response.total) && response.total === rows.length))
                return { data: rows }
        }
        return { data: [] }
    }

    async function providerEndpointTraffic(target, endpoints, revision) {
        if (!owns(revision)) return
        if (endpoints === null) {
            section('endpointTraffic', errorReading('inventory_unavailable'), null, revision)
            return
        }
        const modelCodes = [...new Set(endpoints.map((endpoint) => endpoint.model?.model_code).filter(Boolean))]
        const ownIDs = new Set(endpoints.map((endpoint) => endpoint.id))
        if (!modelCodes.length) {
            const unsupported = endpoints.length > 0
            section(
                'endpointTraffic',
                unsupported
                    ? { state: 'unavailable', coverage: { reason: 'missing_model_identity' } }
                    : { state: 'ok' },
                null,
                revision
            )
            return
        }
        const responses = await Promise.all(
            modelCodes.map(async (model) => {
                try {
                    const response = checked(
                        await api.dashboard.getSnapshot({
                            ...queryWindow(target),
                            traffic: { ...windowOptions(target), model, group_by: 'endpoint' },
                        })
                    )
                    return response.data?.traffic || errorReading('missing_reading')
                } catch {
                    return errorReading()
                }
            })
        )
        if (!owns(revision)) return
        const merged = emptyTraffic()
        let geometry = null
        let usable = 0
        let failed = false
        let partial = endpoints.some((endpoint) => !endpoint.model?.model_code)
        for (const reading of responses) {
            failed ||= reading.state === 'error'
            let data
            try {
                data = monitoringData(reading)
            } catch {
                partial = true
                failed = true
                continue
            }
            if (!data || !Array.isArray(data.series)) {
                failed ||= reading.state === 'ok' || reading.state === 'partial'
                partial = true
                continue
            }
            const identity = JSON.stringify([
                reading.window?.start,
                reading.window?.end,
                reading.window?.timezone,
                (reading.buckets || []).map(({ start, end }) => [start, end]),
            ])
            if (geometry !== null && geometry !== identity) {
                partial = true
                continue
            }
            geometry = identity
            usable++
            partial ||= reading.state !== 'ok'
            merged.times = data.times || []
            for (const row of data.series || []) {
                if (ownIDs.has(row.label) && !merged.series.some((existing) => existing.label === row.label)) {
                    merged.series.push(row)
                }
            }
        }
        section(
            'endpointTraffic',
            {
                state: usable ? (partial ? 'partial' : 'ok') : failed ? 'error' : 'unavailable',
                source: 'mixed',
                ...(partial ? { coverage: { reason: 'missing_endpoint_sources' } } : {}),
            },
            merged,
            revision
        )
    }

    function pageEvents(page = {}) {
        if (!selection) return Promise.resolve()
        const target = selection
        const revision = generation
        const eventRevision = ++eventGeneration
        const eventPage = { ...state.eventPage, ...page }
        const isCurrent = () => owns(revision) && eventRevision === eventGeneration
        emit({
            eventsLoading: true,
            events: [],
            eventPage,
            readings: { ...state.readings, events: null },
        })
        return (async () => {
            try {
                const params = {
                    start_time: target.start_time,
                    end_time: target.end_time,
                    current: eventPage.current,
                    pageSize: eventPage.pageSize,
                    ...(kind === 'model'
                        ? { model_code: target.code }
                        : { provider_ref: target.code || target.id, provider_name_fallback: target.name || undefined }),
                }
                const response = checked(await api.ops.getEvents(params, { enableAbortController: false }))
                if (!Array.isArray(response.data)) throw new Error('Invalid events')
                if (!isCurrent()) return
                // The server remains authoritative for paging totals and filtering.
                emit({
                    events: response.data,
                    eventTotal: response.total || 0,
                    readings: { ...state.readings, events: { state: 'ok' } },
                })
            } catch {
                if (isCurrent()) emit({ events: [], readings: { ...state.readings, events: errorReading() } })
            } finally {
                if (isCurrent()) emit({ eventsLoading: false })
            }
        })()
    }

    function refresh(next) {
        if (!next?.id || (kind === 'model' && !next.code)) {
            stop()
            return Promise.resolve()
        }
        const target = Object.freeze({ ...next })
        const key = JSON.stringify(target)
        if (current?.key === key) return current.promise
        const revision = ++generation
        eventGeneration++
        selection = target
        state = initialState(state.eventPage.pageSize)
        emit({ loading: true })
        const options = windowOptions(target)
        const query = {
            ...queryWindow(target),
            traffic: {
                ...options,
                ...(kind === 'model' ? { model: target.code } : { provider: target.id, group_by: 'provider' }),
            },
            ...(kind === 'model'
                ? {
                      models: { ...options, model: target.code, limit: 1 },
                      performance: { ...options, model: target.code },
                  }
                : { providers: { ...options, provider: target.code || target.id, limit: 1 } }),
        }
        const entry = { key, promise: null }
        current = entry
        entry.promise = (async () => {
            const directory = readList(
                'endpoints',
                () =>
                    kind === 'model'
                        ? modelEndpoints(target, revision)
                        : api.endpoint.getEndpointsByProviderId(target.id),
                revision,
                'catalog'
            )
            const jobs = [
                readSnapshot(
                    query,
                    [
                        ['ranking', kind === 'model' ? 'models' : 'providers', (rows) => rows[0] || null],
                        ['traffic', 'traffic'],
                        ...(kind === 'model' ? [['performance', 'performance']] : []),
                    ],
                    revision
                ),
                directory,
                readList('breakers', () => api.dashboard.getCircuitBreakers(), revision, 'runtime'),
            ]
            if (kind === 'model') {
                jobs.push(
                    readSnapshot(
                        { ...queryWindow(target), traffic: { ...options, model: target.code, group_by: 'endpoint' } },
                        [['endpointTraffic', 'traffic']],
                        revision
                    )
                )
            } else {
                jobs.push(directory.then((endpoints) => providerEndpointTraffic(target, endpoints, revision)))
            }
            const eventJob = pageEvents()
            try {
                await Promise.all(jobs)
            } finally {
                if (owns(revision)) {
                    current = null
                    emit({ loading: false })
                }
            }
            await eventJob
        })()
        return entry.promise
    }

    function stop() {
        generation++
        eventGeneration++
        selection = null
        current = null
        emit(initialState(state.eventPage.pageSize))
    }
    return { getState: () => state, refresh, pageEvents, stop }
}
