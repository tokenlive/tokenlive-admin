// Package upgradehost defines the public click-upgrade contract between the
// embedding host (tokenlive-standalone) and tokenlive-admin. The host owns the
// real local upgrade machinery; admin only authorizes requests and forwards
// them. An admin deployment without a registered host reports unsupported.
//
// The interface is intentionally small: capability discovery, a two-phase
// prepare/submit upgrade flow and task reads. Hosts must treat every request
// as untrusted: re-validate the caller, the installation and the confirmed
// target server-side on every call.
package upgradehost

import (
	"context"
	"errors"
	"sync"
)

// Host is implemented by an embedding runtime that can execute local
// upgrades. All methods must be safe for concurrent use.
type Host interface {
	// Capability reports whether the running installation supports click
	// upgrade, whether the server-side conditions currently allow it, the
	// concrete blocking reasons otherwise, and the active task if one exists.
	// Read-only: it must not mutate state or contact update sources.
	Capability(ctx context.Context) (Capability, error)

	// Prepare validates the requested target against a fresh candidate,
	// creates an awaiting-confirmation task and returns the confirmation
	// material. It must not install, restart or touch the running service.
	Prepare(ctx context.Context, req PrepareRequest) (Preparation, error)

	// Submit accepts a previously prepared confirmation. It re-validates the
	// permission boundary, the credential, the target and the installation,
	// then starts the one-shot upgrade task exactly once.
	Submit(ctx context.Context, req SubmitRequest) (TaskView, error)

	// Task returns one persisted task by ID, or the most recent task when
	// TaskID is empty. Returned views contain no secrets or tokens.
	Task(ctx context.Context, req TaskRequest) (TaskView, error)
}

// PrepareRequest asks the host to prepare an upgrade to the exact stable
// version the user saw and clicked.
type PrepareRequest struct {
	TargetVersion string `json:"target_version"`
	// Initiator is the authenticated username the admin API binds the
	// confirmation credential to.
	Initiator string `json:"-"`
}

// SubmitRequest confirms a prepared task. The credential is the opaque value
// returned by Prepare; the host stores only its digest.
type SubmitRequest struct {
	TaskID     string `json:"task_id"`
	Credential string `json:"credential"`
	Confirm    bool   `json:"confirm"`
	Initiator  string `json:"-"`
}

// TaskRequest selects a task view. An empty TaskID means the most recent task.
type TaskRequest struct {
	TaskID string `json:"task_id,omitempty"`
}

// Capability is the read-only answer shown before any confirm dialog.
type Capability struct {
	Supported bool     `json:"supported"`
	Allowed   bool     `json:"allowed"`
	Reasons   []string `json:"reasons,omitempty"`
	// ActiveTask is non-nil when a task exists that is neither terminal nor
	// fully cleaned up; the UI must show it instead of a new button.
	ActiveTask *TaskView `json:"active_task,omitempty"`
}

// Preparation is the result of a successful Prepare. Credential is returned
// exactly once; the server keeps only a digest bound to Initiator, the
// installation and the target.
type Preparation struct {
	TaskID         string `json:"task_id"`
	CurrentVersion string `json:"current_version,omitempty"`
	TargetVersion  string `json:"target_version,omitempty"`
	// ReleaseURL points at the official release notes for the target.
	ReleaseURL string `json:"release_url,omitempty"`
	// ConfirmExpiresIn is the remaining credential validity in seconds.
	ConfirmExpiresIn int    `json:"confirm_expires_in"`
	Credential       string `json:"credential"`
	// RestartNotice tells the UI which restart impact text to render; the
	// wording itself stays in frontend i18n.
	RestartWarning bool `json:"restart_warning"`
}

// TaskView is the safe, bounded projection of a persisted upgrade task. It
// never contains credentials, tokens, absolute user paths beyond the
// installation prefix or raw external command output.
type TaskView struct {
	TaskID         string `json:"task_id"`
	State          string `json:"state"`
	CurrentVersion string `json:"current_version,omitempty"`
	TargetVersion  string `json:"target_version,omitempty"`
	ReleaseURL     string `json:"release_url,omitempty"`
	Initiator      string `json:"initiator,omitempty"`
	// Phase is the concrete stage inside a non-terminal state, e.g. the
	// brew command currently running; informational only.
	Phase        string `json:"phase,omitempty"`
	FailureStage string `json:"failure_stage,omitempty"`
	ErrorKind    string `json:"error_kind,omitempty"`
	// Detail is a short, sanitized human-readable diagnostic summary.
	Detail      string `json:"detail,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
	SucceededAt string `json:"succeeded_at,omitempty"`
}

// Task states, shared by host implementation and admin API consumers. The
// names mirror the confirmed design so the UI can map them to i18n directly.
const (
	StatePreparing            = "preparing"
	StateAwaitingConfirmation = "awaiting_confirmation"
	StateQueued               = "queued"
	StateDownloading          = "downloading"
	StateInstalling           = "installing"
	StateRestarting           = "restarting"
	StateVerifying            = "verifying"
	StateSucceeded            = "succeeded"
	StateFailed               = "failed"
	StateNeedsAttention       = "needs_attention"
	StateConfirmationExpired  = "confirmation_expired"
)

// Terminal states. needs_attention and failed are terminal for the worker but
// still block new tasks until resolved locally; see the design's failure rules.
func StateTerminal(s string) bool {
	switch s {
	case StateSucceeded, StateFailed, StateNeedsAttention, StateConfirmationExpired:
		return true
	}
	return false
}

// Well-known blocking reasons for Capability.Reasons. Hosts may add their own
// stable snake_case identifiers; the frontend renders them via i18n.
const (
	ReasonHostUnsupported    = "host_unsupported"
	ReasonCheckDisabled      = "update_check_disabled"
	ReasonServiceIdentity    = "service_identity_unverifiable"
	ReasonInstallUnsupported = "install_unsupported"
	ReasonTaskActive         = "upgrade_task_active"
	ReasonTaskPending        = "upgrade_task_needs_attention"
)

var (
	registerMu sync.RWMutex
	registered Host
)

// Sentinel errors hosts must return so admin can map them to stable,
// machine-readable error IDs instead of leaking host internals. Wrap them
// with %w when adding context.
var (
	// ErrTaskConflict: another task blocks this one (active or unresolved).
	ErrTaskConflict = errors.New("upgrade task conflict")
	// ErrTargetChanged: the confirmed target no longer matches the candidate.
	ErrTargetChanged = errors.New("upgrade target changed")
	// ErrConfirmationExpired: the credential expired or was already used.
	ErrConfirmationExpired = errors.New("upgrade confirmation expired")
	// ErrInvalidCredential: the credential is malformed or not bound to the caller.
	ErrInvalidCredential = errors.New("invalid upgrade confirmation credential")
	// ErrCheckDisabled: online update checks are disabled; no new tasks.
	ErrCheckDisabled = errors.New("update checks are disabled")
	// ErrUnsupported: the installation does not support click upgrade.
	ErrUnsupported = errors.New("click upgrade is not supported here")
	// ErrNotAllowed: server-side conditions temporarily reject execution.
	ErrNotAllowed = errors.New("click upgrade is not allowed")
)

// Register installs the process-wide host implementation. Later calls
// overwrite earlier ones; passing nil clears it. Admin calls this once
// during embedding startup, mirroring util.OnConfigChanged.
func Register(h Host) {
	registerMu.Lock()
	registered = h
	registerMu.Unlock()
}

// Current returns the registered host or nil when admin runs standalone
// without an embedding host.
func Current() Host {
	registerMu.RLock()
	defer registerMu.RUnlock()
	return registered
}
