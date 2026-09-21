import test from 'node:test'
import assert from 'node:assert/strict'
import {
    normalizeSmartRouting,
    getSmartModelCandidates,
    validateSmartRouting,
    updateSplitPoint,
    splitRoutingRange,
    removeRoutingSplit,
    buildModelRoutingFields,
    formatRoutingRange,
    loadSmartModelCandidates,
    getModelSaveFeedback,
} from '../src/utils/smart-model.js'
import * as smartModel from '../src/utils/smart-model.js'

const candidates = [
    { id: 'economy', model_type: 'normal', enabled: 1, space_code: 'demo', request_types: '["chat_completion"]' },
    { id: 'strong', model_type: 'normal', enabled: 1, space_code: 'demo', request_types: '["chat_completion"]' },
]
const validConfig = () => ({
    version: 7,
    judge_model_id: 'economy',
    judge_timeout_ms: 5000,
    judge_max_input_bytes: 65536,
    judge_max_output_tokens: 256,
    ranges: [
        { min: 0, max: 40, model_id: 'economy' },
        { min: 40, max: 75, model_id: 'strong' },
        { min: 75, max: 100, model_id: 'strong' },
    ],
})

test('new smart config starts with editable, contiguous ranges and bounded judge defaults', () => {
    assert.deepEqual(normalizeSmartRouting(), {
        version: 0,
        judge_model_id: undefined,
        judge_timeout_ms: 5000,
        judge_max_input_bytes: 65536,
        judge_max_output_tokens: 256,
        ranges: [
            { min: 0, max: 50, model_id: undefined },
            { min: 50, max: 100, model_id: undefined },
        ],
    })
})

test('editing retains optimistic concurrency version and does not mutate the fetched config', () => {
    const source = validConfig()
    const edited = normalizeSmartRouting(source)
    assert.deepEqual(edited, validConfig())
    edited.ranges[0].max = 35
    assert.equal(source.ranges[0].max, 40)
    assert.equal(edited.version, 7)
})

test('selectors expose only enabled, same-space, non-self normal chat models from readable API results', () => {
    const models = [
        ...candidates,
        { ...candidates[0], id: 'legacy', model_type: undefined, request_types: ['chat_completion'] },
        { ...candidates[0], id: 'disabled', enabled: 0 },
        { ...candidates[0], id: 'other-space', space_code: 'other' },
        { ...candidates[0], id: 'composite', model_type: 'smart' },
        { ...candidates[0], id: 'self' },
        { ...candidates[0], id: 'embedding', request_types: '["embedding"]' },
        { ...candidates[0], id: 'bad-json', request_types: '{broken}' },
    ]
    assert.deepEqual(
        getSmartModelCandidates(models, { spaceCode: 'demo', modelId: 'self' })?.map((model) => model.id),
        ['economy', 'strong', 'legacy']
    )
})

test('accepts full partition including zero and 100 with a child reused as judge', () => {
    assert.equal(validateSmartRouting(validConfig(), candidates), null)
})

test('rejects gaps, overlaps, empty, unordered, fractional and out-of-range partitions', () => {
    for (const ranges of [
        [
            { min: 0, max: 40, model_id: 'economy' },
            { min: 41, max: 100, model_id: 'strong' },
        ],
        [
            { min: 0, max: 40, model_id: 'economy' },
            { min: 39, max: 100, model_id: 'strong' },
        ],
        [
            { min: 0, max: 0, model_id: 'economy' },
            { min: 0, max: 100, model_id: 'strong' },
        ],
        [
            { min: 50, max: 100, model_id: 'economy' },
            { min: 0, max: 50, model_id: 'strong' },
        ],
        [
            { min: 0, max: 40.5, model_id: 'economy' },
            { min: 40.5, max: 100, model_id: 'strong' },
        ],
        [
            { min: -1, max: 40, model_id: 'economy' },
            { min: 40, max: 100, model_id: 'strong' },
        ],
        [
            { min: 0, max: 40, model_id: 'economy' },
            { min: 40, max: 101, model_id: 'strong' },
        ],
        [],
    ]) {
        assert.equal(validateSmartRouting({ ...validConfig(), ranges }, candidates), 'partition')
    }
})

