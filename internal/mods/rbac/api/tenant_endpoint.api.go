package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantEndpoint RBAC 模块下的租户-端点关联 API 接口层
type TenantEndpoint struct {
	TenantEndpointBIZ *biz.TenantEndpoint
}

// @Tags TenantEndpointAPI
// @Security ApiKeyAuth
// @Summary 查询租户指定模型下已经允许的端点 ID 列表
// @Param tenantCode query string true "租户英文编码"
// @Param modelId query string true "模型 ID"
// @Success 200 {object} util.ResponseResult{data=[]string}
// @Router /api/v1/tenant-models/endpoints [get]
func (a *TenantEndpoint) GetAllowedEndpointIDs(c *gin.Context) {
	ctx := c.Request.Context()
	tenantCode := c.Query("tenant_code")
	modelID := c.Query("model_id")
	if tenantCode == "" || modelID == "" {
		util.ResError(c, errors.BadRequest("", "tenant_code and model_id are required"))
		return
	}

	list, err := a.TenantEndpointBIZ.GetAllowedEndpointIDs(ctx, tenantCode, modelID)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, list)
}

// @Tags TenantEndpointAPI
// @Security ApiKeyAuth
// @Summary 批量保存租户大模型的可访问端点白名单
// @Param body body schema.TenantEndpointForm true "保存表单"
// @Success 200 {object} util.ResponseResult
// @Router /api/v1/tenant-models/endpoints [post]
func (a *TenantEndpoint) SaveEndpoints(c *gin.Context) {
	ctx := c.Request.Context()
	item := new(schema.TenantEndpointForm)
	if err := util.ParseJSON(c, item); err != nil {
		util.ResError(c, err)
		return
	}

	creator := util.FromUsername(ctx)
	err := a.TenantEndpointBIZ.SaveEndpoints(ctx, item.TenantCode, item.ModelID, item.EndpointIDs, creator)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
