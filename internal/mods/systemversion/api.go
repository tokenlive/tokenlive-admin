package systemversion

import (
	"context"
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
	"github.com/tokenlive/tokenlive-admin/pkg/upgradehost"
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
	result, err := a.Service.Summary(c.Request.Context(), a.canManage(c.Request.Context()), a.canUpgrade(c.Request.Context()))
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

// hostOrUnsupported returns the registered upgrade host, or nil after having
// answered with the honest unsupported capability. Independent admin without
// an embedding host must never fabricate local capability.
func hostOrUnsupported(c *gin.Context) upgradehost.Host {
	host := upgradehost.Current()
	if host == nil {
		util.ResSuccess(c, upgradehost.Capability{
			Supported: false,
			Allowed:   false,
			Reasons:   []string{upgradehost.ReasonHostUnsupported},
		})
		return nil
	}
	return host
}

// UpgradeCapability godoc
// @Tags SystemVersion
// @Summary Click-upgrade capability of the embedding host
// @Description Read-only. Requires update-management or upgrade-execution permission.
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} util.ResponseResult{data=upgradehost.Capability}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/upgrade/capability [get]
func (a *SystemVersion) UpgradeCapability(c *gin.Context) {
	if !a.canManage(c.Request.Context()) && !a.canUpgrade(c.Request.Context()) {
		util.ResError(c, errors.Forbidden("", "Version update or upgrade permission required"))
		return
	}
	host := hostOrUnsupported(c)
	if host == nil {
		return
	}
	capability, err := host.Capability(c.Request.Context())
	if err != nil {
		hostError(c, err)
		return
	}
	// Admin owns the check-switch knowledge: with checks off, execution is
	// honestly reported as not allowed (tasks already running still surface).
	if capability.Supported && !a.Service.Enabled() {
		capability.Allowed = false
		capability.Reasons = append(capability.Reasons, upgradehost.ReasonCheckDisabled)
	}
	util.ResSuccess(c, capability)
}

// checksGate enforces the existing "online checks disabled" semantics for the
// upgrade flow: no new task that requires external validation or download may
// be created while checks are off. Tasks already executing are not interrupted.
func (a *SystemVersion) checksGate(c *gin.Context) bool {
	if !a.Service.Enabled() {
		util.ResError(c, errors.Conflict("update_check_disabled", "%s", "Update checks are disabled"))
		return false
	}
	return true
}

// UpgradePrepare godoc
// @Tags SystemVersion
// @Summary Prepare a click upgrade and create a confirmation credential
// @Description Requires upgrade-execution permission. Validates the target, never installs or restarts. Rejected while online update checks are disabled.
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body upgradehost.PrepareRequest true "Target version"
// @Success 200 {object} util.ResponseResult{data=upgradehost.Preparation}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 409 {object} util.ResponseResult "upgrade_task_conflict | upgrade_target_changed | update_check_disabled"
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/upgrade/prepare [post]
func (a *SystemVersion) UpgradePrepare(c *gin.Context) {
	if !a.authorizeUpgradeExecution(c) {
		return
	}
	if !a.checksGate(c) {
		return
	}
	host := upgradehost.Current()
	if host == nil {
		util.ResError(c, errors.Conflict(upgradehost.ReasonHostUnsupported, "Click upgrade is not supported here"))
		return
	}
	var req upgradehost.PrepareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ResError(c, errors.BadRequest("", "Invalid prepare request: %s", err.Error()))
		return
	}
	req.Initiator = upgradeInitiator(c.Request.Context())
	preparation, err := host.Prepare(c.Request.Context(), req)
	if err != nil {
		hostError(c, err)
		return
	}
	util.ResSuccess(c, preparation)
}

// UpgradeSubmit godoc
// @Tags SystemVersion
// @Summary Submit a confirmed upgrade task
// @Description Requires upgrade-execution permission. Re-validates the credential, target and installation, then starts the one-shot task.
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body upgradehost.SubmitRequest true "Task confirmation"
// @Success 200 {object} util.ResponseResult{data=upgradehost.TaskView}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 409 {object} util.ResponseResult "upgrade_task_conflict | upgrade_target_changed | upgrade_confirmation_expired | update_check_disabled"
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/upgrade/submit [post]
func (a *SystemVersion) UpgradeSubmit(c *gin.Context) {
	if !a.authorizeUpgradeExecution(c) {
		return
	}
	if !a.checksGate(c) {
		return
	}
	host := upgradehost.Current()
	if host == nil {
		util.ResError(c, errors.Conflict(upgradehost.ReasonHostUnsupported, "Click upgrade is not supported here"))
		return
	}
	var req upgradehost.SubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.ResError(c, errors.BadRequest("", "Invalid submit request: %s", err.Error()))
		return
	}
	req.Initiator = upgradeInitiator(c.Request.Context())
	task, err := host.Submit(c.Request.Context(), req)
	if err != nil {
		hostError(c, err)
		return
	}
	util.ResSuccess(c, task)
}

// UpgradeTask godoc
// @Tags SystemVersion
// @Summary Read one persisted upgrade task
// @Description Requires upgrade-execution permission. Empty id returns the most recent task.
// @Security ApiKeyAuth
// @Produce json
// @Param id query string false "Task ID"
// @Success 200 {object} util.ResponseResult{data=upgradehost.TaskView}
// @Failure 401 {object} util.ResponseResult
// @Failure 403 {object} util.ResponseResult
// @Failure 500 {object} util.ResponseResult
// @Router /api/v1/system/upgrade/task [get]
func (a *SystemVersion) UpgradeTask(c *gin.Context) {
	if !a.authorizeUpgradeExecution(c) {
		return
	}
	host := upgradehost.Current()
	if host == nil {
		util.ResError(c, errors.Conflict(upgradehost.ReasonHostUnsupported, "Click upgrade is not supported here"))
		return
	}
	task, err := host.Task(c.Request.Context(), upgradehost.TaskRequest{TaskID: c.Query("id")})
	if err != nil {
		hostError(c, err)
		return
	}
	util.ResSuccess(c, task)
}

func (a *SystemVersion) authorizeUpgradeExecution(c *gin.Context) bool {
	if !a.canUpgrade(c.Request.Context()) {
		util.ResError(c, errors.Forbidden("", "Standalone upgrade execution permission required"))
		return false
	}
	return true
}

// upgradeInitiator names the authenticated caller for credential binding.
// Username when available, else the user ID; never a client-supplied value.
func upgradeInitiator(ctx context.Context) string {
	if name := util.FromUsername(ctx); name != "" {
		return name
	}
	return util.FromUserID(ctx)
}

// hostError maps host sentinel errors to stable public error IDs and keeps
// unexpected host failures as generic server errors.
func hostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, upgradehost.ErrTaskConflict):
		util.ResError(c, errors.Conflict("upgrade_task_conflict", "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrTargetChanged):
		util.ResError(c, errors.Conflict("upgrade_target_changed", "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrConfirmationExpired):
		util.ResError(c, errors.Conflict("upgrade_confirmation_expired", "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrInvalidCredential):
		util.ResError(c, errors.Conflict("upgrade_invalid_credential", "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrCheckDisabled):
		util.ResError(c, errors.Conflict("update_check_disabled", "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrUnsupported):
		util.ResError(c, errors.Conflict(upgradehost.ReasonHostUnsupported, "%s", err.Error()))
	case errors.Is(err, upgradehost.ErrNotAllowed):
		util.ResError(c, errors.Conflict("upgrade_not_allowed", "%s", err.Error()))
	default:
		util.ResError(c, err)
	}
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
