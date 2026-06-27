package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelPriceVersion management
type ModelPriceVersion struct {
	ModelPriceVersionBIZ *biz.ModelPriceVersion
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Query model price versions
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param model_id query string false "Model ID"
// @Param status query string false "Status (active/inactive)"
// @Param currency query string false "Currency"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelPriceVersion}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions [get]
func (m *ModelPriceVersion) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelPriceVersionQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelPriceVersionBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Get model price version by ID
// @Param id path string true "Price version ID"
// @Success 200 {object} util.ResponseResult{data=schema.ModelPriceVersion}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions/{id} [get]
func (m *ModelPriceVersion) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelPriceVersionBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Get current effective price for a model
// @Param model_id query string true "Model ID"
// @Param currency query string false "Currency" default(CNY)
// @Success 200 {object} util.ResponseResult{data=schema.ModelPriceVersion}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions/current [get]
func (m *ModelPriceVersion) GetCurrentPrice(c *gin.Context) {
	ctx := c.Request.Context()
	modelID := c.Query("model_id")
	if modelID == "" {
		util.ResError(c, errors.BadRequest("", "model_id is required"))
		return
	}
	currency := c.DefaultQuery("currency", "CNY")

	item, err := m.ModelPriceVersionBIZ.GetCurrentPrice(ctx, modelID, currency)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Get all price versions for a model
// @Param model_id path string true "Model ID"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelPriceVersion}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{model_id}/prices [get]
func (m *ModelPriceVersion) QueryByModelID(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := m.ModelPriceVersionBIZ.QueryByModelID(ctx, c.Param("model_id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Create model price version
// @Param body body schema.ModelPriceVersionForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelPriceVersion}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions [post]
func (m *ModelPriceVersion) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelPriceVersionForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelPriceVersionBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Update model price version
// @Param id path string true "Price version ID"
// @Param body body schema.ModelPriceVersionForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions/{id} [put]
func (m *ModelPriceVersion) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelPriceVersionForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelPriceVersionBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Deactivate model price version
// @Param id path string true "Price version ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions/{id}/deactivate [put]
func (m *ModelPriceVersion) Deactivate(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelPriceVersionBIZ.Deactivate(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelPriceVersionAPI
// @Security ApiKeyAuth
// @Summary Delete model price version
// @Param id path string true "Price version ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-price-versions/{id} [delete]
func (m *ModelPriceVersion) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelPriceVersionBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
