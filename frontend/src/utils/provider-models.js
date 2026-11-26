export const GEMINI_DEFAULT_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta'

export const ENDPOINT_CODE_MAX_LENGTH = 128

export const GEMINI_OTHER_FAMILY = '__OTHER__'

const GEMINI_METHOD_PRIORITY = [
    'generateContent',
    'countTokens',
    'createCachedContent',
    'batchGenerateContent',
    'bidiGenerateContent',
]

function normalizeModelLabel(value) {
    if (typeof value !== 'string') return ''
    return value
        .trim()
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '')
}

export function getDistinctGeminiDisplayName(model) {
    const displayName = String(model?.display_name || '').trim()
    if (!displayName) return ''
    const normalizedId = normalizeModelLabel(String(model?.id || ''))
    if (normalizedId && normalizeModelLabel(displayName) === normalizedId) {
        return ''
    }
    return displayName
}

export function getDistinctGeminiDescription(model) {
    const description = String(model?.description || '').trim()
    if (!description) return ''
    const normalizedDesc = normalizeModelLabel(description)
    const normalizedDisplay = normalizeModelLabel(String(model?.display_name || ''))
    const normalizedId = normalizeModelLabel(String(model?.id || ''))
    if (
        (normalizedDisplay && normalizedDesc === normalizedDisplay) ||
        (normalizedId && normalizedDesc === normalizedId)
    ) {
        return ''
    }
    return description
}

export function formatTokenLimitCompact(value, unknownLabel = '未知') {
    if (!isKnownTokenLimit(value)) return unknownLabel
    if (value === 0) return '0'
    if (value >= 1048576 && value % 1048576 === 0) {
        return `${value / 1048576}M`
    }
    if (value >= 1000000 && value % 1000000 === 0) {
        return `${value / 1000000}M`
    }
    if (value >= 1000000) {
        return `${Number((value / 1000000).toFixed(1))}M`
    }
    if (value >= 1024 && value % 1024 === 0) {
        return `${value / 1024}K`
    }
    if (value >= 1000 && value % 1000 === 0) {
        return `${value / 1000}K`
    }
    if (value >= 10000) {
        return `${Number((value / 1000).toFixed(1))}K`
    }
    return value.toLocaleString('en-US')
}

export function formatTokenLimitExact(value, unknownLabel = '未知') {
    if (!isKnownTokenLimit(value)) return unknownLabel
    return value.toLocaleString('en-US')
}

export function orderGeminiMethods(methods) {
    if (!Array.isArray(methods)) return []
    const seen = new Set()
    const list = []
    for (const item of methods) {
        if (typeof item !== 'string') continue
        const trimmed = item.trim()
        if (!trimmed || seen.has(trimmed)) continue
        seen.add(trimmed)
        list.push(trimmed)
    }
    return list.sort((a, b) => {
        const idxA = GEMINI_METHOD_PRIORITY.indexOf(a)
        const idxB = GEMINI_METHOD_PRIORITY.indexOf(b)
        if (idxA !== -1 && idxB !== -1) return idxA - idxB
        if (idxA !== -1) return -1
        if (idxB !== -1) return 1
        return a.localeCompare(b)
    })
}

export function extractGeminiModelFamily(model) {
    const id = String(model?.id || '')
        .trim()
        .toLowerCase()
    if (!id) return GEMINI_OTHER_FAMILY
    const versionMatch = id.match(/^gemini-(\d+(?:\.\d+)?)/)
    if (versionMatch) {
        return `Gemini ${versionMatch[1]}`
    }
    if (id.startsWith('gemini-')) return 'Gemini'
    if (/^gemma(?:-|$)/.test(id)) return 'Gemma'
    if (/^imagen(?:-|$)/.test(id)) return 'Imagen'
    if (/^veo(?:-|$)/.test(id)) return 'Veo'
    if (/^(?:text-)?embedding(?:-|$)/.test(id)) return 'Embedding'
    return GEMINI_OTHER_FAMILY
}

function parseGeminiVersion(family) {
    const match = typeof family === 'string' ? family.match(/^Gemini (\d+(?:\.\d+)?)$/) : null
    if (!match) return null
    return match[1].split('.').map((part) => Number(part))
}

