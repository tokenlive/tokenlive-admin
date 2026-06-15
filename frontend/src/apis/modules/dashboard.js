import request from '@/utils/request'

// 获取概览数据（合并 QPS + Metrics + CircuitBreakers）
export const getOverview = () => request.basic.get('/api/v1/dashboard/overview')

// 获取当前熔断隔离的端点节点（保留用于独立查询）
export const getCircuitBreakers = (params) => request.basic.get('/api/v1/dashboard/circuit-breakers', params)

// 获取最近60分钟的成功/失败流量走势（支持分组）
export const getTrends = (params) => request.basic.get('/api/v1/dashboard/trends', params)

// 获取模型使用排行（支持排序，默认 Top 10）
export const getModelRanking = (params) => request.basic.get('/api/v1/dashboard/model-ranking', params)

// 一键全量同步数据库配置至 Redis 缓存
export const syncRedis = () => request.basic.post('/api/v1/dashboard/sync-redis')

// ============ 以下为旧接口（保留兼容，不使用） ============
// 获取全局实时 QPS
export const getQPS = (params) => request.basic.get('/api/v1/dashboard/qps', params)

// 获取大盘遥测核心指标
export const getMetrics = (params) => request.basic.get('/api/v1/dashboard/metrics', params)
