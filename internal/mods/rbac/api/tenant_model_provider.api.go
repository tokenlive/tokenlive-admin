package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantModelProvider RBAC 模块下的租户-模型-供应商白名单 API 接口层
type TenantModelProvider struct {
	TenantModelProviderBIZ *biz.TenantModelProvider
}

// @Tags TenantModelProviderAPI
// @Security ApiKeyAuth
// @Summary 查询租户指定模型下已经允许的供应商 ID 列表
// @Param tenantCode query string true "租户英文编码"
// @Param modelId query string true "模型 ID"
// @Success 200 {object} util.ResponseResult{data=[]string}
// @Router /api/v1/tenant-models/providers [get]
func (a *TenantModelProvider) GetAllowedProviderIDs(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCode := c.Query("tenant_code")
	modelID := c.Query("model_id")
	if tenantCode == "" || modelID == "" {
		util.ResError(c, errors.BadRequest("", "tenant_code and model_id are required"))
		return
	}

	list, err := a.TenantModelProviderBIZ.GetAllowedProviderIDs(ctx, tenantCode, modelID)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, list)
}

// @Tags TenantModelProviderAPI
// @Security ApiKeyAuth
// @Summary 批量保存租户大模型的可访问上游供应商白名单
// @Param body body schema.TenantModelProviderForm true "保存表单"
// @Success 200 {object} util.ResponseResult
// @Router /api/v1/tenant-models/providers [post]
func (a *TenantModelProvider) SaveProviders(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.TenantModelProviderForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}

	creator := util.FromUsername(ctx)
	err := a.TenantModelProviderBIZ.SaveProviders(ctx, item.TenantCode, item.ModelID, item.ProviderIDs, creator)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
