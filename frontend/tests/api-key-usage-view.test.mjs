import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { registerHooks } from 'node:module'
import { parse, compileScript } from '@vue/compiler-sfc'
import { createSSRApp } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { createI18n } from 'vue-i18n'
import antd from 'ant-design-vue'
import zh from '../src/locales/lang/zh-CN/pages.js'
import en from '../src/locales/lang/en-US/pages.js'

const stateStub = `data:text/javascript,${encodeURIComponent(
    'export const useAPIKeyUsage = () => ({state: globalThis.__usageViewState, query: globalThis.__usageViewState.query, setQuery() {}})'
)}`
registerHooks({
    resolve(specifier, context, nextResolve) {
        if (specifier === '@/composables/useAPIKeyUsage') return { url: stateStub, shortCircuit: true }
        return nextResolve(specifier, context)
    },
    load(url, context, nextLoad) {
        if (url.endsWith('/APIKeyUsageRanking.vue')) {
            const descriptor = parse(readFileSync(new URL(url), 'utf8')).descriptor
            return {
                format: 'module',
                source: compileScript(descriptor, { id: 'usage-view', inlineTemplate: true }).content,
                shortCircuit: true,
            }
        }
        return nextLoad(url, context)
    },
})
const { default: Panel } = await import('../src/views/home/components/APIKeyUsageRanking.vue')
const payload = {
    state: 'ready',
    data_source: 'clickhouse',
    generated_at: '2026-09-16T12:00:00+08:00',
    window: { start: '2026-09-16T00:00:00+08:00', end: '2026-09-16T12:00:00+08:00', timezone: 'UTC+8' },
    summary: { request_count: 4, total_tokens: 100, total_cost: '0.300000001', key_count: 1 },
    items: [
        {
            row_id: 'fixture-row-id',
            key_name: 'Production Agent',
            key_display: 'tl_l****abcd',
            source: 'admin_user',
            owner: { kind: 'user', id: 'u', name: 'Developer' },
            key_status: 'disabled',
            metadata_status: 'ready',
            request_count: 3,
            success_rate: 66.67,
            total_tokens: 70,
            input_tokens: 50,
            output_tokens: 20,
            cached_tokens: 10,
            cache_creation_tokens: 5,
            total_cost: '0.300000001',
            token_share: 70,
            api_key_hash: 'MUST-NOT-RENDER',
        },
    ],
    unattributed: { request_count: 1, total_tokens: 30, total_cost: '0', token_share: 30 },
    warnings: ['unattributed_usage'],
}

async function render(state, authorized = true, locale = 'zh-ch') {
    globalThis.__usageViewState = {
        query: { time_range: 'today', sort_by: 'tokens', limit: 10 },
        phase: 'ready',
        stale: false,
        error: null,
        data: payload,
        ...state,
    }
    const app = createSSRApp(Panel, { authorized })
    app.use(antd)
    app.use(
        createI18n({
            legacy: false,
            locale,
            messages: { 'zh-ch': zh, 'en-us': en },
            missingWarn: false,
            fallbackWarn: false,
        })
    )
    return renderToString(app)
}

test('renders real table cells, exact cost, ownership and historical unknown totals', async () => {
    const html = await render({})
    for (const text of [
        'API Key 用量排行',
        'Production Agent',
        'Developer',
        '已禁用',
        '0.300000001',
        '无法归属的历史用量',
        '70.00%',
    ]) {
        assert.ok(html.includes(text), text)
    }
    assert.ok(!html.includes('MUST-NOT-RENDER'))
    assert.ok(!html.includes('ant-pagination'))
})

test('unauthorized users render no ranking or controls', async () => {
    const html = await render({}, false)
    assert.ok(!html.includes('Production Agent'))
    assert.ok(!html.includes('API Key 用量排行'))
    assert.ok(!html.includes('ant-select'))
})

test('disabled, empty and unavailable messages are distinct and translated', async () => {
    for (const [phase, text] of [
        ['disabled', '尚未启用 API Key 用量统计'],
        ['empty', '该时段暂无调用'],
        ['error', '数据暂不可用'],
    ]) {
        const html = await render({ phase, data: null })
        assert.ok(html.includes(text), phase)
        assert.ok(!html.includes('Production Agent'))
    }
    const html = await render({}, true, 'en-us')
    assert.ok(html.includes('API Key Usage'))
    assert.ok(html.includes('Unattributed historical usage'))
})

test('stale data retains its successful timestamp and shows an explicit warning', async () => {
    const html = await render({ stale: true })
    assert.ok(html.includes('更新失败'))
    assert.ok(html.includes('2026-09-16'))
    assert.ok(html.includes('Production Agent'))
})
