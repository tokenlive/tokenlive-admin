package biz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func draftSmartForm() *schema.ModelForm {
	form := smartForm()
	form.Enabled = 0
	form.SmartRouting = nil
	return form
}

func TestSmartDraftCreationAndEnableGuard(t *testing.T) {
	_, b, ctx := smartFixture(t)
	result, err := b.Create(ctx, draftSmartForm())
	require.NoError(t, err)
	require.Zero(t, result.Enabled)
	require.Nil(t, result.SmartRouting)

	require.Error(t, b.ToggleEnabled(ctx, result.ID, &schema.ModelEnabledForm{Enabled: 1}))
	enableThroughEdit := draftSmartForm()
	enableThroughEdit.Enabled = 1
	require.Error(t, b.Update(ctx, result.ID, enableThroughEdit))
	model, err := b.Get(ctx, result.ID)
	require.NoError(t, err)
	require.Zero(t, model.Enabled)
	raw, err := json.Marshal(model)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))
	require.Equal(t, false, wire["smart_routing_ready"])
}

func TestSmartDraftCannotBeCreatedEnabled(t *testing.T) {
	db, b, ctx := smartFixture(t)
	form := draftSmartForm()
	form.Enabled = 1
	_, err := b.Create(ctx, form)
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&schema.Model{}).Where("model_type = ?", "smart").Count(&count).Error)
	require.Zero(t, count)
}

func TestSmartBasicEditPreservesRoutingAndVersion(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model := createSmartFixture(t, b, ctx)
	form := smartForm()
	form.SmartRouting = nil
	form.Description = "basic information only"
	require.NoError(t, b.Update(ctx, model.ID, form))
	got, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, "basic information only", got.Description)
	require.Equal(t, int64(1), got.SmartRouting.Version)
	require.Equal(t, 40, got.SmartRouting.Ranges[0].Max)
	require.Equal(t, "a", got.SmartRouting.JudgeModelID)
}

func TestSmartBasicEditCanSaveOrDisableWithUnavailableDependency(t *testing.T) {
	for _, enabled := range []int{0, 1} {
		t.Run(map[int]string{0: "disable", 1: "metadata only"}[enabled], func(t *testing.T) {
			_, b, ctx := smartFixture(t)
			model := createSmartFixture(t, b, ctx)
			require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 0}))
			form := smartForm()
			form.SmartRouting = nil
			form.Enabled = enabled
			form.Description = "basic edit independent of unchanged routing"
			require.NoError(t, b.Update(ctx, model.ID, form))
			got, err := b.Get(ctx, model.ID)
			require.NoError(t, err)
			require.Equal(t, enabled, got.Enabled)
			require.Equal(t, form.Description, got.Description)
			require.Equal(t, int64(1), got.SmartRouting.Version)
			require.Equal(t, "a", got.SmartRouting.JudgeModelID)
		})
	}
}

func TestSmartDraftBasicEditAndConversion(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model, err := b.Create(ctx, draftSmartForm())
	require.NoError(t, err)
	form := draftSmartForm()
	form.Description = "draft description"
	require.NoError(t, b.Update(ctx, model.ID, form))
	zero := int64(0)
	form.ModelType = schema.ModelTypeNormal
	form.SmartRoutingVersion = &zero
	require.NoError(t, b.Update(ctx, model.ID, form))
	got, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, schema.ModelTypeNormal, got.ModelType)
	require.Nil(t, got.SmartRouting)
}

func TestSmartEnableRevalidatesDependenciesWithoutChangingVersion(t *testing.T) {
	_, b, ctx := smartFixture(t)
	form := smartForm()
	form.Enabled = 0
	model, err := b.Create(ctx, form)
	require.NoError(t, err)
	require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 0}))
	require.Error(t, b.ToggleEnabled(ctx, model.ID, &schema.ModelEnabledForm{Enabled: 1}))
	stillDisabled, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Zero(t, stillDisabled.Enabled)
	require.NoError(t, b.ToggleEnabled(ctx, "a", &schema.ModelEnabledForm{Enabled: 1}))
	require.NoError(t, b.ToggleEnabled(ctx, model.ID, &schema.ModelEnabledForm{Enabled: 1}))
	enabled, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, 1, enabled.Enabled)
	require.Equal(t, int64(1), enabled.SmartRouting.Version)
}

