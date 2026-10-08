// State machine for the Homebrew click-upgrade flow in the About dialog.
// Pure controller, no Vue imports, so node:test can drive it with fakes.
// Server facts (capability, task states) are never inferred client-side.

export const TERMINAL_TASK_STATES = ['succeeded', 'failed', 'needs_attention', 'confirmation_expired']

export function isTerminalTask(task) {
    return TERMINAL_TASK_STATES.includes(task?.state)
}

function unwrap(response) {
    if (response?.success === true && response.data !== undefined) return response.data
    const failure = new Error(response?.error?.detail || 'Invalid upgrade response')
    failure.response = { status: response?.error?.code, data: response }
    throw failure
}

const DEFAULT_POLL_INTERVAL = 2000

export function createUpgradeController({
    api,
    onChange = () => {},
    onSucceeded = () => {},
    setTimeoutFn = setTimeout,
    clearTimeoutFn = clearTimeout,
    pollInterval = DEFAULT_POLL_INTERVAL,
}) {
    const state = {
        capability: null,
        preparation: null,
        task: null,
        busy: false,
        error: null,
    }
    let authorized = false
    let active = false
    let disposed = false
    let pollTimer = null

    const getState = () => ({
        ...state,
        capability: state.capability && { ...state.capability },
        preparation: state.preparation && { ...state.preparation },
        task: state.task && { ...state.task },
    })
    const emit = () => {
        if (!disposed) onChange(getState())
    }
    const clearTimer = () => {
        if (pollTimer !== null) {
            clearTimeoutFn(pollTimer)
            pollTimer = null
        }
    }
    const pollScheduled = () => authorized && active && state.task && !isTerminalTask(state.task)

    function schedulePoll() {
        clearTimer()
        if (pollScheduled()) {
            pollTimer = setTimeoutFn(() => {
                pollTimer = null
                void pollTask()
            }, pollInterval)
        }
    }

    async function pollTask() {
        let finished = false
        try {
            const data = unwrap(await api.task(state.task?.task_id || ''))
            if (data?.task_id === (state.task?.task_id || data.task_id)) {
                state.task = data
                if (isTerminalTask(data)) {
                    finished = true
                    if (data.state === 'succeeded') {
                        // Success changes local versions: refresh capability
                        // and let the layout reload the version summary.
                        await refreshCapability()
                        onSucceeded()
                    }
                }
            }
            state.error = null
        } catch (failure) {
            // Transient poll failures keep polling; only the server can end
            // a running task. Authorization loss stops it.
            if ([401, 403].includes(failure?.response?.status)) {
                state.task = null
                emit()
                return
            }
            state.error = failure
        }
        emit()
        if (!finished) schedulePoll()
    }

    async function refreshCapability() {
        if (!authorized) return
        state.busy = true
        emit()
        try {
            const data = unwrap(await api.capability())
            state.capability = data
            state.error = null
            // Adopt a task another session may have started.
            if (data.active_task && (!state.task || isTerminalTask(state.task))) {
                state.task = data.active_task
            }
        } catch (failure) {
            if ([401, 403].includes(failure?.response?.status)) {
                state.capability = null
            }
            state.error = failure
        }
        state.busy = false
        emit()
        schedulePoll()
    }

    async function prepare(targetVersion) {
        if (!authorized || state.busy || state.preparation) return
        // Frontend gate mirrors the server: an unsupported environment never
        // reaches the confirm flow, whatever the button state says.
        if (state.capability && state.capability.supported !== true) return
        state.busy = true
        state.error = null
        emit()
        try {
            state.preparation = unwrap(await api.prepare({ target_version: targetVersion }))
        } catch (failure) {
            state.error = failure
        }
        state.busy = false
        emit()
    }

    function cancelPreparation() {
        state.preparation = null
        emit()
    }

    async function confirm() {
        const preparation = state.preparation
        if (!preparation || state.busy) return
        state.busy = true
        state.error = null
        emit()
        try {
            state.task = unwrap(
                await api.submit({
                    task_id: preparation.task_id,
                    credential: preparation.credential,
                    confirm: true,
                })
            )
            state.preparation = null
        } catch (failure) {
            state.error = failure
        }
        state.busy = false
        emit()
        schedulePoll()
    }

    function setAuthorized(value) {
        authorized = value === true
        if (!authorized) {
            clearTimer()
            state.capability = null
            state.preparation = null
            state.task = null
            state.error = null
        }
        emit()
    }

    function setActive(value) {
        active = value === true
        if (active && authorized) {
            void refreshCapability()
        } else {
            clearTimer()
        }
        emit()
    }

    function dispose() {
        disposed = true
        clearTimer()
    }

    return { getState, refreshCapability, prepare, cancelPreparation, confirm, setAuthorized, setActive, dispose }
}
