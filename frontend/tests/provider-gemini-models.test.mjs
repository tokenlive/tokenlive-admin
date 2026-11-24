import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { createRenderer, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/lang/zh-CN/pages.js'
import { buildEndpointCode, ENDPOINT_CODE_MAX_LENGTH } from '../src/utils/provider-models.js'

const moduleURL = (source) => `data:text/javascript,${encodeURIComponent(source)}`
const stubs = {
    '@/apis': moduleURL(
        'const domain = () => new Proxy({}, { get: (_, key) => (...args) => globalThis.__providerApi[key](...args) }); export default { model: domain(), provider: domain(), endpoint: domain(), space: domain() }'
    ),
    '@/config': moduleURL('export const config = () => true'),
    'ant-design-vue': moduleURL(
        'export const message = new Proxy({}, {get: (_, type) => text => { globalThis.__providerMessages.push({type,text}); return type === "loading" ? () => {} : undefined }})'
    ),
}
const tags = new Set()
registerHooks({
    resolve(specifier, context, nextResolve) {
        if (stubs[specifier]) return { url: stubs[specifier], shortCircuit: true }
        if (specifier.startsWith('@/')) {
            const path = specifier.slice(2)
            return {
                url: new URL(`../src/${path}${path.endsWith('.vue') ? '' : '.js'}`, import.meta.url).href,
                shortCircuit: true,
            }
        }
        return nextResolve(specifier, context)
    },
    load(url, context, nextLoad) {
        if (url.endsWith('.vue')) {
            const source = readFileSync(new URL(url), 'utf8')
            for (const [, tag] of source.matchAll(/<(a-[a-z-]+)\b/g)) tags.add(tag)
            const descriptor = parse(source).descriptor
            const script = compileScript(descriptor, { id: url, genDefaultAs: '__component' })
            const template = compileTemplate({
                source: descriptor.template.content,
                filename: url,
                id: url,
                compilerOptions: { bindingMetadata: script.bindings, hoistStatic: false },
            })
            return {
                format: 'module',
                source: `${script.content}\n${template.code}\n__component.render = render; export default __component`,
                shortCircuit: true,
            }
        }
        return nextLoad(url, context)
    },
})

const { default: FetchModelsDrawer } = await import('../src/views/resource/ProviderFetchModelsDrawer.vue')
const { default: ImportMappingDialog } = await import('../src/views/resource/ProviderImportMappingDialog.vue')

const renderer = createRenderer({
    createElement: (tag) => ({ tag, children: [], props: {} }),
    createText: (text) => ({ text }),
    createComment: (text) => ({ comment: text }),
    setText: (node, text) => {
        node.text = text
    },
    setElementText: (node, text) => {
        node.children = [{ text }]
    },
    patchProp: (node, key, _, value) => {
        node.props[key] = value
    },
    parentNode: (node) => node.parent,
    nextSibling: (node) => node.parent?.children[node.parent.children.indexOf(node) + 1] || null,
    insert(node, parent, anchor) {
        if (node.parent) node.parent.children.splice(node.parent.children.indexOf(node), 1)
        node.parent = parent
        const index = anchor ? parent.children.indexOf(anchor) : -1
        if (index >= 0) parent.children.splice(index, 0, node)
        else parent.children.push(node)
    },
    remove(node) {
        const children = node.parent?.children
        if (children) children.splice(children.indexOf(node), 1)
    },
})

const flush = async () => {
    await new Promise((resolve) => setImmediate(resolve))
    await nextTick()
}

function mount(Component, props = {}) {
    globalThis.__providerMessages = []
    const app = renderer.createApp(Component, props)
    app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh }, missingWarn: false, fallbackWarn: false }))
    for (const tag of tags) {
        app.component(tag, {
            inheritAttrs: false,
            setup(_, { attrs, slots, expose }) {
                if (tag === 'a-form') {
                    expose({ validateFields: async () => ({ ...attrs.model }), resetFields() {}, clearValidate() {} })
                }
                return () => h(tag, attrs, [slots.default?.(), slots.footer?.(), slots.action?.()])
            },
        })
    }
    const root = { children: [] }
    app.mount(root)
    return { app, state: app._instance.setupState, root }
}

function nodes(root, tag) {
    return root.children?.flatMap((child) => [...(child.tag === tag ? [child] : []), ...nodes(child, tag)]) || []
}

