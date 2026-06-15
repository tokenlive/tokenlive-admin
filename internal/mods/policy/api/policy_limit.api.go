package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Limit policy management
type PolicyLimit struct {
	PolicyLimitBIZ *biz.PolicyLimit
}

// @Tags PolicyLimitAPI
// @Security ApiKeyAuth
// @Summary Query policy limit list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyLimit}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-limits [get]
func (a *PolicyLimit) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyLimitQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyLimitBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyLimitAPI
// @Security ApiKeyAuth
// @Summary Get policy limit record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyLimitForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-limits/{id} [get]
func (a *PolicyLimit) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyLimitBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyLimitAPI
// @Security ApiKeyAuth
// @Summary Create policy limit record
// @Param body body schema.PolicyLimitForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyLimit}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-limits [post]
func (a *PolicyLimit) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyLimitForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyLimitBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyLimitAPI
// @Security ApiKeyAuth
// @Summary Update policy limit record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyLimitForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-limits/{id} [put]
func (a *PolicyLimit) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyLimitForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyLimitBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyLimitAPI
// @Security ApiKeyAuth
// @Summary Delete policy limit record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-limits/{id} [delete]
func (a *PolicyLimit) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyLimitBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
