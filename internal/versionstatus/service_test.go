package versionstatus

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

func TestComposeEmptyRegistryIsUnknownNotAvailable(t *testing.T) {
	result := Compose(professionalIdentity(), nil, updatecheck.CheckResult{
		Enabled: true, Sources: map[string]updatecheck.SourceState{
			"gateway": {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"}},
		},
	})
	gateway := onlyComponent(t, result, "gateway")
	if gateway.Current != "unknown" || gateway.State != "unknown" || gateway.Count != 0 || gateway.Latest != "" {
		t.Fatalf("empty registry pretended to have a comparable Gateway: %+v", gateway)
	}
	disabled := onlyComponent(t, Compose(professionalIdentity(), nil, updatecheck.CheckResult{}), "gateway")
	if disabled.State != "disabled" {
		t.Fatalf("disabled must win: %+v", disabled)
	}
}

func TestComposeDoesNotTrustStaleCandidate(t *testing.T) {
	identity := professionalIdentity()
	state := updatecheck.CheckResult{Enabled: true, Sources: map[string]updatecheck.SourceState{
		"admin": {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"}, Stale: true},
	}}
	got := Compose(identity, nil, state)
	admin := onlyComponent(t, got, "admin")
	if admin.State != "stale" || !admin.Source.Stale || admin.Latest != "" {
		t.Fatalf("stale candidate became actionable: %+v", admin)
	}

	state.Sources["admin"] = updatecheck.SourceState{
		Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"},
	}
	fresh := onlyComponent(t, Compose(identity, nil, state), "admin")
	if fresh.State != "available" || fresh.Latest != "v2.0.0" || fresh.Source.Stale {
		t.Fatalf("fresh candidate was not compared: %+v", fresh)
	}
}

func TestComposePreservesSourceStatesBeforeComparison(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		status  string
		stale   bool
		want    string
	}{
		{"disabled globally", false, "ready", false, "disabled"},
		{"disabled source", true, "disabled", true, "disabled"},
		{"failed with history", true, "unavailable", true, "unavailable"},
		{"no candidate with history", true, "no_candidate", true, "no_candidate"},
		{"checking fresh history", true, "checking", false, "checking"},
		{"never checked", true, "unchecked", false, "unchecked"},
		{"expired candidate", true, "ready", true, "stale"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Compose(professionalIdentity(), nil, updatecheck.CheckResult{
				Enabled: tc.enabled, RetryAfterSeconds: 17,
				Sources: map[string]updatecheck.SourceState{
					"admin": {Status: tc.status, Stale: tc.stale, Candidate: &updatecheck.Candidate{Version: "v2.0.0"}},
				},
			})
			if result.Enabled != tc.enabled || result.RetryAfterSeconds != 17 {
				t.Fatalf("lost checker metadata: %+v", result)
			}
			admin := onlyComponent(t, result, "admin")
			if admin.State != tc.want || admin.Latest != "" || admin.Source.Status != tc.status {
				t.Fatalf("source state was replaced by comparison: %+v", admin)
			}
		})
	}
}

