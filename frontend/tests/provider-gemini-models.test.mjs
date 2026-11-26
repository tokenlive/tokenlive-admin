import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { Comment, createRenderer, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/lang/zh-CN/pages.js'
import {
    GEMINI_OTHER_FAMILY,
    buildEndpointCode,
    compareGeminiFamilies,
    ENDPOINT_CODE_MAX_LENGTH,
    extractGeminiModelFamily,
    formatTokenLimitCompact,
    formatTokenLimitExact,
    getDistinctGeminiDescription,
    getDistinctGeminiDisplayName,
    orderGeminiMethods,
} from '../src/utils/provider-models.js'

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
                return () =>
                    h(tag, { ...attrs, __slots: slots }, [slots.default?.(), slots.footer?.(), slots.action?.()])
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

function vnodeText(node) {
    if (node == null || typeof node === 'boolean' || node.type === Comment) return ''
    if (typeof node === 'string' || typeof node === 'number') return String(node)
    if (Array.isArray(node)) return node.map(vnodeText).join(' ').replace(/\s+/g, ' ').trim()
    if (typeof node.children === 'string' || typeof node.children === 'number') {
        return String(node.children).trim()
    }
    if (Array.isArray(node.children)) {
        return vnodeText(node.children)
    }
    if (node.children && typeof node.children === 'object') {
        return Object.values(node.children)
            .filter((slot) => typeof slot === 'function')
            .map((slot) => vnodeText(slot()))
            .join(' ')
            .replace(/\s+/g, ' ')
            .trim()
    }
    return ''
}

function findVNodes(node, predicate) {
    if (!node || typeof node !== 'object' || node.type === Comment) return []
    if (Array.isArray(node)) return node.flatMap((item) => findVNodes(item, predicate))
    const matched = predicate(node) ? [node] : []
    if (Array.isArray(node.children)) {
        matched.push(...findVNodes(node.children, predicate))
    } else if (node.children && typeof node.children === 'object') {
        for (const slot of Object.values(node.children)) {
            if (typeof slot === 'function') matched.push(...findVNodes(slot(), predicate))
        }
    }
    return matched
}

function renderBodyCell(table, key, record) {
    const column = table.props.columns.find((item) => item.key === key)
    assert.ok(column, `missing column ${key}`)
    return table.props.__slots.bodyCell({ column, record })
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
        getEndpointsByProviderId: async () => ({ success: true, data: [] }),
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

async function fetchDrawer(t, result, provider = {}, apiExtra = {}) {
    importAPI({ fetchProviderModels: async () => ({ success: true, data: result }), ...apiExtra })
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

test('Provider model helpers deduplicate Gemini labels, format token limits, order methods, and extract families', () => {
    assert.equal(getDistinctGeminiDisplayName({ id: 'gemini-2.5-flash', display_name: 'Gemini 2.5 Flash' }), '')
    assert.equal(getDistinctGeminiDisplayName({ id: 'gemma-4-26b-a4b-it', display_name: 'Gemma 4 26B A4B IT' }), '')
    assert.equal(
        getDistinctGeminiDisplayName({ id: 'gemini-2.5-flash-image', display_name: 'Nano Banana' }),
        'Nano Banana'
    )

    assert.equal(
        getDistinctGeminiDescription({
            id: 'gemini-2.5-flash-preview-tts',
            display_name: 'Gemini 2.5 Flash Preview TTS',
            description: 'Gemini 2.5 Flash Preview TTS',
        }),
        ''
    )
    assert.equal(
        getDistinctGeminiDescription({
            id: 'gemini-2.5-flash',
            display_name: 'Gemini 2.5 Flash',
            description: 'gemini-2.5-flash',
        }),
        ''
    )
    assert.equal(
        getDistinctGeminiDescription({
            id: 'gemini-2.5-pro',
            display_name: 'Gemini 2.5 Pro',
            description: 'Stable release of Gemini 2.5 Pro',
        }),
        'Stable release of Gemini 2.5 Pro'
    )

    assert.equal(formatTokenLimitCompact(1048576), '1M')
    assert.equal(formatTokenLimitCompact(2097152), '2M')
    assert.equal(formatTokenLimitCompact(1000000), '1M')
    assert.equal(formatTokenLimitCompact(262144), '256K')
    assert.equal(formatTokenLimitCompact(65536), '64K')
    assert.equal(formatTokenLimitCompact(32768), '32K')
    assert.equal(formatTokenLimitCompact(8192), '8K')
    assert.equal(formatTokenLimitCompact(2048), '2K')
    assert.equal(formatTokenLimitCompact(512), '512')
    assert.equal(formatTokenLimitCompact(0), '0')
    assert.equal(formatTokenLimitCompact(undefined), '未知')
    assert.equal(formatTokenLimitCompact(-1, 'Unknown'), 'Unknown')
    assert.equal(formatTokenLimitExact(1048576), '1,048,576')
    assert.equal(formatTokenLimitExact(0), '0')
    assert.equal(formatTokenLimitExact(null), '未知')

    assert.deepEqual(
        orderGeminiMethods(['batchGenerateContent', 'countTokens', 'generateContent', 'countTokens', 'customMethod']),
        ['generateContent', 'countTokens', 'batchGenerateContent', 'customMethod']
    )
    assert.deepEqual(orderGeminiMethods(undefined), [])

    assert.equal(extractGeminiModelFamily({ id: 'gemini-3-flash-preview' }), 'Gemini 3')
    assert.equal(extractGeminiModelFamily({ id: 'gemini-2.5-pro' }), 'Gemini 2.5')
    assert.equal(extractGeminiModelFamily({ id: 'gemini-flash-latest' }), 'Gemini')
    assert.equal(extractGeminiModelFamily({ id: 'gemma-4-31b-it' }), 'Gemma')
    assert.equal(extractGeminiModelFamily({ id: 'imagen-3.0-generate-002' }), 'Imagen')
    assert.equal(extractGeminiModelFamily({ id: 'veo-2.0-generate-001' }), 'Veo')
    assert.equal(extractGeminiModelFamily({ id: 'gemini-embedding-001' }), 'Gemini')
    assert.equal(extractGeminiModelFamily({ id: 'text-embedding-004' }), 'Embedding')
    assert.equal(extractGeminiModelFamily({ id: 'custom-model' }), GEMINI_OTHER_FAMILY)

    const families = ['Embedding', GEMINI_OTHER_FAMILY, 'Gemma', 'Gemini 2.0', 'Gemini', 'Gemini 3', 'Gemini 2.5']
    assert.deepEqual([...families].sort(compareGeminiFamilies), [
        'Gemini 3',
        'Gemini 2.5',
        'Gemini 2.0',
        'Gemini',
        'Gemma',
        'Embedding',
        GEMINI_OTHER_FAMILY,
    ])
})

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

test('Gemini drawer displays compact searchable native details, merged limits, deduplicated labels, and ordered methods', async (t) => {
    const redundant = nativeModel({
        id: 'gemini-2.5-flash-preview-tts',
        display_name: 'Gemini 2.5 Flash Preview TTS',
        description: 'Gemini 2.5 Flash Preview TTS',
        input_token_limit: 8192,
        output_token_limit: 16384,
        supported_generation_methods: ['countTokens', 'generateContent'],
    })
    const aliased = nativeModel({
        id: 'gemini-2.5-flash-image',
        display_name: 'Nano Banana',
        description: 'Gemini 2.5 Flash Preview Image',
        input_token_limit: 1048576,
        output_token_limit: 65536,
        supported_generation_methods: ['batchGenerateContent', 'countTokens', 'generateContent', 'createCachedContent'],
    })
    const unknown = nativeModel({ id: 'unknown-limits', input_token_limit: undefined, output_token_limit: undefined })
    const drawer = await fetchDrawer(t, { models: [nativeModel(), redundant, aliased, unknown] })
    const table = nodes(drawer.root, 'a-table')[0]
    assert.ok(nodes(drawer.root, 'a-drawer')[0].props.width > 660)
    assert.deepEqual(
        table.props.columns.map((column) => column.key),
        ['id', 'token_limits', 'supported_generation_methods']
    )

    for (const query of [
        'native flash display',
        'useful details',
        'counttokens',
        'models/gemini-2.0-flash',
        'gemini 2.5 flash preview tts',
    ]) {
        nodes(drawer.root, 'a-input-search')[0].props['onUpdate:value'](query)
        await nextTick()
        assert.ok(nodes(drawer.root, 'a-table')[0].props['data-source'].length > 0, `cannot search ${query}`)
    }
    nodes(drawer.root, 'a-input-search')[0].props['onUpdate:value']('')
    await nextTick()

    const redundantIdText = vnodeText(renderBodyCell(table, 'id', redundant))
    assert.equal(redundantIdText, 'gemini-2.5-flash-preview-tts')

    const aliasedIdText = vnodeText(renderBodyCell(table, 'id', aliased))
    assert.match(aliasedIdText, /gemini-2\.5-flash-image/)
    assert.match(aliasedIdText, /Nano Banana/)
    assert.match(aliasedIdText, /Gemini 2\.5 Flash Preview Image/)

    const aliasedLimitCell = renderBodyCell(table, 'token_limits', aliased)
    assert.equal(vnodeText(aliasedLimitCell), '1M / 64K')
    const limitTooltips = findVNodes(aliasedLimitCell, (node) => typeof node.props?.title === 'string')
    assert.match(limitTooltips[0].props.title, /1,048,576.*65,536/)

    const unknownLimitCell = renderBodyCell(table, 'token_limits', unknown)
    assert.equal(vnodeText(unknownLimitCell), '未知 / 未知')

    const aliasedMethodsCell = renderBodyCell(table, 'supported_generation_methods', aliased)
    assert.equal(vnodeText(aliasedMethodsCell), 'generateContent +3')
    const methodTooltip = findVNodes(aliasedMethodsCell, (node) => typeof node.props?.title === 'string')
    assert.equal(
        methodTooltip[0].props.title,
        'generateContent, countTokens, createCachedContent, batchGenerateContent'
    )

    const redundantMethodsCell = renderBodyCell(table, 'supported_generation_methods', redundant)
    assert.equal(vnodeText(redundantMethodsCell), 'generateContent countTokens')

    // Non-Gemini provider retains owned_by column.
    const openaiDrawer = await fetchDrawer(
        t,
        { models: [{ id: 'gpt-4o', owned_by: 'openai' }] },
        { protocol: '', url: 'https://api.openai.com' }
    )
    assert.deepEqual(
        nodes(openaiDrawer.root, 'a-table')[0].props.columns.map((column) => column.key),
        ['id', 'owned_by']
    )
})

test('Gemini drawer defaults to importable models, supports filter switching and sorting, and blocks unsupported selection and bypassed confirm', async (t) => {
    const blocked = [
        nativeModel({ id: 'embed-only', supported_generation_methods: ['embedContent'] }),
        nativeModel({ id: 'gemini-name-without-methods', supported_generation_methods: undefined }),
        nativeModel({ id: 'count-only', supported_generation_methods: ['countTokens'] }),
    ]
    const generator = nativeModel({
        id: 'embed-in-name-but-generates',
        supported_generation_methods: ['generateContent'],
    })
    // Put unsupported models first in upstream response to verify importable-first sorting when viewing all.
    const drawer = await fetchDrawer(t, { models: [...blocked, generator] })
    const table = nodes(drawer.root, 'a-table')[0]
    assert.equal(drawer.state.geminiSupportFilter, 'supported')
    assert.deepEqual(
        table.props['data-source'].map((item) => item.id),
        [generator.id]
    )

    // Switch to 'all': all models are visible and importable ones sort before unsupported ones.
    const filterGroup = nodes(drawer.root, 'a-radio-group')[0]
    filterGroup.props['onUpdate:value']('all')
    await nextTick()
    assert.deepEqual(
        nodes(drawer.root, 'a-table')[0].props['data-source'].map((item) => item.id),
        [generator.id, ...blocked.map((item) => item.id)]
    )
    assert.equal(typeof table.props['row-selection'].getCheckboxProps, 'function')
    assert.equal(table.props['row-selection'].getCheckboxProps(generator).disabled, false)
    for (const item of blocked) {
        const checkbox = table.props['row-selection'].getCheckboxProps(item)
        assert.equal(checkbox.disabled, true)
        assert.match(checkbox.title, /generateContent/)
        const idCellText = vnodeText(renderBodyCell(nodes(drawer.root, 'a-table')[0], 'id', item))
        assert.match(idCellText, /不可导入/)
        drawer.state.selectedModels = [generator.id, item.id]
        drawer.state.handleConfirm()
        assert.deepEqual(drawer.confirmed, [])
        assert.equal(drawer.state.visible, true)
    }

    // Switch to 'unsupported' and back to 'supported', preserving selected importable key.
    drawer.state.selectedModels = [generator.id]
    filterGroup.props['onUpdate:value']('unsupported')
    await nextTick()
    assert.deepEqual(
        nodes(drawer.root, 'a-table')[0].props['data-source'].map((item) => item.id),
        blocked.map((item) => item.id)
    )
    assert.deepEqual(drawer.state.selectedModels, [generator.id])

    filterGroup.props['onUpdate:value']('supported')
    await nextTick()
    await selectAndConfirm(drawer, [generator.id])
    assert.equal(drawer.confirmed[0].models[0].id, generator.id)
})

test('Gemini drawer filters by model family, marks imported endpoints, hides imported models, and ignores stale endpoint loads', async (t) => {
    const models = [
        nativeModel({ id: 'gemini-3-flash-preview' }),
        nativeModel({ id: 'gemini-2.5-pro' }),
        nativeModel({ id: 'gemma-4-31b-it' }),
        nativeModel({ id: 'custom-gen-model' }),
    ]
    const drawer = await fetchDrawer(
        t,
        { models },
        {},
        {
            getEndpointsByProviderId: async () => ({
                success: true,
                data: [{ id: 'ep-1', real_model: 'gemini-2.5-pro' }],
            }),
        }
    )

    assert.deepEqual(
        drawer.state.familyOptions.map((item) => item.value),
        ['Gemini 3', 'Gemini 2.5', 'Gemma', GEMINI_OTHER_FAMILY]
    )
    assert.equal(drawer.state.importedVisibleCount, 1)

    const table = nodes(drawer.root, 'a-table')[0]
    assert.match(vnodeText(renderBodyCell(table, 'id', models[1])), /已导入/)
    assert.doesNotMatch(vnodeText(renderBodyCell(table, 'id', models[0])), /已导入/)

    // Filter by family.
    drawer.state.selectedFamily = 'Gemini 2.5'
    await nextTick()
    assert.deepEqual(
        nodes(drawer.root, 'a-table')[0].props['data-source'].map((m) => m.id),
        ['gemini-2.5-pro']
    )

    // Toggle hideImported on the full supported list.
    drawer.state.selectedFamily = '__ALL__'
    drawer.state.hideImported = true
    await nextTick()
    assert.deepEqual(
        nodes(drawer.root, 'a-table')[0].props['data-source'].map((m) => m.id),
        ['gemini-3-flash-preview', 'gemma-4-31b-it', 'custom-gen-model']
    )

    // Stale endpoint response from provider-a must not pollute provider-b.
    const pendingEndpoints = []
    globalThis.__providerApi.getEndpointsByProviderId = () => new Promise((resolve) => pendingEndpoints.push(resolve))
    globalThis.__providerApi.fetchProviderModels = async () => ({
        success: true,
        data: { models, base_url: GEMINI_BASE_URL },
    })
    drawer.state.handleOpen({ id: 'provider-a', protocol: 'gemini', url: GEMINI_BASE_URL })
    drawer.state.handleOpen({ id: 'provider-b', protocol: 'gemini', url: GEMINI_BASE_URL })
    await flush()
    assert.equal(pendingEndpoints.length, 2)
    pendingEndpoints[1]({ success: true, data: [{ id: 'ep-b', real_model: 'gemini-3-flash-preview' }] })
    await flush()
    pendingEndpoints[0]({ success: true, data: [{ id: 'ep-a', real_model: 'gemma-4-31b-it' }] })
    await flush()
    assert.equal(drawer.state.importedModelIds.has('gemini-3-flash-preview'), true)
    assert.equal(drawer.state.importedModelIds.has('gemma-4-31b-it'), false)

    // Endpoint loading error must degrade gracefully without throwing or breaking open/fetch.
    globalThis.__providerApi.getEndpointsByProviderId = async () => {
        throw new Error('endpoint list failed')
    }
    drawer.state.handleOpen({ id: 'provider-c', protocol: 'gemini', url: GEMINI_BASE_URL })
    await flush()
    assert.equal(drawer.state.visible, true)
    assert.equal(drawer.state.importedModelIds.size, 0)
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
    assert.equal(vnodeText(renderBodyCell(table, 'token_limits', discovered)), '0 / 0')
    for (const invalid of [undefined, null, -1, NaN, Infinity, 1.5]) {
        const invalidModel = nativeModel({ input_token_limit: invalid, output_token_limit: invalid })
        assert.equal(vnodeText(renderBodyCell(table, 'token_limits', invalidModel)), '未知 / 未知')
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

test('JoyCode import uses the catalog Responses capability and preserves the real model ID', async (t) => {
    const upstream = [{ id: 'GPT-6.2 Sol', request_types: ['responses'] }, { id: 'GLM-5.3' }]
    const baseURL = 'https://api-ai.jd.com'
    const drawer = await fetchDrawer(t, { models: upstream }, { protocol: 'joycode', url: baseURL })
    await selectAndConfirm(
        drawer,
        upstream.map((model) => model.id)
    )

    const writes = importAPI()
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())
    await mapping.state.handleOpen(
        importContext(drawer.confirmed[0].models, { protocol: 'joycode', base_url: baseURL })
    )
    await nodes(mapping.root, 'a-modal')[0].props.onOk()

    assert.deepEqual(
        writes
            .filter((write) => write.type === 'model')
            .map(({ payload }) => [payload.model_code, JSON.parse(payload.request_types)]),
        [
            ['GPT-6.2-Sol', ['responses']],
            ['GLM-5.3', ['chat_completion']],
        ]
    )
    assert.equal(writes[1].payload.real_model, 'GPT-6.2 Sol')
})

test('JoyCode Responses import rejects an existing Chat mapping before writing any endpoints', async (t) => {
    const chat = existingModelRecord({ id: 'chat', request_types: '["chat_completion","responses"]' })
    const responses = existingModelRecord({ id: 'responses', request_types: '["responses"]' })
    const writes = importAPI({ getModelList: async () => ({ success: true, data: [chat, responses] }) })
    const mapping = mount(ImportMappingDialog)
    t.after(() => mapping.app.unmount())
    const discovered = { id: 'GPT-6.2 Sol', request_types: ['responses'] }
    await mapping.state.handleOpen(importContext([discovered], { protocol: 'joycode' }))

    mapping.state.mapping[discovered.id] = chat.id
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(writes, [])
    assert.equal(mapping.state.visible, true)
    assert.match(globalThis.__providerMessages.at(-1).text, /Responses/)

    mapping.state.mapping[discovered.id] = responses.id
    await nodes(mapping.root, 'a-modal')[0].props.onOk()
    assert.deepEqual(
        writes.map((write) => write.type),
        ['endpoint']
    )
    assert.equal(writes[0].payload.model_id, responses.id)
})
