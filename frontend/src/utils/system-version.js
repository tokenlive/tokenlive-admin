// Freshness and permission are explicit server facts, never inferred from a version string.
export function hasAvailableUpdate(summary, updates) {
    return (
        summary?.can_manage_updates === true &&
        updates?.enabled === true &&
        Array.isArray(updates.components) &&
        updates.components.some((item) => item?.state === 'available' && item.source?.stale === false)
    )
}

export function formatIdentity(identity, labels = {}) {
    const version = identity?.build?.version || labels.unknown || 'unknown'
    switch (identity?.edition) {
        case 'standalone':
            return `${labels.standalone || 'Standalone'} ${version}`
        case 'professional':
            return `${labels.professional || 'Professional'} · Admin ${version}`
        default:
            return `${labels.unknown || 'Unknown'} · ${labels.version || 'Version'} ${version}`
    }
}

export function copyVersionText(summary, labels = {}) {
    const identity = summary?.identity
    const separator = labels.separator || ': '
    const valueLabel = (value, values) => (values ? values[value] || labels.unknown || 'unknown' : value || 'unknown')
    const lines = [
        `TokenLive ${formatIdentity(identity, labels)}`,
        `${labels.channel || 'Channel'}${separator}${valueLabel(identity?.install_channel, labels.channelValues)}`,
        `${labels.build || 'Build'}${separator}${valueLabel(identity?.build?.kind, labels.buildValues)}`,
    ]
    if (identity?.edition === 'professional') {
        const groups = summary.gateway?.groups
        if (Array.isArray(groups) && groups.length) {
            for (const group of groups) {
                lines.push(
                    `Gateway ${group.version || labels.unknown || 'unknown'} (${valueLabel(group.build_kind, labels.buildValues)}) × ${group.count}`
                )
            }
        } else {
            lines.push(`Gateway${separator}${labels.unknown || 'unknown'}`)
        }
    }
    return lines.join('\n')
}
