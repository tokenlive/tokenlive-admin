package biz

import (
	"testing"

	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
)

func TestCompositeDraftPreviewAndRolloutGuards(t *testing.T) {
	db, b, ctx := smartFixture(t)
	old := config.C.Gateway.CompositeRoutingV2
	config.C.Gateway.CompositeRoutingV2 = false
	t.Cleanup(func() { config.C.Gateway.CompositeRoutingV2 = old })
	f := smartForm()
	f.Enabled = 0
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	id := created.ID
	preview, err := b.PreviewCompositeRouting(ctx, id, &CompositePreviewForm{})
	require.NoError(t, err)
	require.Equal(t, "a", preview.Target)
	require.Equal(t, "primary", preview.Reason)
	require.False(t, preview.RuntimeAvailable)
	_, err = resolveSmartRuntime(ctx, db, created.Model)
	require.Error(t, err, "publication must also reject unsupported formats")
	require.Error(t, b.ToggleEnabled(ctx, id, &schema.ModelEnabledForm{Enabled: 1}))
	saved, err := b.ModelDAL.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, 0, saved.Enabled)
	require.Equal(t, int64(1), saved.SmartRouting.Version)
	require.Error(t, b.Delete(ctx, "a"), "draft references protect child identity")
	// Root still cannot preview credential headers or an unsupported inference mode.
	_, err = b.PreviewCompositeRouting(ctx, id, &CompositePreviewForm{Headers: map[string]string{"Authorization": "secret"}})
	require.Error(t, err)
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Update("enabled", 0).Error)
	preview, err = b.PreviewCompositeRouting(ctx, id, &CompositePreviewForm{})
	require.NoError(t, err)
	require.Equal(t, "disabled", preview.TargetStatus)
	saved, err = b.ModelDAL.Get(ctx, id)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.SmartRouting.Version, "preview cannot write configuration")
	require.NoError(t, b.Delete(ctx, id))
	require.NoError(t, b.Delete(ctx, "a"))
}

func TestCompositeRuntimeResolvesStableIDsAndRules(t *testing.T) {
	db, b, ctx := smartFixture(t)
	old := config.C.Gateway.CompositeRoutingV2
	config.C.Gateway.CompositeRoutingV2 = true
	t.Cleanup(func() { config.C.Gateway.CompositeRoutingV2 = old })
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Update("model_code", "model-a").Error)
	f := smartForm()
	f.Enabled = 0
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Rules, DefaultModelID: "a", Rules: []schema.SmartRule{{ID: "code", Order: 10, Enabled: true, RelationType: "AND", TargetModelID: "b", Conditions: []compositerouting.Condition{{Type: "TAG", Key: "task", OpType: "EQUAL", Values: []string{"code"}}}}}}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	runtime, err := resolveSmartRuntime(ctx, db, created.Model)
	require.NoError(t, err)
	require.Equal(t, "model-a", runtime.DefaultModel)
	require.Equal(t, "b", runtime.Rules[0].TargetModel)
	require.Equal(t, []string{"model-a", "b"}, runtime.ReferenceCodes())
	for _, in := range []compositerouting.Input{{}, {Tags: map[string]string{"task": "code"}}} {
		plan, err := compositerouting.Select(runtime.PlanConfig(), in)
		require.NoError(t, err)
		want := "model-a"
		if len(in.Tags) > 0 {
			want = "b"
		}
		require.Equal(t, want, plan.Target)
	}
	wire, err := json.Marshal(runtime)
	require.NoError(t, err)
	require.NotContains(t, string(wire), "model_id")
	require.NoError(t, b.Delete(ctx, created.ID))
}

func TestCompositePreviewDoesNotExposeHiddenTargets(t *testing.T) {
	db, b, ctx := smartFixture(t)
	f := smartForm()
	f.Enabled = 0
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("data_id = ?", "a").Update("permission", 0).Error)
	preview, err := b.PreviewCompositeRouting(ctx, created.ID, &CompositePreviewForm{})
	require.NoError(t, err)
	require.Empty(t, preview.Target)
	require.Equal(t, "unavailable_or_not_authorized", preview.TargetStatus)
	require.NoError(t, db.Model(&schema.DataPermission{}).Where("data_id = ?", created.ID).Update("permission", 0).Error)
	_, err = b.PreviewCompositeRouting(ctx, created.ID, &CompositePreviewForm{})
	require.Error(t, err)
}
