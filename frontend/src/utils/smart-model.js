export function normalizeSmartRouting(config) {
    return {
        version: config?.version ?? 0,
        judge_model_id: config?.judge_model_id,
        judge_timeout_ms: config?.judge_timeout_ms ?? 5000,
        judge_max_input_bytes: config?.judge_max_input_bytes ?? 65536,
        judge_max_output_tokens: config?.judge_max_output_tokens ?? 256,
        ranges: (
            config?.ranges || [
                { min: 0, max: 50, model_id: undefined },
                { min: 50, max: 100, model_id: undefined },
            ]
        ).map((range) => ({ ...range })),
    }
}

function parseList(value) {
    if (Array.isArray(value)) return value
    try {
        const parsed = JSON.parse(value)
        return Array.isArray(parsed) ? parsed : []
    } catch {
        return []
    }
}

export function getSmartModelCandidates(models, { spaceCode, modelId } = {}) {
    return models.filter(
        (model) =>
            (model.model_type || 'normal') === 'normal' &&
            model.enabled === 1 &&
            model.space_code === spaceCode &&
            model.id !== modelId &&
            parseList(model.request_types).includes('chat_completion')
    )
}

export async function loadSmartModelCandidates(load, { spaceCode, modelId } = {}) {
    if (!spaceCode) return []
    const models = []
    for (let current = 1; ; current++) {
        const result = await load({ pageSize: 100, current, model_type: 'normal', enabled: 1, space_code: spaceCode })
        const page = result.data ?? []
        if (!result.success || !Array.isArray(page)) throw new Error('candidates_load')
        models.push(...page)
        if (page.length === 0 || (Number.isFinite(result.total) ? models.length >= result.total : page.length < 100)) {
            return getSmartModelCandidates(models, { spaceCode, modelId })
        }
    }
}

export function validateSmartRouting(config, candidates) {
    const ranges = config?.ranges
    if (
        !Array.isArray(ranges) ||
        ranges.length < 2 ||
        ranges[0].min !== 0 ||
        ranges[ranges.length - 1].max !== 100 ||
        ranges.some(
            (range, index) =>
                !Number.isInteger(range.min) ||
                !Number.isInteger(range.max) ||
                range.min < 0 ||
                range.max > 100 ||
                range.min >= range.max ||
                (index > 0 && ranges[index - 1].max !== range.min)
        )
    ) {
        return 'partition'
    }
    const ids = new Set(candidates.map((model) => model.id))
    if (!ids.has(config.judge_model_id)) return 'judge_required'
    if (ranges.some((range) => !ids.has(range.model_id))) return 'target_required'
    if (new Set(ranges.map((range) => range.model_id)).size < 2) return 'distinct_models'
    if (
        ['judge_timeout_ms', 'judge_max_input_bytes', 'judge_max_output_tokens'].some(
            (field) => !Number.isInteger(config[field]) || config[field] <= 0
        )
    ) {
        return 'judge_limits'
    }
    return null
}

export function updateSplitPoint(ranges, index, value) {
    if (
        !Number.isInteger(value) ||
        !ranges[index + 1] ||
        value <= ranges[index].min ||
        value >= ranges[index + 1].max
    ) {
        throw new Error('partition')
    }
    return ranges.map((range, position) => ({
        ...range,
        ...(position === index ? { max: value } : {}),
        ...(position === index + 1 ? { min: value } : {}),
    }))
}

export function splitRoutingRange(ranges, index, value) {
    const range = ranges[index]
    if (!range || !Number.isInteger(value) || value <= range.min || value >= range.max) {
        throw new Error('partition')
    }
    return ranges.flatMap((item, position) =>
        position === index
            ? [
                  { ...item, max: value },
                  { min: value, max: item.max, model_id: undefined },
              ]
            : [{ ...item }]
    )
}

export function removeRoutingSplit(ranges, index) {
    if (ranges.length <= 2 || !ranges[index] || !ranges[index + 1]) throw new Error('partition')
    return ranges
        .filter((_, position) => position !== index + 1)
        .map((range, position) => ({ ...range, ...(position === index ? { max: ranges[index + 1].max } : {}) }))
}

export function buildModelRoutingFields(model, original) {
    const isSmart = model.model_type === 'smart'
    const canRetainEnabled =
        original?.model_type === 'smart' &&
        (original.enabled === 1 || (original.smart_routing_ready === true && model.space_code === original.space_code))
    return {
        model_type: isSmart ? 'smart' : 'normal',
        ...(isSmart ? { enabled: canRetainEnabled ? model.enabled : 0 } : { smart_routing: null }),
        ...(!isSmart && original?.model_type === 'smart'
            ? { smart_routing_version: original.smart_routing?.version ?? 0 }
            : {}),
        request_types: JSON.stringify(isSmart ? ['chat_completion'] : parseList(model.request_types)),
        abilities: JSON.stringify(parseList(model.abilities)),
    }
}

export function isModelEnableBlocked(model) {
    return model.model_type === 'smart' && model.enabled !== 1 && model.smart_routing_ready !== true
}

export const formatRoutingRange = (range, isLast) => `[${range.min}, ${range.max}${isLast ? ']' : ')'}`

export function getModelSaveFeedback(data, { deleted = false } = {}) {
    const warnings = data?.warnings || []
    if (data?.sync_status === 'failed') {
        return { type: 'warning', key: deleted ? 'delete_sync_failed' : 'sync_failed', warnings }
    }
    if (warnings.length) return { type: 'warning', key: 'saved_with_warnings', warnings }
    return null
}
