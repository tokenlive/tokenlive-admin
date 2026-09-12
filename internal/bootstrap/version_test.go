package bootstrap_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokenlive/tokenlive-admin/cmd"
	"github.com/tokenlive/tokenlive-admin/internal/bootstrap"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/json"
	"github.com/tokenlive/tokenlive-admin/pkg/encoding/toml"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func TestBootstrapRuntimeIdentity(t *testing.T) {
	cases := []struct {
		name     string
		version  string
		identity *productversion.Identity
		want     productversion.Identity
	}{
		{
			name: "explicit standalone wins over legacy and configured versions", version: "v9.9.9",
			identity: &productversion.Identity{
				Edition: "standalone", InstallChannel: "homebrew",
				Build: productversion.Build{Version: "v1.2.3", Kind: "release"},
			},
			want: productversion.Identity{
				Edition: "standalone", InstallChannel: "homebrew",
				Build: productversion.Build{Version: "v1.2.3", Kind: "release"},
			},
		},
		{
			name: "legacy version remains dev", version: "v1.2.3",
			want: productversion.Identity{
				Edition: "professional", InstallChannel: "release",
				Build: productversion.Build{Version: "v1.2.3", Kind: "dev"},
			},
		},
		{
			name: "configured version is not runtime identity",
			want: productversion.Identity{
				Edition: "professional", InstallChannel: "release",
				Build: productversion.Build{Version: "dev", Kind: "dev"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := identityConfig(t)
			rt, err := bootstrap.Init(context.Background(), bootstrap.RunConfig{
				WorkDir: dir, Configs: "test.json", Version: tc.version, Identity: tc.identity,
			})
			if rt != nil {
				rt.Release(context.Background())
				t.Fatal("startup unexpectedly reached runtime creation")
			}
			assertIdentityStartup(t, err, tc.want)
		})
	}
}

func TestStartCmdRuntimeIdentity(t *testing.T) {
	cases := []struct {
		name    string
		args    []string
		version string
		kind    string
	}{
		{"no arguments are dev", nil, "dev", "dev"},
		{"legacy version is dev", []string{"v1.2.3"}, "v1.2.3", "dev"},
		{"explicit release", []string{"v1.2.3", "release"}, "v1.2.3", "release"},
		{"unknown kind is dev", []string{"v1.2.3", "nightly"}, "v1.2.3", "dev"},
		{"empty kind is dev", []string{"v1.2.3", ""}, "v1.2.3", "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := identityConfig(t)
			app := &cli.App{Commands: []*cli.Command{cmd.StartCmd(tc.args...)}}
			err := app.Run([]string{"admin-test", "start", "--workdir", dir, "--config", "test.json"})
			assertIdentityStartup(t, err, productversion.Identity{
				Edition: "professional", InstallChannel: "release",
				Build: productversion.Build{Version: tc.version, Kind: tc.kind},
			})
		})
	}
}

func TestConfigRuntimeIdentityExcluded(t *testing.T) {
	cases := []struct {
		name   string
		decode func([]byte, interface{}) error
		data   string
	}{
		{
			"json", json.Unmarshal,
			`{"RuntimeIdentity":{"edition":"standalone","install_channel":"homebrew","build":{"version":"v99.0.0","kind":"release"}}}`,
		},
		{
			"toml", toml.Unmarshal,
			"[RuntimeIdentity]\nEdition = 'standalone'\nInstallChannel = 'homebrew'\n[RuntimeIdentity.Build]\nVersion = 'v99.0.0'\nKind = 'release'\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var cfg config.Config
			if err := tc.decode([]byte(tc.data), &cfg); err != nil {
				t.Fatal(err)
			}
			if cfg.RuntimeIdentity != (productversion.Identity{}) {
				t.Fatalf("configuration supplied a runtime identity: %+v", cfg.RuntimeIdentity)
			}
		})
	}
	cfg := config.Config{RuntimeIdentity: productversion.Identity{
		Edition: "standalone", InstallChannel: "homebrew",
		Build: productversion.Build{Version: "runtime-only-version", Kind: "release"},
	}}
	for name, encode := range map[string]func(interface{}) ([]byte, error){
		"json": json.Marshal, "toml": toml.Marshal,
	} {
		data, err := encode(&cfg)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "runtime-only-version") {
			t.Errorf("%s included the runtime-only identity", name)
		}
	}
}

// Stop in DB type validation, before opening storage or starting background
// modules/external checks. Always use fresh temporary config, never configs/dev.
func identityConfig(t *testing.T) string {
	t.Helper()
	previous := config.C
	logger := zap.L()
	t.Cleanup(func() {
		config.C = previous
		zap.ReplaceGlobals(logger)
	})
	config.C = new(config.Config)
	dir := t.TempDir()
	data := []byte(`{
		"General": {"Version": "v99.0.0", "DisablePrintConfig": true},
		"Logger": {"Level": "fatal"},
		"Storage": {"DB": {"Type": "identity-test-no-database"}},
		"Util": {"Captcha": {"Disable": true}, "Prometheus": {"Enable": false}}
	}`)
	if err := os.WriteFile(filepath.Join(dir, "test.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	// MustLoad is process-wide; explicitly reload our isolated config for each case.
	if err := config.Load(dir, "test.json"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func assertIdentityStartup(t *testing.T, err error, want productversion.Identity) {
	t.Helper()
	if err == nil || err.Error() != "unsupported database type: identity-test-no-database" {
		t.Fatalf("expected isolated startup to stop at DB validation, got %v", err)
	}
	if got := config.C.RuntimeIdentity; got != want {
		t.Errorf("runtime identity = %+v, want %+v", got, want)
	}
	if got := config.C.General.Version; got != want.Build.Version {
		t.Errorf("public version = %q, want %q", got, want.Build.Version)
	}
	if want.Build.Kind == "dev" && productversion.Compare(config.C.RuntimeIdentity.Build, "v99.0.0") != "uncomparable" {
		t.Error("legacy/development startup was treated as a comparable release")
	}
}
