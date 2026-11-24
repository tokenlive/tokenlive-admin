package biz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
)

func TestProviderFetchJoyCodeModelsPublishesNativeResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/saas/models/v2/modelList", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":0,"data":[
			{"chatApiModel":"GPT-6.2 Sol","extJson":{"adapterType":"openai-response"}},
			{"label":"GPT-6.2 Astra","ext":"{\"adapterType\":\"openai-response\"}"},
			{"chatApiModel":"GLM-5.3","extJson":{"adapterType":"openai"}},
			{"chatApiModel":"Kimi-K3.1","ext":"malformed"}
		]}`))
	}))
	t.Cleanup(server.Close)

	db := newProviderDeleteTestDB(t)
	require.NoError(t, db.Create(&schema.Provider{
		ID: "joycode", Code: "joycode", Name: "JoyCode", Protocol: "joycode", URL: server.URL, Deleted: "0",
	}).Error)
	provider := &Provider{ProviderDAL: &dal.Provider{DB: db}}
	result, err := provider.FetchModels(context.Background(), "joycode", &schema.FetchModelsForm{APIKey: "test-key"})
	require.NoError(t, err)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	assert.JSONEq(t, `{"models":[
		{"id":"GPT-6.2 Sol","object":"model","owned_by":"jd","request_types":["responses"]},
		{"id":"GPT-6.2 Astra","object":"model","owned_by":"jd","request_types":["responses"]},
		{"id":"GLM-5.3","object":"model","owned_by":"jd"},
		{"id":"Kimi-K3.1","object":"model","owned_by":"jd"}
	]}`, string(encoded))
}