test('rejects fewer than two different child models and inaccessible references', () => {
    const singleTarget = validConfig()
    singleTarget.ranges.forEach((range) => {
        range.model_id = 'economy'
    })
    assert.equal(validateSmartRouting(singleTarget, candidates), 'distinct_models')
    assert.equal(validateSmartRouting({ ...validConfig(), judge_model_id: 'hidden' }, candidates), 'judge_required')
    const invalidChild = validConfig()
    invalidChild.ranges[2].model_id = 'hidden'
    assert.equal(validateSmartRouting(invalidChild, candidates), 'target_required')
})

test('judge limits must be positive integers', () => {
    for (const field of ['judge_timeout_ms', 'judge_max_input_bytes', 'judge_max_output_tokens']) {
        for (const value of [0, -1, 1.5, null, '5000']) {
            assert.equal(validateSmartRouting({ ...validConfig(), [field]: value }, candidates), 'judge_limits')
        }
    }
})

test('changing a split joins adjacent ranges without mutating the source', () => {
    const ranges = validConfig().ranges
    assert.deepEqual(updateSplitPoint(ranges, 0, 35), [
        { min: 0, max: 35, model_id: 'economy' },
        { min: 35, max: 75, model_id: 'strong' },
        { min: 75, max: 100, model_id: 'strong' },
    ])
    assert.equal(ranges[0].max, 40)
    for (const value of [0, 75, 100, 40.5, null]) {
        assert.throws(() => updateSplitPoint(ranges, 0, value), /partition/)
    }
})

test('new splits must be inside the selected range and merging preserves full coverage', () => {
    const ranges = validConfig().ranges
    assert.deepEqual(splitRoutingRange(ranges, 0, 20), [
        { min: 0, max: 20, model_id: 'economy' },
        { min: 20, max: 40, model_id: undefined },
        { min: 40, max: 75, model_id: 'strong' },
        { min: 75, max: 100, model_id: 'strong' },
    ])
    for (const value of [0, 40, 20.5, null]) {
        assert.throws(() => splitRoutingRange(ranges, 0, value), /partition/)
    }
    assert.deepEqual(removeRoutingSplit(ranges, 0), [
        { min: 0, max: 75, model_id: 'economy' },
        { min: 75, max: 100, model_id: 'strong' },
    ])
    assert.throws(() => removeRoutingSplit(ranges.slice(0, 2), 0), /partition/)
})

test('new smart models omit routing and are forced disabled while keeping selected abilities', () => {
    assert.deepEqual(
        buildModelRoutingFields({
            model_type: 'smart',
            smart_routing: validConfig(),
            request_types: ['messages', 'embedding'],
            abilities: ['stream', 'tool_call'],
            enabled: 1,
        }),
        {
            model_type: 'smart',
            enabled: 0,
            request_types: '["chat_completion"]',
            abilities: '["stream","tool_call"]',
        }
    )
    assert.deepEqual(
        buildModelRoutingFields({
            model_type: 'normal',
            smart_routing: validConfig(),
            request_types: ['messages', 'chat_completion'],
            abilities: ['stream', 'tool_call'],
        }),
        {
            model_type: 'normal',
            smart_routing: null,
            request_types: '["messages","chat_completion"]',
            abilities: '["stream","tool_call"]',
        }
    )
    assert.deepEqual(buildModelRoutingFields({ request_types: ['embedding'] }), {
        model_type: 'normal',
        smart_routing: null,
        request_types: '["embedding"]',
        abilities: '[]',
    })
})

test('basic edits omit smart routing and its version to preserve independently saved configuration', () => {
    const fields = buildModelRoutingFields(
        { model_type: 'smart', enabled: 1, smart_routing: validConfig() },
        { model_type: 'smart', enabled: 1, smart_routing_ready: true, smart_routing: { version: 3 } }
    )
    assert.equal(Object.hasOwn(fields, 'smart_routing'), false)
    assert.equal(Object.hasOwn(fields, 'smart_routing_version'), false)
    assert.equal(fields.enabled, 1)
})

test('normal to smart conversions are drafts even if the form contains an old routing value', () => {
    const fields = buildModelRoutingFields(
        { model_type: 'smart', enabled: 1, smart_routing: validConfig() },
        { model_type: 'normal', enabled: 1 }
    )
    assert.equal(fields.enabled, 0)
    assert.equal(Object.hasOwn(fields, 'smart_routing'), false)
})

test('clearing a smart draft carries revision zero rather than an undefined revision', () => {
    assert.equal(
        buildModelRoutingFields({ model_type: 'normal' }, { model_type: 'smart', smart_routing: null })
            .smart_routing_version,
        0
    )
})

