import request from '@/utils/request'

// 获取租户列表（支持分页与过滤条件）
export const getList = (params) => request.basic.get('/api/v1/tenants', params)

// 获取特定租户详情
export const get = (id) => request.basic.get(`/api/v1/tenants/${id}`)

// 创建新租户记录
export const create = (params) => request.basic.post('/api/v1/tenants', params)

// 更新租户属性
export const update = (id, params) => request.basic.put(`/api/v1/tenants/${id}`, params)

// 删除租户
export const del = (id) => request.basic.delete(`/api/v1/tenants/${id}`)

// 获取租户已授权的模型 ID 列表
export const getAuthorizedModelIds = (tenantCode) => request.basic.get(`/api/v1/tenant-models/${tenantCode}`)

// 批量保存租户与模型的绑定关系
export const saveTenantModels = (params) => request.basic.post('/api/v1/tenant-models/bindings', params)

// 获取租户指定模型的可访问供应商白名单 ID 列表
export const getTenantModelProviders = (tenantCode, modelId) => {
    return request.basic.get(
        '/api/v1/tenant-models/providers',
        { tenant_code: tenantCode, model_id: modelId },
        { enableAbortController: false }
    )
}

// 保存租户模型供应商绑定白名单
export const saveTenantModelProviders = (params) => request.basic.post('/api/v1/tenant-models/providers', params)

// 获取租户指定模型的可访问端点 ID 列表（新）
export const getTenantModelEndpoints = (tenantCode, modelId) => {
    return request.basic.get(
        '/api/v1/tenant-models/endpoints',
        { tenant_code: tenantCode, model_id: modelId },
        { enableAbortController: false }
    )
}

// 保存租户模型端点绑定白名单（新）
export const saveTenantModelEndpoints = (params) => request.basic.post('/api/v1/tenant-models/endpoints', params)
