/**
 *  端点 (Endpoint) 接口
 */
import request from '@/utils/request'
// 获取 endpoint 列表
export const getEndpointList = (params) => request.basic.get('/api/v1/endpoints', params)
// 获取 endpoint 单条数据
export const getEndpoint = (id) => request.basic.get(`/api/v1/endpoints/${id}`)
// 添加 endpoint
export const createEndpoint = (params) => request.basic.post('/api/v1/endpoints', params)
// 更新 endpoint
export const updateEndpoint = (id, params) => request.basic.put(`/api/v1/endpoints/${id}`, params)
// 切换 endpoint 启用状态
export const toggleEndpointEnabled = (id, params) => request.basic.put(`/api/v1/endpoints/${id}/enabled`, params)
// 删除 endpoint
export const delEndpoint = (id) => request.basic.delete(`/api/v1/endpoints/${id}`)
// 查询 Model 关联的 Endpoint 列表
export const getEndpointsByModelId = (modelId) => request.basic.get(`/api/v1/models/${modelId}/endpoints`)
// 查询 Provider 关联的 Endpoint 列表
export const getEndpointsByProviderId = (providerId) => request.basic.get(`/api/v1/providers/${providerId}/endpoints`)
// 测试指定 Endpoint 的连通性 (已有)
export const testEndpoint = (id) => request.basic.post(`/api/v1/endpoints/${id}/test`)
// 测试临时 Endpoint 配置 (草稿)
export const testEndpointDraft = (params) => request.basic.post('/api/v1/endpoints/test', params)
