package mods

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/dashboard"
	"github.com/tokenlive/tokenlive-admin/internal/mods/ops"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource"
	"github.com/tokenlive/tokenlive-admin/internal/mods/space"
	"github.com/tokenlive/tokenlive-admin/internal/mods/systemversion"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

func TestReleaseStopsSystemVersionChecker(t *testing.T) {
	entered := make(chan struct{})
	exited := make(chan struct{})
	checker, err := updatecheck.NewChecker(context.Background(), updatecheck.Options{
		Enabled: true, Timeout: time.Minute,
	}, map[string]updatecheck.Source{"admin": lifecycleSource{entered: entered, exited: exited}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(checker.Close)
	service := versionstatus.New(productversion.Identity{Edition: "professional"},
		versionregistry.NewMemoryStore("", nil), checker, "this_admin")
	modules := &Mods{
		RBAC:          &rbac.RBAC{Casbinx: &rbac.Casbinx{}},
		Resource:      &resource.Resource{},
		Space:         &space.Space{},
		Policy:        &policy.Policy{},
		Dashboard:     &dashboard.Dashboard{},
		Ops:           &ops.Ops{},
		SystemVersion: &systemversion.SystemVersion{Service: service},
	}
	service.Start()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("controlled source did not start")
	}
	if err := modules.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("module release left version-check work running")
	}
	if _, err := service.Check(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("released module still permits checks: %v", err)
	}
}

type lifecycleSource struct {
	entered chan struct{}
	exited  chan struct{}
}

func (s lifecycleSource) Latest(ctx context.Context) (updatecheck.Candidate, error) {
	close(s.entered)
	<-ctx.Done()
	close(s.exited)
	return updatecheck.Candidate{}, ctx.Err()
}
