package versionstatus

import (
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

// GatewayView exposes aggregate versions only, never node identities.
type GatewayView struct {
	Status string                  `json:"status"`
	Scope  string                  `json:"scope"`
	Groups []versionregistry.Group `json:"groups"`
}

// Summary is the local identity and currently observed Gateway distribution.
// Update-source details are deliberately absent from this view.
type Summary struct {
	Identity         productversion.Identity `json:"identity"`
	Gateway          GatewayView             `json:"gateway"`
	CanManageUpdates bool                    `json:"can_manage_updates"`
}

type ComponentView struct {
	Component string                  `json:"component"`
	Current   string                  `json:"current"`
	Latest    string                  `json:"latest,omitempty"`
	State     string                  `json:"state"`
	Count     int                     `json:"count,omitempty"`
	Source    updatecheck.SourceState `json:"source"`
}

type Updates struct {
	Enabled           bool            `json:"enabled"`
	Components        []ComponentView `json:"components"`
	RetryAfterSeconds int             `json:"retry_after_seconds"`
}

// Compose compares the supplied current builds with fresh, ready candidates.
// Gateway groups must represent active reports at the time of the read.
func Compose(identity productversion.Identity, groups []versionregistry.Group, state updatecheck.CheckResult) Updates {
	result := Updates{
		Enabled: state.Enabled, RetryAfterSeconds: state.RetryAfterSeconds,
		Components: make([]ComponentView, 0, len(groups)+1),
	}
	if identity.Edition == "standalone" {
		result.Components = append(result.Components, composeComponent("standalone", identity.Build, 0, state))
		return result
	}
	result.Components = append(result.Components, composeComponent("admin", identity.Build, 0, state))
	if len(groups) == 0 {
		gateway := composeComponent("gateway", productversion.Build{Version: "unknown", Kind: "dev"}, 0, state)
		gateway.Latest = ""
		if state.Enabled {
			gateway.State = "unknown"
		}
		result.Components = append(result.Components, gateway)
	}
	for _, group := range groups {
		result.Components = append(result.Components, composeComponent("gateway",
			productversion.Build{Version: group.Version, Kind: group.BuildKind}, group.Count, state))
	}
	return result
}

func composeComponent(name string, current productversion.Build, count int, result updatecheck.CheckResult) ComponentView {
	source := result.Sources[name]
	if source.Status == "" {
		source.Status = "unavailable"
	}
	if source.Candidate != nil {
		candidate := *source.Candidate
		source.Candidate = &candidate
	}
	view := ComponentView{
		Component: name, Current: current.Version, Count: count, Source: source,
		State: source.Status,
	}
	switch {
	case !result.Enabled:
		view.State = "disabled"
	case source.Status != "ready":
		// Failed, disabled and in-flight candidates remain non-actionable.
	case source.Stale:
		view.State = "stale"
	case source.Candidate == nil:
		view.State = "no_candidate"
	default:
		view.State = productversion.Compare(current, source.Candidate.Version)
		view.Latest = source.Candidate.Version
	}
	return view
}
