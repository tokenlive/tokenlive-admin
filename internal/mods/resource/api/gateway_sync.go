package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type GatewaySync struct {
	GatewaySyncBIZ *biz.GatewaySync
	EventBiz       *opsBiz.EventBiz
}

func (a *GatewaySync) validateToken(c *gin.Context) bool {
	expectedToken := os.Getenv("GATEWAY_SYNC_TOKEN")
	if expectedToken == "" {
		util.ResError(c, errors.Unauthorized("", "GATEWAY_SYNC_TOKEN is not configured on the server"))
		return false
	}

	token := c.GetHeader("X-Sync-Token")
	if token != expectedToken {
		util.ResError(c, errors.Unauthorized("", "invalid sync token"))
		return false
	}
	return true
}

func (a *GatewaySync) handleResponse(c *gin.Context, data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		util.ResError(c, err)
		return
	}

	hash := md5.Sum(jsonData)
	etag := `W/"` + hex.EncodeToString(hash[:]) + `"`

	ifMatch := c.GetHeader("If-None-Match")
	if ifMatch == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("ETag", etag)
	c.Data(http.StatusOK, "application/json", jsonData)
}

// GetConfig returns the gateway routing configurations (models, endpoints, providers)
func (a *GatewaySync) GetConfig(c *gin.Context) {
	if !a.validateToken(c) {
		return
	}

	modelCode := c.Query("model_code")
	data, err := a.GatewaySyncBIZ.GetGatewayConfig(c.Request.Context(), modelCode)
	if err != nil {
		util.ResError(c, err)
		return
	}

	a.handleResponse(c, data)
}

// GetPolicies returns the gateway governance policies
func (a *GatewaySync) GetPolicies(c *gin.Context) {
	if !a.validateToken(c) {
		return
	}

	modelCode := c.Query("model_code")
	data, err := a.GatewaySyncBIZ.GetGatewayPolicies(c.Request.Context(), modelCode)
	if err != nil {
		util.ResError(c, err)
		return
	}

	a.handleResponse(c, data)
}

// GetApiKeys returns the API Keys (users and tenants)
func (a *GatewaySync) GetApiKeys(c *gin.Context) {
	if !a.validateToken(c) {
		return
	}

	apiKey := c.Query("apikey")
	data, err := a.GatewaySyncBIZ.GetGatewayApiKeys(c.Request.Context(), apiKey)
	if err != nil {
		util.ResError(c, err)
		return
	}

	a.handleResponse(c, data)
}

// ReportMetrics receives the metrics and circuit breaker statuses from the gateway
func (a *GatewaySync) ReportMetrics(c *gin.Context) {
	if !a.validateToken(c) {
		return
	}

	var payload struct {
		Metrics       []metrics.RequestMetric `json:"metrics"`
		OpenEndpoints []string                `json:"open_endpoints"`
		OpenServices  []string                `json:"open_services"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		util.ResError(c, err)
		return
	}

	for _, m := range payload.Metrics {
		metrics.GlobalStore.Record(m)
	}

	metrics.GlobalStore.UpdateCircuitBreakers(payload.OpenEndpoints, payload.OpenServices)

	c.Status(http.StatusOK)
}

// ReportEvent receives a single policy event from the gateway and saves it to the database
func (a *GatewaySync) ReportEvent(c *gin.Context) {
	if !a.validateToken(c) {
		return
	}

	var payload struct {
		EventType    string   `json:"event_type"`
		TenantCode   string   `json:"tenant_code"`
		ModelCode    string   `json:"model_code"`
		EndpointID   string   `json:"endpoint_id"`
		EndpointCode string   `json:"endpoint_code"`
		ProviderName string   `json:"provider_name"`
		PolicyID     string   `json:"policy_id"`
		PolicyName   string   `json:"policy_name"`
		Threshold    *float64 `json:"threshold,omitempty"`
		CurrentValue *float64 `json:"current_value,omitempty"`
		RequestID    string   `json:"request_id"`
		TraceID      string   `json:"trace_id"`
		Message      string   `json:"message"`
		Timestamp    int64    `json:"ts"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		util.ResError(c, err)
		return
	}

	eventTime := time.Unix(payload.Timestamp, 0)
	item := &opsSchema.EventLog{
		EventType:    payload.EventType,
		TenantCode:   payload.TenantCode,
		ModelCode:    payload.ModelCode,
		EndpointID:   payload.EndpointID,
		EndpointCode: payload.EndpointCode,
		ProviderName: payload.ProviderName,
		PolicyID:     payload.PolicyID,
		PolicyName:   payload.PolicyName,
		Threshold:    payload.Threshold,
		CurrentValue: payload.CurrentValue,
		RequestID:    payload.RequestID,
		TraceID:      payload.TraceID,
		Message:      payload.Message,
		EventTime:    eventTime,
	}

	if err := a.EventBiz.CreateEvent(c.Request.Context(), item); err != nil {
		util.ResError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
