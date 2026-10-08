import request from '@/utils/request'

// 获取日志列表
export const getLoggers = (params) => request.basic.get('/api/v1/loggers', params)

// These GETs read server-side snapshots; only the POST initiates an external check.
export const getVersionSummary = () => request.basic.get('/api/v1/current/version')
export const getUpdates = () => request.basic.get('/api/v1/system/updates')
export const checkUpdates = () => request.basic.post('/api/v1/system/updates/check')

// Click upgrade: capability is read-only; prepare/submit drive the
// double-confirmed one-shot upgrade task on the standalone host.
export const getUpgradeCapability = () => request.basic.get('/api/v1/system/upgrade/capability')
export const prepareUpgrade = (data) => request.basic.post('/api/v1/system/upgrade/prepare', data)
export const submitUpgrade = (data) => request.basic.post('/api/v1/system/upgrade/submit', data)
export const getUpgradeTask = (id) => request.basic.get('/api/v1/system/upgrade/task', id ? { id } : undefined)
