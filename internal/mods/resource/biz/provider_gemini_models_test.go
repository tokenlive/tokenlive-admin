package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestProviderFetchGeminiModels(t *testing.T) {
	var pageTokens []string
	var pageTokensMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1beta/models", r.URL.Path)
		assert.Equal(t, "60", r.URL.Query().Get("pageSize"))
		assert.Equal(t, "override-key", r.Header.Get("x-goog-api-key"))
		assert.Empty(t, r.Header.Get("Authorization"))
		assert.Empty(t, r.URL.Query().Get("key"))
		pageToken := r.URL.Query().Get("pageToken")
		pageTokensMu.Lock()
		pageTokens = append(pageTokens, pageToken)
		pageTokensMu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if pageToken == "" {
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-test","displayName":"Gemini Test","description":"Native generation model","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent","countTokens"]}],"nextPageToken":"next/+ page"}`))
			return
		}
		assert.Equal(t, "next/+ page", pageToken)
		_, _ = w.Write([]byte(`{"models":[{"name":"models/embedding-test","inputTokenLimit":2048,"supportedGenerationMethods":["embedContent"]}]}`))
	}))
	t.Cleanup(server.Close)
	provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta/")

	result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{APIKey: "override-key"})

	require.NoError(t, err)
	require.Len(t, result.Models, 2)
	pageTokensMu.Lock()
	assert.Equal(t, []string{"", "next/+ page"}, pageTokens)
	pageTokensMu.Unlock()
	assert.Equal(t, "gemini-test", result.Models[0].ID)
	assert.Equal(t, "model", result.Models[0].Object)
	assert.Equal(t, "google", result.Models[0].OwnedBy)
	assert.Equal(t, "embedding-test", result.Models[1].ID)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	var response map[string]any
	require.NoError(t, json.Unmarshal(encoded, &response))
	assert.Equal(t, server.URL+"/v1beta", response["base_url"])
	model := response["models"].([]any)[0].(map[string]any)
	assert.Equal(t, "models/gemini-test", model["name"])
	assert.Equal(t, "Gemini Test", model["display_name"])
	assert.Equal(t, "Native generation model", model["description"])
	assert.Equal(t, float64(1048576), model["input_token_limit"])
	assert.Equal(t, float64(65536), model["output_token_limit"])
	assert.Equal(t, []any{"generateContent", "countTokens"}, model["supported_generation_methods"])
}

func TestProviderFetchGeminiModelsNormalizesBaseURL(t *testing.T) {
	for _, tc := range []struct {
		name, basePath, catalogPath, normalizedPath string
	}{
		{"root", "", "/v1beta/models", "/v1beta"},
		{"beta", "/v1beta/", "/v1beta/models", "/v1beta"},
		{"stable", "/v1/", "/v1/models", "/v1"},
		{"uppercase", "/V1BETA/", "/v1beta/models", "/v1beta"},
		{"models_suffix", "/v1beta/models/", "/v1beta/models", "/v1beta"},
		{"openai", "/v1beta/openai/", "/v1beta/models", "/v1beta"},
		{"openai_models", "/v1beta/openai/models", "/v1beta/models", "/v1beta"},
		{"proxy", "/proxy/v1beta/", "/proxy/v1beta/models", "/proxy/v1beta"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.catalogPath, r.URL.Path)
				assert.Equal(t, "provider-key", r.Header.Get("x-goog-api-key"))
				_, _ = w.Write([]byte(`{"models":[]}`))
			}))
			t.Cleanup(server.Close)
			provider := newGeminiModelsTestProvider(t, server.URL)
			result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{BaseURL: server.URL + tc.basePath})
			require.NoError(t, err)
			require.NotNil(t, result.Models)
			assert.Empty(t, result.Models)
			encoded, err := json.Marshal(result)
			require.NoError(t, err)
			var response map[string]any
			require.NoError(t, json.Unmarshal(encoded, &response))
			assert.Equal(t, server.URL+tc.normalizedPath, response["base_url"])
		})
	}
}