function formBoundary(state) {
    state.formRef = { validateFields: async () => ({ ...state.formData }), resetFields() {}, clearValidate() {} }
}

const GEMINI_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta'
const nativeModel = (extra = {}) => ({
    id: 'gemini-2.0-flash',
    object: 'model',
    owned_by: 'google',
    name: 'models/gemini-2.0-flash',
    display_name: 'Native Flash Display',
    description: 'Native description with useful details. '.repeat(12),
    input_token_limit: 1048576,
    output_token_limit: 8192,
    supported_generation_methods: ['generateContent', 'countTokens'],
    ...extra,
})
const nativeMetadata = {
    name: 'models/gemini-2.0-flash',
    display_name: 'Native Flash Display',
    description: 'Native description with useful details. '.repeat(12),
    input_token_limit: '1048576',
    output_token_limit: '8192',
    supported_generation_methods: '["generateContent","countTokens"]',
}
const existingModelRecord = (extra = {}) => ({
    id: 'existing',
    model_name: 'Existing Model',
    model_code: 'existing-code',
    model_type: 'normal',
    space_code: 'demo',
    enabled: 1,
    request_types: '["gemini_generate_content"]',
    abilities: '[]',
    ...extra,
})
const importContext = (models, extra = {}) => ({
    providerId: 'google-provider',
    providerCode: 'google',
    protocol: 'gemini',
    space_code: 'demo',
    base_url: GEMINI_BASE_URL,
    keysToCreate: [{ id: 'test-key', value: '' }],
    models,
    ...extra,
})

function importAPI(extra = {}) {
    const writes = []
    globalThis.__providerApi = {
        getModelList: async () => ({ success: true, data: [], total: 0 }),
        getSpaceList: async () => ({ success: true, data: [{ code: 'demo', name: 'Demo' }] }),
        createModel: async (payload) => {
            writes.push({ type: 'model', payload })
            return { success: true, data: { id: `created-${writes.length}` } }
        },
        updateModel: async (...args) => {
            writes.push({ type: 'update', args })
            return { success: true }
        },
        createEndpoint: async (payload) => {
            writes.push({ type: 'endpoint', payload })
            return { success: true }
        },
        ...extra,
    }
    return writes
}

async function fetchDrawer(t, result, provider = {}) {
    importAPI({ fetchProviderModels: async () => ({ success: true, data: result }) })
    const confirmed = []
    const mounted = mount(FetchModelsDrawer, { onConfirm: (payload) => confirmed.push(payload) })
    t.after(() => mounted.app.unmount())
    mounted.state.handleOpen({ id: 'google-provider', protocol: 'gemini', url: GEMINI_BASE_URL, ...provider })
    formBoundary(mounted.state)
    await nodes(mounted.root, 'a-button')[0].props.onClick()
    await flush()
    return { ...mounted, confirmed }
}

async function selectAndConfirm({ root }, ids) {
    nodes(root, 'a-table')[0].props['row-selection'].onChange(ids)
    await nextTick()
    await nodes(root, 'a-button').at(-1).props.onClick()
}

test('Gemini drawer confirms complete native model objects and preserves selection across search filters', async (t) => {
    const discovered = nativeModel()
    const drawer = await fetchDrawer(t, { models: [discovered] })
    const rowSelection = nodes(drawer.root, 'a-table')[0].props['row-selection']
    assert.equal(rowSelection.preserveSelectedRowKeys, true)
    await selectAndConfirm(drawer, [discovered.id])
    assert.equal(drawer.confirmed.length, 1)
    assert.deepEqual(drawer.confirmed[0].models, [discovered])
})

test('Gemini drawer defaults the native base URL and imports the normalized fetch result base_url', async (t) => {
    const discovered = nativeModel()
    const drawer = await fetchDrawer(t, { models: [discovered], base_url: GEMINI_BASE_URL }, { url: '' })
    assert.equal(drawer.state.formData.base_url, GEMINI_BASE_URL)
    await selectAndConfirm(drawer, [discovered.id])
    assert.equal(drawer.confirmed[0].base_url, GEMINI_BASE_URL)

    const normalized = await fetchDrawer(
        t,
        { models: [discovered], base_url: GEMINI_BASE_URL },
        { url: `${GEMINI_BASE_URL}/openai/` }
    )
    await selectAndConfirm(normalized, [discovered.id])
    assert.equal(normalized.confirmed[0].base_url, GEMINI_BASE_URL)
    const writes = importAPI()
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())
    await mapping.state.handleOpen(importContext(normalized.confirmed[0].models, normalized.confirmed[0]))
    await nextTick()
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.equal(writes.find((write) => write.type === 'endpoint').payload.url, GEMINI_BASE_URL)
})

