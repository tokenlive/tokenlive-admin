package wirex

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Provider wiring injects EndpointDAL and updates bound endpoints atomically by
// stable key ID, so swap and overlapping replacements never clobber one another.
func TestBuildInjectorPreservesProviderEndpointKeysForOverlappingUpdates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		newVals  [2]string
		wantEpA  string
		wantEpB  string
	}{
		{"swap", [2]string{"fixture-b", "fixture-a"}, "fixture-b", "fixture-a"},
		{"overlap", [2]string{"fixture-b", "fixture-c"}, "fixture-b", "fixture-c"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			injector := providerWiringInjector(t)
			db := injector.DB
			if err := db.AutoMigrate(&schema.Provider{}, &schema.Endpoint{}, &schema.DataPermission{}, &opsSchema.AuditLog{}); err != nil {
				t.Fatal(err)
			}
			ctx := util.NewTenant(util.NewUsername(context.Background(), "fixture-user"), "fixture-tenant")
			provider := injector.M.Resource.ProviderAPI.ProviderBIZ
			if provider.EndpointDAL == nil {
				t.Fatal("expected EndpointDAL to be wired into ProviderBIZ")
			}
			form := &schema.ProviderForm{
				Code: "fixture-provider", Name: "Fixture Provider", Protocol: "fixture",
				ApiKeys: []schema.ApiKeyItem{{Value: "fixture-a"}, {Value: "fixture-b"}},
			}
			created, err := provider.Create(ctx, form)
			if err != nil {
				t.Fatal(err)
			}
			createdKeys := created.GetApiKeys()
			if len(createdKeys) != 2 || createdKeys[0].ID == "" || createdKeys[1].ID == "" {
				t.Fatalf("expected created provider keys to have IDs, got %+v", createdKeys)
			}
			endpoints := []schema.Endpoint{
				{ID: "fixture-ep-a", Code: "fixture-ep-a", ProviderID: created.ID, ModelID: "fixture-model", ProviderKeyID: createdKeys[0].ID, ApiKey: "fixture-a", Enabled: 1, Deleted: "0"},
				{ID: "fixture-ep-b", Code: "fixture-ep-b", ProviderID: created.ID, ModelID: "fixture-model", ProviderKeyID: createdKeys[1].ID, ApiKey: "fixture-b", Enabled: 1, Deleted: "0"},
				{ID: "fixture-inherit", Code: "fixture-inherit", ProviderID: created.ID, ModelID: "fixture-model", Enabled: 1, Deleted: "0"},
				{ID: "fixture-other", Code: "fixture-other", ProviderID: "other-provider", ModelID: "fixture-model", ApiKey: "fixture-a", Enabled: 1, Deleted: "0"},
			}
			t.Cleanup(func() {
				if err := db.Unscoped().Where("id IN ?", []string{"fixture-ep-a", "fixture-ep-b", "fixture-inherit", "fixture-other"}).
					Delete(&schema.Endpoint{}).Error; err != nil {
					t.Error(err)
				}
				if err := db.Unscoped().Where("data_id = ?", created.ID).Delete(&schema.DataPermission{}).Error; err != nil {
					t.Error(err)
				}
				if err := db.Unscoped().Where("resource_id = ? OR resource_id IN ?", created.ID, []string{"fixture-ep-a", "fixture-ep-b"}).Delete(&opsSchema.AuditLog{}).Error; err != nil {
					t.Error(err)
				}
				if err := db.Unscoped().Where("id = ?", created.ID).Delete(&schema.Provider{}).Error; err != nil {
					t.Error(err)
				}
			})
			for i := range endpoints {
				if err := db.Create(&endpoints[i]).Error; err != nil {
					t.Fatal(err)
				}
			}

			form.ApiKeys = []schema.ApiKeyItem{
				{ID: createdKeys[0].ID, Value: tc.newVals[0]},
				{ID: createdKeys[1].ID, Value: tc.newVals[1]},
			}
			if err := provider.Update(ctx, created.ID, form); err != nil {
				t.Fatal(err)
			}
			updated, err := provider.Get(ctx, created.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got := updated.GetApiKeys(); !reflect.DeepEqual(got, form.ApiKeys) {
				t.Fatalf("provider update itself did not persist: got %+v, want %+v", got, form.ApiKeys)
			}
			var actual []schema.Endpoint
			if err := db.Order("id").Find(&actual).Error; err != nil {
				t.Fatal(err)
			}
			gotKeys := make(map[string]string, len(actual))
			for _, endpoint := range actual {
				gotKeys[endpoint.ID] = endpoint.ApiKey
			}
			wantKeys := map[string]string{
				"fixture-ep-a": tc.wantEpA, "fixture-ep-b": tc.wantEpB,
				"fixture-inherit": "", "fixture-other": "fixture-a",
			}
			if !reflect.DeepEqual(gotKeys, wantKeys) {
				t.Errorf("endpoint cascade mismatch: got %v, want %v", gotKeys, wantKeys)
			}
			var providerAuditCount int64
			if err := db.Model(&opsSchema.AuditLog{}).Where("resource_id = ?", created.ID).Count(&providerAuditCount).Error; err != nil {
				t.Fatal(err)
			}
			if providerAuditCount != 2 {
				t.Errorf("provider create/update audit dependency lost: got %d records, want 2", providerAuditCount)
			}
		})
	}
}

func providerWiringInjector(t *testing.T) *Injector {
	t.Helper()
	previousConfig := config.C
	t.Cleanup(func() { config.C = previousConfig })
	config.C = new(config.Config)
	if err := config.Load(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	config.C.Storage.DB.Type = "sqlite3"
	config.C.Storage.DB.DSN = filepath.Join(t.TempDir(), "provider-wiring.sqlite")
	config.C.Storage.DB.MaxOpenConns = 1
	config.C.Storage.Cache.Type = "memory"
	config.C.Storage.Cache.Redis.Addr = ""
	config.C.Middleware.Auth.Store.Type = "memory"
	t.Setenv("UPDATE_CHECK_ENABLED", "false")
	t.Setenv("UPDATE_CHECK_INTERVAL_SECONDS", "21600")
	t.Setenv("GATEWAY_VERSION_NAMESPACE", "provider-wiring-test")
	previousTransport := http.DefaultTransport
	http.DefaultTransport = providerWiringTransport{}
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	// Exercise the generated production construction path only. Mods.Init is
	// never called, so this fixture starts no application or external workers.
	injector, cleanup, err := BuildInjector(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	return injector
}

type providerWiringTransport struct{}

func (providerWiringTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("external HTTP is disabled in provider wiring tests")
}
