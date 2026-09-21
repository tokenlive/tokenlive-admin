package biz

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func smartFixture(t *testing.T) (*gorm.DB, *Model, context.Context) {
	t.Helper()
	db := newModelDeleteTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { ClearGatewayConfigCache(); _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&schema.Provider{}))
	b := newModelDeleteTestBiz(db)
	ctx := newModelDeleteTestContext()
	for _, id := range []string{"a", "b"} {
		require.NoError(t, db.Create(&schema.Model{ID: id, ModelCode: id, ModelName: id, SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}).Error)
		require.NoError(t, b.DataPermissionBIZ.CreateByOwner(ctx, schema.DataPermissionTypeModel, id, "tenant-a"))
	}
	return db, b, ctx
}

func smartForm() *schema.ModelForm {
	return &schema.ModelForm{
		ModelCode: "smart", ModelName: "Smart", ModelType: "smart", SpaceCode: "default", Enabled: 1,
		RequestTypes: `["chat_completion"]`, Abilities: `[]`,
		SmartRouting: &schema.SmartRouting{
			JudgeModelID: "a",
			Ranges:       []schema.SmartRange{{Min: 0, Max: 40, ModelID: "a"}, {Min: 40, Max: 100, ModelID: "b"}},
		},
	}
}

func createSmartFixture(t *testing.T, b *Model, ctx context.Context) *schema.Model {
	t.Helper()
	result, err := b.Create(ctx, smartForm())
	require.NoError(t, err)
	return result.Model
}

func TestSmartCreateRejectsInvalidReferencesWithoutSaving(t *testing.T) {
	// Removing any reference guard would persist an unauthorized or unusable graph.
	for _, tc := range []struct {
		name string
		edit func(*testing.T, *gorm.DB, *schema.ModelForm)
	}{
		{"missing", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) { f.SmartRouting.JudgeModelID = "absent" }},
		{"disabled", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) {
			require.NoError(t, db.Model(&schema.Model{}).Where("id = 'a'").Update("enabled", 0).Error)
		}},
		{"cross space", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) { f.SpaceCode = "other" }},
		{"unauthorized", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) {
			require.NoError(t, db.Model(&schema.DataPermission{}).Where("data_id = 'a'").Update("permission", 0).Error)
		}},
		{"wrong protocol", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) {
			require.NoError(t, db.Model(&schema.Model{}).Where("id = 'a'").Update("request_types", `["embedding"]`).Error)
		}},
		{"nested", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) {
			require.NoError(t, db.Model(&schema.Model{}).Where("id = 'a'").Update("model_type", "smart").Error)
		}},
		{"invalid partition", func(t *testing.T, db *gorm.DB, f *schema.ModelForm) { f.SmartRouting.Ranges[1].Min = 41 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, b, ctx := smartFixture(t)
			f := smartForm()
			tc.edit(t, db, f)
			_, err := b.Create(ctx, f)
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&schema.Model{}).Where("model_code = 'smart'").Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestSmartVersionRejectsStaleWriteAndKeepsStableReferences(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	require.Equal(t, int64(1), model.SmartRouting.Version)
	f := smartForm()
	f.SmartRouting.Version = 1
	f.SmartRouting.Ranges[0].Max, f.SmartRouting.Ranges[1].Min = 60, 60
	require.NoError(t, b.Update(ctx, model.ID, f))
	stale := smartForm()
	stale.SmartRouting.Version = 1
	require.Error(t, b.Update(ctx, model.ID, stale))
	got, err := b.ModelDAL.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), got.SmartRouting.Version)
	require.Equal(t, 60, got.SmartRouting.Ranges[0].Max)
	require.Equal(t, "a", got.SmartRouting.JudgeModelID)
	self := smartForm()
	self.SmartRouting.Version = 2
	self.SmartRouting.JudgeModelID = model.ID
	require.Error(t, b.Update(ctx, model.ID, self))
}

func TestSmartReferencedModelDeletionAndConversionBlocked(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	require.Error(t, b.Delete(ctx, "a"))
	f := smartForm()
	f.ModelCode, f.ModelName = "a", "a"
	require.Error(t, b.Update(ctx, "a", f))
	require.NoError(t, b.Delete(ctx, model.ID))
	child, err := b.ModelDAL.Get(ctx, "a")
	require.NoError(t, err)
	require.NotNil(t, child, "deleting a composite must not delete its children")
	require.NoError(t, b.Delete(ctx, "a"))
}