test('Drawer blocks confirm when base_url changed after fetch, clears stale models on failure, and ignores stale races', async (t) => {
    const discovered = nativeModel()
    const drawer = await fetchDrawer(t, { models: [discovered], base_url: GEMINI_BASE_URL })
    nodes(drawer.root, 'a-table')[0].props['row-selection'].onChange([discovered.id])
    await nextTick()

    // Editing base_url without refetching must block confirm.
    drawer.state.formData.base_url = 'https://proxy.example/v1beta/openai'
    await nodes(drawer.root, 'a-button').at(-1).props.onClick()
    assert.deepEqual(drawer.confirmed, [])
    assert.equal(globalThis.__providerMessages.at(-1).type, 'warning')

    // A failed fetch clears stale modelList and selectedModels.
    globalThis.__providerApi.fetchProviderModels = async () => {
        throw new Error('network error')
    }
    await nodes(drawer.root, 'a-button')[0].props.onClick()
    await flush()
    assert.deepEqual(drawer.state.modelList, [])
    assert.deepEqual(drawer.state.selectedModels, [])

    // Stale concurrent response from an earlier provider is ignored.
    const pending = []
    globalThis.__providerApi.fetchProviderModels = () => new Promise((resolve) => pending.push(resolve))
    drawer.state.handleOpen({ id: 'provider-a', protocol: 'gemini', url: 'https://a.example/v1beta' })
    formBoundary(drawer.state)
    const first = drawer.state.handleFetchModels()
    await flush()
    drawer.state.handleOpen({ id: 'provider-b', protocol: 'gemini', url: 'https://b.example/v1beta' })
    formBoundary(drawer.state)
    const second = drawer.state.handleFetchModels()
    await flush()
    assert.equal(pending.length, 2)
    pending[1]({
        success: true,
        data: { models: [nativeModel({ id: 'model-b' })], base_url: 'https://b.example/v1beta' },
    })
    await second
    pending[0]({
        success: true,
        data: { models: [nativeModel({ id: 'model-a' })], base_url: 'https://a.example/v1beta' },
    })
    await first
    assert.deepEqual(
        drawer.state.modelList.map((m) => m.id),
        ['model-b']
    )
    assert.equal(drawer.state.formData.base_url, 'https://b.example/v1beta')
})

test('Gemini drawer displays searchable native details, limits, and methods with unknown missing limits', async (t) => {
    const unknown = nativeModel({ id: 'unknown-limits', input_token_limit: undefined, output_token_limit: undefined })
    const drawer = await fetchDrawer(t, { models: [nativeModel(), unknown] })
    const table = nodes(drawer.root, 'a-table')[0]
    assert.ok(nodes(drawer.root, 'a-drawer')[0].props.width > 660)
    for (const field of ['input_token_limit', 'output_token_limit', 'supported_generation_methods']) {
        assert.ok(
            table.props.columns.some((column) => column.dataIndex === field),
            `missing ${field} column`
        )
    }
    for (const query of ['native flash display', 'useful details', 'counttokens', 'models/gemini-2.0-flash']) {
        nodes(drawer.root, 'a-input-search')[0].props['onUpdate:value'](query)
        await nextTick()
        assert.ok(nodes(drawer.root, 'a-table')[0].props['data-source'].length > 0, `cannot search ${query}`)
    }
    const limitColumn = table.props.columns.find((column) => column.dataIndex === 'input_token_limit')
    assert.equal(limitColumn.customRender({ text: undefined, record: unknown }), '未知')
    const modelColumn = table.props.columns.find((column) => column.dataIndex === 'id')
    const details = modelColumn.customRender({ text: unknown.id, record: unknown })
    assert.ok(details.children.some((child) => child?.children === unknown.display_name))
    assert.ok(details.children.some((child) => child?.children === unknown.description))
})

