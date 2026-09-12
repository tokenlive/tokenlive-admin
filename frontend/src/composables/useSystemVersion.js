import { computed, onMounted, onUnmounted, ref } from 'vue'
import { checkUpdates, getUpdates, getVersionSummary } from '@/apis/modules/system'

function responseData(response) {
    if (response?.success === true && response.data) return response.data
    const failure = new Error(response?.error?.detail || 'Invalid version response')
    failure.response = { status: response?.error?.code, data: response }
    throw failure
}

function isAuthorizationFailure(failure) {
    return [401, 403].includes(failure?.response?.status)
}

function hasRejectedSnapshot(failure) {
    const { status, data } = failure?.response || {}
    return (
        data?.success === false &&
        data.data &&
        ((status === 409 && data.error?.id === 'update_check_disabled') ||
            (status === 429 && data.error?.id === 'update_check_cooldown'))
    )
}

// Own one instance at layout level. About-open calls load(); it never calls check().
export function useSystemVersion() {
    const summary = ref(null)
    const updates = ref(null)
    const checking = ref(false)
    const error = ref(null)
    const now = ref(Date.now())
    const retryDeadline = ref(0)
    // This is frontend build information only. Consumers must label it as such.
    const fallbackVersion = computed(() =>
        summary.value ? '' : typeof __APP_INFO__ === 'undefined' ? 'unknown' : __APP_INFO__.version || 'unknown'
    )
    const retryAfterSeconds = computed(() => Math.max(0, Math.ceil((retryDeadline.value - now.value) / 1000)))
    let disposed = false
    let permissionEpoch = 0
    let updateRevision = 0
    let loadPromise = null
    let checkPromise = null
    let pollTimer
    let clockTimer

    const isCurrent = (epoch) => !disposed && epoch === permissionEpoch

    function clearUpdates() {
        updates.value = null
        retryDeadline.value = 0
    }

    function revokePermission() {
        permissionEpoch++
        updateRevision++
        clearUpdates()
        error.value = null
        if (summary.value) summary.value = { ...summary.value, can_manage_updates: false }
    }

    function applyUpdates(snapshot) {
        updates.value = snapshot
        const seconds = snapshot.retry_after_seconds
        now.value = Date.now()
        retryDeadline.value = now.value + (Number.isFinite(seconds) && seconds > 0 ? Math.ceil(seconds) * 1000 : 0)
    }

    function load() {
        if (disposed) return Promise.resolve()
        if (loadPromise) return loadPromise
        const epoch = permissionEpoch
        if (!checking.value) error.value = null
        loadPromise = (async () => {
            let nextSummary
            try {
                nextSummary = responseData(await getVersionSummary())
            } catch (failure) {
                if (isCurrent(epoch)) {
                    summary.value = null
                    revokePermission()
                    error.value = failure
                }
                return
            }
            if (!isCurrent(epoch)) return
            summary.value = nextSummary
            if (nextSummary.can_manage_updates !== true) {
                revokePermission()
                return
            }
            const revision = updateRevision
            try {
                const snapshot = responseData(await getUpdates())
                if (isCurrent(epoch) && revision === updateRevision) applyUpdates(snapshot)
            } catch (failure) {
                if (!isCurrent(epoch)) return
                // A 401/403 always revokes permission, even if a newer check completed.
                if (isAuthorizationFailure(failure)) {
                    revokePermission()
                    error.value = failure
                } else if (revision === updateRevision) {
                    clearUpdates()
                    error.value = failure
                }
            }
        })().finally(() => {
            loadPromise = null
        })
        return loadPromise
    }

    function check() {
        if (disposed) return Promise.resolve()
        if (checkPromise) return checkPromise
        now.value = Date.now()
        if (
            summary.value?.can_manage_updates !== true ||
            updates.value?.enabled !== true ||
            retryAfterSeconds.value > 0
        ) {
            return Promise.resolve()
        }
        const epoch = permissionEpoch
        updateRevision++
        checking.value = true
        error.value = null
        checkPromise = (async () => {
            try {
                const snapshot = responseData(await checkUpdates())
                if (isCurrent(epoch)) applyUpdates(snapshot)
            } catch (failure) {
                if (!isCurrent(epoch)) return
                if (isAuthorizationFailure(failure)) {
                    revokePermission()
                } else if (hasRejectedSnapshot(failure)) {
                    // R15 retains useful cached state, but remains a failed check.
                    applyUpdates(failure.response.data.data)
                } else {
                    clearUpdates()
                }
                error.value = failure
            } finally {
                // Invalidate cache GETs started either before or during this check.
                updateRevision++
                checking.value = false
                checkPromise = null
            }
        })()
        return checkPromise
    }

    const isVisible = () => typeof document !== 'undefined' && document.visibilityState === 'visible'

    function stopTimers() {
        clearInterval(pollTimer)
        clearInterval(clockTimer)
        pollTimer = undefined
        clockTimer = undefined
    }

    function startTimers() {
        stopTimers()
        if (!isVisible()) return
        now.value = Date.now()
        pollTimer = setInterval(() => void load(), 30_000)
        clockTimer = setInterval(() => {
            now.value = Date.now()
        }, 1000)
    }

    function onFocus() {
        if (isVisible()) {
            now.value = Date.now()
            void load()
        }
    }

    function onVisibilityChange() {
        startTimers()
        if (isVisible()) void load()
    }

    onMounted(() => {
        void load()
        window.addEventListener('focus', onFocus)
        document.addEventListener('visibilitychange', onVisibilityChange)
        startTimers()
    })
    onUnmounted(() => {
        disposed = true
        revokePermission()
        stopTimers()
        window.removeEventListener('focus', onFocus)
        document.removeEventListener('visibilitychange', onVisibilityChange)
    })

    return { summary, updates, fallbackVersion, load, check, checking, error, retryAfterSeconds }
}
