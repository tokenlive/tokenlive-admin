package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Circuit break policy management
type PolicyCircuitBreak struct {
	PolicyCircuitBreakBIZ *biz.PolicyCircuitBreak
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Query policy circuit break list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyCircuitBreak}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks [get]
func (a *PolicyCircuitBreak) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyCircuitBreakQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyCircuitBreakBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Get policy circuit break record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyCircuitBreakForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks/{id} [get]
func (a *PolicyCircuitBreak) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyCircuitBreakBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Create policy circuit break record
// @Param body body schema.PolicyCircuitBreakForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyCircuitBreak}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks [post]
func (a *PolicyCircuitBreak) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyCircuitBreakForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyCircuitBreakBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Update policy circuit break record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyCircuitBreakForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks/{id} [put]
func (a *PolicyCircuitBreak) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyCircuitBreakForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyCircuitBreakBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Toggle policy circuit break enabled status by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyEnabledForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks/{id}/enabled [put]
func (a *PolicyCircuitBreak) UpdateEnabled(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyEnabledForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyCircuitBreakBIZ.ToggleEnabled(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Delete policy circuit break record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks/{id} [delete]
func (a *PolicyCircuitBreak) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyCircuitBreakBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyCircuitBreakAPI
// @Security ApiKeyAuth
// @Summary Copy policy circuit break template to model
// @Param id path string true "template id"
// @Param body body schema.PolicyCopyToModelForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyCircuitBreak}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-circuit-breaks/{id}/copy-to-model [post]
func (a *PolicyCircuitBreak) CopyToModel(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyCopyToModelForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyCircuitBreakBIZ.CopyTemplateToModel(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}
