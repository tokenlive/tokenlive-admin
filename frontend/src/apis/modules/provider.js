/**
 *  供应商 (Provider) 接口
 */
import request from '@/utils/request'
// 获取 provider 列表
export const getProviderList = (params) => request.basic.get('/api/v1/providers', params)
// 获取 provider 单条数据
export const getProvider = (id) => request.basic.get(`/api/v1/providers/${id}`)
// 添加 provider
export const createProvider = (params) => request.basic.post('/api/v1/providers', params)
// 更新 provider
export const updateProvider = (id, params) => request.basic.put(`/api/v1/providers/${id}`, params)
// 删除 provider
export const delProvider = (id) => request.basic.delete(`/api/v1/providers/${id}`)
// 获取供应商上游模型列表
export const fetchProviderModels = (id, params) => request.basic.post(`/api/v1/providers/${id}/fetch-models`, params)
