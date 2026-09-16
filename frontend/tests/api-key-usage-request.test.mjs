import test from 'node:test'
import assert from 'node:assert/strict'
import { createAPIKeyUsageLoader } from '../src/utils/api-key-usage-request.js'

const query = { time_range: 'today', sort_by: 'tokens', limit: 10 }

test('sends an authenticated cancellable query and unwraps the response', async () => {
    const controller = new AbortController()
    const load = createAPIKeyUsageLoader({
        send: async (request) => {
            assert.equal(request.url, '/api/v1/dashboard/api-key-ranking')
            assert.equal(request.method, 'get')
            assert.deepEqual(request.params, query)
            assert.equal(request.headers.Authorization, 'fixture-token')
            assert.equal(request.signal, controller.signal)
            return { data: { success: true, data: { state: 'disabled' } } }
        },
        getSession: () => ({ token: 'fixture-token', hasRefreshToken: false }),
    })
    assert.deepEqual(await load(query, { signal: controller.signal }), { state: 'disabled' })
})

for (const status of [403, 503]) {
    test(`${status} stays local and does not refresh or invalidate the session`, async () => {
        const failure = { response: { status } }
        let refreshes = 0
        let invalidations = 0
        const load = createAPIKeyUsageLoader({
            send: async () => {
                throw failure
            },
            getSession: () => ({ token: 'token', hasRefreshToken: true }),
            refreshAccessToken: async () => {
                refreshes++
                return true
            },
            invalidateLocalSession: () => {
                invalidations++
            },
        })
        await assert.rejects(load(query), (error) => error === failure)
        assert.equal(refreshes, 0)
        assert.equal(invalidations, 0)
    })
}

test('401 retries only once with the refreshed access token', async () => {
    let token = 'old'
    let calls = 0
    let refreshes = 0
    const failure = { response: { status: 401 } }
    const load = createAPIKeyUsageLoader({
        send: async (request) => {
            calls++
            assert.equal(request.headers.Authorization, calls === 1 ? 'old' : 'new')
            throw failure
        },
        getSession: () => ({ token, hasRefreshToken: true }),
        refreshAccessToken: async () => {
            token = 'new'
            refreshes++
            return true
        },
    })
    await assert.rejects(load(query), (error) => error === failure)
    assert.equal(calls, 2)
    assert.equal(refreshes, 1)
})

test('refresh failure marks usage authorization uncertain without invalidating the session', async () => {
    let calls = 0
    const refreshFailure = new Error('temporary refresh outage')
    const load = createAPIKeyUsageLoader({
        send: async () => {
            calls++
            throw { response: { status: 401 } }
        },
        getSession: () => ({ token: 'old', hasRefreshToken: true }),
        refreshAccessToken: async () => {
            throw refreshFailure
        },
    })
    await assert.rejects(
        load(query),
        (error) => error.cause === refreshFailure && error.usageAuthenticationFailure === true
    )
    assert.equal(calls, 1)
})

test('401 without refresh credentials invalidates only the local session', async () => {
    let invalidations = 0
    const load = createAPIKeyUsageLoader({
        send: async () => {
            throw { response: { status: 401 } }
        },
        getSession: () => ({ token: 'old', hasRefreshToken: false }),
        invalidateLocalSession: () => {
            invalidations++
        },
    })
    await assert.rejects(load(query), (error) => error.response.status === 401)
    assert.equal(invalidations, 1)
})

test('an already cancelled load never calls the transport', async () => {
    const controller = new AbortController()
    controller.abort()
    let calls = 0
    const load = createAPIKeyUsageLoader({
        send: async () => {
            calls++
        },
        getSession: () => ({ token: 'token' }),
    })
    await assert.rejects(load(query, { signal: controller.signal }), (error) => error.name === 'AbortError')
    assert.equal(calls, 0)
})

test('malformed successful HTTP response is not converted to empty usage', async () => {
    const load = createAPIKeyUsageLoader({
        send: async () => ({ status: 200, data: { success: false } }),
        getSession: () => ({ token: 'token' }),
    })
    await assert.rejects(load(query), /Invalid usage response/)
})
