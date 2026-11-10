package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type UsageTrendsQuerier interface {
	Query(context.Context, schema.UsageTrendsQuery) (*schema.UsageTrendsResponse, error)
}

type UsageTrends struct {
	Service UsageTrendsQuerier
}

// Query returns aggregate-only trends under the dashboard's normal RBAC permission.
// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query request, token, latency and failure trends from request logs
// @Param time_range query string false "today, 1h, 6h, 24h, 7d"
// @Success 200 {object} util.ResponseResult{data=schema.UsageTrendsResponse}
// @Failure 400,401,403,503 {object} util.ResponseResult
// @Router /api/v1/dashboard/usage-trends [get]
func (a *UsageTrends) Query(c *gin.Context) {
	ctx := c.Request.Context()
	if util.FromUserID(ctx) == "" {
		util.ResError(c, errors.New("usage_trends_unauthorized", "Authentication required", 401))
		return
	}
	q, err := schema.ParseUsageTrendsQuery(c.Request.URL.Query())
	if err != nil {
		util.ResError(c, errors.BadRequest("invalid_usage_trends_query", "Invalid trends query"))
		return
	}
	response, err := a.Service.Query(ctx, q)
	if err != nil {
		logging.Context(ctx).Warn("Usage trends source unavailable")
		util.ResError(c, errors.New("usage_trends_unavailable", "Usage trends are temporarily unavailable", 503))
		return
	}
	util.ResSuccess(c, response)
}
