package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Route policy detail management
type PolicyRouteDetail struct {
	PolicyRouteDetailBIZ *biz.PolicyRouteDetail
}

// @Tags PolicyRouteDetailAPI
// @Security ApiKeyAuth
// @Summary Query policy route detail list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyRouteDetail}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-route-details [get]
func (a *PolicyRouteDetail) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyRouteDetailQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyRouteDetailBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyRouteDetailAPI
// @Security ApiKeyAuth
// @Summary Get policy route detail record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyRouteDetailForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-route-details/{id} [get]
func (a *PolicyRouteDetail) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyRouteDetailBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyRouteDetailAPI
// @Security ApiKeyAuth
// @Summary Create policy route detail record
// @Param body body schema.PolicyRouteDetailForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyRouteDetail}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-route-details [post]
func (a *PolicyRouteDetail) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyRouteDetailForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyRouteDetailBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyRouteDetailAPI
// @Security ApiKeyAuth
// @Summary Update policy route detail record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyRouteDetailForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-route-details/{id} [put]
func (a *PolicyRouteDetail) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyRouteDetailForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyRouteDetailBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyRouteDetailAPI
// @Security ApiKeyAuth
// @Summary Delete policy route detail record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-route-details/{id} [delete]
func (a *PolicyRouteDetail) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyRouteDetailBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
