import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

import { countProviderEndpointHealth } from '../src/utils/providerEndpointHealth.js'

test('available count is all endpoints minus disabled, over all endpoints', () => {
    const { total, up, breakerCount } = countProviderEndpointHealth(
        [
            { id: 'ep-1', enabled: 1 },
            { id: 'ep-2', enabled: 1 },
            { id: 'ep-3', enabled: 0 },
            { id: 'ep-4', enabled: 0 },
        ],
        []
    )

    assert.equal(total, 4)
    assert.equal(up, 2)
    assert.equal(breakerCount, 0)
})

test('a circuit breaker on a disabled endpoint does not reduce available count', () => {
    const { total, up, breakerCount } = countProviderEndpointHealth(
        [
            { id: 'ep-1', enabled: 1 },
            { id: 'ep-2', enabled: 0 },
        ],
        [{ id: 'ep-2', provider_id: 'provider-1' }]
    )

    assert.equal(total, 2)
    assert.equal(up, 1)
    assert.equal(breakerCount, 0)
})

test('circuit breakers are counted separately and do not change the available ratio', () => {
    const { total, up, breakerCount } = countProviderEndpointHealth(
        [
            { id: 'ep-1', enabled: 1 },
            { id: 'ep-2', enabled: 1 },
            { id: 'ep-3', enabled: 0 },
        ],
        [{ id: 'ep-2', provider_id: 'provider-1' }]
    )

    assert.equal(total, 3)
    assert.equal(up, 2)
    assert.equal(breakerCount, 1)
})

test('provider detail uses enabled-only endpoint health counts', async () => {
    const source = await readFile(new URL('../src/views/resource/ProviderDetail.vue', import.meta.url), 'utf8')
    assert.match(
        source,
        /countProviderEndpointHealth\(endpointListData\.value, healthBreakers\.value, providerId\.value\)/
    )
    assert.doesNotMatch(source, /healthTotalEndpoints = computed\(\(\) => endpointListData\.value\.length\)/)
})
