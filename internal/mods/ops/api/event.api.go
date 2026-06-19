package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// EventAPI handles HTTP requests for event operations.
type EventAPI struct {
	EventBIZ *biz.EventBiz
	Hub      *WSHub
}

// @Tags EventAPI
// @Security ApiKeyAuth
// @Summary Query event logs with pagination and filtering
// @Param current query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(20)
// @Param event_type query string false "Event type filter"
// @Param tenant_code query string false "Tenant code filter"
// @Param model_code query string false "Model code filter"
// @Param endpoint_id query string false "Endpoint ID filter"
// @Param policy_id query string false "Policy ID filter"
// @Param start_time query string false "Start time (ISO 8601)"
// @Param end_time query string false "End time (ISO 8601)"
// @Success 200 {object} util.ResponseResult{data=schema.EventLogs}
// @Router /api/v1/ops/events [get]
func (a *EventAPI) Query(c *gin.Context) {
	var params schema.EventQueryParam
	if err := c.ShouldBindQuery(&params); err != nil {
		util.ResError(c, err)
		return
	}

	result, err := a.EventBIZ.QueryEvents(c.Request.Context(), params)
	if err != nil {
		util.ResError(c, err)
		return
	}

	util.ResPage(c, result.Data, result.PageResult)
}

// @Tags EventAPI
// @Security ApiKeyAuth
// @Summary Get event statistics (counts, trend, rankings)
// @Param time_range query string false "Time range" Enums(1h, 6h, 24h, 7d, today) default(24h)
// @Success 200 {object} util.ResponseResult{data=schema.EventStatistics}
// @Router /api/v1/ops/events/statistics [get]
func (a *EventAPI) GetStatistics(c *gin.Context) {
	timeRange := c.DefaultQuery("time_range", "24h")

	stats, err := a.EventBIZ.GetStatistics(c.Request.Context(), timeRange)
	if err != nil {
		util.ResError(c, err)
		return
	}

	util.ResSuccess(c, stats)
}

// @Tags EventAPI
// @Security ApiKeyAuth
// @Summary WebSocket endpoint for real-time event push
// @Router /api/v1/ops/events/ws [get]
func (a *EventAPI) HandleWebSocket(c *gin.Context) {
	HandleWebSocket(a.Hub)(c)
}
