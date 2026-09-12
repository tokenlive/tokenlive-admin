package systemversion

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/updatecheck"
	"github.com/tokenlive/tokenlive-admin/internal/versionstatus"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/tokenlive/tokenlive-admin/pkg/versionregistry"
)

func TestProviderDefaultsAndConstructionDoesNotFetch(t *testing.T) {
	isolatedConfig(t)
	if got := config.C.UpdateCheck; !got.Enabled || got.IntervalSeconds != 21600 || got.TimeoutSeconds != 5 || got.CooldownSeconds != 60 {
		t.Fatalf("wrong update defaults: %+v", got)
	}
	options, err := checkerOptions(config.C.UpdateCheck)
	if err != nil || options.Interval != 6*time.Hour {
		t.Fatalf("omitted configuration must retain its six-hour interval: %+v, %v", options, err)
	}
	var requests atomic.Int32
	previous := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		requests.Add(1)
		return nil, errors.New("test forbids HTTP")
	})}
	t.Cleanup(func() { http.DefaultClient = previous })
	service, cleanup, err := ProvideService(context.Background(), nil)
	if err != nil || service == nil || cleanup == nil {
		t.Fatalf("provider: service=%v cleanup=%v err=%v", service, cleanup != nil, err)
	}
	t.Cleanup(cleanup)
	summary, err := service.Summary(context.Background(), false)
	if err != nil || summary.Identity != testIdentity("professional", "release") || summary.Gateway.Scope != "this_admin" {
		t.Fatalf("bad local summary: %+v, %v", summary, err)
	}
	updates, err := service.Updates(context.Background())
	if err != nil || !updates.Enabled || len(updates.Components) != 2 ||
		updates.Components[0].Source.Status != "unchecked" || updates.Components[1].Source.Status != "unchecked" {
		t.Fatalf("provider did not leave checks unstarted: %+v, %v", updates, err)
	}
	cleanup()
	cleanup()
	if _, err := service.Check(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wire cleanup did not close checker: %v", err)
	}
	if requests.Load() != 0 {
		t.Fatalf("construction/cleanup fetched external sources: %d", requests.Load())
	}
}

func TestProviderDisabledEnvironmentOverrideKeepsReportService(t *testing.T) {
	isolatedConfig(t)
	config.C.Gateway.VersionNamespace = "config-ns"
	t.Setenv("UPDATE_CHECK_ENABLED", "false")
	t.Setenv("UPDATE_CHECK_INTERVAL_SECONDS", "42")
	t.Setenv("GATEWAY_VERSION_NAMESPACE", "env-ns")
	service, cleanup, err := ProvideService(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	service.Start()
	node := providerNode("env-ns")
	if err := service.Report(context.Background(), node); err != nil {
		t.Fatalf("environment namespace did not override config: %v", err)
	}
	// The isolated in-memory report owns a TTL and disappears with this service.
	node.Namespace = "config-ns"
	if err := service.Report(context.Background(), node); err == nil {
		t.Fatal("configured namespace overrode explicit environment")
	}
	updates, err := service.Check(context.Background())
	if !errors.Is(err, updatecheck.ErrDisabled) || updates.Enabled {
		t.Fatalf("environment did not disable checks: %+v, %v", updates, err)
	}
	if len(updates.Components) != 2 || updates.Components[1].Current != "v1.0.0" ||
		updates.Components[1].State != "disabled" || updates.Components[1].Count != 1 {
		t.Fatalf("disabled service lost internal reporting: %+v", updates)
	}
	if !config.C.UpdateCheck.Enabled || config.C.Gateway.VersionNamespace != "config-ns" {
		t.Fatal("provider mutated shared configuration instead of resolving a private copy")
	}
}

func TestProviderResolvesEnvironmentAndConfigDurations(t *testing.T) {
	isolatedConfig(t)
	config.C.UpdateCheck.Enabled = false
	config.C.UpdateCheck.IntervalSeconds = 19
	config.C.UpdateCheck.TimeoutSeconds = 7
	config.C.UpdateCheck.CooldownSeconds = 11
	options, err := checkerOptions(config.C.UpdateCheck)
	if err != nil || options.Enabled || options.Interval != 19*time.Second ||
		options.Timeout != 7*time.Second || options.Cooldown != 11*time.Second {
		t.Fatalf("bad configured checker options: %+v, %v", options, err)
	}
	t.Setenv("UPDATE_CHECK_ENABLED", "true")
	t.Setenv("UPDATE_CHECK_INTERVAL_SECONDS", "42")
	options, err = checkerOptions(config.C.UpdateCheck)
	if err != nil || !options.Enabled || options.Interval != 42*time.Second ||
		options.Timeout != 7*time.Second || options.Cooldown != 11*time.Second {
		t.Fatalf("bad resolved checker options: %+v, %v", options, err)
	}
}

func TestProviderRejectsInvalidCheckConfigurationWithoutStarting(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		value  string
		change func()
	}{
		{name: "bad enable flag", env: "UPDATE_CHECK_ENABLED", value: "sometimes"},
		{name: "bad interval", env: "UPDATE_CHECK_INTERVAL_SECONDS", value: "tomorrow"},
		{name: "negative env interval", env: "UPDATE_CHECK_INTERVAL_SECONDS", value: "-1"},
		{name: "zero env interval", env: "UPDATE_CHECK_INTERVAL_SECONDS", value: "0"},
		{name: "zero config interval", change: func() { config.C.UpdateCheck.IntervalSeconds = 0 }},
		{name: "overflow interval", env: "UPDATE_CHECK_INTERVAL_SECONDS", value: "9223372036854775807"},
		{name: "negative config timeout", change: func() { config.C.UpdateCheck.TimeoutSeconds = -1 }},
		{name: "negative config cooldown", change: func() { config.C.UpdateCheck.CooldownSeconds = -1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolatedConfig(t)
			if tc.env != "" {
				t.Setenv(tc.env, tc.value)
			}
			if tc.change != nil {
				tc.change()
			}
			service, cleanup, err := ProvideService(context.Background(), nil)
			if cleanup != nil {
				cleanup()
			}
			if err == nil || service != nil {
				t.Fatalf("invalid check config was accepted: service=%v, err=%v", service, err)
			}
		})
	}
}

