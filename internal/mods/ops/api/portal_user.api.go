package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type PortalUserAPI struct {
	PortalUserBIZ *biz.PortalUser
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Search portal users
// @Param keyword query string false "search keyword"
// @Param limit query int false "limit" default(20)
// @Success 200 {object} util.ResponseResult{data=[]biz.PortalUserResult}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/users [get]
func (a *PortalUserAPI) Query(c *gin.Context) {
	ctx := c.Request.Context()
	keyword := c.Query("keyword")
	limitStr := c.Query("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	result, err := a.PortalUserBIZ.Search(ctx, keyword, limit)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary List portal workspace API keys
// @Param workspace_id path string true "workspace ID"
// @Success 200 {object} util.ResponseResult{data=[]biz.PortalWorkspaceAPIKey}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/api-keys [get]
func (a *PortalUserAPI) ListWorkspaceAPIKeys(c *gin.Context) {
	result, err := a.PortalUserBIZ.ListWorkspaceAPIKeys(c.Request.Context(), c.Param("workspace_id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Sync portal workspace API keys to gateway runtime
// @Param workspace_id path string true "workspace ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/runtime-sync [post]
func (a *PortalUserAPI) SyncWorkspaceRuntime(c *gin.Context) {
	if err := a.PortalUserBIZ.SyncWorkspaceRuntime(c.Request.Context(), c.Param("workspace_id")); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

type bindWorkspaceTenantRequest struct {
	TenantCode string `json:"tenant_code"`
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Bind portal workspace to admin tenant
// @Param workspace_id path string true "workspace ID"
// @Param body body bindWorkspaceTenantRequest true "tenant code"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/bind-tenant [post]
func (a *PortalUserAPI) BindWorkspaceTenant(c *gin.Context) {
	var req bindWorkspaceTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ResError(c, err)
		return
	}
	tenantCode := strings.TrimSpace(req.TenantCode)
	if tenantCode == "" {
		util.ResError(c, errors.BadRequest("", "tenant_code is required"))
		return
	}
	if err := a.PortalUserBIZ.BindWorkspaceTenant(c.Request.Context(), c.Param("workspace_id"), tenantCode); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Unbind portal workspace tenant
// @Param workspace_id path string true "workspace ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/unbind-tenant [post]
func (a *PortalUserAPI) UnbindWorkspaceTenant(c *gin.Context) {
	if err := a.PortalUserBIZ.UnbindWorkspaceTenant(c.Request.Context(), c.Param("workspace_id")); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