test('Gemini drawer keeps embed-only and undeclared methods visible but blocks selection and bypassed confirm', async (t) => {
    const blocked = [
        nativeModel({ id: 'embed-only', supported_generation_methods: ['embedContent'] }),
        nativeModel({ id: 'gemini-name-without-methods', supported_generation_methods: undefined }),
        nativeModel({ id: 'count-only', supported_generation_methods: ['countTokens'] }),
    ]
    const generator = nativeModel({
        id: 'embed-in-name-but-generates',
        supported_generation_methods: ['generateContent'],
    })
    const drawer = await fetchDrawer(t, { models: [generator, ...blocked] })
    const table = nodes(drawer.root, 'a-table')[0]
    assert.deepEqual(
        table.props['data-source'].map((item) => item.id),
        [generator.id, ...blocked.map((item) => item.id)]
    )
    assert.equal(typeof table.props['row-selection'].getCheckboxProps, 'function')
    assert.equal(table.props['row-selection'].getCheckboxProps(generator).disabled, false)
    for (const item of blocked) {
        const checkbox = table.props['row-selection'].getCheckboxProps(item)
        assert.equal(checkbox.disabled, true)
        assert.match(checkbox.title, /generateContent/)
        drawer.state.selectedModels = [generator.id, item.id]
        drawer.state.handleConfirm()
        assert.deepEqual(drawer.confirmed, [])
        assert.equal(drawer.state.visible, true)
    }
    await selectAndConfirm(drawer, [generator.id])
    assert.equal(drawer.confirmed[0].models[0].id, generator.id)
})

test('Gemini mapping uses imported limits, no inferred abilities, edited system code/name, and canonical native metadata', async (t) => {
    const writes = importAPI()
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())
    await mapping.state.handleOpen(importContext([nativeModel({ output_token_limit: 65536 })]))
    mapping.state.newModelCodes['gemini-2.0-flash'] = 'my-edited-code'
    await nextTick()
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(
        writes.map((write) => write.type),
        ['model', 'endpoint']
    )
    const created = writes[0].payload
    assert.equal(created.model_code, 'my-edited-code')
    assert.equal(created.model_name, 'my-edited-code')
    assert.deepEqual(JSON.parse(created.request_types), ['gemini_generate_content'])
    assert.equal(created.context_length, 1048576)
    assert.equal(created.max_output_tokens, 65536)
    assert.deepEqual(JSON.parse(created.abilities), [])
    assert.ok(created.description.length <= 255)
    const endpoint = writes[1].payload
    assert.equal(endpoint.real_model, 'gemini-2.0-flash')
    assert.deepEqual(endpoint.metadata, { ...nativeMetadata, output_token_limit: '65536' })
    assert.equal(Object.hasOwn(endpoint.metadata, 'context_length'), false)
    assert.equal(Object.hasOwn(endpoint.metadata, 'max_output_tokens'), false)
    assert.ok(Object.values(endpoint.metadata).every((value) => typeof value === 'string'))
})

test('Gemini mapping adds native metadata to existing model endpoints, excludes smart models, and warns on request_types mismatch', async (t) => {
    const existing = existingModelRecord({
        id: 'existing',
        model_code: 'system-code',
        model_name: 'User model name',
        request_types: '["chat_completion"]',
        context_length: 32768,
        max_output_tokens: 2048,
        abilities: '["vision"]',
        description: 'User description',
    })
    const smartModel = existingModelRecord({
        id: 'smart-1',
        model_type: 'smart',
        model_code: 'smart-router',
        model_name: 'Smart Router',
    })
    const before = structuredClone(existing)
    const writes = importAPI({ getModelList: async () => ({ success: true, data: [existing, smartModel] }) })
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())
    await mapping.state.handleOpen(importContext([nativeModel()]))
    assert.deepEqual(
        mapping.state.mappableExistingModels.map((item) => item.id),
        ['existing']
    )
    mapping.state.mapping['gemini-2.0-flash'] = existing.id
    await nextTick()
    assert.equal(mapping.state.hasIncompatibleExistingRequestTypes(existing.id), true)
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(
        writes.map((write) => write.type),
        ['endpoint']
    )
    assert.deepEqual(existing, before)
    assert.equal(writes[0].payload.model_id, 'existing')
    assert.equal(writes[0].payload.real_model, 'gemini-2.0-flash')
    assert.ok(writes[0].payload.code.startsWith('ep-system-code-'))
    assert.deepEqual(writes[0].payload.metadata, nativeMetadata)
    assert.ok(globalThis.__providerMessages.some((m) => m.type === 'warning' && /gemini_generate_content/.test(m.text)))
})

