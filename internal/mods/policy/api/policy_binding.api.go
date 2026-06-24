package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// PolicyBinding 策略绑定 API 接口
type PolicyBinding struct {
	PolicyBindingBIZ *biz.PolicyBinding
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Query policy binding list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyBinding}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings [get]
func (a *PolicyBinding) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyBindingQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyBindingBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Get policy binding record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyBindingForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings/{id} [get]
func (a *PolicyBinding) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyBindingBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Create policy binding record
// @Param body body schema.PolicyBindingForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyBinding}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings [post]
func (a *PolicyBinding) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyBindingForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyBindingBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Update policy binding record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyBindingForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings/{id} [put]
func (a *PolicyBinding) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyBindingForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyBindingBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Toggle policy binding enabled status by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyBindingEnabledForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings/{id}/enabled [put]
func (a *PolicyBinding) UpdateEnabled(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyBindingEnabledForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyBindingBIZ.ToggleEnabled(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyBindingAPI
// @Security ApiKeyAuth
// @Summary Delete policy binding record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-bindings/{id} [delete]
func (a *PolicyBinding) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyBindingBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