func TestSmartDraftConversionCannotBypassEndpointGuard(t *testing.T) {
	db, b, ctx := smartFixture(t)
	require.NoError(t, db.Create(&schema.Endpoint{
		ID: "direct-a", Code: "direct-a", ModelID: "a", ProviderID: "provider", URL: "http://example.test",
	}).Error)
	form := draftSmartForm()
	form.ModelName, form.ModelCode = "a", "a"
	require.Error(t, b.Update(ctx, "a", form))
}

func TestSmartDetailRoutingSavePreservesBasicFieldsAndEnabledState(t *testing.T) {
	db, b, ctx := smartFixture(t)
	b.ConfigRedisSync, _ = smartRedisFixture(t, db)
	seedSmartEndpoints(t, db)
	form := draftSmartForm()
	form.Description = "keep basic description"
	model, err := b.Create(ctx, form)
	require.NoError(t, err)
	routing := smartForm().SmartRouting
	result, err := b.UpdateSmartRouting(ctx, model.ID, routing)
	require.NoError(t, err)
	require.True(t, result.Saved)
	got, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Zero(t, got.Enabled, "saving configuration must not auto-enable")
	require.Equal(t, "keep basic description", got.Description)
	require.Equal(t, int64(1), got.SmartRouting.Version)
	require.True(t, got.SmartRoutingReady)
	_, err = b.UpdateSmartRouting(ctx, model.ID, smartForm().SmartRouting)
	require.Error(t, err, "stale first-save version must not replace configured routing")
	require.NoError(t, b.ToggleEnabled(ctx, model.ID, &schema.ModelEnabledForm{Enabled: 1}))
	next := got.SmartRouting.Clone()
	next.Ranges[0].Max, next.Ranges[1].Min = 70, 70
	_, err = b.UpdateSmartRouting(ctx, model.ID, next)
	require.NoError(t, err)
	updated, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Equal(t, 1, updated.Enabled)
	require.Equal(t, int64(2), updated.SmartRouting.Version)
	require.Equal(t, "keep basic description", updated.Description)
}

func TestSmartDetailRoutingSaveRequiresSmartTypeAndWritePermission(t *testing.T) {
	_, b, ctx := smartFixture(t)
	model, err := b.Create(ctx, draftSmartForm())
	require.NoError(t, err)
	_, err = b.UpdateSmartRouting(ctx, "a", smartForm().SmartRouting)
	require.Error(t, err, "ordinary model cannot accept a route")
	_, err = b.UpdateSmartRouting(util.NewUsername(ctx, "other"), model.ID, smartForm().SmartRouting)
	require.Error(t, err)
	invalid := smartForm().SmartRouting
	invalid.Ranges[1].Min = 99
	_, err = b.UpdateSmartRouting(ctx, model.ID, invalid)
	require.Error(t, err)
	got, err := b.Get(ctx, model.ID)
	require.NoError(t, err)
	require.Nil(t, got.SmartRouting)
}

func TestSmartDraftIsNotPublishedUntilExplicitEnable(t *testing.T) {
	db, b, ctx := smartFixture(t)
	sync, server := smartRedisFixture(t, db)
	b.ConfigRedisSync = sync
	seedSmartEndpoints(t, db)
	model, err := b.Create(ctx, draftSmartForm())
	require.NoError(t, err)
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	export := &GatewaySync{DB: db}
	config, err := export.GetGatewayConfig(ctx, "")
	require.NoError(t, err)
	require.NotContains(t, config.Models, "smart")
	_, err = b.UpdateSmartRouting(ctx, model.ID, smartForm().SmartRouting)
	require.NoError(t, err)
	require.False(t, server.Exists("aigw:config:smart_routing:smart"))
	require.NoError(t, b.ToggleEnabled(ctx, model.ID, &schema.ModelEnabledForm{Enabled: 1}))
	require.True(t, server.Exists("aigw:config:smart_routing:smart"))
	config, err = export.GetGatewayConfig(ctx, "")
	require.NoError(t, err)
	require.Contains(t, config.Models, "smart")
}
