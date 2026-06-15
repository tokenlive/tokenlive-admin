/**
 *  空间接口
 */
import request from '@/utils/request'
// 获取space列表
export const getSpaceList = (params) => request.basic.get('/api/v1/spaces', params)
// 获取space单条数据
export const getSpace = (id) => request.basic.get(`/api/v1/spaces/${id}`)
// 添加space
export const createSpace = (params) => request.basic.post('/api/v1/spaces', params)
// 更新space
export const updateSpace = (id, params) => request.basic.put(`/api/v1/spaces/${id}`, params)
// 删除space
export const delSpace = (id) => request.basic.delete(`/api/v1/spaces/${id}`)
