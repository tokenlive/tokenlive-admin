const defaultQuery = { time_range: 'today', sort_by: 'tokens', limit: 10 }
const queryKey = (query) => JSON.stringify([query.time_range, query.sort_by, query.limit])

function phaseFor(data) {
    if (data?.state === 'disabled') return 'disabled'
    if (
        data?.state !== 'ready' ||
        !Array.isArray(data.items) ||
        !Number.isFinite(data.summary?.request_count) ||
        data.summary.request_count < 0
    ) {
        throw new Error('Invalid usage data')
    }
    return data.summary.request_count === 0 ? 'empty' : 'ready'
}

function validateQuery(query) {
    if (
        !['today', '1h', '6h', '24h', '7d'].includes(query.time_range) ||
        !['tokens', 'request_count', 'cost'].includes(query.sort_by) ||
        ![10, 20, 50].includes(query.limit)
    ) {
        throw new Error('Invalid usage query')
    }
}

export function createAPIKeyUsageController({
    load,
    onChange = () => {},
    setTimeoutFn = setTimeout,
    clearTimeoutFn = clearTimeout,
}) {
    const state = { phase: 'idle', data: null, stale: false, error: null, query: { ...defaultQuery } }
    const cache = new Map()
    let authorized = false
    let active = false
    let visible = false
    let disposed = false
    let timer = null
    let sequence = 0
    let current = null

    const getState = () => ({ ...state, query: { ...state.query } })
    const emit = () => {
        if (!disposed) onChange(getState())
    }
    const enabled = () => !disposed && authorized && active && visible
    const clearTimer = () => {
        if (timer !== null) clearTimeoutFn(timer)
        timer = null
    }
    const cancelCurrent = () => {
        clearTimer()
        sequence++
        current?.abort.abort()
        current = null
    }
    const clearData = () => {
        cache.clear()
        state.data = null
        state.stale = false
        state.error = null
    }
    const schedule = () => {
        clearTimer()
        if (enabled()) {
            timer = setTimeoutFn(() => {
                timer = null
                refresh()
            }, 30000)
        }
    }

    function refresh() {
        if (!enabled()) return Promise.resolve()
        const key = queryKey(state.query)
        if (current?.key === key) return current.promise
        clearTimer()
        const id = ++sequence
        const abort = new AbortController()
        const query = { ...state.query }
        state.error = null
        if (!state.data) state.phase = 'loading'
        emit()
        // Install the in-flight entry before a synchronously throwing loader can
        // finish. All errors are local state, never an unhandled polling rejection.
        const entry = { id, key, abort, promise: null }
        current = entry
        entry.promise = (async () => {
            try {
                const data = await load(query, { signal: abort.signal })
                if (disposed || id !== sequence || abort.signal.aborted) return
                const phase = phaseFor(data)
                if (phase === 'disabled') cache.delete(key)
                else cache.set(key, data)
                state.data = phase === 'disabled' ? null : data
                state.phase = phase
                state.stale = false
                state.error = null
            } catch (error) {
                if (disposed || id !== sequence || abort.signal.aborted) return
                const status = error?.response?.status || 0
                if (status === 401 || status === 403 || error?.usageAuthenticationFailure === true) {
                    authorized = false
                    clearData()
                    state.phase = 'forbidden'
                } else {
                    const previous = cache.get(key)
                    state.data = previous || null
                    state.phase = previous ? phaseFor(previous) : 'error'
                    state.stale = !!previous
                    // Do not retain Axios request headers or response internals.
                    state.error = { status, reason: 'unavailable' }
                }
            } finally {
                if (!disposed && id === sequence) {
                    current = null
                    emit()
                    schedule()
                }
            }
        })()
        return entry.promise
    }

    const setQuery = (patch) => {
        if (disposed) return
        const next = { ...state.query, ...patch }
        validateQuery(next)
        if (queryKey(next) === queryKey(state.query)) return
        cancelCurrent()
        state.query = next
        state.data = authorized ? cache.get(queryKey(next)) || null : null
        state.stale = false
        state.error = null
        state.phase = state.data ? phaseFor(state.data) : 'idle'
        emit()
        refresh()
    }
    const setAuthorized = (value) => {
        if (disposed || authorized === (value === true)) return
        authorized = value === true
        if (!authorized) {
            cancelCurrent()
            clearData()
            state.phase = 'forbidden'
            emit()
        } else {
            refresh()
        }
    }
    const setActive = (value) => {
        if (disposed || active === (value === true)) return
        active = value === true
        if (enabled()) refresh()
        else cancelCurrent()
    }
    const setVisible = (value) => {
        if (disposed || visible === (value === true)) return
        visible = value === true
        if (enabled()) refresh()
        else cancelCurrent()
    }
    const dispose = () => {
        if (disposed) return
        cancelCurrent()
        clearData()
        state.phase = 'idle'
        disposed = true
    }
    return { getState, refresh, setQuery, setAuthorized, setActive, setVisible, dispose }
}
