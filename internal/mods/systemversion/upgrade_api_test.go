package systemversion

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/tokenlive/tokenlive-admin/pkg/upgradehost"
)

// fakeUpgradeHost records calls so tests can assert permission boundaries and
// error mapping without any local upgrade machinery.
type fakeUpgradeHost struct {
	capErr     error
	capability upgradehost.Capability
	prepErr    error
	prep       upgradehost.Preparation
	prepReqs   []upgradehost.PrepareRequest
	submitErr  error
	task       upgradehost.TaskView
	submitReqs []upgradehost.SubmitRequest
}

func (f *fakeUpgradeHost) Capability(context.Context) (upgradehost.Capability, error) {
	return f.capability, f.capErr
}

func (f *fakeUpgradeHost) Prepare(_ context.Context, req upgradehost.PrepareRequest) (upgradehost.Preparation, error) {
	f.prepReqs = append(f.prepReqs, req)
	return f.prep, f.prepErr
}

func (f *fakeUpgradeHost) Submit(_ context.Context, req upgradehost.SubmitRequest) (upgradehost.TaskView, error) {
	f.submitReqs = append(f.submitReqs, req)
	return f.task, f.submitErr
}

func (f *fakeUpgradeHost) Task(context.Context, upgradehost.TaskRequest) (upgradehost.TaskView, error) {
	return f.task, nil
}

