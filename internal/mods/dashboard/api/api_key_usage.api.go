package api

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type UsageQuerier interface {
	Query(context.Context, schema.Query) (*schema.Response, error)
}

type APIKeyUsage struct {
	Service UsageQuerier
}

// Query returns the usage ranking only to the authenticated Root principal.
// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Query Root-only client API key usage ranking
// @Param time_range query string false "today, 1h, 6h, 24h, 7d"
// @Param sort_by query string false "tokens, request_count, cost"
// @Param limit query int false "10, 20, 50"
// @Success 200 {object} util.ResponseResult{data=schema.Response}
// @Failure 400,401,403,503 {object} util.ResponseResult
// @Router /api/v1/dashboard/api-key-ranking [get]
func (a *APIKeyUsage) Query(c *gin.Context) {
	ctx := c.Request.Context()
	if !util.FromIsRootUser(ctx) {
		util.ResError(c, errors.Forbidden("api_key_usage_forbidden", "Root access required"))
		return
	}
	query, err := schema.ParseQuery(c.Request.URL.Query())
	if err != nil {
		util.ResError(c, errors.BadRequest("invalid_api_key_usage_query", "Invalid ranking query"))
		return
	}
	result, err := a.Service.Query(ctx, query)
	if err != nil {
		// Driver errors may contain connection details. Never log credentials,
		// query parameters, raw key identities or an unfiltered source error.
		logging.Context(ctx).Warn("API key usage source unavailable")
		util.ResError(c, errors.New("api_key_usage_unavailable", "API key usage is temporarily unavailable", 503))
		return
	}
	util.ResSuccess(c, result)
}
