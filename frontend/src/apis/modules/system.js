import request from '@/utils/request'

// 获取日志列表
export const getLoggers = (params) => request.basic.get('/api/v1/loggers', params)

// These GETs read server-side snapshots; only the POST initiates an external check.
export const getVersionSummary = () => request.basic.get('/api/v1/current/version')
export const getUpdates = () => request.basic.get('/api/v1/system/updates')
export const checkUpdates = () => request.basic.post('/api/v1/system/updates/check')
