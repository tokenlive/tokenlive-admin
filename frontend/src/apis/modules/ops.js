import request from '@/utils/request'

// 获取事件列表（分页 + 筛选）
export const getEvents = (params) => request.basic.get('/api/v1/ops/events', params)

// 获取事件统计数据（计数 + 趋势 + 排行）
export const getEventStatistics = (params) => request.basic.get('/api/v1/ops/events/statistics', params)
