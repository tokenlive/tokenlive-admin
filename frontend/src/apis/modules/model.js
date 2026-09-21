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
// 独立保存智能模型路由配置，不改变模型启用状态
export const updateSmartRouting = (id, params) => request.basic.put(`/api/v1/models/${id}/smart-routing`, params)
// 切换 model 启用状态
export const toggleModelEnabled = (id, params) => request.basic.put(`/api/v1/models/${id}/enabled`, params)
// 删除 model
export const delModel = (id) => request.basic.delete(`/api/v1/models/${id}`)
// 同步 model
export const syncModel = (id) => request.basic.post(`/api/v1/models/${id}/sync`)
