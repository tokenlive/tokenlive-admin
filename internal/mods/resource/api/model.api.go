package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Model management for LLM models
type Model struct {
	ModelBIZ *biz.Model
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Query model list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param model_name query string false "Model name (like)"
// @Param model_code query string false "Model code (exact)"
// @Param space_code query string false "Space code"
// @Success 200 {object} util.ResponseResult{data=[]schema.Model}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models [get]
func (m *Model) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Get model record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.Model}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id} [get]
func (m *Model) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Create model record
// @Param body body schema.ModelForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.Model}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models [post]
func (m *Model) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Update model record by ID
// @Param id path string true "unique id"
// @Param body body schema.ModelForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id} [put]
func (m *Model) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Toggle model enabled status by ID
// @Param id path string true "unique id"
// @Param body body schema.ModelEnabledForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id}/enabled [put]
func (m *Model) UpdateEnabled(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelEnabledForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelBIZ.ToggleEnabled(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Delete model record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id} [delete]
func (m *Model) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelAPI
// @Security ApiKeyAuth
// @Summary Sync model's Redis cache by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id}/sync [post]
func (m *Model) Sync(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelBIZ.Sync(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
