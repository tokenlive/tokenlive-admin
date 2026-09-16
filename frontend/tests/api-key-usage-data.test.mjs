import test from 'node:test'
import assert from 'node:assert/strict'
import { registerHooks } from 'node:module'
import { createRenderer, h, KeepAlive, nextTick, ref } from 'vue'

const apiStub = `data:text/javascript,${encodeURIComponent(
    'export const getAPIKeyRanking = (...args) => globalThis.__usageLoad(...args)'
)}`
registerHooks({
    resolve(specifier, context, nextResolve) {
        if (specifier === '@/apis/modules/api-key-usage') return { url: apiStub, shortCircuit: true }
        if (specifier.startsWith('@/')) {
            return nextResolve(new URL(`../src/${specifier.slice(2)}.js`, import.meta.url).href, context)
        }
        return nextResolve(specifier, context)
    },
})
const { useAPIKeyUsage } = await import('../src/composables/useAPIKeyUsage.js')

test('real Vue activation and visibility lifecycle controls polling and cleanup', async (t) => {
    t.mock.timers.enable({ apis: ['setTimeout'] })
    const previous = globalThis.document
    const doc = new EventTarget()
    doc.visibilityState = 'visible'
    globalThis.document = doc
    let calls = 0
    globalThis.__usageLoad = async () => {
        calls++
        return { state: 'disabled' }
    }
    const authorized = ref(false)
    const shown = ref(true)
    let data
    const component = {
        setup() {
            data = useAPIKeyUsage(authorized)
            return () => null
        },
    }
    const empty = { render: () => null }
    const renderer = createRenderer({
        createElement: () => ({}),
        createComment: () => ({}),
        createText: () => ({}),
        insert() {},
        remove() {},
        setText() {},
        setElementText() {},
        patchProp() {},
        parentNode: () => null,
        nextSibling: () => null,
    })
    const app = renderer.createApp({
        render: () => h(KeepAlive, null, { default: () => h(shown.value ? component : empty) }),
    })
    app.mount({})
    let unmounted = false
    const unmount = () => {
        if (!unmounted) app.unmount()
        unmounted = true
    }
    t.after(() => {
        unmount()
        globalThis.document = previous
        delete globalThis.__usageLoad
    })
    await nextTick()
    assert.equal(calls, 0)
    authorized.value = true
    await nextTick()
    assert.equal(calls, 1)
    shown.value = false
    await nextTick()
    t.mock.timers.tick(60000)
    assert.equal(calls, 1)
    shown.value = true
    await nextTick()
    assert.equal(calls, 2)
    doc.visibilityState = 'hidden'
    doc.dispatchEvent(new Event('visibilitychange'))
    t.mock.timers.tick(60000)
    assert.equal(calls, 2)
    doc.visibilityState = 'visible'
    doc.dispatchEvent(new Event('visibilitychange'))
    await nextTick()
    assert.equal(calls, 3)
    authorized.value = false
    await nextTick()
    assert.equal(data.state.data, null)
    unmount()
    doc.dispatchEvent(new Event('visibilitychange'))
    t.mock.timers.tick(60000)
    assert.equal(calls, 3)
})
