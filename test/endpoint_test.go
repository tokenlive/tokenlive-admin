package test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// setupEndpointDeps creates a provider and model for endpoint tests and returns them.
func setupEndpointDeps(t *testing.T, e *httpexpect.Expect, suffix string) (schema.Provider, schema.Model) {
	assert := assert.New(t)

	providerFormItem := schema.ProviderForm{
		Code:        fmt.Sprintf("test-endpoint-provider-code-%s", suffix),
		Name:        fmt.Sprintf("test-endpoint-provider-%s", suffix),
		Protocol:    "openai",
		ApiKeys:     []string{"sk-test-key"},
		Enabled:     1,
		Description: "Provider for endpoint tests",
	}

	var provider schema.Provider
	e.POST(baseAPI + "/providers").WithJSON(providerFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &provider})
	assert.NotEmpty(provider.ID)
	assert.Equal(providerFormItem.Name, provider.Name)
	assert.Equal(providerFormItem.Enabled, provider.Enabled)

	modelFormItem := schema.ModelForm{
		ModelName:     fmt.Sprintf("test-endpoint-model-%s", suffix),
		ModelCode:     fmt.Sprintf("test-endpoint-model-code-%s", suffix),
		SpaceCode:     "default",
		RequestTypes:  `["chat_completion"]`,
		ContextLength: 128000,
		Enabled:       1,
		Description:   "Model for endpoint tests",
	}

	var model schema.Model
	e.POST(baseAPI + "/models").WithJSON(modelFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &model})
	assert.NotEmpty(model.ID)
	assert.Equal(modelFormItem.ModelName, model.ModelName)
	assert.Equal(modelFormItem.Enabled, model.Enabled)

	return provider, model
}

func TestEndpoint_CreateWithModelID(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	provider, model := setupEndpointDeps(t, e, "create")

	// ===== Create Endpoint with new fields (ModelID, RealModel, Priority) =====
	endpointFormItem := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.openai.com/v1/chat/completions",
		RealModel:   "gpt-4-0613",
		Priority:    10,
		Weight:      1,
		Enabled:     1,
		Description: "Primary endpoint",
	}

	var endpoint schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint})
	assert.NotEmpty(endpoint.ID)
	assert.Equal(endpointFormItem.ProviderID, endpoint.ProviderID)
	assert.Equal(endpointFormItem.ModelID, endpoint.ModelID)
	assert.Equal(endpointFormItem.URL, endpoint.URL)
	assert.Equal(endpointFormItem.RealModel, endpoint.RealModel)
	assert.Equal(endpointFormItem.Priority, endpoint.Priority)
	assert.Equal(endpointFormItem.Weight, endpoint.Weight)
	assert.Equal(endpointFormItem.Enabled, endpoint.Enabled)

	// ===== Get Endpoint by ID =====
	var getEndpoint schema.Endpoint
	e.GET(baseAPI + "/endpoints/" + endpoint.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &getEndpoint})
	assert.Equal(endpoint.ID, getEndpoint.ID)
	assert.Equal(endpoint.ModelID, getEndpoint.ModelID)
	assert.Equal(endpoint.RealModel, getEndpoint.RealModel)
	assert.Equal(endpoint.Priority, getEndpoint.Priority)

	// ===== Update Endpoint =====
	updateFormItem := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.openai.com/v1/chat/completions",
		RealModel:   "gpt-4-turbo",
		Priority:    5,
		Weight:      3,
		Enabled:     1,
		Description: "Updated primary endpoint",
	}

	e.PUT(baseAPI + "/endpoints/" + endpoint.ID).WithJSON(updateFormItem).
		Expect().Status(http.StatusOK)

	var updatedEndpoint schema.Endpoint
	e.GET(baseAPI + "/endpoints/" + endpoint.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &updatedEndpoint})
	assert.Equal("gpt-4-turbo", updatedEndpoint.RealModel)
	assert.Equal(5, updatedEndpoint.Priority)
	assert.Equal(3, updatedEndpoint.Weight)
	assert.Equal("Updated primary endpoint", updatedEndpoint.Description)

	// ===== Delete Endpoint =====
	e.DELETE(baseAPI + "/endpoints/" + endpoint.ID).Expect().Status(http.StatusOK)
	e.GET(baseAPI + "/endpoints/" + endpoint.ID).Expect().Status(http.StatusNotFound)

	// ===== Cleanup =====
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/providers/" + provider.ID).Expect().Status(http.StatusOK)
}

func TestEndpoint_QueryByModelID(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	provider, model := setupEndpointDeps(t, e, "query")

	// Create two endpoints under the same model
	endpointFormItem1 := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.openai.com/v1/chat/completions",
		RealModel:   "gpt-4-0613",
		Priority:    10,
		Weight:      1,
		Enabled:     1,
		Description: "Primary endpoint",
	}

	var endpoint1 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem1).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint1})
	assert.NotEmpty(endpoint1.ID)

	endpointFormItem2 := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.backup.com/v1/chat/completions",
		RealModel:   "gpt-4-0314",
		Priority:    20,
		Weight:      2,
		Enabled:     1,
		Description: "Backup endpoint",
	}

	var endpoint2 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem2).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint2})
	assert.NotEmpty(endpoint2.ID)

	// ===== Query Endpoints with model_id filter =====
	var queryResult schema.EndpointQueryResult
	e.GET(baseAPI+"/endpoints").WithQuery("model_id", model.ID).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &queryResult.Data})
	assert.GreaterOrEqual(len(queryResult.Data), 2)

	for _, ep := range queryResult.Data {
		assert.Equal(model.ID, ep.ModelID)
	}

	// ===== Query Endpoints by Model ID (via /models/:id/endpoints) =====
	var modelEndpoints []*schema.Endpoint
	e.GET(baseAPI + "/models/" + model.ID + "/endpoints").
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &modelEndpoints})
	assert.GreaterOrEqual(len(modelEndpoints), 2)

	// ===== Query Endpoints by Provider ID (via /providers/:id/endpoints) =====
	var providerEndpoints []*schema.Endpoint
	e.GET(baseAPI + "/providers/" + provider.ID + "/endpoints").
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &providerEndpoints})
	assert.GreaterOrEqual(len(providerEndpoints), 2)

	// ===== Cleanup =====
	e.DELETE(baseAPI + "/endpoints/" + endpoint1.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/endpoints/" + endpoint2.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/providers/" + provider.ID).Expect().Status(http.StatusOK)
}

