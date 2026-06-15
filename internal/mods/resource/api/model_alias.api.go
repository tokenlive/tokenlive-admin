package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelAlias management
type ModelAlias struct {
	ModelAliasBIZ *biz.ModelAlias
}

// @Tags ModelAliasAPI
// @Security ApiKeyAuth
// @Summary Query model alias list
// @Param space_code query string false "Space code"
// @Param alias query string false "Model alias"
// @Param model_id query string false "Model ID"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelAlias}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-aliases [get]
func (m *ModelAlias) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelAliasQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelAliasBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelAliasAPI
// @Security ApiKeyAuth
// @Summary Get model alias record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.ModelAlias}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-aliases/{id} [get]
func (m *ModelAlias) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelAliasBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelAliasAPI
// @Security ApiKeyAuth
// @Summary Create model alias record
// @Param body body schema.ModelAliasForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelAlias}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-aliases [post]
func (m *ModelAlias) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelAliasForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelAliasBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelAliasAPI
// @Security ApiKeyAuth
// @Summary Update model alias record by ID
// @Param id path string true "unique id"
// @Param body body schema.ModelAliasForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-aliases/{id} [put]
func (m *ModelAlias) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelAliasForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelAliasBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelAliasAPI
// @Security ApiKeyAuth
// @Summary Delete model alias record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-aliases/{id} [delete]
func (m *ModelAlias) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelAliasBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
