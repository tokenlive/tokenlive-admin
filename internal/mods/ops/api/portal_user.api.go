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

type activateWorkspaceRuntimeAccessRequest struct {
	ScopeType string `json:"scope_type"`
	ScopeCode string `json:"scope_code"`
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Get portal workspace runtime access
// @Param workspace_id path string true "workspace ID"
// @Success 200 {object} util.ResponseResult{data=biz.PortalWorkspaceRuntimeAccess}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/runtime-access [get]
func (a *PortalUserAPI) GetWorkspaceRuntimeAccess(c *gin.Context) {
	result, err := a.PortalUserBIZ.GetWorkspaceRuntimeAccess(c.Request.Context(), c.Param("workspace_id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Activate portal workspace runtime access
// @Param workspace_id path string true "workspace ID"
// @Param body body activateWorkspaceRuntimeAccessRequest true "runtime access scope"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/runtime-access [put]
func (a *PortalUserAPI) ActivateWorkspaceRuntimeAccess(c *gin.Context) {
	var req activateWorkspaceRuntimeAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ResError(c, err)
		return
	}
	scopeType := strings.TrimSpace(req.ScopeType)
	if scopeType == "" {
		scopeType = "tenant"
	}
	scopeCode := strings.TrimSpace(req.ScopeCode)
	if scopeType != "tenant" || scopeCode == "" {
		util.ResError(c, errors.BadRequest("", "scope_type must be tenant and scope_code is required"))
		return
	}
	if err := a.PortalUserBIZ.ActivateWorkspaceRuntimeAccess(c.Request.Context(), c.Param("workspace_id"), scopeType, scopeCode); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}

// @Tags PortalUserAPI
// @Security ApiKeyAuth
// @Summary Disable portal workspace runtime access
// @Param workspace_id path string true "workspace ID"
// @Success 200 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/ops/portal/workspaces/{workspace_id}/runtime-access/disable [post]
func (a *PortalUserAPI) DisableWorkspaceRuntimeAccess(c *gin.Context) {
	if err := a.PortalUserBIZ.DisableWorkspaceRuntimeAccess(c.Request.Context(), c.Param("workspace_id")); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResOK(c)
}
