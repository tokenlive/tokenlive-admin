import request from '@/utils/request'
import { monitoringQuery, monitoringResponse } from '@/utils/monitoring-snapshot'

export { getAPIKeyRanking } from './api-key-usage'

// 按需读取统一监控快照
export const getSnapshot = (query) =>
    request.basic.post('/api/v1/dashboard/snapshot', query, { enableAbortController: false })

// 获取当前熔断隔离的端点节点（保留用于独立查询）
export const getCircuitBreakers = (params) =>
    request.basic.get('/api/v1/dashboard/circuit-breakers', params, { enableAbortController: false })

// 获取供应商上游健康排行（支持排序，默认 Top 10；时间范围上限 7 天）
export const getProviderRanking = async (params) =>
    monitoringResponse(await getSnapshot(monitoringQuery('providers', params)), 'providers')

// 一键全量同步数据库配置至 Redis 缓存
export const syncRedis = () => request.basic.post('/api/v1/dashboard/sync-redis')