func TestEndpoint_SelectEndpoint(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	provider, model := setupEndpointDeps(t, e, "select")

	// Create endpoints with different priorities and weights
	endpointFormItem1 := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.openai.com/v1/chat/completions",
		RealModel:   "gpt-4-0613",
		Priority:    10,
		Weight:      1,
		Enabled:     1,
		Description: "Primary endpoint",
	}

	var endpoint1 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem1).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint1})
	assert.NotEmpty(endpoint1.ID)

	endpointFormItem2 := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.backup.com/v1/chat/completions",
		RealModel:   "gpt-4-0314",
		Priority:    20,
		Weight:      2,
		Enabled:     1,
		Description: "Backup endpoint",
	}

	var endpoint2 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem2).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint2})
	assert.NotEmpty(endpoint2.ID)

	// ===== Select Endpoints via /models/:id/endpoints and verify ordering =====
	var modelEndpoints []*schema.Endpoint
	e.GET(baseAPI + "/models/" + model.ID + "/endpoints").
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &modelEndpoints})
	assert.GreaterOrEqual(len(modelEndpoints), 2)

	// Verify ordering: priority ASC, weight DESC
	for i := 1; i < len(modelEndpoints); i++ {
		prev := modelEndpoints[i-1]
		curr := modelEndpoints[i]
		if prev.Priority == curr.Priority {
			assert.GreaterOrEqual(prev.Weight, curr.Weight)
		} else {
			assert.Less(prev.Priority, curr.Priority)
		}
	}

	// ===== Cleanup =====
	e.DELETE(baseAPI + "/endpoints/" + endpoint1.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/endpoints/" + endpoint2.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/providers/" + provider.ID).Expect().Status(http.StatusOK)
}

func TestEndpoint_TestConnectivity(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	provider, model := setupEndpointDeps(t, e, "test-conn")

	// 1. 测试未保存的临时草稿配置 (POST /api/v1/endpoints/test)
	testDraftForm := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://invalid-domain-xxx-yyy.com/v1/chat/completions",
		RealModel:   "gpt-4",
		Priority:    1,
		Weight:      1,
		Enabled:     1,
		Description: "Test Draft",
	}

	var testResult schema.EndpointTestResult
	e.POST(baseAPI + "/endpoints/test").WithJSON(testDraftForm).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &testResult})

	// 由于使用的是无效的域名，应该测试失败，但是测试流程应该顺利走完
	assert.False(testResult.Success)
	assert.Contains(testResult.Message, "发送请求失败")

	// 2. 测试已保存的端点配置 (POST /api/v1/endpoints/:id/test)
	endpointFormItem := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://invalid-domain-xxx-yyy.com/v1/chat/completions",
		RealModel:   "gpt-4",
		Priority:    1,
		Weight:      1,
		Enabled:     1,
		Description: "Saved Endpoint",
	}

	var endpoint schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint})
	assert.NotEmpty(endpoint.ID)

	var testResultByID schema.EndpointTestResult
	e.POST(baseAPI + "/endpoints/" + endpoint.ID + "/test").
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &testResultByID})

	assert.False(testResultByID.Success)
	assert.Contains(testResultByID.Message, "发送请求失败")

	// ===== Cleanup =====
	e.DELETE(baseAPI + "/endpoints/" + endpoint.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/providers/" + provider.ID).Expect().Status(http.StatusOK)
}

func TestEndpoint_DuplicateCheck(t *testing.T) {
	e := tester(t)
	assert := assert.New(t)

	provider, model := setupEndpointDeps(t, e, "dup")

	endpointFormItem1 := schema.EndpointForm{
		ProviderID:  provider.ID,
		ModelID:     model.ID,
		URL:         "https://api.openai.com/v1/chat/completions",
		ApiKey:      "sk-test-key-123",
		RealModel:   "gpt-4",
		Priority:    1,
		Weight:      1,
		Enabled:     1,
		Description: "Endpoint A",
	}

	// 1. First endpoint creation should succeed
	var endpoint1 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem1).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint1})
	assert.NotEmpty(endpoint1.ID)

	// 2. Duplicate endpoint creation (same model, provider, url, api_key, real_model) should fail with Conflict (409)
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem1).
		Expect().Status(http.StatusConflict)

	// 3. Creation with different RealModel should succeed
	endpointFormItem2 := endpointFormItem1
	endpointFormItem2.RealModel = "gpt-4-turbo"
	var endpoint2 schema.Endpoint
	e.POST(baseAPI + "/endpoints").WithJSON(endpointFormItem2).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &endpoint2})
	assert.NotEmpty(endpoint2.ID)

	// ===== Cleanup =====
	e.DELETE(baseAPI + "/endpoints/" + endpoint1.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/endpoints/" + endpoint2.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/providers/" + provider.ID).Expect().Status(http.StatusOK)
}
