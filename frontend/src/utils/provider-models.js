export const GEMINI_DEFAULT_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta'

export const ENDPOINT_CODE_MAX_LENGTH = 128

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
