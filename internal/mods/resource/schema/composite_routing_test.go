package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/pkg/compositerouting"
)

func TestCompositeRoutingModesAndStrictDecoding(t *testing.T) {
	for _, body := range []string{
		`{"schema_version":2,"mode":"priority","primary_model_id":"a"}`,
		`{"schema_version":2,"mode":"rules","default_model_id":"a","rules":[]}`,
	} {
		var r SmartRouting
		require.NoError(t, json.Unmarshal([]byte(body), &r))
		require.NoError(t, r.Validate())
		require.Equal(t, []string{"a"}, r.ReferenceIDs())
		require.Zero(t, r.JudgeTimeoutMS, "deterministic selection must not acquire judge settings")
	}
	for _, body := range []string{
		`{"schema_version":3,"mode":"priority","primary_model_id":"a"}`,
		`{"mode":"priority","primary_model_id":"a"}`,
		`{"schema_version":2,"mode":"priority","primary_model_id":"a","judge_model_id":"b"}`,
		`{"schema_version":2,"mode":"priority","primary_model_id":"a","default_model_id":"b"}`,
		`{"schema_version":2,"mode":"other","primary_model_id":"a"}`,
	} {
		var r SmartRouting
		require.NoError(t, json.Unmarshal([]byte(body), &r))
		require.Error(t, r.Validate(), body)
	}
	var r SmartRouting
	require.Error(t, json.Unmarshal([]byte(`{"schema_version":2,"mode":"priority","primary_model_id":"a","primay_model_id":"b"}`), &r))
}

func TestCompositeRuleSnapshotDeepCopy(t *testing.T) {
	r := &SmartRouting{SchemaVersion: 2, Mode: compositerouting.Rules, DefaultModelID: "a", Rules: []SmartRule{{ID: "rule", Order: 1, Enabled: true, RelationType: "AND", TargetModelID: "b", Conditions: []compositerouting.Condition{{Type: "TAG", Key: "task", OpType: "EQUAL", Values: []string{"code"}}}}}}
	require.NoError(t, r.Validate())
	clone := r.Clone()
	clone.Rules[0].Conditions[0].Values[0] = "chat"
	require.Equal(t, "code", r.Rules[0].Conditions[0].Values[0])
	require.Equal(t, []string{"a", "b"}, r.ReferenceIDs())
	clone.Rules = append(clone.Rules, clone.Rules[0])
	require.Error(t, clone.Validate(), "duplicate rule identity/order is ambiguous")
}
