package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalogI18n management
type ModelCatalogI18n struct {
	ModelCatalogI18nBIZ *biz.ModelCatalogI18n
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Query model catalog i18n entries
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param model_id query string false "Model ID"
// @Param locale query string false "Locale"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalogI18n}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n [get]
func (m *ModelCatalogI18n) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelCatalogI18nQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogI18nBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Get model catalog i18n entry by model_id and locale
// @Param model_id path string true "Model ID"
// @Param locale path string true "Locale"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalogI18n}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n/{model_id}/{locale} [get]
func (m *ModelCatalogI18n) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelCatalogI18nBIZ.Get(ctx, c.Param("model_id"), c.Param("locale"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Get all i18n entries for a model
// @Param model_id path string true "Model ID"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalogI18n}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{model_id}/i18n [get]
func (m *ModelCatalogI18n) QueryByModelID(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := m.ModelCatalogI18nBIZ.QueryByModelID(ctx, c.Param("model_id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Create model catalog i18n entry
// @Param body body schema.ModelCatalogI18nForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalogI18n}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n [post]
func (m *ModelCatalogI18n) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogI18nForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogI18nBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Update model catalog i18n entry
// @Param model_id path string true "Model ID"
// @Param locale path string true "Locale"
// @Param body body schema.ModelCatalogI18nForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n/{model_id}/{locale} [put]
func (m *ModelCatalogI18n) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogI18nForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogI18nBIZ.Update(ctx, c.Param("model_id"), c.Param("locale"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Batch upsert i18n entries for a model
// @Param body body schema.ModelCatalogI18nBatchForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n/batch [put]
func (m *ModelCatalogI18n) BatchUpsert(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogI18nBatchForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogI18nBIZ.BatchUpsert(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogI18nAPI
// @Security ApiKeyAuth
// @Summary Delete model catalog i18n entry
// @Param model_id path string true "Model ID"
// @Param locale path string true "Locale"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalog-i18n/{model_id}/{locale} [delete]
func (m *ModelCatalogI18n) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelCatalogI18nBIZ.Delete(ctx, c.Param("model_id"), c.Param("locale"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