func TestProviderFetchModelsChecksPermissionsAndBlocksCrossHostCredentialLeak(t *testing.T) {
	var attackerCalls atomic.Int32
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerCalls.Add(1)
		_, _ = w.Write([]byte(`{"models":[{"name":"models/leaked"}]}`))
	}))
	t.Cleanup(attacker.Close)

	trusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "provider-key", r.Header.Get("x-goog-api-key"))
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-ok"}]}`))
	}))
	t.Cleanup(trusted.Close)

	provider := newGeminiModelsTestProvider(t, trusted.URL+"/v1beta")
	provider.DataPermissionBIZ = &DataPermission{
		DataPermissionDAL: &dal.DataPermission{DB: provider.ProviderDAL.DB},
	}
	require.NoError(t, provider.ProviderDAL.DB.Create(&schema.DataPermission{
		ID:         "perm-1",
		Type:       schema.DataPermissionTypeProvider,
		DataId:     "provider-1",
		User:       "alice",
		Tenant:     "tenant-a",
		Role:       "owner",
		Permission: 1,
		Deleted:    "0",
	}).Error)

	// Non-root user without permission is rejected.
	bobCtx := util.NewTenant(util.NewUsername(context.Background(), "bob"), "tenant-b")
	res, err := provider.FetchModels(bobCtx, "provider-1", &schema.FetchModelsForm{})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "Provider not found")

	// Authorized user cannot exfiltrate stored provider key to a different host.
	aliceCtx := util.NewTenant(util.NewUsername(context.Background(), "alice"), "tenant-a")
	res, err = provider.FetchModels(aliceCtx, "provider-1", &schema.FetchModelsForm{BaseURL: attacker.URL + "/v1beta"})
	require.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "explicit api_key")
	assert.Zero(t, attackerCalls.Load())

	// Cross-host override succeeds when an explicit API key is supplied.
	res, err = provider.FetchModels(aliceCtx, "provider-1", &schema.FetchModelsForm{
		BaseURL: attacker.URL + "/v1beta",
		APIKey:  "explicit-key",
	})
	require.NoError(t, err)
	require.Len(t, res.Models, 1)
	assert.Equal(t, int32(1), attackerCalls.Load())

	// Authorized user calling the trusted origin succeeds with stored key.
	res, err = provider.FetchModels(aliceCtx, "provider-1", &schema.FetchModelsForm{})
	require.NoError(t, err)
	require.Len(t, res.Models, 1)
	assert.Equal(t, "gemini-ok", res.Models[0].ID)
}

func TestProviderFetchGeminiModelsDeduplicatesSanitizesAndBoundsPagination(t *testing.T) {
	t.Run("dedup_and_sanitize_negative_limits", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("pageToken") == "" {
				_, _ = w.Write([]byte(`{"models":[{"name":" models/gemini-dup ","inputTokenLimit":-5,"outputTokenLimit":1024}],"nextPageToken":"p2"}`))
				return
			}
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-dup","inputTokenLimit":4096},{"name":"models/gemini-unique","inputTokenLimit":2048,"outputTokenLimit":-1}]}`))
		}))
		t.Cleanup(server.Close)
		provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")

		result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})
		require.NoError(t, err)
		require.Len(t, result.Models, 2)
		assert.Equal(t, "gemini-dup", result.Models[0].ID)
		assert.Equal(t, "models/gemini-dup", result.Models[0].Name)
		assert.Nil(t, result.Models[0].InputTokenLimit)
		require.NotNil(t, result.Models[0].OutputTokenLimit)
		assert.Equal(t, 1024, *result.Models[0].OutputTokenLimit)
		assert.Equal(t, "gemini-unique", result.Models[1].ID)
		assert.Nil(t, result.Models[1].OutputTokenLimit)
	})

	t.Run("max_pages_guard", func(t *testing.T) {
		var pages atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			n := pages.Add(1)
			_, _ = fmt.Fprintf(w, `{"models":[{"name":"models/m-%d"}],"nextPageToken":"tok-%d"}`, n, n)
		}))
		t.Cleanup(server.Close)
		provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")

		result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "pagination limit")
		assert.Equal(t, int32(geminiModelsMaxPages), pages.Load())
	})

	t.Run("oversized_response_rejected", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"models":[{"name":"models/big","description":"` + strings.Repeat("a", geminiModelsMaxResponseBytes) + `"}]}`))
		}))
		t.Cleanup(server.Close)
		provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")

		result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "size limit")
	})
}

func TestProviderFetchGeminiModelsRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name, body, message string
		status              int
	}{
		{"upstream_error", `{"error":{"message":"invalid test key"}}`, "status 403", http.StatusForbidden},
		{"invalid_json", `{`, "parse", http.StatusOK},
		{"openai_shape", `{"data":[{"id":"gemini-test"}]}`, "models", http.StatusOK},
		{"missing_name", `{"models":[{"displayName":"Missing ID"}]}`, "name", http.StatusOK},
		{"repeated_page", `{"models":[],"nextPageToken":"same"}`, "pagination", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)
			provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")
			result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tc.message)
			assert.LessOrEqual(t, requests.Load(), int32(2))
		})
	}
}

func TestProviderFetchGeminiModelsHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Query().Get("pageToken") == "" {
			_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-test"}],"nextPageToken":"next"}`))
			return
		}
		cancel()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")

	result, err := provider.FetchModels(ctx, "provider-1", &schema.FetchModelsForm{})

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Equal(t, int32(2), requests.Load())
}

