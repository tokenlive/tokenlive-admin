package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Tenant management for RBAC
type Tenant struct {
	TenantBIZ *biz.Tenant
}

// @Tags TenantAPI
// @Security ApiKeyAuth
// @Summary Query tenant list
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param code query string false "Tenant code"
// @Param name query string false "Name of tenant"
// @Param status query string false "Status of tenant (activated, freezed)"
// @Success 200 {object} util.ResponseResult{data=[]schema.Tenant}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenants [get]
func (a *Tenant) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.TenantQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.TenantBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags TenantAPI
// @Security ApiKeyAuth
// @Summary Get tenant record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult{data=schema.Tenant}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenants/{id} [get]
func (a *Tenant) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.TenantBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}

// @Tags TenantAPI
// @Security ApiKeyAuth
// @Summary Create tenant record
// @Param body body schema.TenantForm true "Request body"
// @Success 200 {object} util.ResponseResult{data=schema.Tenant}
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenants [post]
func (a *Tenant) Create(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.TenantForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.TenantBIZ.Create(ctx, item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags TenantAPI
// @Security ApiKeyAuth
// @Summary Update tenant record by ID
// @Param id path string true "unique id"
// @Param body body schema.TenantForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenants/{id} [put]
func (a *Tenant) Update(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.TenantForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	} else if err := item.Validate(); err != nil {
		util.ResError(c, err)
		return
	}

	err := a.TenantBIZ.Update(ctx, c.Param("id"), item)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags TenantAPI
// @Security ApiKeyAuth
// @Summary Delete tenant record by ID
// @Param id path string true "unique id"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenants/{id} [delete]
func (a *Tenant) Delete(c *gin.Context) {
	ctx := c.Request.Context()
	err := a.TenantBIZ.Delete(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