func TestSmartRejectsDirectEndpointsAndEndpointConversion(t *testing.T) {
	db, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	ep := newEndpointQueryTestBiz(db)
	ep.ModelDAL = &dal.Model{DB: db}
	ep.ProviderDAL = &dal.Provider{DB: db}
	_, err := ep.Create(ctx, &schema.EndpointForm{Code: "direct", ModelID: model.ID, ProviderID: "provider", URL: "https://example.test", AuthType: "api_key"})
	require.Error(t, err)
	require.NoError(t, db.Create(&schema.Endpoint{ID: "existing", Code: "existing", ModelID: "a", ProviderID: "provider", URL: "https://example.test"}).Error)
	require.Error(t, ep.Update(ctx, "existing", &schema.EndpointForm{Code: "existing", ModelID: model.ID, ProviderID: "provider", URL: "https://example.test", AuthType: "api_key"}))
	require.NoError(t, b.Delete(ctx, model.ID))
	f := smartForm()
	f.ModelCode, f.ModelName = "a", "a"
	f.SmartRouting.JudgeModelID = "b"
	require.Error(t, b.Update(ctx, "a", f))
}

func TestSmartDisableWarnsAuthorizedCallers(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	got, err := b.ModelDAL.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.SmartRouting.Version)
	require.Equal(t, "a", got.SmartRouting.JudgeModelID)
	require.ErrorContains(t, b.Update(ctx, "a", &schema.ModelForm{ModelCode: "renamed", ModelName: "a", SpaceCode: "default", Enabled: 1, RequestTypes: `["chat_completion"]`}), "模型编码创建后不可修改")
	require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 0}))
	detail, err := b.Get(ctx, "a")
	require.NoError(t, err)
	raw, err := json.Marshal(detail)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))
	require.NotEmpty(t, wire["referenced_by"], "disable impact must be visible to authorized users")
	hidden, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.NotNil(t, hidden)
}

func TestSmartMutationsRequireDataPermission(t *testing.T) {
	for _, operation := range []string{"update", "disable", "delete", "sync"} {
		t.Run(operation, func(t *testing.T) {
			_, b, ctx := smartFixture(t)
			model := createSmartFixture(t, b, ctx)
			other := util.NewUsername(ctx, "unauthorized-user")
			for _, id := range []string{"a", "b"} {
				require.NoError(t, b.DataPermissionBIZ.CreateByOwner(other, schema.DataPermissionTypeModel, id, "tenant-a"))
			}
			var err error
			switch operation {
			case "update":
				form := smartForm()
				form.SmartRouting.Version = 1
				err = b.Update(other, model.ID, form)
			case "disable":
				err = b.ToggleEnabled(other, model.ID, &schema.ModelEnabledForm{Enabled: 0})
			case "delete":
				err = b.Delete(other, model.ID)
			case "sync":
				err = b.Sync(other, model.ID)
			}
			require.Error(t, err)
			got, getErr := b.ModelDAL.Get(ctx, model.ID)
			require.NoError(t, getErr)
			require.NotNil(t, got)
			require.Equal(t, 1, got.Enabled)
		})
	}
}

func TestSmartConversionRequiresCurrentVersion(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	update := smartForm()
	update.SmartRouting.Version = 1
	require.NoError(t, b.Update(ctx, model.ID, update))
	var form schema.ModelForm
	require.NoError(t, json.Unmarshal([]byte(`{"model_code":"smart","model_name":"Smart","model_type":"normal","request_types":"[\"chat_completion\"]","smart_routing":null,"smart_routing_version":1}`), &form))
	require.Error(t, b.Update(ctx, model.ID, &form), "stale conversion must not erase a newer routing configuration")
	got, err := b.ModelDAL.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, "smart", got.ModelType)
}

func TestSmartConversionRequiresWritePermissionOnOriginalModel(t *testing.T) {
	db, b, ctx := smartFixture(t)
	require.NoError(t, db.Create(&schema.Model{ID: "unowned", ModelCode: "unowned", ModelName: "Unowned", SpaceCode: "default"}).Error)
	form := smartForm()
	form.ModelCode, form.ModelName = "unowned", "Unowned"
	require.Error(t, b.Update(ctx, "unowned", form))
	got, err := b.ModelDAL.Get(ctx, "unowned")
	require.NoError(t, err)
	require.Equal(t, "normal", got.ModelType)
}

func TestSmartSelectorFiltersAndHidesUnauthorizedReferenceDetails(t *testing.T) {
	db, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	enabled := 1
	result, err := b.Query(ctx, schema.ModelQueryParam{ModelType: "normal", SpaceCode: "default", Enabled: &enabled})
	require.NoError(t, err)
	require.Len(t, result.Data, 2)
	for _, normal := range result.Data {
		require.Equal(t, "normal", normal.ModelType)
		require.Len(t, normal.ReferencedBy, 1)
		require.Equal(t, model.ID, normal.ReferencedBy[0].ID)
	}
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("data_id = ?", model.ID).Update("permission", 0).Error)
	child, err := b.Get(ctx, "a")
	require.NoError(t, err)
	require.Empty(t, child.ReferencedBy, "hidden composites must not leak through reference metadata")
}
