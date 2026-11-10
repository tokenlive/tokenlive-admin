import request from '@/utils/request'
import { monitoringQuery, monitoringResponse } from '@/utils/monitoring-snapshot'

export { getAPIKeyRanking } from './api-key-usage'

// 获取概览数据（合并 QPS + Metrics + CircuitBreakers）
export const getSnapshot = (query) =>
    request.basic.post('/api/v1/dashboard/snapshot', query, { enableAbortController: false })

const read = async (kind, params) => monitoringResponse(await getSnapshot(monitoringQuery(kind, params)), kind)

export const getOverview = (params) => read('overview', params)

// 获取当前熔断隔离的端点节点（保留用于独立查询）
export const getCircuitBreakers = (params) => request.basic.get('/api/v1/dashboard/circuit-breakers', params)

// 获取最近60分钟的成功/失败流量走势（支持分组）
export const getTrends = (params) => read('traffic', params)

// 获取单个模型的 TTFT 与请求期 OTPS 趋势
export const getModelPerformanceTrends = (params) => read('performance', params)

// 获取模型使用排行（支持排序，默认 Top 10）
export const getModelRanking = (params) => read('models', params)

// 获取供应商上游健康排行（支持排序，默认 Top 10；时间范围上限 7 天）
export const getProviderRanking = (params) => read('providers', params)

// 一键全量同步数据库配置至 Redis 缓存
export const syncRedis = () => request.basic.post('/api/v1/dashboard/sync-redis')
