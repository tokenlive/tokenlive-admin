package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Loadbalance policy management
type PolicyLoadbalance struct {
	PolicyLoadbalanceBIZ *biz.PolicyLoadbalance
}

// @Tags PolicyLoadbalanceAPI
// @Security ApiKeyAuth
// @Summary Query policy loadbalance list
// @Success 200 {object} util.ResponseResult{data=[]schema.PolicyLoadbalance}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-loadbalances [get]
func (a *PolicyLoadbalance) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.PolicyLoadbalanceQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyLoadbalanceBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags PolicyLoadbalanceAPI
// @Security ApiKeyAuth
// @Summary Get policy loadbalance record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyLoadbalanceForm}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-loadbalances/{id} [get]
func (a *PolicyLoadbalance) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.PolicyLoadbalanceBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags PolicyLoadbalanceAPI
// @Security ApiKeyAuth
// @Summary Create policy loadbalance record
// @Param body body schema.PolicyLoadbalanceForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.PolicyLoadbalance}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-loadbalances [post]
func (a *PolicyLoadbalance) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyLoadbalanceForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.PolicyLoadbalanceBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PolicyLoadbalanceAPI
// @Security ApiKeyAuth
// @Summary Update policy loadbalance record by ID
// @Param id path string true "unique id"
// @Param body body schema.PolicyLoadbalanceForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-loadbalances/{id} [put]
func (a *PolicyLoadbalance) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.PolicyLoadbalanceForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.PolicyLoadbalanceBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PolicyLoadbalanceAPI
// @Security ApiKeyAuth
// @Summary Delete policy loadbalance record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/policy/policy-loadbalances/{id} [delete]
func (a *PolicyLoadbalance) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.PolicyLoadbalanceBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