func TestProviderFetchGeminiModelsEmptyAndMissingLimits(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		count      int
	}{
		{"omitted_empty_list", `{}`, 0},
		{"missing_limits", `{"models":[{"name":"models/gemini-test","supportedGenerationMethods":["generateContent"]}]}`, 1},
		{"zero_output", `{"models":[{"name":"models/embedding-test","inputTokenLimit":2048,"outputTokenLimit":0,"supportedGenerationMethods":["embedContent"]}]}`, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(tc.body))
			}))
			t.Cleanup(server.Close)
			provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")
			result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})
			require.NoError(t, err)
			require.NotNil(t, result.Models)
			require.Len(t, result.Models, tc.count)
			if tc.name == "missing_limits" {
				assert.Nil(t, result.Models[0].InputTokenLimit)
				assert.Nil(t, result.Models[0].OutputTokenLimit)
			}
			if tc.name == "zero_output" {
				require.NotNil(t, result.Models[0].OutputTokenLimit)
				assert.Zero(t, *result.Models[0].OutputTokenLimit)
			}
		})
	}
}

func TestNormalizeGeminiModelsBaseURL(t *testing.T) {
	for _, tc := range []struct{ baseURL, expected string }{
		{"", geminiDefaultBaseURL},
		{"  https://generativelanguage.googleapis.com/  ", geminiDefaultBaseURL},
		{"https://generativelanguage.googleapis.com/v1beta/models/", geminiDefaultBaseURL},
		{"https://generativelanguage.googleapis.com/v1beta/openai/models", geminiDefaultBaseURL},
		{"https://example.test/proxy", "https://example.test/proxy/v1beta"},
		{"https://example.test/v1alpha/", "https://example.test/v1alpha"},
		{"https://example.test/V1BETA/", "https://example.test/v1beta"},
		{"https://example.test/v2beta1/", "https://example.test/v2beta1"},
	} {
		normalized, err := normalizeGeminiModelsBaseURL(tc.baseURL)
		require.NoError(t, err)
		assert.Equal(t, tc.expected, normalized)
	}
	for _, baseURL := range []string{
		"example.test/v1beta", "ftp://example.test/v1beta", "https://",
		"https://example.test/v1beta?key=test", "https://example.test/v1beta#models", "https://user:pass@example.test/v1beta",
	} {
		_, err := normalizeGeminiModelsBaseURL(baseURL)
		require.Error(t, err, baseURL)
	}
}

func TestProviderFetchGeminiModelsRejectsRedirects(t *testing.T) {
	var redirectedRequests atomic.Int32
	var receivedAPIKey atomic.Value
	receivedAPIKey.Store("")
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectedRequests.Add(1)
		receivedAPIKey.Store(r.Header.Get("x-goog-api-key"))
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-test"}]}`))
	}))
	t.Cleanup(target.Close)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/v1beta/models", http.StatusFound)
	}))
	t.Cleanup(server.Close)
	provider := newGeminiModelsTestProvider(t, server.URL+"/v1beta")

	result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Zero(t, redirectedRequests.Load(), "A redirect must not receive the provider API key")
	assert.Empty(t, receivedAPIKey.Load())
}

