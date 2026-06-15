/**
 *  模型别名 (ModelAlias) 接口
 */
import request from '@/utils/request'
// 获取 model_alias 列表
export const getModelAliasList = (params) => request.basic.get('/api/v1/model-aliases', params)
// 获取 model_alias 单条数据
export const getModelAlias = (id) => request.basic.get(`/api/v1/model-aliases/${id}`)
// 添加 model_alias
export const createModelAlias = (params) => request.basic.post('/api/v1/model-aliases', params)
// 更新 model_alias
export const updateModelAlias = (id, params) => request.basic.put(`/api/v1/model-aliases/${id}`, params)
// 删除 model_alias
export const delModelAlias = (id) => request.basic.delete(`/api/v1/model-aliases/${id}`)
