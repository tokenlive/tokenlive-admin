import test from 'node:test'
import assert from 'node:assert/strict'

import {
    createRefreshCoordinator,
    getRefreshFailureAction,
    getStartupAction,
    isAuthenticationFailure,
} from '../src/utils/session.js'

test('selects the correct startup action for stored credentials', () => {
    assert.equal(getStartupAction({ hasAccessToken: true, hasRefreshToken: true }), 'initialize')
    assert.equal(getStartupAction({ hasAccessToken: false, hasRefreshToken: true }), 'refresh')
    assert.equal(getStartupAction({ hasAccessToken: false, hasRefreshToken: false }), 'login')
})

test('does not classify transient refresh failures as authentication failures', () => {
    assert.equal(isAuthenticationFailure({ response: { status: 500 } }), false)
    assert.equal(isAuthenticationFailure(new Error('Network Error')), false)
    assert.equal(isAuthenticationFailure({ response: { status: 401 } }), true)
})

test('only a refresh 401 invalidates local credentials', () => {
    assert.equal(getRefreshFailureAction({ response: { status: 401 } }), 'invalidate')
    assert.equal(getRefreshFailureAction({ response: { status: 500 } }), 'preserve')
    assert.equal(getRefreshFailureAction(new Error('Network Error')), 'preserve')
})

test('shares one refresh across concurrent callers', async () => {
    let calls = 0
    const coordinator = createRefreshCoordinator()
    const refresh = async () => {
        calls += 1
        await Promise.resolve()
        return 'new-token'
    }

    assert.deepEqual(await Promise.all([coordinator.run(refresh), coordinator.run(refresh)]), [
        'new-token',
        'new-token',
    ])
    assert.equal(calls, 1)
})

test('rejects every caller when the shared refresh fails', async () => {
    const failure = new Error('temporary failure')
    const coordinator = createRefreshCoordinator()
    const refresh = async () => {
        throw failure
    }

    const results = await Promise.allSettled([coordinator.run(refresh), coordinator.run(refresh)])

    assert.deepEqual(
        results.map(({ status }) => status),
        ['rejected', 'rejected']
    )
    assert.equal(results[0].reason, failure)
    assert.equal(results[1].reason, failure)
})