func TestGeminiModelMetadataExportsNativeLimits(t *testing.T) {
	db, _, ctx := smartFixture(t)
	seedSmartEndpoints(t, db)
	require.NoError(t, db.Model(&schema.Model{}).Where("id = ?", "a").Updates(map[string]any{
		"context_length": 64000, "max_output_tokens": 8192, "abilities": `["stream"]`,
	}).Error)
	// Native input_token_limit / output_token_limit alone must drive endpoint capabilities without alias duplication.
	metadata := map[string]string{
		"name":                         "models/gemini-test",
		"input_token_limit":            "32000",
		"output_token_limit":           "0",
		"supported_generation_methods": `["generateContent"]`,
	}
	raw, err := json.Marshal(metadata)
	require.NoError(t, err)
	require.NoError(t, db.Model(&schema.Endpoint{}).Where("id = ?", "ep-a").Update("metadata", string(raw)).Error)

	result, err := (&GatewaySync{DB: db}).GetGatewayConfig(ctx, "a")

	require.NoError(t, err)
	model := result.Models["a"]
	assert.EqualValues(t, 64000, model.ContextLength)
	assert.EqualValues(t, 8192, model.MaxOutputTokens)
	require.Len(t, model.Endpoints, 1)
	assert.EqualValues(t, 32000, model.Endpoints[0].ContextLength)
	assert.Zero(t, model.Endpoints[0].MaxOutputTokens)
	assert.Equal(t, []string{"stream"}, model.Endpoints[0].Abilities)
	encoded, err := json.Marshal(model.Endpoints[0])
	require.NoError(t, err)
	var endpoint map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &endpoint))
	assert.Equal(t, "32000", string(endpoint["context_length"]))
	_, hasMaxOutput := endpoint["max_output_tokens"]
	assert.False(t, hasMaxOutput, "zero max_output_tokens must not inherit model cap 8192")
	var exportedMetadata map[string]string
	require.NoError(t, json.Unmarshal(endpoint["metadata"], &exportedMetadata))
	assert.Equal(t, metadata, exportedMetadata)
}

func TestProviderFetchOpenAIModelsRemainsCompatible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/models", r.URL.Path)
		assert.Equal(t, "Bearer provider-key", r.Header.Get("Authorization"))
		assert.Empty(t, r.Header.Get("x-goog-api-key"))
		assert.Empty(t, r.URL.RawQuery)
		_, _ = w.Write([]byte(`{"data":[{"id":"openai-test","object":"model","owned_by":"openai"}]}`))
	}))
	t.Cleanup(server.Close)
	provider := newGeminiModelsTestProvider(t, server.URL+"/v1")
	require.NoError(t, provider.ProviderDAL.DB.Model(&schema.Provider{}).Where("id = ?", "provider-1").Update("protocol", "openai").Error)

	result, err := provider.FetchModels(context.Background(), "provider-1", &schema.FetchModelsForm{})

	require.NoError(t, err)
	assert.Equal(t, []schema.UpstreamModel{{ID: "openai-test", Object: "model", OwnedBy: "openai"}}, result.Models)
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	assert.JSONEq(t, `{"models":[{"id":"openai-test","object":"model","owned_by":"openai"}]}`, string(encoded))
}

func newGeminiModelsTestProvider(t *testing.T, baseURL string) *Provider {
	t.Helper()
	db := newProviderDeleteTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.Create(&schema.Provider{
		ID:       "provider-1",
		Code:     "gemini-test",
		Name:     "Gemini Test",
		Protocol: "gemini",
		URL:      baseURL,
		AuthType: "api_key",
		ApiKeys:  util.RawJSON(`[{"id":"key-1","value":"provider-key"}]`),
		Deleted:  "0",
	}).Error)
	return &Provider{ProviderDAL: &dal.Provider{DB: db}}
}
