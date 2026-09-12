package adminapp_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tokenlive/tokenlive-admin/adminapp"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/pkg/productversion"
	"go.uber.org/zap"
)

func TestNewRuntimeIdentity(t *testing.T) {
	cases := []struct {
		name     string
		version  string
		identity *productversion.Identity
		want     productversion.Identity
	}{
		{
			name: "standalone host identity takes priority", version: "v9.9.9",
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
			name: "legacy embed is not a release", version: "v1.2.3",
			want: productversion.Identity{
				Edition: "professional", InstallChannel: "release",
				Build: productversion.Build{Version: "v1.2.3", Kind: "dev"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			previous := config.C
			logger := zap.L()
			t.Cleanup(func() {
				config.C = previous
				zap.ReplaceGlobals(logger)
			})
			config.C = new(config.Config)
			dir := t.TempDir()
			// Fail before opening storage or starting background modules/external
			// checks, while exercising the real New -> bootstrap.Init identity chain.
			data := []byte(`{
				"General": {"Version": "v99.0.0", "DisablePrintConfig": true},
				"Logger": {"Level": "fatal"},
				"Storage": {"DB": {"Type": "identity-test-no-database"}},
				"Util": {"Captcha": {"Disable": true}, "Prometheus": {"Enable": false}}
			}`)
			if err := os.WriteFile(filepath.Join(dir, "test.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			// MustLoad is process-wide, so reload only this isolated fixture per case.
			if err := config.Load(dir, "test.json"); err != nil {
				t.Fatal(err)
			}
			app, err := adminapp.New(context.Background(), adminapp.Options{
				WorkDir: dir, Configs: "test.json", Version: tc.version, Identity: tc.identity,
			})
			if app != nil {
				_ = app.Shutdown(context.Background())
				t.Fatal("startup unexpectedly reached runtime creation")
			}
			if err == nil || err.Error() != "unsupported database type: identity-test-no-database" {
				t.Fatalf("expected isolated startup to stop at DB validation, got %v", err)
			}
			if got := config.C.RuntimeIdentity; got != tc.want {
				t.Errorf("runtime identity = %+v, want %+v", got, tc.want)
			}
			if got := config.C.General.Version; got != tc.want.Build.Version {
				t.Errorf("public version = %q, want %q", got, tc.want.Build.Version)
			}
			if tc.identity == nil && productversion.Compare(config.C.RuntimeIdentity.Build, "v1.2.4") != "uncomparable" {
				t.Error("legacy embed was treated as a comparable release")
			}
		})
	}
}