func TestProviderBorrowsRedisAndUsesNamespace(t *testing.T) {
	isolatedConfig(t)
	t.Setenv("UPDATE_CHECK_ENABLED", "false")
	t.Setenv("GATEWAY_VERSION_NAMESPACE", "scoped")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	service, cleanup, err := ProvideService(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	node := providerNode("scoped")
	if err := service.Report(context.Background(), node); err != nil {
		t.Fatal(err)
	}
	store := versionregistry.NewRedisStore(client, "scoped")
	t.Cleanup(func() { _ = store.Delete(context.Background(), node.InstanceID) })
	nodes, err := store.List(context.Background())
	if err != nil || len(nodes) != 1 || nodes[0] != node {
		t.Fatalf("provider did not use shared Redis namespace: %+v, %v", nodes, err)
	}
	summary, err := service.Summary(context.Background(), false)
	if err != nil || summary.Gateway.Scope != "shared" || summary.Gateway.Status != "observed" {
		t.Fatalf("Redis scope not represented: %+v, %v", summary, err)
	}
	cleanup()
	cleanup()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("cleanup closed borrowed Redis client: %v", err)
	}
	if err := store.Delete(context.Background(), node.InstanceID); err != nil {
		t.Fatal(err)
	}
	nodes, err = store.List(context.Background())
	if err != nil || len(nodes) != 0 {
		t.Fatalf("test report was not removable: %+v, %v", nodes, err)
	}
}

func TestProviderInvalidNamespaceDoesNotTouchRedis(t *testing.T) {
	isolatedConfig(t)
	t.Setenv("UPDATE_CHECK_ENABLED", "false")
	t.Setenv("GATEWAY_VERSION_NAMESPACE", "invalid:namespace")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	service, cleanup, err := ProvideService(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	if err := service.Report(context.Background(), providerNode("default")); err == nil {
		t.Fatal("invalid namespace accepted an internal report")
	}
	summary, err := service.Summary(context.Background(), false)
	if err != nil || summary.Gateway.Status != "unavailable" || summary.Identity != testIdentity("professional", "release") {
		t.Fatalf("invalid namespace lost safe local summary: %+v, %v", summary, err)
	}
	if server.CommandCount() != 0 {
		t.Fatalf("invalid namespace issued %d Redis commands", server.CommandCount())
	}
}

func TestProviderStandaloneHidesSharedGatewayReports(t *testing.T) {
	isolatedConfig(t)
	config.C.RuntimeIdentity = testIdentity("standalone", "homebrew")
	t.Setenv("UPDATE_CHECK_ENABLED", "false")
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	service, cleanup, err := ProvideService(context.Background(), client)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	summary, err := service.Summary(context.Background(), true)
	if err != nil || summary.Gateway.Status != "not_applicable" || len(summary.Gateway.Groups) != 0 {
		t.Fatalf("standalone exposed Gateway distribution: %+v, %v", summary, err)
	}
	updates, err := service.Updates(context.Background())
	if err != nil || len(updates.Components) != 1 || updates.Components[0].Component != "standalone" {
		t.Fatalf("standalone components: %+v, %v", updates, err)
	}
	cleanup()
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("standalone cleanup closed borrowed Redis: %v", err)
	}
}

