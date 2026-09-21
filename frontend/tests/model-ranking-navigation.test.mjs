import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { compile } from '@vue/compiler-dom'
import { babelParse, parse } from '@vue/compiler-sfc'
import * as Vue from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'

const source = await readFile(new URL('../src/views/home/index.vue', import.meta.url), 'utf8')
const { descriptor } = parse(source)
const script = descriptor.scriptSetup.content
const functions = babelParse(script, { sourceType: 'module' }).program.body.filter(
    (node) => node.type === 'FunctionDeclaration'
)
const tableStart = descriptor.template.content.indexOf('<a-table')
const tableEnd = descriptor.template.content.indexOf('</a-table>', tableStart) + '</a-table>'.length
const { code } = compile(descriptor.template.content.slice(tableStart, tableEnd), { mode: 'function' })
const render = new Function('Vue', code)(Vue)

// Compile the real ranking template and navigation functions, leaving the
// dashboard's unrelated network requests and chart lifecycle out of this test.
function rankingTable(t, router, records) {
    const handlers = new Function(
        'router',
        `${functions.map((node) => script.slice(node.start, node.end)).join('\n')}
        return { ${functions.map((node) => node.id.name).join(', ')} }`
    )(router)
    let table
    let modelCell
    const renderer = Vue.createRenderer({
        createComment: () => ({}),
        insert() {},
        remove() {},
        parentNode: () => null,
        nextSibling: () => null,
    })
    const app = renderer.createApp({
        render() {
            table = render({ ...handlers, modelRanking: records, columns: [] }, [])
            modelCell = table.children.bodyCell({ column: { key: 'model_name' }, record: records[0] })
            return null
        },
    })
    for (const name of ['a-table', 'a-tag', 'a-tooltip']) {
        app.component(name, { render: () => null })
    }
    app.mount({})
    t.after(() => app.unmount())
    return { table, modelCell }
}

const records = [
    { model_id: 'model-101', model_name: 'First model', model_code: 'first-code' },
    { model_id: 'model-202', model_name: 'Second model', model_code: 'second-code' },
]

test('each ranking row navigates to its own model monitoring tab', async (t) => {
    const router = createRouter({
        history: createMemoryHistory(),
        routes: [
            { path: '/', component: {} },
            { path: '/space/model/:id', name: 'modelDetail', component: {} },
        ],
    })
    await router.push('/')
    const { table } = rankingTable(t, router, records)
    const customRow = table.props.customRow || table.props['custom-row']
    assert.equal(typeof customRow, 'function', 'The ranking must bind clicks at the row level')

    for (const record of records) {
        const row = customRow(record)
        assert.equal(row.style.cursor, 'pointer')
        await row.onClick()
        // Vue Router navigation is asynchronous even if the click handler
        // does not return its promise.
        await new Promise((resolve) => setImmediate(resolve))
        assert.equal(router.currentRoute.value.name, 'modelDetail')
        assert.deepEqual(router.currentRoute.value.params, { id: record.model_id })
        assert.deepEqual(router.currentRoute.value.query, { tab: 'monitor' })
    }
})

test('clicking the model name has only one navigation handler along its bubbling path', (t) => {
    const destinations = []
    const { table, modelCell } = rankingTable(t, { push: (destination) => destinations.push(destination) }, records)
    const customRow = table.props.customRow || table.props['custom-row']
    assert.equal(typeof customRow, 'function', 'The ranking must bind clicks at the row level')

    function clickHandlers(nodes) {
        return nodes.flatMap((node) => [
            ...(node.props?.onClick ? [node.props.onClick] : []),
            ...(Array.isArray(node.children) ? clickHandlers(node.children) : []),
        ])
    }

    for (const click of [...clickHandlers(modelCell), customRow(records[0]).onClick]) {
        click()
    }
    assert.deepEqual(destinations, [{ name: 'modelDetail', params: { id: 'model-101' }, query: { tab: 'monitor' } }])
})
