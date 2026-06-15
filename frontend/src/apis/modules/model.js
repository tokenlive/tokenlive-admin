/**
 *  模型 (Model) 接口
 */
import request from '@/utils/request'
// 获取 model 列表
export const getModelList = (params) => request.basic.get('/api/v1/models', params)
// 获取 model 单条数据
export const getModel = (id) => request.basic.get(`/api/v1/models/${id}`)
// 添加 model
export const createModel = (params) => request.basic.post('/api/v1/models', params)
// 更新 model
export const updateModel = (id, params) => request.basic.put(`/api/v1/models/${id}`, params)
// 删除 model
export const delModel = (id) => request.basic.delete(`/api/v1/models/${id}`)
// 同步 model
export const syncModel = (id) => request.basic.post(`/api/v1/models/${id}/sync`)
