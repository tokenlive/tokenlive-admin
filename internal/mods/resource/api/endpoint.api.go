package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Endpoint management
type Endpoint struct {
	EndpointBIZ *biz.Endpoint
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Query endpoint list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param provider_id query string false "Provider ID"
// @Param url query string false "URL (like)"
// @Success 200 {object} util.ResponseResult{data=[]schema.Endpoint}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints [get]
func (e *Endpoint) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.EndpointQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := e.EndpointBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Get endpoint record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.Endpoint}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/{id} [get]
func (e *Endpoint) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := e.EndpointBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Create endpoint record
// @Param body body schema.EndpointForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.Endpoint}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints [post]
func (e *Endpoint) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.EndpointForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := e.EndpointBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Update endpoint record by ID
// @Param id path string true "unique id"
// @Param body body schema.EndpointForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/{id} [put]
func (e *Endpoint) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.EndpointForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := e.EndpointBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Toggle endpoint enabled status by ID
// @Param id path string true "unique id"
// @Param body body schema.EndpointEnabledForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/{id}/enabled [put]
func (e *Endpoint) UpdateEnabled(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.EndpointEnabledForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := e.EndpointBIZ.ToggleEnabled(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Delete endpoint record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/{id} [delete]
func (e *Endpoint) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := e.EndpointBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Test endpoint connectivity (draft configuration)
// @Param body body schema.EndpointForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.EndpointTestResult}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/test [post]
func (e *Endpoint) Test(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.EndpointForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := e.EndpointBIZ.Test(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Test endpoint connectivity by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.EndpointTestResult}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/endpoints/{id}/test [post]
func (e *Endpoint) TestByID(c *gin.Context) {
	ctx := c.Request.Context()
	result, err := e.EndpointBIZ.TestByID(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Query endpoints by Model ID
// @Param id path string true "Model ID"
// @Success 200 {object} util.ResponseResult{data=[]schema.Endpoint}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/models/{id}/endpoints [get]
func (e *Endpoint) QueryEndpointsByModelID(c *gin.Context) {
	ctx := c.Request.Context()
	modelID := c.Param("id")
	if modelID == "" {
		util.ResError(c, errors.BadRequest("", "Model ID is required"))
		return
	}

	result, err := e.EndpointBIZ.QueryEndpointsByModelID(ctx, modelID)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags EndpointAPI
// @Security ApiKeyAuth
// @Summary Query endpoints by Provider ID
// @Param id path string true "Provider ID"
// @Success 200 {object} util.ResponseResult{data=[]schema.Endpoint}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/providers/{id}/endpoints [get]
func (e *Endpoint) QueryEndpointsByProviderID(c *gin.Context) {
	ctx := c.Request.Context()
	providerID := c.Param("id")
	if providerID == "" {
		util.ResError(c, errors.BadRequest("", "Provider ID is required"))
		return
	}

	result, err := e.EndpointBIZ.QueryEndpointsByProviderID(ctx, providerID)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
