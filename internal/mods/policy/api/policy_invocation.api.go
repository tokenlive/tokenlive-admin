package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Invocation policy management
type PolicyInvocation struct {
	PolicyInvocationBIZ *biz.PolicyInvocation
}

// @Tags PolicyInvocationAPI
// @Security ApiKeyAuth
// @Summary Query policy invocation list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyInvocation}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-invocations [get]
func (a *PolicyInvocation) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyInvocationQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyInvocationBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyInvocationAPI
// @Security ApiKeyAuth
// @Summary Get policy invocation record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyInvocationForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-invocations/{id} [get]
func (a *PolicyInvocation) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyInvocationBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyInvocationAPI
// @Security ApiKeyAuth
// @Summary Create policy invocation record
// @Param body body schema.PolicyInvocationForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyInvocation}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-invocations [post]
func (a *PolicyInvocation) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyInvocationForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyInvocationBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyInvocationAPI
// @Security ApiKeyAuth
// @Summary Update policy invocation record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyInvocationForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-invocations/{id} [put]
func (a *PolicyInvocation) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyInvocationForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyInvocationBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyInvocationAPI
// @Security ApiKeyAuth
// @Summary Delete policy invocation record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-invocations/{id} [delete]
func (a *PolicyInvocation) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyInvocationBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
