import request from '@/utils/request'

// 获取事件列表（分页 + 筛选）
export const getEvents = (params) => request.basic.get('/api/v1/ops/events', params)

// 获取事件统计数据（计数 + 趋势 + 排行）
export const getEventStatistics = (params) => request.basic.get('/api/v1/ops/events/statistics', params)

// 获取 Portal 工作空间 API Key 安全元数据
export const getPortalWorkspaceAPIKeys = (workspaceId) =>
    request.basic.get(`/api/v1/ops/portal/workspaces/${workspaceId}/api-keys`)

// 触发 Portal 工作空间 API Key 运行态重同步
export const syncPortalWorkspaceRuntime = (workspaceId) =>
    request.basic.post(`/api/v1/ops/portal/workspaces/${workspaceId}/runtime-sync`)
