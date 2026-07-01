function formatDateTime(value) {
    if (!value) return '-'
    const date = new Date(value)
    if (Number.isNaN(date.getTime())) return '-'

    const pad = (num) => String(num).padStart(2, '0')
    return (
        [date.getUTCFullYear(), pad(date.getUTCMonth() + 1), pad(date.getUTCDate())].join('-') +
        ` ${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}:${pad(date.getUTCSeconds())}`
    )
}

export function normalizePortalAPIKeys(items = []) {
    return items.map((item) => ({
        id: item.id || '',
        name: item.name || '',
        key_prefix: item.key_prefix || '',
        secret_last4: item.secret_last4 || '',
        status: item.status || '',
        expires_at: formatDateTime(item.expires_at),
        last_used_at: formatDateTime(item.last_used_at),
        created_at: formatDateTime(item.created_at),
        updated_at: formatDateTime(item.updated_at),
    }))
}

export function portalAPIKeyStatusColor(status) {
    const colorMap = {
        active: 'green',
        disabled: 'orange',
        revoked: 'red',
        expired: 'default',
    }
    return colorMap[status] || 'default'
}