func TestComposeComparesActiveGatewayGroupsAndKeepsSourceFailuresIndependent(t *testing.T) {
	result := Compose(professionalIdentity(), []versionregistry.Group{
		{Version: "v1.0.0", BuildKind: "release", Count: 2},
		{Version: "v3.0.0", BuildKind: "release", Count: 1},
		{Version: "dev", BuildKind: "dev", Count: 1},
	}, updatecheck.CheckResult{Enabled: true, Sources: map[string]updatecheck.SourceState{
		"admin":   {Status: "unavailable", ErrorCode: "check_failed"},
		"gateway": {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"}},
	}})
	if len(result.Components) != 4 {
		t.Fatalf("expected admin plus three active groups, got %+v", result.Components)
	}
	if got := onlyComponent(t, result, "admin"); got.State != "unavailable" {
		t.Fatalf("admin failure lost: %+v", got)
	}
	for i, want := range []struct {
		current string
		state   string
		count   int
	}{{"v1.0.0", "available", 2}, {"v3.0.0", "ahead", 1}, {"dev", "uncomparable", 1}} {
		got := result.Components[i+1]
		if got.Component != "gateway" || got.Current != want.current || got.State != want.state || got.Count != want.count {
			t.Fatalf("gateway group %d = %+v, want %+v", i, got, want)
		}
	}
}

func TestComposeStandaloneHasOnlyOneUnit(t *testing.T) {
	identity := productversion.Identity{
		Edition: "standalone", InstallChannel: "homebrew",
		Build: productversion.Build{Version: "v1.0.0", Kind: "release"},
	}
	got := Compose(identity, []versionregistry.Group{{Version: "v0.1.0", BuildKind: "release", Count: 3}},
		updatecheck.CheckResult{Enabled: true, Sources: map[string]updatecheck.SourceState{
			"standalone": {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"}},
			"admin":      {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v3.0.0"}},
			"gateway":    {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v4.0.0"}},
		}})
	if len(got.Components) != 1 || got.Components[0].Component != "standalone" ||
		got.Components[0].Current != "v1.0.0" || got.Components[0].Latest != "v2.0.0" ||
		got.Components[0].State != "available" {
		t.Fatalf("standalone was split or compared against another source: %+v", got)
	}
}

func TestComposeDevelopmentIdentityAndMissingSourceAreNotComparable(t *testing.T) {
	identity := professionalIdentity()
	identity.Build.Kind = "dev"
	result := Compose(identity, nil, updatecheck.CheckResult{Enabled: true, Sources: map[string]updatecheck.SourceState{
		"admin": {Status: "ready", Candidate: &updatecheck.Candidate{Version: "v2.0.0"}},
	}})
	if got := onlyComponent(t, result, "admin"); got.State != "uncomparable" {
		t.Fatalf("development identity became an update target: %+v", got)
	}
	missing := onlyComponent(t, Compose(identity, nil, updatecheck.CheckResult{Enabled: true}), "admin")
	if missing.State != "unavailable" || missing.Source.Status != "unavailable" || missing.Latest != "" {
		t.Fatalf("missing source invented an update: %+v", missing)
	}
}

func TestServiceExpiresGatewayNoticesWithoutFetchingAgain(t *testing.T) {
	now := time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)
	store := versionregistry.NewMemoryStore("test", func() time.Time { return now })
	var calls atomic.Int32
	source := sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
		calls.Add(1)
		return updatecheck.Candidate{Version: "v2.0.0"}, nil
	})
	checker := testChecker(t, updatecheck.Options{Enabled: true}, map[string]updatecheck.Source{
		"admin": source, "gateway": source,
	})
	service := New(professionalIdentity(), store, checker, "this_admin")
	node := testNode()
	if err := service.Report(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Delete(context.Background(), node.InstanceID); err != nil {
			t.Error(err)
		}
	})
	first, err := service.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gateway := onlyComponent(t, first, "gateway"); gateway.State != "available" || gateway.Count != 1 {
		t.Fatalf("active Gateway not compared: %+v", gateway)
	}
	summary, err := service.Summary(context.Background(), true)
	if err != nil || summary.Gateway.Status != "observed" || len(summary.Gateway.Groups) != 1 ||
		summary.Gateway.Scope != "this_admin" || !summary.CanManageUpdates {
		t.Fatalf("bad active summary: %+v, %v", summary, err)
	}
	now = now.Add(versionregistry.TTL)
	for range 2 {
		got, err := service.Updates(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		gateway := onlyComponent(t, got, "gateway")
		if gateway.State != "unknown" || gateway.Current != "unknown" || gateway.Count != 0 || gateway.Latest != "" {
			t.Fatalf("expired Gateway still produces an alert: %+v", gateway)
		}
		if admin := onlyComponent(t, got, "admin"); admin.State != "available" {
			t.Fatalf("Gateway expiry erased admin result: %+v", admin)
		}
	}
	summary, err = service.Summary(context.Background(), false)
	if err != nil || summary.Gateway.Status != "unknown" || len(summary.Gateway.Groups) != 0 || summary.CanManageUpdates {
		t.Fatalf("expired summary: %+v, %v", summary, err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("read-only aggregation fetched sources again: %d calls", got)
	}
}

func TestServiceRegistryFailurePreservesLocalDataWithoutLeakingDiagnostics(t *testing.T) {
	const diagnostic = "Redis at private.example:6379 key instance-12345 is corrupt"
	store := failingStore{err: errors.New(diagnostic)}
	checker := testChecker(t, updatecheck.Options{Enabled: true}, map[string]updatecheck.Source{
		"admin": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			return updatecheck.Candidate{Version: "v2.0.0"}, nil
		}),
		"gateway": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			return updatecheck.Candidate{Version: "v3.0.0"}, nil
		}),
	})
	service := New(professionalIdentity(), store, checker, "shared")
	summary, err := service.Summary(context.Background(), false)
	if err != nil || summary.Identity != professionalIdentity() || summary.Gateway.Status != "unavailable" ||
		summary.Gateway.Scope != "shared" || summary.Gateway.Groups == nil || len(summary.Gateway.Groups) != 0 {
		t.Fatalf("registry outage erased local identity or pretended success: %+v, %v", summary, err)
	}
	updates, err := service.Check(context.Background())
	if err != nil {
		t.Fatalf("registry failure discarded completed checks: %v", err)
	}
	if admin := onlyComponent(t, updates, "admin"); admin.State != "available" || admin.Latest != "v2.0.0" {
		t.Fatalf("registry failure erased admin result: %+v", admin)
	}
	gateway := onlyComponent(t, updates, "gateway")
	if gateway.Current != "unknown" || gateway.State != "unavailable" || gateway.Count != 0 || gateway.Latest != "" {
		t.Fatalf("registry failure became an empty or comparable registry: %+v", gateway)
	}
	read, err := service.Updates(context.Background())
	if err != nil || onlyComponent(t, read, "gateway").State != "unavailable" {
		t.Fatalf("read lost registry failure: %+v, %v", read, err)
	}
	for _, value := range []any{summary, updates, read} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		for _, private := range []string{diagnostic, "private.example", "instance-12345"} {
			if strings.Contains(string(data), private) {
				t.Fatalf("private registry diagnostics leaked into DTO: %s", data)
			}
		}
	}
}

