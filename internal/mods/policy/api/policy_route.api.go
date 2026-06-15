package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Route policy management
type PolicyRoute struct {
	PolicyRouteBIZ *biz.PolicyRoute
}

// @Tags PolicyRouteAPI
// @Security ApiKeyAuth
// @Summary Query policy route list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyRoute}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-routes [get]
func (a *PolicyRoute) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyRouteQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyRouteBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyRouteAPI
// @Security ApiKeyAuth
// @Summary Get policy route record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyRouteForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-routes/{id} [get]
func (a *PolicyRoute) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyRouteBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyRouteAPI
// @Security ApiKeyAuth
// @Summary Create policy route record
// @Param body body schema.PolicyRouteForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyRoute}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-routes [post]
func (a *PolicyRoute) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyRouteForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyRouteBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyRouteAPI
// @Security ApiKeyAuth
// @Summary Update policy route record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyRouteForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-routes/{id} [put]
func (a *PolicyRoute) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyRouteForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyRouteBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyRouteAPI
// @Security ApiKeyAuth
// @Summary Delete policy route record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-routes/{id} [delete]
func (a *PolicyRoute) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyRouteBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
