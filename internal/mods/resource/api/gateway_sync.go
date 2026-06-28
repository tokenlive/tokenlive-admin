package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/biz"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

type GatewaySync struct {
	GatewaySyncBIZ *biz.GatewaySync
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
