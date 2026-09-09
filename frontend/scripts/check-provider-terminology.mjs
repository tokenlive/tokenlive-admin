import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

const root = new URL('../', import.meta.url)
const readSource = (path) => readFile(new URL(path, root), 'utf8')

const [
    protocolSource,
    zhCNSource,
    enUSSource,
    providerFormSource,
    providerListSource,
    providerDetailSource,
    endpointFormSource,
    modelDetailSource,
] = await Promise.all([
    readSource('src/enums/provider.js'),
    readSource('src/locales/lang/zh-CN/pages.js'),
    readSource('src/locales/lang/en-US/pages.js'),
    readSource('src/views/resource/ProviderEditDialog.vue'),
    readSource('src/views/resource/provider.vue'),
    readSource('src/views/resource/ProviderDetail.vue'),
    readSource('src/views/resource/EndpointEditDialog.vue'),
    readSource('src/views/resource/ModelDetail.vue'),
])

const protocolValues = [...protocolSource.matchAll(/value:\s*'([^']+)'/g)].map((match) => match[1])
assert.deepEqual(protocolValues, ['openai', 'anthropic', 'gemini', 'joycode'])

for (const [source, expected] of [
    [
        zhCNSource,
        [
            '上游 API 协议',
            '凭证类型',
            'OpenAI 兼容协议',
            'Anthropic Messages 协议',
            'Gemini Native 协议',
            '继承供应商默认协议',
        ],
    ],
    [
        enUSSource,
        [
            'Upstream API Protocol',
            'Credential Type',
            'OpenAI-compatible API',
            'Anthropic Messages API',
            'Gemini Native API',
            'Inherit Provider Default Protocol',
        ],
    ],
]) {
    for (const text of expected) {
        assert.ok(source.includes(text), `Missing localized provider terminology: ${text}`)
    }
}

for (const source of [providerFormSource, endpointFormSource]) {
    assert.ok(source.includes('getProviderProtocolOptions'), 'Protocol selectors must use shared option metadata')
    assert.ok(!source.includes('<a-select-option value="openai">'), 'Protocol values must not be duplicated in forms')
}

for (const source of [providerListSource, providerDetailSource, modelDetailSource]) {
    assert.ok(source.includes('getProviderProtocolLabel'), 'Protocol displays must use descriptive shared labels')
}

for (const source of [providerDetailSource, modelDetailSource]) {
    assert.ok(
        !source.includes('pages.endpoint.form.protocol.inherited'),
        'Inherited protocol rows must not display an additional inherited suffix'
    )
}

assert.ok(
    providerFormSource.includes(':label="$t(\'pages.provider.form.auth_type\')"'),
    'Provider credential type label must be localized'
)
assert.ok(
    !providerFormSource.includes('pages.provider.form.protocol.hint'),
    'Provider protocol helper text must not be displayed'
)
assert.ok(
    !endpointFormSource.includes('pages.endpoint.form.protocol.hint'),
    'Endpoint protocol helper text must not be displayed separately'
)
assert.ok(
    endpointFormSource.includes('value=""'),
    'Endpoint protocol selector must preserve the empty override value used for inheritance'
)
for (const source of [providerFormSource, endpointFormSource]) {
    assert.ok(source.includes('value="api_key"'), 'API Key credential enum must remain api_key')
    assert.ok(source.includes('value="oauth_token"'), 'OAuth credential enum must remain oauth_token')
}
assert.ok(providerFormSource.includes('...values'), 'Provider form payload must preserve existing field names')
assert.ok(endpointFormSource.includes('...values'), 'Endpoint form payload must preserve existing field names')

console.log('Provider terminology and protocol values are consistent')