test('incomplete smart models cannot be enabled, but enabled legacy models can be disabled', () => {
    assert.equal(typeof smartModel.isModelEnableBlocked, 'function')
    for (const record of [
        { model_type: 'smart', enabled: 0, smart_routing_ready: false },
        { model_type: 'smart', enabled: 0 },
    ]) {
        assert.equal(smartModel.isModelEnableBlocked(record), true)
    }
    for (const record of [
        { model_type: 'normal', enabled: 0 },
        { model_type: 'smart', enabled: 0, smart_routing_ready: true },
        { model_type: 'smart', enabled: 1, smart_routing_ready: false },
    ]) {
        assert.equal(smartModel.isModelEnableBlocked(record), false)
    }
    assert.equal(
        buildModelRoutingFields(
            { model_type: 'smart', enabled: 1 },
            { model_type: 'smart', enabled: 0, smart_routing_ready: false }
        ).enabled,
        0
    )
    assert.equal(
        buildModelRoutingFields(
            { model_type: 'smart', enabled: 0 },
            { model_type: 'smart', enabled: 1, smart_routing_ready: false }
        ).enabled,
        0
    )
})

test('range descriptions make only the last upper bound inclusive', () => {
    assert.equal(formatRoutingRange({ min: 0, max: 40 }, false), '[0, 40)')
    assert.equal(formatRoutingRange({ min: 75, max: 100 }, true), '[75, 100]')
})

test('smart-to-normal conversion carries the fetched config version while clearing smart routing', () => {
    assert.deepEqual(
        buildModelRoutingFields(
            {
                model_type: 'normal',
                request_types: ['chat_completion'],
                abilities: [],
                smart_routing: validConfig(),
            },
            { model_type: 'smart', smart_routing: { ...validConfig(), version: 4 } }
        ),
        {
            model_type: 'normal',
            smart_routing: null,
            smart_routing_version: 4,
            request_types: '["chat_completion"]',
            abilities: '[]',
        }
    )
})

test('candidate loading paginates readable normal models and rejects failed pages', async () => {
    const queries = []
    const result = await loadSmartModelCandidates(
        async (query) => {
            queries.push(query)
            return query.current === 1
                ? { success: true, data: [candidates[0]], total: 2 }
                : { success: true, data: [candidates[1]], total: 2 }
        },
        { spaceCode: 'demo', modelId: 'self' }
    )
    assert.deepEqual(result, candidates)
    assert.deepEqual(queries, [
        { pageSize: 100, current: 1, model_type: 'normal', enabled: 1, space_code: 'demo' },
        { pageSize: 100, current: 2, model_type: 'normal', enabled: 1, space_code: 'demo' },
    ])
    await assert.rejects(
        () =>
            loadSmartModelCandidates(async () => ({ success: false }), {
                spaceCode: 'demo',
            }),
        /candidates_load/
    )
})

test('a saved model with failed publication reports warning instead of success', () => {
    assert.deepEqual(
        getModelSaveFeedback({
            saved: true,
            sync_status: 'failed',
            warnings: ['Redis unavailable'],
        }),
        { type: 'warning', key: 'sync_failed', warnings: ['Redis unavailable'] }
    )
    assert.deepEqual(getModelSaveFeedback({ saved: true, sync_status: 'synced', warnings: ['Dependency disabled'] }), {
        type: 'warning',
        key: 'saved_with_warnings',
        warnings: ['Dependency disabled'],
    })
    assert.equal(getModelSaveFeedback({ saved: true, sync_status: 'synced' }), null)
})

test('empty eligible model responses are not reported as a load failure, and no-space forms do not query', async () => {
    assert.deepEqual(
        await loadSmartModelCandidates(async () => ({ success: true, data: null, total: 0 }), {
            spaceCode: 'demo',
        }),
        []
    )
    let calls = 0
    assert.deepEqual(
        await loadSmartModelCandidates(async () => {
            calls++
            return { success: true, data: candidates, total: 2 }
        }),
        []
    )
    assert.equal(calls, 0)
})

test('delete publication failure uses recovery feedback that does not point to the removed model row', () => {
    assert.deepEqual(
        getModelSaveFeedback(
            {
                saved: true,
                sync_status: 'failed',
                warnings: ['Redis unavailable'],
            },
            { deleted: true }
        ),
        {
            type: 'warning',
            key: 'delete_sync_failed',
            warnings: ['Redis unavailable'],
        }
    )
})
