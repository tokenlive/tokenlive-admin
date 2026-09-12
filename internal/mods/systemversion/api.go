package systemversion

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

const maxReportBytes = 4 << 10

// CurrentVersion godoc
// @Tags SystemVersion
// @Summary Current local version and aggregated Gateway builds
// @Description Requires login, but not update-management permission. Contains no update targets or node identities.
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} util.ResponseResult{data=versionstatus.Summary}
// @Failure 401 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/current/version [get]
func (a *SystemVersion) CurrentVersion(c *gin.Context) {
	result, err := a.Service.Summary(c.Request.Context(), a.canManage(c.Request.Context()))
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// Updates godoc
// @Tags SystemVersion
// @Summary Read cached update results
// @Description Requires the POST /api/v1/system/updates/check capability. Does not contact update sources.
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} util.ResponseResult{data=versionstatus.Updates}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/updates [get]
func (a *SystemVersion) Updates(c *gin.Context) {
	if !a.authorizeUpdates(c) {
		return
	}
	result, err := a.Service.Updates(c.Request.Context())
	if err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, result)
}

// Check godoc
// @Tags SystemVersion
// @Summary Check for stable version updates
// @Description Requires update-management permission. Disabled and cooldown requests do not contact sources; they return success=false with the current Updates in data.
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} util.ResponseResult{data=versionstatus.Updates}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 409 {object} util.ResponseResult{data=versionstatus.Updates} "update_check_disabled"
// @Failure 429 {object} util.ResponseResult{data=versionstatus.Updates} "update_check_cooldown"
// @Header 429 {integer} Retry-After "Seconds until manual checks are permitted"
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/updates/check [post]
func (a *SystemVersion) Check(c *gin.Context) {
	if !a.authorizeUpdates(c) {
		return
	}
	result, err := a.Service.Check(c.Request.Context())
	switch {
	case errors.Is(err, updatecheck.ErrDisabled):
		checkRejected(c, result, errors.Conflict("update_check_disabled", "Update checks are disabled"))
	case errors.Is(err, updatecheck.ErrCooldown):
		c.Header("Retry-After", strconv.Itoa(result.RetryAfterSeconds))
		checkRejected(c, result, errors.TooManyRequests("update_check_cooldown", "Manual update check is cooling down"))
	case err != nil:
		util.ResError(c, err)
	default:
		util.ResSuccess(c, result)
	}
}

func (a *SystemVersion) authorizeUpdates(c *gin.Context) bool {
	if !a.canManage(c.Request.Context()) {
		util.ResError(c, errors.Forbidden("", "Version update management permission required"))
		return false
	}
	return true
}

// checkRejected retains the standard error envelope while returning the current
// snapshot, so disabled/cooldown cannot be mistaken for a completed check.
func checkRejected(c *gin.Context, result versionstatus.Updates, err error) {
	apiError := errors.FromError(err)
	util.ResJSON(c, int(apiError.Code), util.ResponseResult{Data: result, Error: apiError})
}

// Report godoc
// @Tags SystemVersion
// @Summary Receive a Gateway version report
// @Description Separate deployment-token boundary; no user JWT. Requires a configured GATEWAY_SYNC_TOKEN, an exact X-Sync-Token match and a valid report in the configured namespace. Reports expire after three minutes.
// @Accept json
// @Produce json
// @Param X-Sync-Token header string true "Deployment synchronization token"
// @Param node body versionregistry.Node true "Gateway report (maximum 4096 bytes)"
// @Success 200 {object} util.ResponseResult
// @Failure 400 {object} util.ResponseResult
// @Failure 401 {object} util.ResponseResult
// @Failure 413 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/gateway/version [post]
func (a *SystemVersion) Report(c *gin.Context) {
	expected := os.Getenv("GATEWAY_SYNC_TOKEN")
	// Compare fixed-size digests so token length does not create a comparison
	// shortcut. An unconfigured or missing token always fails closed.
	expectedHash := sha256.Sum256([]byte(expected))
	token := c.GetHeader("X-Sync-Token")
	tokenHash := sha256.Sum256([]byte(token))
	if expected == "" || token == "" || subtle.ConstantTimeCompare(expectedHash[:], tokenHash[:]) != 1 {
		util.ResError(c, errors.Unauthorized("", "Invalid deployment synchronization token"))
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxReportBytes))
	if err != nil {
		util.ResError(c, errors.RequestEntityTooLarge("", "Gateway version report exceeds 4096 bytes"))
		return
	}
	var node versionregistry.Node
	if err := json.Unmarshal(body, &node); err != nil {
		util.ResError(c, errors.BadRequest("", "Invalid Gateway version report JSON"))
		return
	}
	namespace := config.C.Gateway.VersionNamespace
	if value, ok := os.LookupEnv("GATEWAY_VERSION_NAMESPACE"); ok {
		namespace = value
	}
	if err := versionregistry.Validate(node, namespace); err != nil {
		util.ResError(c, errors.BadRequest("", "Invalid Gateway version report"))
		return
	}
	if err := a.Service.Report(c.Request.Context(), node); err != nil {
		util.ResError(c, err)
		return
	}
	util.ResSuccess(c, nil)
}
