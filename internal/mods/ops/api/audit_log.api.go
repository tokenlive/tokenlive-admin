package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// AuditLog management
type AuditLog struct {
	AuditLogBIZ *biz.AuditLog
}

// @Tags AuditLogAPI
// @Security ApiKeyAuth
// @Summary Query audit logs
// @Param current query int true "pagination index" default(1)
// @Param pageSize query int true "pagination size" default(10)
// @Param tenant_code query string false "Tenant code"
// @Param actor_user_id query string false "Actor user ID"
// @Param action query string false "Action"
// @Param resource_type query string false "Resource type"
// @Param resource_id query string false "Resource ID"
// @Param trace_id query string false "Trace ID"
// @Param start_time query string false "Start time"
// @Param end_time query string false "End time"
// @Success 200 {object} util.ResponseResult{data=[]schema.AuditLog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/audit-logs [get]
func (a *AuditLog) Query(c *gin.Context) {
	ctx := c.Request.Context()
	var params schema.AuditLogQueryParam
	if err := util.ParseQuery(c, &params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.AuditLogBIZ.Query(ctx, params)
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags AuditLogAPI
// @Security ApiKeyAuth
// @Summary Get audit log by ID
// @Param id path string true "Audit log ID"
// @Success 200 {object} util.ResponseResult{data=schema.AuditLog}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/audit-logs/{id} [get]
func (a *AuditLog) Get(c *gin.Context) {
	ctx := c.Request.Context()
	item, err := a.AuditLogBIZ.Get(ctx, c.Param("id"))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, item)
}
