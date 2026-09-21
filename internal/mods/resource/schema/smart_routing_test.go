package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const validSmartForm = `{"model_code":"smart-model","model_type":"smart","request_types":"[\"chat_completion\"]","smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":40,"model_id":"a"},{"min":40,"max":100,"model_id":"b"}]}}`

func TestSmartFormRejectsInvalidConfiguration(t *testing.T) {
	// Missing validation would allow a non-partition or an unsupported request protocol.
	for _, tc := range []struct{ name, patch string }{
		{"enabled model missing config", `{"smart_routing":null,"enabled":1}`},
		{"unknown type", `{"model_type":"other"}`},
		{"gap", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":40,"model_id":"a"},{"min":41,"max":100,"model_id":"b"}]}}`},
		{"overlap", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":50,"model_id":"a"},{"min":40,"max":100,"model_id":"b"}]}}`},
		{"empty interval", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":0,"model_id":"a"},{"min":0,"max":100,"model_id":"b"}]}}`},
		{"out of bounds", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":-1,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`},
		{"missing end", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":99,"model_id":"b"}]}}`},
		{"one target", `{"smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"a"}]}}`},
		{"judge missing", `{"smart_routing":{"ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`},
		{"negative timeout", `{"smart_routing":{"judge_model_id":"a","judge_timeout_ms":-1,"ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`},
		{"negative input", `{"smart_routing":{"judge_model_id":"a","judge_max_input_bytes":-1,"ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`},
		{"negative output", `{"smart_routing":{"judge_model_id":"a","judge_max_output_tokens":-1,"ranges":[{"min":0,"max":50,"model_id":"a"},{"min":50,"max":100,"model_id":"b"}]}}`},
		{"other protocol", `{"request_types":"[\"embedding\"]"}`},
		{"normal config", `{"model_type":"normal"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var base, patch map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(validSmartForm), &base))
			require.NoError(t, json.Unmarshal([]byte(tc.patch), &patch))
			for k, v := range patch {
				base[k] = v
			}
			b, err := json.Marshal(base)
			require.NoError(t, err)
			var form ModelForm
			require.NoError(t, json.Unmarshal(b, &form))
			require.Error(t, form.Validate())
		})
	}
}

func TestSmartFormAllowsConfiguredAbilities(t *testing.T) {
	var form ModelForm
	require.NoError(t, json.Unmarshal([]byte(`{"model_code":"smart-model","model_type":"smart","request_types":"[\"chat_completion\"]","abilities":"[\"stream\",\"tool_call\"]","smart_routing":{"judge_model_id":"a","ranges":[{"min":0,"max":40,"model_id":"a"},{"min":40,"max":100,"model_id":"b"}]}}`), &form))
	require.NoError(t, form.Validate())
}

func TestSmartFormAllowsOnlyDisabledUnconfiguredDraft(t *testing.T) {
	var form ModelForm
	require.NoError(t, json.Unmarshal([]byte(`{"model_code":"draft","model_type":"smart","enabled":0,"request_types":"[\"chat_completion\"]","smart_routing":null}`), &form))
	require.NoError(t, form.Validate())
	form.Enabled = 1
	require.Error(t, form.Validate())
}

func TestSmartFormRejectsFractionalBoundaries(t *testing.T) {
	var form ModelForm
	require.Error(t, json.Unmarshal([]byte(`{"model_type":"smart","smart_routing":{"ranges":[{"min":0.5,"max":100,"model_id":"a"}]}}`), &form))
}

func TestSmartModelDefaultsRoundTripAndRepeatMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	for i := 0; i < 2; i++ {
		require.NoError(t, db.AutoMigrate(&Model{}))
	}
	var form ModelForm
	require.NoError(t, json.Unmarshal([]byte(validSmartForm), &form))
	require.NoError(t, form.Validate())
	model := &Model{ID: "smart", ModelName: "Smart"}
	require.NoError(t, form.FillTo(model))
	require.NoError(t, db.Create(model).Error)
	var got Model
	require.NoError(t, db.First(&got, "id = ?", "smart").Error)
	b, err := json.Marshal(got)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, "smart", wire["model_type"])
	routing, ok := wire["smart_routing"].(map[string]any)
	require.True(t, ok, "smart_routing must be an object, not an encoded string")
	require.Equal(t, float64(5000), routing["judge_timeout_ms"])
	require.Equal(t, float64(65536), routing["judge_max_input_bytes"])
	require.Equal(t, float64(256), routing["judge_max_output_tokens"])

	normal := &Model{ID: "normal", ModelName: "Normal", ModelCode: "normal"}
	require.NoError(t, db.Create(normal).Error)
	got = Model{}
	require.NoError(t, db.First(&got, "id = ?", "normal").Error)
	b, err = json.Marshal(normal)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &wire))
	require.Equal(t, "normal", wire["model_type"])
}

func TestSmartModelMigrationUpgradesLegacyRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(&Model{}))
	require.NoError(t, db.Migrator().DropColumn(&Model{}, "ModelType"))
	require.NoError(t, db.Migrator().DropColumn(&Model{}, "SmartRouting"))
	require.NoError(t, db.Exec(`INSERT INTO model (id, model_name, model_code, space_code) VALUES ('legacy', 'Legacy', 'legacy', 'default')`).Error)
	// Existing AutoMigrate is the repository's migration mechanism. Re-run it
	// against an old row to ensure the new default is not merely a create hook.
	require.NoError(t, db.AutoMigrate(&Model{}))
	require.NoError(t, db.AutoMigrate(&Model{}))
	var got Model
	require.NoError(t, db.First(&got, "id = ?", "legacy").Error)
	require.Equal(t, "normal", got.ModelType)
	require.Nil(t, got.SmartRouting)
}
