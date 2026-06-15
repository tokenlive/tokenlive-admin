import request from '@/utils/request'

// 获取 API Key 列表
export const getList = (params) => request.basic.get('/api/v1/user-api-keys', params)
// 获取 API Key 单条数据
export const get = (id) => request.basic.get(`/api/v1/user-api-keys/${id}`)
// 获取 API Key 明文（用于复制操作）
export const getPlaintext = (id) => request.basic.get(`/api/v1/user-api-keys/${id}/plaintext`)
// 添加 API Key
export const create = (params) => request.basic.post('/api/v1/user-api-keys', params)
// 更新 API Key
export const update = (id, params) => request.basic.put(`/api/v1/user-api-keys/${id}`, params)
// 删除 API Key
export const del = (id) => request.basic.delete(`/api/v1/user-api-keys/${id}`)
