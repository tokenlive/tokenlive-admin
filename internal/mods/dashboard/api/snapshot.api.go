package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

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
	a.snapshot(c, q)
}

func (a *Dashboard) snapshot(c *gin.Context, q monitor.Query) {
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

// Existing single-reading routes are transport adapters to the same query seam.
// They return the canonical snapshot, not a second independently defined DTO.
func (a *Dashboard) query(c *gin.Context, selection string) {
	q := monitor.Query{Timezone: c.Query("timezone")}
	if value := c.Query("end_time"); value != "" {
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			util.ResError(c, errors.BadRequest("invalid_monitoring_time", "Invalid end_time"))
			return
		}
		q.At = &at
	}
	w := monitor.WindowQuery{Range: c.DefaultQuery("time_range", "1h")}
	if value := c.Query("start_time"); value != "" {
		at, err := time.Parse(time.RFC3339, value)
		if err != nil {
			util.ResError(c, errors.BadRequest("invalid_monitoring_time", "Invalid start_time"))
			return
		}
		w.Start = &at
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		util.ResError(c, errors.BadRequest("invalid_monitoring_limit", "Invalid limit"))
		return
	}
	switch selection {
	case "overview":
		q.Overview = true
	case "traffic":
		q.Traffic = &monitor.TrafficQuery{WindowQuery: w, GroupBy: c.Query("group_by"), Model: c.Query("model")}
	case "performance":
		q.Performance = &monitor.PerformanceQuery{WindowQuery: w, Model: c.Query("model")}
	case "models":
		q.Models = &monitor.ModelQuery{WindowQuery: w, Model: c.Query("model"), Sort: c.DefaultQuery("sort_by", "request_count"), Limit: limit}
	case "providers":
		q.Providers = &monitor.ProviderQuery{WindowQuery: w, Provider: c.Query("provider"), Sort: c.DefaultQuery("sort_by", "request_count"), Limit: limit}
	}
	a.snapshot(c, q)
}

func (a *Dashboard) QueryOverview(c *gin.Context)               { a.query(c, "overview") }
func (a *Dashboard) QueryTrends(c *gin.Context)                 { a.query(c, "traffic") }
func (a *Dashboard) QueryModelPerformanceTrends(c *gin.Context) { a.query(c, "performance") }
func (a *Dashboard) QueryModelRanking(c *gin.Context)           { a.query(c, "models") }
func (a *Dashboard) QueryProviderRanking(c *gin.Context)        { a.query(c, "providers") }
func (a *Dashboard) QueryCircuitBreakers(c *gin.Context) {
	if a.Monitor == nil {
		util.ResError(c, fmt.Errorf("monitoring is unavailable"))
		return
	}
	util.ResSuccess(c, a.Monitor.CircuitBreakers(c.Request.Context()))
}
func (a *Dashboard) QueryQPS(c *gin.Context) { a.query(c, "overview") }
