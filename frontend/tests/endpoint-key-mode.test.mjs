import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

import {
    formatProviderKeyLabel,
    maskApiKey,
    resolveEndpointKeyDisplay,
    resolveEndpointKeyMode,
} from '../src/utils/endpointKeyMode.js'

test('maskApiKey never returns full secret', () => {
    assert.equal(maskApiKey(''), '')
    assert.equal(maskApiKey('short'), 'sh****')
    assert.equal(maskApiKey('sk-1234567890abcdef'), 'sk-123...cdef')
})

test('resolveEndpointKeyMode distinguishes oauth, bind, override, and inherit', () => {
    assert.equal(resolveEndpointKeyMode({ auth_type: 'oauth_token', provider_key_id: 'k1', api_key: 'sk-1' }), 'oauth')
    assert.equal(resolveEndpointKeyMode({ auth_type: 'api_key', provider_key_id: 'k1', api_key: 'sk-1' }), 'bind')
    assert.equal(
        resolveEndpointKeyMode({ auth_type: 'api_key', provider_key_id: '', api_key: 'sk-custom' }),
        'override'
    )
    assert.equal(resolveEndpointKeyMode({ auth_type: 'api_key', provider_key_id: '', api_key: '' }), 'inherit')
})

test('resolveEndpointKeyDisplay formats bound key description without leaking full key', () => {
    const provider = {
        api_keys: [{ id: 'k1', value: 'sk-proj-secret-123456', description: 'Main Key' }],
    }
    const bound = resolveEndpointKeyDisplay(
        { auth_type: 'api_key', provider_key_id: 'k1', api_key: 'sk-proj-secret-123456' },
        provider
    )
    assert.equal(bound.mode, 'bind')
    assert.equal(bound.color, 'cyan')
    assert.match(bound.label, /Main Key/)
    assert.doesNotMatch(bound.label, /sk-proj-secret-123456/)

    const override = resolveEndpointKeyDisplay({ auth_type: 'api_key', api_key: 'sk-manual-99999999' }, provider)
    assert.equal(override.mode, 'override')
    assert.equal(override.color, 'orange')
    assert.doesNotMatch(override.label, /sk-manual-99999999/)

    const inherit = resolveEndpointKeyDisplay({ auth_type: 'api_key' }, provider)
    assert.equal(inherit.mode, 'inherit')
})

test('formatProviderKeyLabel handles description-only or key-only items', () => {
    assert.equal(
        formatProviderKeyLabel({ id: 'k1', value: 'sk-1234567890', description: 'Prod' }),
        'Prod (sk-123...7890)'
    )
    assert.equal(formatProviderKeyLabel({ id: 'k1', value: 'sk-1234567890', description: '' }), 'sk-123...7890')
})

test('key binding and preflight confirmation are wired into resource views', async () => {
    const [providerEditSrc, importDialogSrc, endpointEditSrc, providerDetailSrc, modelDetailSrc] = await Promise.all([
        readFile(new URL('../src/views/resource/ProviderEditDialog.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/views/resource/ProviderImportMappingDialog.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/views/resource/EndpointEditDialog.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/views/resource/ProviderDetail.vue', import.meta.url), 'utf8'),
        readFile(new URL('../src/views/resource/ModelDetail.vue', import.meta.url), 'utf8'),
    ])

    assert.match(providerEditSrc, /previewProviderKeyImpact/)
    assert.match(importDialogSrc, /provider_key_id/)
    assert.match(endpointEditSrc, /provider_key_id/)
    assert.match(endpointEditSrc, /keyMode/)
    assert.match(providerDetailSrc, /resolveEndpointKeyDisplay/)
    assert.match(providerDetailSrc, /onProviderEdited/)
    assert.match(modelDetailSrc, /resolveEndpointKeyDisplay/)
})