test('Mapping prevalidates duplicate new codes and conflicts with existing models before writing', async (t) => {
    const existing = existingModelRecord({ id: 'existing-1', model_code: 'taken-code', model_name: 'Taken Name' })
    const writes = importAPI({ getModelList: async () => ({ success: true, data: [existing] }) })
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())

    // Duplicate new codes within the same batch
    await mapping.state.handleOpen(importContext([nativeModel({ id: 'gemini_a' }), nativeModel({ id: 'gemini-a' })]))
    await nextTick()
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(writes, [])
    assert.equal(mapping.state.visible, true)
    assert.equal(globalThis.__providerMessages.at(-1).type, 'error')
    assert.match(globalThis.__providerMessages.at(-1).text, /gemini-a/)

    // Conflict with an existing system model code
    await mapping.state.handleOpen(importContext([nativeModel({ id: 'taken-code' })]))
    await nextTick()
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(writes, [])
    assert.equal(mapping.state.visible, true)
    assert.equal(globalThis.__providerMessages.at(-1).type, 'error')
    assert.match(globalThis.__providerMessages.at(-1).text, /taken-code/)
})

test('buildEndpointCode enforces backend 128-char maximum while preserving uniqueness suffix', () => {
    const longModel = 'gemini-2.5-flash-preview-native-audio-dialog-thinking-experimental'
    const longProvider = 'enterprise-google-vertex-gemini-production-us-central1-gateway-provider-cluster'
    const code = buildEndpointCode(longModel, longProvider, 1732432100, '-2')
    assert.ok(code.length <= ENDPOINT_CODE_MAX_LENGTH)
    assert.ok(code.startsWith('ep-'))
    assert.ok(code.endsWith('-1732432100-2'))
})

test('Gemini mapping prevalidates the whole batch before writes, including bypassed embed-only and missing methods', async (t) => {
    for (const methods of [['embedContent'], ['countTokens'], [], undefined]) {
        const writes = importAPI()
        const mapping = mount(ImportMappingDialog)
        t.after(() => mapping.app.unmount())
        await mapping.state.handleOpen(
            importContext([
                nativeModel(),
                nativeModel({ id: 'gemini-unsupported', supported_generation_methods: methods }),
            ])
        )
        await nextTick()
        await nodes(mapping.root, 'a-modal')[0].props.onOk()
        assert.deepEqual(writes, [], `unsupported methods ${JSON.stringify(methods)} must prevent all mutations`)
        assert.equal(mapping.state.visible, true)
        assert.equal(globalThis.__providerMessages.at(-1).type, 'error')
        assert.match(globalThis.__providerMessages.at(-1).text, /generateContent/)
    }
})

test('Gemini mapping falls back independently for invalid limits without publishing fabricated limit metadata', async (t) => {
    for (const invalid of [undefined, null, -1, NaN, Infinity, 1.5, '1048576']) {
        const writes = importAPI()
        const mapping = mount(ImportMappingDialog)
        t.after(() => mapping.app.unmount())
        await mapping.state.handleOpen(
            importContext([nativeModel({ input_token_limit: invalid, output_token_limit: 4096 })])
        )
        await nextTick()
        await nodes(mapping.root, 'a-modal')[0].props.onOk()
        const created = writes.find((write) => write.type === 'model').payload
        const endpoint = writes.find((write) => write.type === 'endpoint').payload
        assert.equal(created.context_length, 8192)
        assert.equal(created.max_output_tokens, 4096)
        assert.deepEqual(JSON.parse(created.abilities), [])
        assert.equal(Object.hasOwn(endpoint.metadata, 'input_token_limit'), false)
        assert.equal(Object.hasOwn(endpoint.metadata, 'context_length'), false)
        assert.equal(endpoint.metadata.output_token_limit, '4096')
    }
})

