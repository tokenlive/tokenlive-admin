/**
 *  数据权限 (DataPermission) 接口
 */
import request from '@/utils/request'
// 获取数据权限列表
export const getDataPermissionList = (params) => request.basic.get('/api/v1/data-permissions', params)
// 创建数据权限
export const createDataPermission = (params) => request.basic.post('/api/v1/data-permissions', params)
// 更新数据权限
export const updateDataPermission = (id, params) => request.basic.put(`/api/v1/data-permissions/${id}`, params)
// 删除数据权限
export const delDataPermission = (id) => request.basic.delete(`/api/v1/data-permissions/${id}`)
