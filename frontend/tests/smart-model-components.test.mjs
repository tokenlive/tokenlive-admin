import test from 'node:test'
import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { createRenderer, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/lang/zh-CN/pages.js'

const moduleURL = (source) => `data:text/javascript,${encodeURIComponent(source)}`
const stubs = {
    '@/apis': moduleURL(
        'export default { model: new Proxy({}, { get: (_, key) => (...args) => globalThis.__smartApi[key](...args) }) }'
    ),
    '@/utils/request': moduleURL('export default {basic: {put: (path, body) => globalThis.__smartHttp(path, body)}}'),
    '@/config': moduleURL('export const config = () => true'),
    'ant-design-vue': moduleURL(
        'export const message = new Proxy({}, {get: (_, type) => text => globalThis.__smartMessages.push({type,text})})'
    ),
    '@/utils/spaceStorage': moduleURL(
        'export const initSpaceCode = () => "demo"; export const setCurrentSpaceCode = () => {}'
    ),
    '@/hooks': moduleURL(
        `export {default as useForm} from ${JSON.stringify(new URL('../src/hooks/useForm.js', import.meta.url).href)}; export {default as useModal} from ${JSON.stringify(new URL('../src/hooks/useModal.js', import.meta.url).href)}`
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
const { default: Dialog } = await import('../src/views/resource/ModelEditDialog.vue')
const modelAPI = await import('../src/apis/modules/model.js')
const editorPath = new URL('../src/views/resource/SmartRoutingEditor.vue', import.meta.url)
const Editor = existsSync(editorPath) ? (await import(editorPath.href)).default : null
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
const candidates = [
    {
        id: 'economy',
        model_name: 'Economy',
        model_code: 'economy',
        model_type: 'normal',
        enabled: 1,
        space_code: 'demo',
        request_types: '["chat_completion"]',
    },
    {
        id: 'strong',
        model_name: 'Strong',
        model_code: 'strong',
        model_type: 'normal',
        enabled: 1,
        space_code: 'demo',
        request_types: '["chat_completion"]',
    },
]
const routing = (version = 7) => ({
    version,
    judge_model_id: 'economy',
    judge_timeout_ms: 5000,
    judge_max_input_bytes: 65536,
    judge_max_output_tokens: 256,
    ranges: [
        { min: 0, max: 50, model_id: 'economy' },
        { min: 50, max: 100, model_id: 'strong' },
    ],
})
const model = (extra = {}) => ({
    id: 'smart',
    model_name: 'Smart',
    model_code: 'smart',
    model_type: 'smart',
    space_code: 'demo',
    enabled: 0,
    smart_routing: null,
    smart_routing_ready: false,
    request_types: '["chat_completion"]',
    abilities: '[]',
    ...extra,
})
const flush = async () => {
    await new Promise((resolve) => setImmediate(resolve))
    await nextTick()
}
function mount(Component, props = {}) {
    assert.ok(Component, 'the detail routing editor must exist')
    globalThis.__smartMessages = []
    const app = renderer.createApp(Component, props)
    app.use(createI18n({ legacy: false, locale: 'zh', messages: { zh }, missingWarn: false, fallbackWarn: false }))
    for (const tag of tags) {
        app.component(tag, {
            inheritAttrs: false,
            setup(_, { attrs, slots }) {
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
function api(extra = {}) {
    globalThis.__smartApi = {
        getModelList: async () => ({ success: true, data: candidates, total: 2 }),
        ...extra,
    }
}

test('basic create shows no routing controls and submits a disabled smart draft without loading candidates', async () => {
    const requests = []
    let candidateCalls = 0
    api({
        getModelList: async () => {
            candidateCalls++
            return { success: true, data: candidates, total: 2 }
        },
        createModel: async (payload) => {
            requests.push(payload)
            return { success: true, data: { saved: true, sync_status: 'synced' } }
        },
    })
    const { app, state, root } = mount(Dialog, { spaceOptions: [] })
    state.handleCreate()
    state.handleModelTypeChange({ target: { value: 'smart' } })
    await flush()
    assert.equal(state.formData.enabled, 0)
    assert.equal(
        nodes(root, 'a-input-number').some((node) => node.props['onUpdate:value'] && node.props.min === 1),
        false
    )
    assert.equal(nodes(root, 'a-switch')[0].props.disabled, true)
    formBoundary(state)
    state.handleOk()
    await flush()
    assert.equal(requests.length, 1)
    assert.equal(requests[0].enabled, 0)
    assert.equal(Object.hasOwn(requests[0], 'smart_routing'), false)
    assert.equal(candidateCalls, 0)
    app.unmount()
})

test('basic smart edit submits without route config/version and allows disabling invalid legacy records', async () => {
    const writes = []
    api({
        getModel: async () => ({ success: true, data: model({ enabled: 1, smart_routing: routing() }) }),
        updateModel: async (id, payload) => {
            writes.push({ id, payload })
            return { success: true, data: { saved: true, sync_status: 'synced' } }
        },
    })
    const { app, state, root } = mount(Dialog)
    await state.handleEdit({ id: 'smart' })
    await flush()
    assert.equal(Boolean(nodes(root, 'a-switch')[0].props.disabled), false)
    state.formData.enabled = 0
    formBoundary(state)
    state.handleOk()
    await flush()
    assert.equal(writes.length, 1)
    assert.equal(writes[0].payload.enabled, 0)
    assert.equal(Object.hasOwn(writes[0].payload, 'smart_routing'), false)
    assert.equal(Object.hasOwn(writes[0].payload, 'smart_routing_version'), false)
    app.unmount()
})

test('detail editor saves initial routing through dedicated API at revision zero without enabling the model', async () => {
    const writes = []
    const saved = []
    globalThis.__smartHttp = async (path, payload) => {
        writes.push({ path, payload })
        return { success: true, data: { saved: true, sync_status: 'synced' } }
    }
    api({ updateSmartRouting: modelAPI.updateSmartRouting })
    const record = model()
    const { app, state } = mount(Editor, { model: record, onSaved: () => saved.push(true) })
    await flush()
    state.startEditing()
    state.draft.judge_model_id = 'economy'
    state.draft.ranges[0].model_id = 'economy'
    state.draft.ranges[1].model_id = 'strong'
    await state.save()
    assert.deepEqual(writes, [{ path: '/api/v1/models/smart/smart-routing', payload: routing(0) }])
    assert.equal(record.enabled, 0)
    assert.equal(record.smart_routing, null)
    assert.deepEqual(saved, [true])
    assert.equal(state.editing, false)
    app.unmount()
})

test('detail editing retains fetched revision, cancel discards changes, and errors retain the unsaved draft', async () => {
    const writes = []
    api({
        updateSmartRouting: async (id, payload) => {
            writes.push({ id, payload })
            throw new Error('version conflict')
        },
    })
    const record = model({ smart_routing: routing(9), smart_routing_ready: true })
    const { app, state } = mount(Editor, { model: record })
    await flush()
    state.startEditing()
    state.draft.judge_timeout_ms = 1234
    state.cancelEditing()
    state.startEditing()
    assert.equal(state.draft.judge_timeout_ms, 5000)
    state.draft.judge_timeout_ms = 2345
    await state.save()
    assert.equal(writes[0].payload.version, 9)
    assert.equal(writes[0].payload.judge_timeout_ms, 2345)
    assert.equal(state.editing, true)
    assert.equal(state.draft.version, 9)
    assert.equal(record.smart_routing.judge_timeout_ms, 5000)
    app.unmount()
})

test('detail editor cannot save during candidate loading or with fewer than two distinct targets', async () => {
    let resolveCandidates
    const writes = []
    api({
        getModelList: () =>
            new Promise((resolve) => {
                resolveCandidates = resolve
            }),
        updateSmartRouting: async (...args) => {
            writes.push(args)
            return { success: true }
        },
    })
    const { app, state } = mount(Editor, { model: model({ smart_routing: routing() }) })
    state.startEditing()
    await state.save()
    assert.deepEqual(writes, [])
    resolveCandidates({ success: true, data: candidates, total: 2 })
    await flush()
    state.draft.ranges[1].model_id = 'economy'
    await state.save()
    assert.deepEqual(writes, [])
    assert.equal(state.validationError, 'distinct_models')
    app.unmount()
})

test('detail editor ignores older candidate responses when the model space changes', async () => {
    const pending = []
    api({ getModelList: () => new Promise((resolve) => pending.push(resolve)) })
    const { app, state } = mount(Editor, { model: model() })
    app._instance.props.model = model({ id: 'new-smart', space_code: 'other' })
    await nextTick()
    pending[1]({
        success: true,
        data: candidates.map((item) => ({ ...item, id: `other-${item.id}`, space_code: 'other' })),
        total: 2,
    })
    await flush()
    pending[0]({ success: true, data: candidates, total: 2 })
    await flush()
    assert.deepEqual(
        state.candidateModels.map((item) => item.id),
        ['other-economy', 'other-strong']
    )
    assert.equal(state.candidatesLoading, false)
    app.unmount()
})

test('detail editor retries failed candidate loading and reports sync failure as a saved warning', async () => {
    let failed = true
    let writes = 0
    const saved = []
    api({
        getModelList: async () => {
            if (failed) throw new Error('offline')
            return { success: true, data: candidates, total: 2 }
        },
        updateSmartRouting: async () => {
            writes++
            return { success: true, data: { saved: true, sync_status: 'failed', warnings: ['Redis unavailable'] } }
        },
    })
    const { app, state } = mount(Editor, {
        model: model({ smart_routing: routing() }),
        onSaved: () => saved.push(true),
    })
    await flush()
    state.startEditing()
    await state.save()
    assert.equal(writes, 0)
    assert.equal(state.candidatesError, true)
    failed = false
    await state.loadCandidates()
    await state.save()
    assert.equal(writes, 1)
    assert.deepEqual(saved, [true])
    assert.equal(globalThis.__smartMessages.at(-1).type, 'warning')
    assert.match(globalThis.__smartMessages.at(-1).text, /Redis unavailable/)
    app.unmount()
})

test('detail editor coalesces save clicks and ignores completion after switching models', async () => {
    const pending = []
    const saved = []
    api({
        updateSmartRouting: () => new Promise((resolve) => pending.push(resolve)),
    })
    const { app, state } = mount(Editor, {
        model: model({ smart_routing: routing() }),
        onSaved: () => saved.push(true),
    })
    await flush()
    state.startEditing()
    const saving = state.save()
    await state.save()
    assert.equal(pending.length, 1)
    assert.equal(state.saving, true)
    app._instance.props.model = model({ id: 'another-smart' })
    await flush()
    pending[0]({ success: true, data: { saved: true, sync_status: 'synced' } })
    await saving
    assert.deepEqual(saved, [])
    assert.deepEqual(globalThis.__smartMessages, [])
    assert.equal(state.editing, false)
    assert.equal(state.saving, false)
    app.unmount()
})
