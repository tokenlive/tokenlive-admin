// A local-error reader: session recovery is delegated to the existing user store.
// No global toast or independent refresh coordinator belongs in this module.
export function createAPIKeyUsageLoader({ send, getSession, refreshAccessToken, invalidateLocalSession }) {
    return async function load(params, { signal } = {}) {
        for (let attempt = 0; attempt < 2; attempt++) {
            if (signal?.aborted) throw new DOMException('Aborted', 'AbortError')
            const session = getSession()
            try {
                const response = await send({
                    method: 'get',
                    url: '/api/v1/dashboard/api-key-ranking',
                    params,
                    signal,
                    headers: { Authorization: session.token },
                })
                if (response.data?.success !== true || !['ready', 'disabled'].includes(response.data?.data?.state)) {
                    throw Object.assign(new Error('Invalid usage response'), { response })
                }
                return response.data.data
            } catch (error) {
                if (signal?.aborted || error.response?.status !== 401 || attempt === 1) throw error
                if (!session.hasRefreshToken) {
                    invalidateLocalSession()
                    throw error
                }
                let refreshed
                try {
                    refreshed = await refreshAccessToken()
                } catch (cause) {
                    throw Object.assign(new Error('Unable to verify usage session'), {
                        cause,
                        usageAuthenticationFailure: true,
                    })
                }
                if (!refreshed) throw error
            }
        }
    }
}
