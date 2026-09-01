import request from '@/utils/request'

// 获取公开版本信息
export const getVersion = () => request.basic.get('/api/v1/pub/version')
