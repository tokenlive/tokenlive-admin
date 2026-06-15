package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Space management for microservice spaces
type Space struct {
	SpaceBIZ *biz.Space
}

// @Tags SpaceAPI
// @Security ApiKeyAuth
// @Summary Query space list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param name query string false "Name of space"
// @Param code query string false "Code of space"
// @Param tenant query string false "Tenant"
// @Success 200 {object} util.ResponseResult{data=[]schema.Space}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/spaces [get]
func (a *Space) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.SpaceQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.SpaceBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags SpaceAPI
// @Security ApiKeyAuth
// @Summary Get space record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.Space}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/spaces/{id} [get]
func (a *Space) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.SpaceBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags SpaceAPI
// @Security ApiKeyAuth
// @Summary Create space record
// @Param body body schema.SpaceForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.Space}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/spaces [post]
func (a *Space) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.SpaceForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.SpaceBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags SpaceAPI
// @Security ApiKeyAuth
// @Summary Update space record by ID
// @Param id path string true "unique id"
// @Param body body schema.SpaceForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/spaces/{id} [put]
func (a *Space) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.SpaceForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.SpaceBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags SpaceAPI
// @Security ApiKeyAuth
// @Summary Delete space record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/spaces/{id} [delete]
func (a *Space) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.SpaceBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