func TestServiceDisabledChecksStillAllowLocalReports(t *testing.T) {
	var calls atomic.Int32
	checker := testChecker(t, updatecheck.Options{Enabled: false}, map[string]updatecheck.Source{
		"admin": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			calls.Add(1)
			return updatecheck.Candidate{Version: "v2.0.0"}, nil
		}),
	})
	store := versionregistry.NewMemoryStore("test", nil)
	service := New(professionalIdentity(), store, checker, "this_admin")
	service.Start()
	node := testNode()
	if err := service.Report(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), node.InstanceID) })
	result, err := service.Check(context.Background())
	if !errors.Is(err, updatecheck.ErrDisabled) || result.Enabled || calls.Load() != 0 {
		t.Fatalf("disabled check performed work: %+v, %v, calls=%d", result, err, calls.Load())
	}
	if gateway := onlyComponent(t, result, "gateway"); gateway.Current != "v1.0.0" || gateway.Count != 1 || gateway.State != "disabled" {
		t.Fatalf("disabled check hid local report: %+v", gateway)
	}
	service.Close()
	service.Close()
}

func TestServiceDisabledWinsRegistryFailure(t *testing.T) {
	checker := testChecker(t, updatecheck.Options{}, nil)
	service := New(professionalIdentity(), failingStore{err: errors.New("registry offline")}, checker, "shared")
	updates, err := service.Updates(context.Background())
	if err != nil || updates.Enabled {
		t.Fatalf("disabled partial updates: %+v, %v", updates, err)
	}
	gateway := onlyComponent(t, updates, "gateway")
	if gateway.State != "disabled" || gateway.Current != "unknown" {
		t.Fatalf("registry failure overrode disabled checks: %+v", gateway)
	}
}

func TestServiceOneSourceFailureDoesNotHideAnotherResult(t *testing.T) {
	store := versionregistry.NewMemoryStore("test", nil)
	node := testNode()
	if err := store.Upsert(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Delete(context.Background(), node.InstanceID) })
	checker := testChecker(t, updatecheck.Options{Enabled: true}, map[string]updatecheck.Source{
		"admin": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			return updatecheck.Candidate{Version: "v2.0.0"}, nil
		}),
		"gateway": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			return updatecheck.Candidate{}, errors.New("private upstream failure")
		}),
	})
	service := New(professionalIdentity(), store, checker, "this_admin")
	result, err := service.Check(context.Background())
	if err != nil {
		t.Fatalf("one source failure discarded all results: %v", err)
	}
	if got := onlyComponent(t, result, "admin"); got.State != "available" || got.Latest != "v2.0.0" {
		t.Fatalf("healthy source lost: %+v", got)
	}
	if got := onlyComponent(t, result, "gateway"); got.State != "unavailable" || got.Source.ErrorCode != "check_failed" {
		t.Fatalf("failed source was not independently represented: %+v", got)
	}
}

func TestServiceStandaloneNeverReadsGatewayDistribution(t *testing.T) {
	identity := productversion.Identity{
		Edition: "standalone", InstallChannel: "homebrew",
		Build: productversion.Build{Version: "v1.0.0", Kind: "release"},
	}
	checker := testChecker(t, updatecheck.Options{Enabled: false}, nil)
	service := New(identity, failingStore{err: errors.New("distribution must not be read")}, checker, "this_admin")
	summary, err := service.Summary(context.Background(), true)
	if err != nil || summary.Gateway.Status != "not_applicable" || len(summary.Gateway.Groups) != 0 || summary.Identity != identity {
		t.Fatalf("standalone exposed Gateway distribution: %+v, %v", summary, err)
	}
	got, err := service.Updates(context.Background())
	if err != nil || len(got.Components) != 1 || got.Components[0].Component != "standalone" {
		t.Fatalf("standalone updates read Gateway registry: %+v, %v", got, err)
	}
}

