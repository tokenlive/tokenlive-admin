export function getStartupAction({ hasAccessToken, hasRefreshToken }) {
    if (hasAccessToken) return 'initialize'
    if (hasRefreshToken) return 'refresh'
    return 'login'
}

export function isAuthenticationFailure(error) {
    return error?.response?.status === 401
}

export function getRefreshFailureAction(error) {
    return isAuthenticationFailure(error) ? 'invalidate' : 'preserve'
}

export function createRefreshCoordinator() {
    let inFlight = null

    return {
        run(refresh) {
            if (!inFlight) {
                inFlight = Promise.resolve()
                    .then(refresh)
                    .finally(() => {
                        inFlight = null
                    })
            }
            return inFlight
        },
    }
}
