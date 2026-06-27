package api

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalog management
type ModelCatalog struct {
	ModelCatalogBIZ *biz.ModelCatalog
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Query model catalog list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param slug query string false "Slug (like)"
// @Param status query string false "Status (available/paused)"
// @Param visibility query string false "Visibility (public/private)"
// @Param featured query bool false "Featured"
// @Param model_code query string false "Model code"
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs [get]
func (m *ModelCatalog) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.ModelCatalogQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Get model catalog by ID
// @Param id path string true "Model ID"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [get]
func (m *ModelCatalog) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelCatalogBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Get model catalog by slug
// @Param slug path string true "Slug"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/slug/{slug} [get]
func (m *ModelCatalog) GetBySlug(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := m.ModelCatalogBIZ.GetBySlug(ctx, c.Param("slug"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Create model catalog
// @Param body body schema.ModelCatalogForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.ModelCatalog}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs [post]
func (m *ModelCatalog) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := m.ModelCatalogBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Update model catalog
// @Param id path string true "Model ID"
// @Param body body schema.ModelCatalogForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [put]
func (m *ModelCatalog) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Publish model catalog
// @Param id path string true "Model ID"
// @Param body body schema.ModelCatalogPublishForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id}/publish [put]
func (m *ModelCatalog) Publish(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.ModelCatalogPublishForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := m.ModelCatalogBIZ.Publish(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Delete model catalog
// @Param id path string true "Model ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/{id} [delete]
func (m *ModelCatalog) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := m.ModelCatalogBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags ModelCatalogAPI
// @Security ApiKeyAuth
// @Summary Query public model catalogs (for portal)
// @Param limit query int false "Limit" default(50)
// @Success 200 {object} util.ResponseResult{data=[]schema.ModelCatalog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/model-catalogs/public [get]
func (m *ModelCatalog) QueryPublic(c *gin.Context) {
	ctx := c.Request.Context()
	limit := 50
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	result, err := m.ModelCatalogBIZ.QueryPublic(ctx, limit)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