func TestServiceCancellationIsNotSuccessfulPartialData(t *testing.T) {
	checker := testChecker(t, updatecheck.Options{Enabled: true}, nil)
	service := New(professionalIdentity(), versionregistry.NewMemoryStore("test", nil), checker, "this_admin")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Summary(ctx, false); !errors.Is(err, context.Canceled) {
		t.Fatalf("summary swallowed cancellation: %v", err)
	}
	if _, err := service.Updates(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("updates swallowed cancellation: %v", err)
	}
	if _, err := service.Check(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("manual check swallowed cancellation: %v", err)
	}
}

func TestServiceLifecycleStartsAsynchronouslyAndCloseCancelsWork(t *testing.T) {
	entered := make(chan struct{})
	exited := make(chan struct{})
	var calls atomic.Int32
	checker := testChecker(t, updatecheck.Options{Enabled: true, Timeout: time.Minute}, map[string]updatecheck.Source{
		"admin": sourceFunc(func(ctx context.Context) (updatecheck.Candidate, error) {
			calls.Add(1)
			close(entered)
			<-ctx.Done()
			close(exited)
			return updatecheck.Candidate{}, ctx.Err()
		}),
	})
	service := New(professionalIdentity(), versionregistry.NewMemoryStore("test", nil), checker, "this_admin")
	if calls.Load() != 0 {
		t.Fatal("constructor fetched a source")
	}
	service.Start()
	service.Start()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("start did not launch a check asynchronously")
	}
	service.Close()
	service.Close()
	select {
	case <-exited:
	default:
		t.Fatal("close returned without waiting for the source")
	}
	service.Start()
	if _, err := service.Check(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("closed service accepted a manual check: %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("duplicate lifecycle work: %d", calls.Load())
	}
}

func TestServiceManualCheckUsesSharedCooldown(t *testing.T) {
	checker := testChecker(t, updatecheck.Options{Enabled: true}, map[string]updatecheck.Source{
		"admin": sourceFunc(func(context.Context) (updatecheck.Candidate, error) {
			return updatecheck.Candidate{Version: "v2.0.0"}, nil
		}),
	})
	service := New(professionalIdentity(), versionregistry.NewMemoryStore("test", nil), checker, "this_admin")
	if _, err := service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := service.Check(context.Background())
	if !errors.Is(err, updatecheck.ErrCooldown) || got.RetryAfterSeconds < 1 {
		t.Fatalf("manual check bypassed cooldown: %+v, %v", got, err)
	}
}

type sourceFunc func(context.Context) (updatecheck.Candidate, error)

func (f sourceFunc) Latest(ctx context.Context) (updatecheck.Candidate, error) { return f(ctx) }

func testChecker(t *testing.T, options updatecheck.Options, sources map[string]updatecheck.Source) *updatecheck.Checker {
	t.Helper()
	checker, err := updatecheck.NewChecker(context.Background(), options, sources)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(checker.Close)
	return checker
}

type failingStore struct{ err error }

func (s failingStore) Upsert(context.Context, versionregistry.Node) error { return s.err }
func (s failingStore) List(context.Context) ([]versionregistry.Node, error) {
	// Even partial data from a failed read must not drive a comparison.
	return []versionregistry.Node{testNode()}, s.err
}
func (s failingStore) Delete(context.Context, string) error { return s.err }

func testNode() versionregistry.Node {
	return versionregistry.Node{
		SchemaVersion: 1, Namespace: "test", InstanceID: "3b5680f1-8e6c-4d96-bb6f-c3c53a62dfda",
		Version: "v1.0.0", BuildKind: "release",
	}
}

func professionalIdentity() productversion.Identity {
	return productversion.Identity{
		Edition: "professional", InstallChannel: "release",
		Build: productversion.Build{Version: "v1.0.0", Kind: "release"},
	}
}

func onlyComponent(t *testing.T, updates Updates, name string) ComponentView {
	t.Helper()
	var got ComponentView
	count := 0
	for _, component := range updates.Components {
		if component.Component == name {
			count++
			got = component
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one %s component, got %d in %+v", name, count, updates.Components)
	}
	return got
}
