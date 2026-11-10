package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard/monitor"
	resourcebiz "github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type Dashboard struct {
	Monitor   *monitor.Reader
	RedisSync *resourcebiz.ConfigRedisSync
}

// QuerySnapshot reads all selected results at one evaluation cutoff.
// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Read a monitoring snapshot with per-reading windows and coverage
// @Param query body monitor.Query true "Selected monitoring readings"
// @Success 200 {object} util.ResponseResult{data=monitor.Snapshot}
// @Failure 400 {object} util.ResponseResult
// @Router /api/v1/dashboard/snapshot [post]
func (a *Dashboard) QuerySnapshot(c *gin.Context) {
	var q monitor.Query
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&q); err != nil {
		util.ResError(c, errors.BadRequest("invalid_monitoring_query", "Invalid monitoring query"))
		return
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		util.ResError(c, errors.BadRequest("invalid_monitoring_query", "Expected one monitoring query"))
		return
	}
	if a.Monitor == nil {
		util.ResError(c, errors.New("monitoring_unavailable", "Monitoring is unavailable", 503))
		return
	}
	res, err := a.Monitor.Read(c.Request.Context(), q)
	if err != nil {
		util.ResError(c, errors.BadRequest("invalid_monitoring_query", "%s", err.Error()))
		return
	}
	util.ResSuccess(c, res)
}

// QueryCircuitBreakers reads currently isolated endpoints and services.
// @Tags DashboardAPI
// @Security ApiKeyAuth
// @Summary Read active circuit breakers
// @Success 200 {object} util.ResponseResult{data=[]monitor.CircuitBreakerInfo}
// @Failure 503 {object} util.ResponseResult
// @Router /api/v1/dashboard/circuit-breakers [get]
func (a *Dashboard) QueryCircuitBreakers(c *gin.Context) {
	if a.Monitor == nil {
		util.ResError(c, errors.New("monitoring_unavailable", "Monitoring is unavailable", 503))
		return
	}
	data, err := a.Monitor.CircuitBreakers(c.Request.Context())
	if err != nil {
		util.ResError(c, errors.New("source_unavailable", "Circuit breaker status is unavailable", 503))
		return
	}
	util.ResSuccess(c, data)
}
