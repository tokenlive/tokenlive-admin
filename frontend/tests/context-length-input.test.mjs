import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'
import { parse, compileScript, compileTemplate } from '@vue/compiler-sfc'
import { createRenderer, h, nextTick } from 'vue'
import { createI18n } from 'vue-i18n'
import zh from '../src/locales/lang/zh-CN/pages.js'
import {
    CONTEXT_LENGTH_OPTIONS,
    filterContextLengthOption,
    getContextLengthDropdownOptions,
} from '../src/enums/model.js'

const moduleURL = (source) => `data:text/javascript,${encodeURIComponent(source)}`
const stubs = {
    '@/apis': moduleURL(
        'const domain = () => new Proxy({}, { get: (_, key) => (...args) => globalThis.__contextLengthApi[key](...args) }); export default { model: domain(), model_catalog: domain(), provider: domain(), endpoint: domain(), space: domain() }'
    ),
    '@/config': moduleURL('export const config = () => true'),
    'ant-design-vue': moduleURL(
        'export const message = new Proxy({}, {get: (_, type) => text => { globalThis.__contextLengthMessages.push({type,text}); return type === "loading" ? () => {} : undefined }})'
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

const { default: ContextLengthInput } = await import('../src/components/ContextLengthInput.vue')
const { default: ModelEditDialog } = await import('../src/views/resource/ModelEditDialog.vue')
const { default: ModelCatalogEditDialog } = await import('../src/views/resource/ModelCatalogEditDialog.vue')

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
    globalThis.__contextLengthMessages = []
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

test('filterContextLengthOption and getContextLengthDropdownOptions handle filtering and unfiltered states', () => {
    assert.equal(filterContextLengthOption(null, CONTEXT_LENGTH_OPTIONS[0]), true)
    assert.equal(filterContextLengthOption('  ', CONTEXT_LENGTH_OPTIONS[0]), true)
    assert.equal(filterContextLengthOption('128k', { value: '128000', label: '128K (128,000)' }), true)
    assert.equal(filterContextLengthOption('128,000', { value: '128000', label: '128K (128,000)' }), true)

    assert.equal(getContextLengthDropdownOptions(CONTEXT_LENGTH_OPTIONS, '1000000', false).length, 11)
    assert.equal(getContextLengthDropdownOptions(CONTEXT_LENGTH_OPTIONS, '   ', true).length, 11)
    assert.deepEqual(
        getContextLengthDropdownOptions(CONTEXT_LENGTH_OPTIONS, '200', true).map((o) => o.value),
        ['200000', '2000000']
    )
    assert.deepEqual(
        getContextLengthDropdownOptions(CONTEXT_LENGTH_OPTIONS, ' 1m ', true).map((o) => o.value),
        ['1000000']
    )
    assert.deepEqual(getContextLengthDropdownOptions(CONTEXT_LENGTH_OPTIONS, '131072', true), [])
})

test('ContextLengthInput renders all options when closed/preset/custom/null and resets filtering cleanly', async () => {
    const emittedUpdates = []
    const { app, root } = mount(ContextLengthInput, {
        value: '1000000',
        placeholder: '请选择或输入上下文长度',
        'onUpdate:value': (v) => emittedUpdates.push(v),
    })

    const getAutoComplete = () => nodes(root, 'a-auto-complete')[0]
    let ac = getAutoComplete()
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)
    assert.equal(ac.props['default-active-first-option'], false)
    assert.equal(ac.props['show-arrow'], true)
    assert.equal(ac.props['allow-clear'], true)
    assert.equal(ac.props.disabled, undefined)
    assert.equal(ac.props.value, '1000000')

    ac.props.onSearch('200')
    await nextTick()
    ac = getAutoComplete()
    assert.deepEqual(
        ac.props.options.map((o) => o.value),
        ['200000', '2000000']
    )

    ac.props.onSelect('200000', { value: '200000' })
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, CONTEXT_LENGTH_OPTIONS.length)

    ac.props.onSearch('200')
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, 2)
    ac.props.onDropdownVisibleChange(false)
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, CONTEXT_LENGTH_OPTIONS.length)

    ac.props.onSearch('128k')
    await nextTick()
    assert.deepEqual(
        getAutoComplete().props.options.map((o) => o.value),
        ['128000']
    )
    ac.props.onBlur({})
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, CONTEXT_LENGTH_OPTIONS.length)

    ac.props.onSearch('128k')
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, 1)
    ac.props.onSearch('')
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, CONTEXT_LENGTH_OPTIONS.length)

    ac.props.onSearch('131072')
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, 0)
    ac.props.onMousedown({})
    await nextTick()
    assert.equal(getAutoComplete().props.options.length, CONTEXT_LENGTH_OPTIONS.length)

    ac.props['onUpdate:value']('131072')
    assert.deepEqual(emittedUpdates, ['131072'])
    app.unmount()

    const customMount = mount(ContextLengthInput, { value: '1048576' })
    assert.equal(nodes(customMount.root, 'a-auto-complete')[0].props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    customMount.app.unmount()

    const nullMount = mount(ContextLengthInput, { value: null })
    const nullAc = nodes(nullMount.root, 'a-auto-complete')[0]
    assert.equal(nullAc.props.value, undefined)
    assert.equal(nullAc.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    nullMount.app.unmount()
})

test('ModelEditDialog exposes all context length options on create and edit', async () => {
    globalThis.__contextLengthApi = {
        getModel: async (id) => ({
            success: true,
            data: {
                id,
                model_name: 'Model',
                model_code: 'model-code',
                model_type: 'normal',
                space_code: 'demo',
                request_types: '["chat_completion"]',
                abilities: '[]',
                context_length: id === 'custom' ? 1048576 : 128000,
                enabled: 1,
            },
        }),
    }

    const { app, state, root } = mount(ModelEditDialog, { spaceOptions: [] })
    state.handleCreate()
    await flush()
    let ac = nodes(root, 'a-auto-complete')[0]
    assert.equal(ac.props.value, '1000000')
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)

    await state.handleEdit({ id: 'preset' })
    await flush()
    ac = nodes(root, 'a-auto-complete')[0]
    assert.equal(ac.props.value, '128000')
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)

    await state.handleEdit({ id: 'custom' })
    await flush()
    ac = nodes(root, 'a-auto-complete')[0]
    assert.equal(ac.props.value, '1048576')
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)

    app.unmount()
})

test('ModelCatalogEditDialog exposes all context length options on create and edit', async () => {
    const { app, state, root } = mount(ModelCatalogEditDialog)
    state.handleCreate()
    await flush()
    let ac = nodes(root, 'a-auto-complete')[0]
    assert.equal(ac.props.value, undefined)
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)

    state.handleEdit({
        model_id: 'gpt-4o',
        slug: 'gpt-4o',
        status: 'available',
        visibility: 'public',
        context_length: 128000,
    })
    await flush()
    ac = nodes(root, 'a-auto-complete')[0]
    assert.equal(ac.props.value, '128000')
    assert.equal(ac.props.options.length, CONTEXT_LENGTH_OPTIONS.length)
    assert.equal(ac.props['filter-option'], false)

    app.unmount()
})