func TestSourcesFollowEditionAndConfirmedInstallChannel(t *testing.T) {
	for _, tc := range []struct {
		name     string
		identity productversion.Identity
		keys     []string
	}{
		{"professional", testIdentity("professional", "release"), []string{"admin", "gateway"}},
		{"homebrew", testIdentity("standalone", "homebrew"), []string{"standalone"}},
		{"unknown edition", testIdentity("unknown", "release"), []string{}},
		{"unknown professional channel", testIdentity("professional", "unknown"), []string{}},
		{"unknown standalone channel", testIdentity("standalone", "unknown"), []string{}},
		{"standalone release is not confirmed Homebrew", testIdentity("standalone", "release"), []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				paths = append(paths, request.URL.Path)
				if request.URL.Scheme != "https" || request.URL.Host != "api.github.com" {
					t.Fatalf("unexpected source URL: %s", request.URL)
				}
				body := `{"tag_name":"v2.0.0","draft":false,"prerelease":false}`
				if strings.Contains(request.URL.Path, "/contents/Formula/") {
					body = `{"type":"file","encoding":"base64","content":"` +
						base64.StdEncoding.EncodeToString([]byte("version \"2.0.0\"\n")) + `"}`
				}
				return &http.Response{
					StatusCode: http.StatusOK, Header: make(http.Header),
					Body: io.NopCloser(strings.NewReader(body)),
				}, nil
			})}
			sources := sourcesForIdentity(tc.identity, client)
			keys := make([]string, 0, len(sources))
			for key := range sources {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			if !reflect.DeepEqual(keys, tc.keys) || len(paths) != 0 {
				t.Fatalf("wrong sources or eager HTTP: keys=%v requests=%v", keys, paths)
			}
			for _, key := range keys {
				candidate, err := sources[key].Latest(context.Background())
				if err != nil || candidate.Version != "v2.0.0" {
					t.Fatalf("selected %s source cannot consume its channel: %+v, %v", key, candidate, err)
				}
			}
			wantPaths := map[string]string{
				"admin":      "/repos/tokenlive/tokenlive-admin/releases/latest",
				"gateway":    "/repos/tokenlive/tokenlive-gateway/releases/latest",
				"standalone": "/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb",
			}
			if len(paths) != len(keys) {
				t.Fatalf("selected sources made %d requests for %d keys", len(paths), len(keys))
			}
			for i, key := range keys {
				if paths[i] != wantPaths[key] {
					t.Fatalf("%s uses wrong publishing channel: %s", key, paths[i])
				}
			}
		})
	}
}

func TestSystemVersionLifecycle(t *testing.T) {
	entered := make(chan struct{})
	exited := make(chan struct{})
	checker, err := updatecheck.NewChecker(context.Background(), updatecheck.Options{
		Enabled: true, Timeout: time.Minute,
	}, map[string]updatecheck.Source{"admin": controlledSource(func(ctx context.Context) (updatecheck.Candidate, error) {
		close(entered)
		<-ctx.Done()
		close(exited)
		return updatecheck.Candidate{}, ctx.Err()
	})})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(checker.Close)
	module := &SystemVersion{Service: versionstatus.New(
		testIdentity("professional", "release"), versionregistry.NewMemoryStore("", nil), checker, "this_admin",
	)}
	if err := module.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("module initialization did not start asynchronous checking")
	}
	if err := module.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := module.Release(context.Background()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
	default:
		t.Fatal("module release returned before checker shutdown")
	}
}

func isolatedConfig(t *testing.T) {
	t.Helper()
	previous := config.C
	t.Cleanup(func() { config.C = previous })
	config.C = new(config.Config)
	// A regression in disabled/lifecycle handling must still never contact a
	// real release server. Tests needing responses supply a controlled client.
	previousHTTP := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("external HTTP is disabled in provider tests")
	})}
	t.Cleanup(func() { http.DefaultClient = previousHTTP })
	for _, name := range []string{"UPDATE_CHECK_ENABLED", "UPDATE_CHECK_INTERVAL_SECONDS", "GATEWAY_VERSION_NAMESPACE"} {
		t.Setenv(name, "")
		if err := os.Unsetenv(name); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.Load(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	config.C.RuntimeIdentity = testIdentity("professional", "release")
}

func testIdentity(edition, channel string) productversion.Identity {
	return productversion.Identity{
		Edition: edition, InstallChannel: channel,
		Build: productversion.Build{Version: "v1.0.0", Kind: "release"},
	}
}

func providerNode(namespace string) versionregistry.Node {
	return versionregistry.Node{
		SchemaVersion: 1, Namespace: namespace, InstanceID: "be6d62c6-bc98-459f-b2fe-4eb5f2a338e6",
		Version: "v1.0.0", BuildKind: "release",
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

type controlledSource func(context.Context) (updatecheck.Candidate, error)

func (f controlledSource) Latest(ctx context.Context) (updatecheck.Candidate, error) { return f(ctx) }
