package biz

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestCompositePromiseRejectsUnsupportedCapacityAndAbilities(t *testing.T) {
	for _, tc := range []struct {
		name            string
		context, output int
		abilities       string
	}{
		{"unknown context", -1, 100, `[]`},
		{"unknown output", 4096, -1, `[]`},
		{"too much context", 256000, 8192, `[]`},
		{"too much output", 128000, 16384, `[]`},
		{"unsupported tools", 128000, 8192, `["tool_call"]`},
		{"unsupported structured output", 128000, 8192, `["structured_output"]`},
		{"unknown stream support", 128000, 8192, `["stream"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, b, ctx := smartFixture(t)
			f := smartForm()
			f.Enabled = 0
			f.ContextLength, f.MaxOutputTokens, f.Abilities = tc.context, tc.output, tc.abilities
			f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
			_, err := b.Create(ctx, f)
			require.Error(t, err)
			var count int64
			require.NoError(t, db.Model(&schema.Model{}).Where("model_code = ?", "smart").Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestCompositePromiseCheckedOnPublicationAndBasicEdit(t *testing.T) {
	db, b, ctx := smartFixture(t)
	old := config.C.Gateway.CompositeRoutingV2
	config.C.Gateway.CompositeRoutingV2 = true
	t.Cleanup(func() { config.C.Gateway.CompositeRoutingV2 = old })
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Update("abilities", `["stream"]`).Error)
	f := smartForm()
	f.Abilities = `["stream"]`
	f.SmartRouting = &schema.SmartRouting{SchemaVersion: 2, Mode: compositerouting.Priority, PrimaryModelID: "a"}
	created, err := b.Create(ctx, f)
	require.NoError(t, err)
	f.SmartRouting = nil // Basic editing must not bypass the promise guard.
	f.MaxOutputTokens = 16000
	require.Error(t, b.Update(ctx, created.ID, f))
	fetched, err := b.ModelDAL.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, 8192, fetched.MaxOutputTokens)
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Update("max_output_tokens", 1024).Error)
	_, err = resolveSmartRuntime(ctx, db, fetched)
	require.Error(t, err, "publication must revalidate target capacity")
	require.NoError(t, b.Delete(ctx, created.ID))
}

func TestCompositePromiseIgnoresDisabledRuleTargets(t *testing.T) {
	model := &schema.Model{ContextLength: 16000, MaxOutputTokens: 4000, Abilities: `[]`, SmartRouting: &schema.SmartRouting{
		Mode: compositerouting.Rules, DefaultModelID: "small", Rules: []schema.SmartRule{{TargetModelID: "large", Enabled: false}},
	}}
	refs := []schema.Model{{ID: "small", ContextLength: 8000, MaxOutputTokens: 2000}, {ID: "large", ContextLength: 32000, MaxOutputTokens: 8000}}
	require.Error(t, validateCompositePromise(model, refs))
	model.SmartRouting.Rules[0].Enabled = true
	require.NoError(t, validateCompositePromise(model, refs))
}

func TestCompositeHTTPExportsResolvedCapabilities(t *testing.T) {
	db, _, ctx := smartFixture(t)
	seedSmartEndpoints(t, db)
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Updates(map[string]any{
		"context_length": 64000, "max_output_tokens": 8192, "abilities": `["stream"]`,
	}).Error)
	require.NoError(t, db.Model(&schema.Endpoint{}).Where("id = ?", "ep-a").Update("metadata", `{"context_length":32000,"max_output_tokens":"4096"}`).Error)
	got, err := (&GatewaySync{DB: db}).GetGatewayConfig(ctx, "a")
	require.NoError(t, err)
	m := got.Models["a"]
	require.EqualValues(t, 64000, m.ContextLength)
	require.EqualValues(t, 8192, m.MaxOutputTokens)
	require.Equal(t, []string{"stream"}, m.Abilities)
	require.Len(t, m.Endpoints, 1)
	require.EqualValues(t, 32000, m.Endpoints[0].ContextLength)
	require.EqualValues(t, 4096, m.Endpoints[0].MaxOutputTokens)
	require.Equal(t, []string{"stream"}, m.Endpoints[0].Abilities)
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"abilities":["stream"]`)
}

func TestEndpointCapabilitiesSharedOverrideSemantics(t *testing.T) {
	model := &schema.Model{ContextLength: 32000, MaxOutputTokens: 4096, Abilities: `["stream"]`}
	for _, raw := range []string{`{"context_length":16000,"max_output_tokens":"2048"}`, `{"context_length":"16000","max_output_tokens":2048}`} {
		context, output, abilities := endpointCapabilities(&schema.Endpoint{Model: model, Metadata: util.RawJSON(raw)})
		require.EqualValues(t, 16000, context)
		require.EqualValues(t, 2048, output)
		require.Equal(t, []string{"stream"}, abilities)
	}
	context, output, abilities := endpointCapabilities(&schema.Endpoint{})
	require.Zero(t, context)
	require.Zero(t, output)
	require.Empty(t, abilities)
}