func TestUpgradeAPIPermissionAndHostBoundaries(t *testing.T) {
	// The harness disables Casbin, so only root holds either capability.
	t.Run("no host reports unsupported", func(t *testing.T) {
		t.Cleanup(func() { upgradehost.Register(nil) })
		_, _, e := newAPITest(t, true, nil)

		rec := apiRequest(e, "GET", "/api/v1/system/upgrade/capability", "root-token", "", "")
		if rec.Code != 200 {
			t.Fatalf("capability: %d %s", rec.Code, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), upgradehost.ReasonHostUnsupported) {
			t.Fatalf("capability body missing unsupported reason: %s", rec.Body)
		}
		for _, tc := range []struct{ method, path, body string }{
			{"POST", "/api/v1/system/upgrade/prepare", `{"target_version":"2.0.0"}`},
			{"POST", "/api/v1/system/upgrade/submit", `{"task_id":"t","credential":"c","confirm":true}`},
			{"GET", "/api/v1/system/upgrade/task", ""},
		} {
			rec := apiRequest(e, tc.method, tc.path, "root-token", "", tc.body)
			if rec.Code != 409 || !strings.Contains(rec.Body.String(), upgradehost.ReasonHostUnsupported) {
				t.Fatalf("%s %s without host: %d %s", tc.method, tc.path, rec.Code, rec.Body)
			}
		}
	})

	t.Run("viewer is forbidden from every upgrade route", func(t *testing.T) {
		t.Cleanup(func() { upgradehost.Register(nil) })
		upgradehost.Register(&fakeUpgradeHost{})
		_, _, e := newAPITest(t, true, nil)
		for _, tc := range []struct{ method, path, body string }{
			{"GET", "/api/v1/system/upgrade/capability", ""},
			{"POST", "/api/v1/system/upgrade/prepare", `{"target_version":"2.0.0"}`},
			{"POST", "/api/v1/system/upgrade/submit", `{"task_id":"t","credential":"c","confirm":true}`},
			{"GET", "/api/v1/system/upgrade/task", ""},
		} {
			rec := apiRequest(e, tc.method, tc.path, "viewer-token", "", tc.body)
			if rec.Code != 403 {
				t.Fatalf("%s %s as viewer: %d %s", tc.method, tc.path, rec.Code, rec.Body)
			}
		}
	})

	t.Run("root flows carry initiator and map sentinel errors", func(t *testing.T) {
		t.Cleanup(func() { upgradehost.Register(nil) })
		host := &fakeUpgradeHost{
			capability: upgradehost.Capability{Supported: true, Allowed: true},
			prep: upgradehost.Preparation{
				TaskID: "task-1", TargetVersion: "2.0.0", Credential: "cred",
				ConfirmExpiresIn: 300, RestartWarning: true,
			},
			task: upgradehost.TaskView{TaskID: "task-1", State: upgradehost.StateQueued},
		}
		upgradehost.Register(host)
		_, _, e := newAPITest(t, true, nil)

		rec := apiRequest(e, "GET", "/api/v1/system/upgrade/capability", "root-token", "", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"supported":true`) {
			t.Fatalf("capability: %d %s", rec.Code, rec.Body)
		}

		rec = apiRequest(e, "POST", "/api/v1/system/upgrade/prepare", "root-token", "", `{"target_version":"2.0.0"}`)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"task-1"`) {
			t.Fatalf("prepare: %d %s", rec.Code, rec.Body)
		}
		if len(host.prepReqs) != 1 || host.prepReqs[0].Initiator != "root" || host.prepReqs[0].TargetVersion != "2.0.0" {
			t.Fatalf("prepare request not bound to root/target: %+v", host.prepReqs)
		}

		rec = apiRequest(e, "POST", "/api/v1/system/upgrade/submit", "root-token", "",
			`{"task_id":"task-1","credential":"cred","confirm":true}`)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), upgradehost.StateQueued) {
			t.Fatalf("submit: %d %s", rec.Code, rec.Body)
		}

		rec = apiRequest(e, "GET", "/api/v1/system/upgrade/task", "root-token", "", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"task-1"`) {
			t.Fatalf("task: %d %s", rec.Code, rec.Body)
		}

		for sentinel, wantID := range map[error]string{
			upgradehost.ErrTaskConflict:        "upgrade_task_conflict",
			upgradehost.ErrTargetChanged:       "upgrade_target_changed",
			upgradehost.ErrConfirmationExpired: "upgrade_confirmation_expired",
			upgradehost.ErrInvalidCredential:   "upgrade_invalid_credential",
			upgradehost.ErrCheckDisabled:       "update_check_disabled",
			upgradehost.ErrUnsupported:         "host_unsupported",
			upgradehost.ErrNotAllowed:          "upgrade_not_allowed",
		} {
			host.prepErr = fmt.Errorf("wrapped: %w", sentinel)
			rec := apiRequest(e, "POST", "/api/v1/system/upgrade/prepare", "root-token", "", `{"target_version":"2.0.0"}`)
			if rec.Code != 409 || !strings.Contains(rec.Body.String(), wantID) {
				t.Fatalf("sentinel %v: %d %s, want 409 with %s", sentinel, rec.Code, rec.Body, wantID)
			}
		}
		host.prepErr = errors.New("unexpected internal failure")
		rec = apiRequest(e, "POST", "/api/v1/system/upgrade/prepare", "root-token", "", `{"target_version":"2.0.0"}`)
		if rec.Code != 500 {
			t.Fatalf("unknown host error: %d %s", rec.Code, rec.Body)
		}
	})

	t.Run("summary exposes separate upgrade capability", func(t *testing.T) {
		t.Cleanup(func() { upgradehost.Register(nil) })
		_, _, e := newAPITest(t, true, nil)
		rec := apiRequest(e, "GET", "/api/v1/current/version", "root-token", "", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"can_manage_upgrades":true`) {
			t.Fatalf("summary for root: %d %s", rec.Code, rec.Body)
		}
		rec = apiRequest(e, "GET", "/api/v1/current/version", "viewer-token", "", "")
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"can_manage_upgrades":false`) {
			t.Fatalf("summary for viewer: %d %s", rec.Code, rec.Body)
		}
	})
}

func TestUpgradeBlockedWhileChecksDisabled(t *testing.T) {
	t.Cleanup(func() { upgradehost.Register(nil) })
	// newAPITest(false, ...) constructs a disabled checker: prepare and submit
	// must be rejected without ever reaching the host.
	host := &fakeUpgradeHost{capability: upgradehost.Capability{Supported: true, Allowed: true}}
	upgradehost.Register(host)
	_, _, e := newAPITest(t, false, nil)
	for _, tc := range []struct {
		method, path, body string
	}{
		{"POST", "/api/v1/system/upgrade/prepare", `{"target_version":"2.0.0"}`},
		{"POST", "/api/v1/system/upgrade/submit", `{"task_id":"t","credential":"c","confirm":true}`},
	} {
		rec := apiRequest(e, tc.method, tc.path, "root-token", "", tc.body)
		if rec.Code != 409 || !strings.Contains(rec.Body.String(), "update_check_disabled") {
			t.Fatalf("%s %s with checks disabled: %d %s", tc.method, tc.path, rec.Code, rec.Body)
		}
	}
	if len(host.prepReqs) != 0 || len(host.submitReqs) != 0 {
		t.Fatal("disabled gate must stop requests before the host")
	}
	// Capability stays readable in disabled mode: honest reasons, no crash,
	// and allowed must drop to false with the stable reason attached.
	rec := apiRequest(e, "GET", "/api/v1/system/upgrade/capability", "root-token", "", "")
	if rec.Code != 200 {
		t.Fatalf("capability with checks disabled: %d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(rec.Body.String(), `"allowed":false`) ||
		!strings.Contains(rec.Body.String(), "update_check_disabled") {
		t.Fatalf("capability must report disabled state: %s", rec.Body)
	}
}
