package biz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestSelectEndpointProbeKindGemini(t *testing.T) {
	assert.Equal(t, endpointProbeGemini, selectEndpointProbeKind("gemini", []string{"gemini_generate_content"}, false))
	assert.Equal(t, endpointProbeGemini, selectEndpointProbeKind("gemini", []string{"chat_completion"}, false))
	assert.Equal(t, endpointProbeGemini, selectEndpointProbeKind("", []string{"gemini_generate_content"}, false))
	assert.Equal(t, endpointProbeChat, selectEndpointProbeKind("openai", []string{"chat_completion"}, false))
}

func TestEndpointTestGeminiProbe(t *testing.T) {
	t.Run("success_with_normalized_url_and_native_header", func(t *testing.T) {
		var gotPath, gotAPIKey, gotAuth string
		var gotBody map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotAPIKey = r.Header.Get("x-goog-api-key")
			gotAuth = r.Header.Get("Authorization")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}`))
		}))
		t.Cleanup(server.Close)

		endpointBiz := newGeminiEndpointTestBiz(t, server.URL+"/v1beta")
		res, err := endpointBiz.Test(context.Background(), &schema.EndpointForm{
			ModelID:    "model-gemini",
			ProviderID: "provider-gemini",
			URL:        server.URL + "/v1beta/openai/",
			RealModel:  "models/gemini-2.0-flash",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Success)
		assert.Equal(t, "pong", res.Detail)
		assert.Equal(t, "/v1beta/models/gemini-2.0-flash:generateContent", gotPath)
		assert.Equal(t, "gemini-secret", gotAPIKey)
		assert.Empty(t, gotAuth)
		assert.NotEmpty(t, gotBody["contents"])
	})

	t.Run("upstream_error_and_missing_candidates", func(t *testing.T) {
		for _, tc := range []struct {
			name, body, wantMsg string
			status              int
		}{
			{"forbidden", `{"error":{"code":403,"message":"API key invalid"}}`, "API key invalid", http.StatusForbidden},
			{"missing_candidates", `{"candidates":[]}`, "candidates", http.StatusOK},
		} {
			t.Run(tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
				}))
				t.Cleanup(server.Close)

				endpointBiz := newGeminiEndpointTestBiz(t, server.URL+"/v1beta")
				res, err := endpointBiz.Test(context.Background(), &schema.EndpointForm{
					ModelID:    "model-gemini",
					ProviderID: "provider-gemini",
					URL:        server.URL + "/v1beta",
					RealModel:  "gemini-2.0-flash",
				})
				require.NoError(t, err)
				require.NotNil(t, res)
				assert.False(t, res.Success)
				assert.Contains(t, res.Message, tc.wantMsg)
			})
		}
	})

	t.Run("redirect_does_not_leak_api_key", func(t *testing.T) {
		var redirectedCalls atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			redirectedCalls.Add(1)
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"pong"}]}}]}`))
		}))
		t.Cleanup(target.Close)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+"/v1beta/models/gemini-2.0-flash:generateContent", http.StatusFound)
		}))
		t.Cleanup(server.Close)

		endpointBiz := newGeminiEndpointTestBiz(t, server.URL+"/v1beta")
		res, err := endpointBiz.Test(context.Background(), &schema.EndpointForm{
			ModelID:    "model-gemini",
			ProviderID: "provider-gemini",
			URL:        server.URL + "/v1beta",
			RealModel:  "gemini-2.0-flash",
		})
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.Success)
		assert.Zero(t, redirectedCalls.Load())
	})
}

func newGeminiEndpointTestBiz(t *testing.T, baseURL string) *Endpoint {
	t.Helper()
	db := newProviderDeleteTestDB(t)
	require.NoError(t, db.AutoMigrate(&schema.Model{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, db.Create(&schema.Provider{
		ID:       "provider-gemini",
		Code:     "google-gemini",
		Name:     "Google Gemini",
		Protocol: "gemini",
		URL:      baseURL,
		AuthType: "api_key",
		ApiKeys:  util.RawJSON(`[{"id":"key-1","value":"gemini-secret"}]`),
		Enabled:  1,
		Deleted:  "0",
	}).Error)
	require.NoError(t, db.Create(&schema.Model{
		ID:           "model-gemini",
		ModelName:    "gemini-2.0-flash",
		ModelCode:    "gemini-2.0-flash",
		ModelType:    schema.ModelTypeNormal,
		SpaceCode:    "demo",
		RequestTypes: `["gemini_generate_content"]`,
		Enabled:      1,
		Deleted:      "0",
	}).Error)

	return &Endpoint{
		ModelDAL:    &dal.Model{DB: db},
		ProviderDAL: &dal.Provider{DB: db},
	}
}