export function compareGeminiFamilies(a, b) {
    if (a === b) return 0
    if (a === GEMINI_OTHER_FAMILY) return 1
    if (b === GEMINI_OTHER_FAMILY) return -1

    const verA = parseGeminiVersion(a)
    const verB = parseGeminiVersion(b)
    if (verA && verB) {
        const len = Math.max(verA.length, verB.length)
        for (let i = 0; i < len; i++) {
            const diff = (verB[i] || 0) - (verA[i] || 0)
            if (diff !== 0) return diff
        }
        return 0
    }
    if (verA) return -1
    if (verB) return 1

    if (a === 'Gemini') return -1
    if (b === 'Gemini') return 1
    if (a === 'Gemma') return -1
    if (b === 'Gemma') return 1

    return String(a).localeCompare(String(b))
}

export function supportsGeminiGenerateContent(model) {
    return (
        Array.isArray(model?.supported_generation_methods) &&
        model.supported_generation_methods.includes('generateContent')
    )
}

export function isKnownTokenLimit(value) {
    return Number.isSafeInteger(value) && value >= 0
}

// System model limits still require positive values; native metadata preserves known zeroes.
export function isValidTokenLimit(value) {
    return isKnownTokenLimit(value) && value > 0
}

export function modelSupportsRequestType(model, targetType, { exclusive = false } = {}) {
    if (!model || !targetType) return false
    let types = model.request_types
    if (typeof types === 'string') {
        try {
            types = JSON.parse(types)
        } catch {
            return false
        }
    }
    return (
        Array.isArray(types) && types.includes(targetType) && (!exclusive || types.every((type) => type === targetType))
    )
}

// Endpoint publication requires a map of strings, not native numeric/array values.
// Backend endpointCapabilities reads input_token_limit / output_token_limit directly,
// so native limits do not need to be duplicated under alias keys.
export function geminiEndpointMetadata(model) {
    const metadata = {}
    for (const field of ['name', 'display_name', 'description']) {
        if (typeof model[field] === 'string') metadata[field] = model[field]
    }
    if (Array.isArray(model.supported_generation_methods)) {
        metadata.supported_generation_methods = JSON.stringify(model.supported_generation_methods)
    }
    for (const field of ['input_token_limit', 'output_token_limit']) {
        if (isKnownTokenLimit(model[field])) {
            metadata[field] = String(model[field])
        }
    }
    return metadata
}

// Keep generated endpoint codes within the backend's 128-char limit while preserving
// the timestamp and key suffix for uniqueness.
export function buildEndpointCode(
    modelCode,
    providerCode,
    timestamp,
    keySuffix = '',
    maxLength = ENDPOINT_CODE_MAX_LENGTH
) {
    const modelPart = String(modelCode || '').trim()
    const providerPart = String(providerCode || '').trim()
    const prefix = 'ep-'
    const suffix = `-${timestamp}${keySuffix || ''}`
    const full = `${prefix}${modelPart}-${providerPart}${suffix}`
    if (full.length <= maxLength) {
        return full
    }
    const budget = Math.max(1, maxLength - prefix.length - suffix.length)
    if (!providerPart) {
        return `${prefix}${modelPart.slice(0, budget).replace(/-+$/, '')}${suffix}`.slice(0, maxLength)
    }
    const separatorLen = 1
    const available = Math.max(2, budget - separatorLen)
    const half = Math.floor(available / 2)
    let modelBudget = Math.min(modelPart.length, half)
    let providerBudget = Math.min(providerPart.length, available - modelBudget)
    if (modelBudget + providerBudget < available) {
        if (modelPart.length > modelBudget) {
            modelBudget = Math.min(modelPart.length, available - providerBudget)
        } else if (providerPart.length > providerBudget) {
            providerBudget = Math.min(providerPart.length, available - modelBudget)
        }
    }
    const trimmedModel = modelPart.slice(0, modelBudget).replace(/-+$/, '')
    const trimmedProvider = providerPart.slice(0, providerBudget).replace(/-+$/, '')
    const middle = [trimmedModel, trimmedProvider].filter(Boolean).join('-')
    return `${prefix}${middle}${suffix}`.slice(0, maxLength)
}
