// 供应商详情「端点可用」=（全部端点 − 禁用端点数）/ 全部端点。
// 禁用端点已退出服役，不计入可用；熔断数单独展示，不改这个比例。
export function countProviderEndpointHealth(endpoints = [], breakers = [], providerId) {
    const list = endpoints || []
    const enabledEndpoints = list.filter((ep) => Number(ep?.enabled) === 1)
    const enabledIds = new Set(enabledEndpoints.map((ep) => ep.id).filter(Boolean))
    const endpointIds = new Set(list.map((ep) => ep.id).filter(Boolean))

    const breakerCount = (breakers || []).filter((item) => {
        if (item?.id && enabledIds.has(item.id)) return true
        if (providerId && item?.provider_id === providerId) {
            if (item.id && endpointIds.has(item.id) && !enabledIds.has(item.id)) return false
            return true
        }
        return false
    }).length

    const total = list.length
    return {
        total,
        breakerCount,
        up: enabledEndpoints.length,
    }
}
