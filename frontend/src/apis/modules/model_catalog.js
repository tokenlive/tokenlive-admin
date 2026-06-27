/**
 *  模型目录 (Model Catalog) 接口
 */
import request from '@/utils/request'

// 获取模型目录列表
export const getModelCatalogList = (params) => request.basic.get('/api/v1/model-catalogs', params)
// 获取公开模型目录列表
export const getPublicModelCatalogs = (params) => request.basic.get('/api/v1/model-catalogs/public', params)
// 获取模型目录详情
export const getModelCatalog = (id) => request.basic.get(`/api/v1/model-catalogs/${id}`)
// 通过 slug 获取模型目录
export const getModelCatalogBySlug = (slug) => request.basic.get(`/api/v1/model-catalogs/slug/${slug}`)
// 创建模型目录
export const createModelCatalog = (params) => request.basic.post('/api/v1/model-catalogs', params)
// 更新模型目录
export const updateModelCatalog = (id, params) => request.basic.put(`/api/v1/model-catalogs/${id}`, params)
// 发布模型目录
export const publishModelCatalog = (id, params) => request.basic.put(`/api/v1/model-catalogs/${id}/publish`, params)
// 删除模型目录
export const delModelCatalog = (id) => request.basic.delete(`/api/v1/model-catalogs/${id}`)

// 获取模型目录的多语言列表
export const getModelCatalogI18nList = (params) => request.basic.get('/api/v1/model-catalog-i18n', params)
// 获取模型的所有多语言
export const getModelI18nByModelId = (modelId) => request.basic.get(`/api/v1/model-catalogs/${modelId}/i18n`)
// 获取指定多语言
export const getModelCatalogI18n = (modelId, locale) =>
    request.basic.get(`/api/v1/model-catalog-i18n/${modelId}/${locale}`)
// 创建多语言
export const createModelCatalogI18n = (params) => request.basic.post('/api/v1/model-catalog-i18n', params)
// 更新多语言
export const updateModelCatalogI18n = (modelId, locale, params) =>
    request.basic.put(`/api/v1/model-catalog-i18n/${modelId}/${locale}`, params)
// 批量更新多语言
export const batchUpsertModelCatalogI18n = (params) => request.basic.put('/api/v1/model-catalog-i18n/batch', params)
// 删除多语言
export const delModelCatalogI18n = (modelId, locale) =>
    request.basic.delete(`/api/v1/model-catalog-i18n/${modelId}/${locale}`)

// 获取价格版本列表
export const getModelPriceVersionList = (params) => request.basic.get('/api/v1/model-price-versions', params)
// 获取当前生效价格
export const getCurrentPrice = (params) => request.basic.get('/api/v1/model-price-versions/current', params)
// 获取模型的所有价格版本
export const getPricesByModelId = (modelId) => request.basic.get(`/api/v1/model-catalogs/${modelId}/prices`)
// 获取价格版本详情
export const getModelPriceVersion = (id) => request.basic.get(`/api/v1/model-price-versions/${id}`)
// 创建价格版本
export const createModelPriceVersion = (params) => request.basic.post('/api/v1/model-price-versions', params)
// 更新价格版本
export const updateModelPriceVersion = (id, params) => request.basic.put(`/api/v1/model-price-versions/${id}`, params)
// 停用价格版本
export const deactivatePriceVersion = (id) => request.basic.put(`/api/v1/model-price-versions/${id}/deactivate`)
// 删除价格版本
export const delModelPriceVersion = (id) => request.basic.delete(`/api/v1/model-price-versions/${id}`)

// 获取服务指标列表
export const getModelServiceMetricList = (params) => request.basic.get('/api/v1/model-service-metrics', params)
// 获取模型的所有服务指标
export const getMetricsByModelId = (modelId) => request.basic.get(`/api/v1/model-catalogs/${modelId}/metrics`)
// 获取指定指标
export const getModelServiceMetric = (modelId, window) =>
    request.basic.get(`/api/v1/model-service-metrics/${modelId}/${window}`)
// 更新服务指标
export const upsertModelServiceMetric = (params) => request.basic.put('/api/v1/model-service-metrics', params)
// 删除模型的所有指标
export const delModelServiceMetrics = (modelId) => request.basic.delete(`/api/v1/model-service-metrics/${modelId}`)
