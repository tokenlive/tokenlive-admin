package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantModel management for RBAC
type TenantModel struct {
	TenantModelBIZ *biz.TenantModel
}

// @Tags TenantModelAPI
// @Security ApiKeyAuth
// @Summary Get all authorized model IDs of the tenant
// @Param tenantCode path string true "Tenant Code"
// @Success 200 {object} util.ResponseResult{data=[]string}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenant-models/{tenantCode} [get]
func (a *TenantModel) GetAuthorizedModelIDs(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCode := c.Param("tenantCode")
	list, err := a.TenantModelBIZ.GetAuthorizedModelIDs(ctx, tenantCode)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, list)
}

// @Tags TenantModelAPI
// @Security ApiKeyAuth
// @Summary Batch save tenant model bindings
// @Param body body schema.TenantModelForm true "Request body"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/tenant-models/bindings [post]
func (a *TenantModel) SaveBindings(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.TenantModelForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}

	creator := util.FromUsername(ctx)
	err := a.TenantModelBIZ.SaveBindings(ctx, item.TenantCode, item.ModelIDs, creator)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
