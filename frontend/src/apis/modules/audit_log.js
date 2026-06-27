/**
 *  审计日志 (Audit Log) 接口
 */
import request from '@/utils/request'

// 获取审计日志列表
export const getAuditLogList = (params) => request.basic.get('/api/v1/audit-logs', params)
// 获取审计日志详情
export const getAuditLog = (id) => request.basic.get(`/api/v1/audit-logs/${id}`)
