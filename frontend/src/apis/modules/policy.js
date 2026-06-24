/**
 *  治理策略接口
 */
import request from '@/utils/request'
// 获取负载均衡策略列表
export const getLoadbalanceList = (params) => request.basic.get('/api/v1/policy/policy-loadbalances', params)
// 获取负载均衡策略单条数据
export const getLoadbalance = (id) => request.basic.get(`/api/v1/policy/policy-loadbalances/${id}`)
// 添加负载均衡策略
export const createLoadbalance = (params) => request.basic.post('/api/v1/policy/policy-loadbalances', params)
// 更新负载均衡策略
export const updateLoadbalance = (id, params) => request.basic.put(`/api/v1/policy/policy-loadbalances/${id}`, params)
// 删除负载均衡策略
export const delLoadbalance = (id) => request.basic.delete(`/api/v1/policy/policy-loadbalances/${id}`)
// 获取路由策略列表
export const getRouteList = (params) => request.basic.get('/api/v1/policy/policy-routes', params)
// 获取路由策略单条数据
export const getRoute = (id) => request.basic.get(`/api/v1/policy/policy-routes/${id}`)
// 添加路由策略
export const createRoute = (params) => request.basic.post('/api/v1/policy/policy-routes', params)
// 更新路由策略
export const updateRoute = (id, params) => request.basic.put(`/api/v1/policy/policy-routes/${id}`, params)
// 删除路由策略
export const delRoute = (id) => request.basic.delete(`/api/v1/policy/policy-routes/${id}`)
// 获取路由详情列表
export const getRouteDetailList = (params) => request.basic.get('/api/v1/policy/policy-route-details', params)
// 获取路由详情单条数据
export const getRouteDetail = (id) => request.basic.get(`/api/v1/policy/policy-route-details/${id}`)
// 添加路由详情
export const createRouteDetail = (params) => request.basic.post('/api/v1/policy/policy-route-details', params)
// 更新路由详情
export const updateRouteDetail = (id, params) => request.basic.put(`/api/v1/policy/policy-route-details/${id}`, params)
// 删除路由详情
export const delRouteDetail = (id) => request.basic.delete(`/api/v1/policy/policy-route-details/${id}`)
// 获取限流策略列表
export const getLimitList = (params) => request.basic.get('/api/v1/policy/policy-limits', params)
// 获取限流策略单条数据
export const getLimit = (id) => request.basic.get(`/api/v1/policy/policy-limits/${id}`)
// 添加限流策略
export const createLimit = (params) => request.basic.post('/api/v1/policy/policy-limits', params)
// 更新限流策略
export const updateLimit = (id, params) => request.basic.put(`/api/v1/policy/policy-limits/${id}`, params)
// 删除限流策略
export const delLimit = (id) => request.basic.delete(`/api/v1/policy/policy-limits/${id}`)
// 获取熔断策略列表
export const getCircuitBreakList = (params) => request.basic.get('/api/v1/policy/policy-circuit-breaks', params)
// 获取熔断策略单条数据
export const getCircuitBreak = (id) => request.basic.get(`/api/v1/policy/policy-circuit-breaks/${id}`)
// 添加熔断策略
export const createCircuitBreak = (params) => request.basic.post('/api/v1/policy/policy-circuit-breaks', params)
// 更新熔断策略
export const updateCircuitBreak = (id, params) =>
    request.basic.put(`/api/v1/policy/policy-circuit-breaks/${id}`, params)
// 删除熔断策略
export const delCircuitBreak = (id) => request.basic.delete(`/api/v1/policy/policy-circuit-breaks/${id}`)

// 获取请求容错策略列表
export const getInvocationList = (params) => request.basic.get('/api/v1/policy/policy-invocations', params)
// 获取请求容错策略单条数据
export const getInvocation = (id) => request.basic.get(`/api/v1/policy/policy-invocations/${id}`)
// 添加请求容错策略
export const createInvocation = (params) => request.basic.post('/api/v1/policy/policy-invocations', params)
// 更新请求容错策略
export const updateInvocation = (id, params) => request.basic.put(`/api/v1/policy/policy-invocations/${id}`, params)
// 删除请求容错策略
export const delInvocation = (id) => request.basic.delete(`/api/v1/policy/policy-invocations/${id}`)

// 获取策略绑定列表
export const getPolicyBindingList = (params) => request.basic.get('/api/v1/policy/policy-bindings', params)
// 获取策略绑定单条数据
export const getPolicyBinding = (id) => request.basic.get(`/api/v1/policy/policy-bindings/${id}`)
// 添加策略绑定
export const createPolicyBinding = (params) => request.basic.post('/api/v1/policy/policy-bindings', params)
// 更新策略绑定
export const updatePolicyBinding = (id, params) => request.basic.put(`/api/v1/policy/policy-bindings/${id}`, params)
// 切换策略绑定启用状态
export const togglePolicyBindingEnabled = (id, params) =>
    request.basic.put(`/api/v1/policy/policy-bindings/${id}/enabled`, params)
// 删除策略绑定
export const delPolicyBinding = (id) => request.basic.delete(`/api/v1/policy/policy-bindings/${id}`)

// 获取流量染色策略列表
export const getTaggingList = (params) => request.basic.get('/api/v1/policy/policy-taggings', params)
// 获取流量染色策略单条数据
export const getTagging = (id) => request.basic.get(`/api/v1/policy/policy-taggings/${id}`)
// 添加流量染色策略
export const createTagging = (params) => request.basic.post('/api/v1/policy/policy-taggings', params)
// 更新流量染色策略
export const updateTagging = (id, params) => request.basic.put(`/api/v1/policy/policy-taggings/${id}`, params)
// 删除流量染色策略
export const delTagging = (id) => request.basic.delete(`/api/v1/policy/policy-taggings/${id}`)
