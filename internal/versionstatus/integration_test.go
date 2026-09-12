package versionstatus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

type releaseTransport struct {
	target    string
	transport http.RoundTripper
}

func (r releaseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	target, err := http.NewRequestWithContext(req.Context(), req.Method, r.target+req.URL.Path, nil)
	if err != nil {
		return nil, err
	}
	clone.URL = target.URL
	return r.transport.RoundTrip(clone)
}

// This catches cached node comparisons surviving TTL expiry, and view reads
// accidentally fetching sources. The parser, checker, registry and service are
// real; only GitHub's network destination is replaced with a local server.
func TestReleaseSnapshotRecomparesLiveMemoryRegistry(t *testing.T) {
	ctx := context.Background()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/repos/tokenlive/tokenlive-gateway/releases/latest" {
			t.Errorf("unexpected release path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`))
	}))
	defer server.Close()
	client := &http.Client{Transport: releaseTransport{server.URL, server.Client().Transport}}
	checker, err := updatecheck.NewChecker(ctx, updatecheck.Options{Enabled: true}, map[string]updatecheck.Source{
		"gateway": updatecheck.NewGitHubSource(client, "gateway"),
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC)
	store := versionregistry.NewMemoryStore("integration", func() time.Time { return now })
	service := versionstatus.New(productversion.Identity{
		Edition: "professional", InstallChannel: "release",
		Build: productversion.Build{Version: "v1.2.4", Kind: "release"},
	}, store, checker, "this_admin")
	defer service.Close()
	if _, err := service.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("initial check made %d source calls, want 1", calls.Load())
	}
	for i, version := range []string{"v1.2.3", "v1.2.3", "v1.2.4"} {
		id := []string{
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
			"33333333-3333-4333-8333-333333333333",
		}[i]
		t.Cleanup(func() {
			if err := store.Delete(ctx, id); err != nil {
				t.Error(err)
			}
		})
		if err := service.Report(ctx, versionregistry.Node{
			SchemaVersion: 1, Namespace: "integration", InstanceID: id, Version: version, BuildKind: "release",
		}); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := service.Summary(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Gateway.Status != "observed" || summary.Gateway.Scope != "this_admin" ||
		len(summary.Gateway.Groups) != 2 || summary.Gateway.Groups[0].Version != "v1.2.3" ||
		summary.Gateway.Groups[0].Count != 2 || summary.Gateway.Groups[1].Version != "v1.2.4" ||
		summary.Gateway.Groups[1].Count != 1 {
		t.Fatalf("incorrect active groups: %+v", summary.Gateway)
	}
	updates, err := service.Updates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(updates.Components) != 3 ||
		updates.Components[1].State != "available" || updates.Components[1].Count != 2 ||
		updates.Components[2].State != "current" || updates.Components[2].Count != 1 {
		t.Fatalf("incorrect per-group comparison: %+v", updates)
	}
	if !strings.HasSuffix(updates.Components[1].Source.Candidate.ReleaseURL, "/releases/tag/v1.2.4") {
		t.Fatalf("release adapter was not used: %+v", updates.Components[1].Source)
	}
	now = now.Add(3 * time.Minute)
	summary, err = service.Summary(ctx, false)
	if err != nil || summary.Gateway.Status != "unknown" || len(summary.Gateway.Groups) != 0 {
		t.Fatalf("expired registry = %+v, error = %v", summary.Gateway, err)
	}
	updates, err = service.Updates(ctx)
	if err != nil || len(updates.Components) != 2 || updates.Components[1].State != "unknown" ||
		updates.Components[1].Current != "unknown" || updates.Components[1].Latest != "" {
		t.Fatalf("expired update notice = %+v, error = %v", updates, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("local reads fetched the source again: %d calls", calls.Load())
	}
}