test('Gemini drawer renders known zero token limits as zero and preserves them through selection emit', async (t) => {
    const discovered = nativeModel({ input_token_limit: 0, output_token_limit: 0 })
    const drawer = await fetchDrawer(t, { models: [discovered] })
    const table = nodes(drawer.root, 'a-table')[0]
    for (const field of ['input_token_limit', 'output_token_limit']) {
        const column = table.props.columns.find((item) => item.dataIndex === field)
        assert.equal(column.customRender({ text: discovered[field], record: discovered }), 0)
        for (const invalid of [undefined, null, -1, NaN, Infinity, 1.5]) {
            assert.equal(column.customRender({ text: invalid, record: discovered }), '未知')
        }
    }
    await selectAndConfirm(drawer, [discovered.id])
    assert.deepEqual(drawer.confirmed[0].models, [discovered])
})

for (const target of ['new', 'existing']) {
    test(`Gemini ${target} mapping round-trips native zero limits and canonical endpoint metadata without inheriting system caps`, async (t) => {
        const discovered = nativeModel({ input_token_limit: 0, output_token_limit: 0 })
        const drawer = await fetchDrawer(t, { models: [discovered] })
        await selectAndConfirm(drawer, [discovered.id])
        const existing = existingModelRecord({
            id: 'existing',
            context_length: 8192,
            max_output_tokens: 8192,
        })
        const before = structuredClone(existing)
        const writes = importAPI({ getModelList: async () => ({ success: true, data: [existing] }) })
        const mapping = mount(ImportMappingDialog)
        t.after(() => mapping.app.unmount())
        await mapping.state.handleOpen(importContext(drawer.confirmed[0].models))
        if (target === 'existing') mapping.state.mapping[discovered.id] = existing.id
        await nextTick()
        await nodes(mapping.root, 'a-modal')[0].props.onOk()
        assert.deepEqual(
            writes.map((write) => write.type),
            target === 'new' ? ['model', 'endpoint'] : ['endpoint']
        )
        assert.deepEqual(existing, before)
        if (target === 'new') {
            assert.equal(writes[0].payload.context_length, 8192)
            assert.equal(writes[0].payload.max_output_tokens, 8192)
        }
        const endpoint = writes.find((write) => write.type === 'endpoint').payload
        assert.equal(endpoint.real_model, discovered.id)
        assert.deepEqual(JSON.parse(JSON.stringify(endpoint)).metadata, {
            ...nativeMetadata,
            input_token_limit: '0',
            output_token_limit: '0',
        })
    })
}

test('OpenAI drawer and mapping preserve legacy request types, defaults, capabilities, and endpoint payloads', async (t) => {
    const cases = [
        { id: 'gpt-4', requestType: 'chat_completion' },
        { id: 'text-embedding-3-small', requestType: 'embedding' },
        { id: 'gpt-responses', requestType: 'responses' },
        {
            id: 'gpt-5',
            requestType: 'responses',
            base_url: 'https://chatgpt.com/backend-api/codex',
            auth_type: 'oauth_token',
        },
    ]
    for (const item of cases) {
        const discovered = { id: item.id, object: 'model', owned_by: 'openai' }
        const drawer = await fetchDrawer(t, { models: [discovered] }, { protocol: '', url: 'https://api.openai.com' })
        await selectAndConfirm(drawer, [item.id])
        assert.equal(drawer.confirmed[0].base_url, 'https://api.openai.com')
        assert.equal(drawer.confirmed[0].models[0].id, item.id)
        const writes = importAPI()
        const mapping = mount(ImportMappingDialog)
        t.after(() => mapping.app.unmount())
        await mapping.state.handleOpen(
            importContext(drawer.confirmed[0].models, {
                protocol: '',
                base_url: item.base_url || 'https://api.openai.com',
                auth_type: item.auth_type,
            })
        )
        await nextTick()
        await nodes(mapping.root, 'a-modal')[0].props.onOk()
        const created = writes[0].payload
        assert.deepEqual(JSON.parse(created.request_types), [item.requestType])
        assert.equal(created.context_length, 8192)
        assert.equal(created.max_output_tokens, 8192)
        assert.deepEqual(JSON.parse(created.abilities), ['stream', 'tool_call'])
        assert.equal(created.description, 'Imported from provider model fetch')
        const endpoint = writes[1].payload
        assert.equal(endpoint.real_model, item.id)
        assert.equal(Object.hasOwn(endpoint, 'metadata'), false)
        assert.equal(endpoint.protocol, '')
        assert.equal(endpoint.provider_key_id, item.auth_type === 'oauth_token' ? '' : 'test-key')
    }
})
