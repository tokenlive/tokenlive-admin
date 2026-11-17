/**
 * 脱敏展示 API Key（仅保留前后缀，避免完整明文暴露）
 */
export function maskApiKey(key = '') {
    const val = String(key || '').trim()
    if (!val) return ''
    if (val.length <= 8) return `${val.slice(0, 2)}****`
    return `${val.slice(0, 6)}...${val.slice(-4)}`
}

/**
 * 格式化供应商密钥选项显示文本
 */
export function formatProviderKeyLabel(item = {}) {
    const masked = maskApiKey(item.value)
    const desc = (item.description || '').trim()
    if (desc && masked) return `${desc} (${masked})`
    return desc || masked || item.id || ''
}

/**
 * 推断端点当前的密钥模式：oauth / bind / override / inherit
 */
export function resolveEndpointKeyMode(endpoint = {}) {
    if ((endpoint.auth_type || 'api_key') === 'oauth_token') {
        return 'oauth'
    }
    if ((endpoint.provider_key_id || '').trim() !== '') {
        return 'bind'
    }
    if ((endpoint.api_key || '').trim() !== '') {
        return 'override'
    }
    return 'inherit'
}

/**
 * 生成端点列表中的密钥来源展示信息
 */
export function resolveEndpointKeyDisplay(endpoint = {}, provider = null, t = (k, d) => d || k) {
    const mode = resolveEndpointKeyMode(endpoint)
    if (mode === 'oauth') {
        return {
            mode,
            color: 'purple',
            label: t('pages.endpoint.key_mode.oauth', 'OAuth 托管'),
            detail: '',
        }
    }
    if (mode === 'bind') {
        const keyId = String(endpoint.provider_key_id || '').trim()
        const keys = Array.isArray(provider?.api_keys) ? provider.api_keys : []
        const matched = keys.find((item) => item?.id === keyId)
        const suffix = matched ? formatProviderKeyLabel(matched) : maskApiKey(endpoint.api_key)
        return {
            mode,
            color: 'cyan',
            label: suffix
                ? `${t('pages.endpoint.key_mode.bind_prefix', '绑定')}: ${suffix}`
                : t('pages.endpoint.key_mode.bind', '绑定供应商密钥'),
            detail: matched?.value ? maskApiKey(matched.value) : maskApiKey(endpoint.api_key),
        }
    }
    if (mode === 'override') {
        const masked = maskApiKey(endpoint.api_key)
        return {
            mode,
            color: 'orange',
            label: masked
                ? `${t('pages.endpoint.key_mode.override', '自定义覆盖')} (${masked})`
                : t('pages.endpoint.key_mode.override', '自定义覆盖'),
            detail: masked,
        }
    }
    return {
        mode: 'inherit',
        color: 'default',
        label: t('pages.endpoint.key_mode.inherit', '继承供应商'),
        detail: '',
    }
}
