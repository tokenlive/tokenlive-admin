const ranges = ['today', '1h', '6h', '24h', '7d']

export function createUsageTrendsController({
    load,
    onChange = () => {},
    setTimeoutFn = setTimeout,
    clearTimeoutFn = clearTimeout,
}) {
    const state = { phase: 'idle', data: null, timeRange: 'today', refreshing: false }
    let authorized = false
    let active = false
    let visible = false
    let disposed = false
    let sequence = 0
    let current = null
    let timer = null
    const getState = () => ({ ...state })
    const emit = () => !disposed && onChange(getState())
    const enabled = () => authorized && active && visible && !disposed
    const cancel = () => {
        sequence++
        if (timer !== null) clearTimeoutFn(timer)
        timer = null
        current?.abort.abort()
        current = null
        state.refreshing = false
    }
    const refresh = () => {
        if (!enabled()) return Promise.resolve()
        if (current) return current.promise
        if (timer !== null) clearTimeoutFn(timer)
        timer = null
        const id = ++sequence
        const abort = new AbortController()
        const entry = { abort, promise: null }
        current = entry
        state.refreshing = true
        if (!state.data) state.phase = 'loading'
        emit()
        entry.promise = (async () => {
            try {
                const data = await load({ time_range: state.timeRange }, { signal: abort.signal })
                if (disposed || id !== sequence || abort.signal.aborted) return
                if (data?.state === 'disabled') {
                    state.data = null
                    state.phase = 'disabled'
                } else if (
                    data?.state === 'ready' &&
                    Number.isFinite(data.summary?.requests) &&
                    data.summary.requests >= 0 &&
                    Array.isArray(data.buckets) &&
                    data.window?.start &&
                    data.window?.end
                ) {
                    state.data = data
                    state.phase = data.summary.requests === 0 ? 'empty' : 'ready'
                } else {
                    throw new Error('Invalid trends response')
                }
            } catch (error) {
                if (disposed || id !== sequence || abort.signal.aborted) return
                const status = error?.response?.status
                const forbidden = status === 401 || status === 403 || error?.usageAuthenticationFailure === true
                state.data = null
                state.phase = forbidden ? 'forbidden' : 'error'
                if (forbidden) authorized = false
            } finally {
                if (!disposed && id === sequence) {
                    current = null
                    state.refreshing = false
                    emit()
                    if (enabled()) {
                        timer = setTimeoutFn(() => {
                            timer = null
                            refresh()
                        }, 30000)
                    }
                }
            }
        })()
        return entry.promise
    }
    const setTimeRange = (value) => {
        if (!ranges.includes(value)) throw new Error('Invalid trends time range')
        if (disposed || state.timeRange === value) return
        cancel()
        state.timeRange = value
        state.data = null
        state.phase = 'idle'
        emit()
        refresh()
    }
    const setFlag = (get, set, value) => {
        if (disposed || get() === (value === true)) return
        set(value === true)
        if (enabled()) refresh()
        else {
            cancel()
            emit()
        }
    }
    return {
        getState,
        refresh,
        setTimeRange,
        setActive: (value) =>
            setFlag(
                () => active,
                (next) => (active = next),
                value
            ),
        setVisible: (value) =>
            setFlag(
                () => visible,
                (next) => (visible = next),
                value
            ),
        setAuthorized(value) {
            setFlag(
                () => authorized,
                (next) => (authorized = next),
                value
            )
            if (!authorized && !disposed) {
                state.data = null
                state.phase = 'forbidden'
                emit()
            }
        },
        dispose() {
            cancel()
            state.data = null
            disposed = true
        },
    }
}
