// Freshness and permission are explicit server facts, never inferred from a version string.
export function hasAvailableUpdate(summary, updates) {
    return (
        summary?.can_manage_updates === true &&
        updates?.enabled === true &&
        Array.isArray(updates.components) &&
        updates.components.some((item) => item?.state === 'available' && item.source?.stale === false)
    )
}

export function formatIdentity(identity) {
    const version = identity?.build?.version || 'unknown'
    switch (identity?.edition) {
        case 'standalone':
            return `Standalone ${version}`
        case 'professional':
            return `Professional · Admin ${version}`
        default:
            return `Unknown · Version ${version}`
    }
}

export function copyVersionText(summary) {
    const identity = summary?.identity
    const lines = [
        `TokenLive ${formatIdentity(identity)}`,
        `Channel: ${identity?.install_channel || 'unknown'}`,
        `Build: ${identity?.build?.kind || 'unknown'}`,
    ]
    if (identity?.edition === 'professional') {
        const groups = summary.gateway?.groups
        if (Array.isArray(groups) && groups.length) {
            for (const group of groups) {
                lines.push(`Gateway ${group.version || 'unknown'} (${group.build_kind || 'unknown'}) × ${group.count}`)
            }
        } else {
            lines.push('Gateway: unknown')
        }
    }
    return lines.join('\n')
}
