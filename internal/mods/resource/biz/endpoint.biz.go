package biz

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/gatewaycontract"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
)

// Endpoint business logic layer
type Endpoint struct {
	Trans             *util.Trans
	EndpointDAL       *dal.Endpoint
	DataPermissionBIZ *DataPermission
	ModelDAL          *dal.Model
	ProviderDAL       *dal.Provider
	ConfigRedisSync   *ConfigRedisSync
	RedisClient       *redis.Client
	AuditLogBIZ       *opsBiz.AuditLog
}

// Query endpoints.
func (e *Endpoint) Query(ctx context.Context, params schema.EndpointQueryParam) (*schema.EndpointQueryResult, error) {
	params.Pagination = true

	result, err := e.EndpointDAL.Query(ctx, params, schema.EndpointQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Data) > 0 {
		e.fillEndpointsStatusPoints(ctx, result.Data)
	}

	return result, nil
}

func (e *Endpoint) fillEndpointsStatusPoints(ctx context.Context, endpoints []*schema.Endpoint) {
	if len(endpoints) == 0 {
		return
	}

	currentMin := time.Now().Unix() / 60
	numEndpoints := len(endpoints)
	numMinutes := 100
	keysPerMinute := 6
	numKeys := numEndpoints * numMinutes * keysPerMinute
	keys := make([]string, numKeys)

	idx := 0
	for _, ep := range endpoints {
		for i := 0; i < numMinutes; i++ {
			minute := currentMin - int64(numMinutes-1-i)
			keys[idx] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricSuccess)
			keys[idx+1] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricFailure)
			keys[idx+2] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricTTFTSum)
			keys[idx+3] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricTTFTCount)
			keys[idx+4] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricOutputTokens)
			keys[idx+5] = gatewaycontract.Keys.Status.Endpoint(ep.ID, minute, gatewaycontract.MetricDurationMs)
			idx += keysPerMinute
		}
	}

	var values []interface{}
	var err error
	if e.RedisClient != nil {
		batchSize := 500
		values = make([]interface{}, 0, len(keys))
		for i := 0; i < len(keys); i += batchSize {
			end := i + batchSize
			if end > len(keys) {
				end = len(keys)
			}
			batchKeys := keys[i:end]
			batchValues, batchErr := e.RedisClient.MGet(ctx, batchKeys...).Result()
			if batchErr != nil {
				err = batchErr
				break
			}
			values = append(values, batchValues...)
		}
		if err != nil {
			logging.Context(ctx).Error("Failed to MGet endpoint status points from Redis", zap.Error(err))
		} else {
			limit := 5
			if len(keys) < limit {
				limit = len(keys)
			}
			logging.Context(ctx).Info("Successfully MGet endpoint status points from Redis",
				zap.Int("keysCount", len(keys)),
				zap.Int("valuesCount", len(values)),
				zap.Any("firstFewKeys", keys[0:limit]),
				zap.Any("firstFewValues", values[0:limit]))
		}
	} else {
		// 从内存获取
		values = make([]interface{}, len(keys))
		idx = 0
		for _, ep := range endpoints {
			for i := 0; i < numMinutes; i++ {
				minute := currentMin - int64(numMinutes-1-i)
				perf := metrics.GlobalStore.GetEndpointMinutePerf(ep.ID, minute)
				if perf.Success > 0 {
					values[idx] = strconv.FormatInt(perf.Success, 10)
				}
				if perf.Fail > 0 {
					values[idx+1] = strconv.FormatInt(perf.Fail, 10)
				}
				if perf.TTFTSum > 0 {
					values[idx+2] = strconv.FormatInt(perf.TTFTSum, 10)
				}
				if perf.TTFTCount > 0 {
					values[idx+3] = strconv.FormatInt(perf.TTFTCount, 10)
				}
				if perf.Output > 0 {
					values[idx+4] = strconv.FormatInt(perf.Output, 10)
				}
				if perf.DurationMs > 0 {
					values[idx+5] = strconv.FormatInt(perf.DurationMs, 10)
				}
				idx += keysPerMinute
			}
		}
	}

	idx = 0
	for _, ep := range endpoints {
		perMinute := make([]schema.EndpointMinutePerf, numMinutes)

		if err == nil && len(values) == numKeys {
			for i := 0; i < numMinutes; i++ {
				perMinute[i] = schema.EndpointMinutePerf{
					Success:    parseRedisInt(values[idx]),
					Fail:       parseRedisInt(values[idx+1]),
					TTFTSum:    parseRedisInt(values[idx+2]),
					TTFTCount:  parseRedisInt(values[idx+3]),
					Output:     parseRedisInt(values[idx+4]),
					DurationMs: parseRedisInt(values[idx+5]),
				}
				idx += keysPerMinute
			}
		}

		points := make([]schema.StatusPoint, 10)
		for pIdx := 0; pIdx < 10; pIdx++ {
			start := pIdx * 10
			end := start + 10
			if end > len(perMinute) {
				end = len(perMinute)
			}
			startSec := (currentMin - int64(numMinutes-1-pIdx*10)) * 60
			endSec := (currentMin - int64(numMinutes-1-(pIdx*10+9)) + 1) * 60
			points[pIdx] = schema.AggregateEndpointStatusPoint(
				perMinute[start:end],
				time.Unix(startSec, 0).Format("15:04"),
				time.Unix(endSec, 0).Format("15:04"),
			)
		}
		ep.StatusPoints = points
	}
}

func parseRedisInt(val interface{}) int64 {
	if val == nil {
		return 0
	}
	s, ok := val.(string)
	if !ok {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// Get the specified endpoint.
func (e *Endpoint) Get(ctx context.Context, id string) (*schema.Endpoint, error) {
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if endpoint == nil {
		return nil, errors.NotFound("", "Endpoint not found")
	}

	if !util.FromIsRootUser(ctx) {
		ok, err := e.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeModel, id)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.NotFound("", "Endpoint not found")
		}
	}

	e.fillEndpointsStatusPoints(ctx, []*schema.Endpoint{endpoint})

	return endpoint, nil
}

// Create a new endpoint.
func (e *Endpoint) Create(ctx context.Context, formItem *schema.EndpointForm) (*schema.Endpoint, error) {
	if err := rejectSmartEndpoint(ctx, e.ModelDAL, formItem.ModelID); err != nil {
		return nil, err
	}
	// Exists check for duplicate endpoint
	exists, err := e.EndpointDAL.ExistsDuplicate(ctx, formItem.ModelID, formItem.ProviderID, formItem.URL, formItem.ApiKey, formItem.Protocol, formItem.RealModel, "")
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errors.Conflict("", "已经存在相同模型、相同供应商、相同 URL、相同 API Key、相同协议类型和相同真实模型的 Endpoint 记录")
	}

	// Exists check for code uniqueness
	existsCode, err := e.EndpointDAL.ExistsByCode(ctx, formItem.Code, "")
	if err != nil {
		return nil, err
	} else if existsCode {
		return nil, errors.Conflict("", "端点编码已存在")
	}

	endpoint := &schema.Endpoint{
		ID:        util.NewXID(),
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(endpoint); err != nil {
		return nil, err
	}

	if endpoint.AuthType == "" {
		if provider, _ := e.ProviderDAL.Get(ctx, endpoint.ProviderID); provider != nil {
			endpoint.AuthType = provider.AuthType
		}
	}
	if endpoint.AuthType == "" {
		endpoint.AuthType = "api_key"
	}

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := rejectSmartEndpoint(ctx, e.ModelDAL, endpoint.ModelID); err != nil {
			return err
		}
		if err := e.EndpointDAL.Create(ctx, endpoint); err != nil {
			return err
		}
		return e.DataPermissionBIZ.CreateByOwner(ctx, schema.DataPermissionTypeModel, endpoint.ID, util.FromTenant(ctx))
	})
	if err != nil {
		return nil, err
	}

	if model, _ := e.ModelDAL.Get(ctx, endpoint.ModelID); model != nil {
		if err := e.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode); err != nil {
			// Redis 同步失败不影响主流程，但记录日志便于排查
			fmt.Printf("[WARN] Redis sync failed for model %s: %v\n", model.ModelCode, err)
		}
	}

	e.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypeEndpoint, endpoint.ID, endpoint.Code, nil, endpoint)
	return endpoint, nil
}

// Update the specified endpoint.
func (e *Endpoint) Update(ctx context.Context, id string, formItem *schema.EndpointForm) error {
	if err := rejectSmartEndpoint(ctx, e.ModelDAL, formItem.ModelID); err != nil {
		return err
	}
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if endpoint == nil {
		return errors.NotFound("", "Endpoint not found")
	}

	// Exists check (excluding self)
	exists, err := e.EndpointDAL.ExistsDuplicate(ctx, formItem.ModelID, formItem.ProviderID, formItem.URL, formItem.ApiKey, formItem.Protocol, formItem.RealModel, id)
	if err != nil {
		return err
	} else if exists {
		return errors.Conflict("", "已经存在相同模型、相同供应商、相同 URL、相同 API Key、相同协议类型和相同真实模型的 Endpoint 记录")
	}

	// Exists check for code uniqueness (excluding self)
	existsCode, err := e.EndpointDAL.ExistsByCode(ctx, formItem.Code, id)
	if err != nil {
		return err
	} else if existsCode {
		return errors.Conflict("", "端点编码已存在")
	}

	var oldModelCode string
	if oldModel, _ := e.ModelDAL.Get(ctx, endpoint.ModelID); oldModel != nil {
		oldModelCode = oldModel.ModelCode
	}

	beforeEndpoint := *endpoint

	if err := formItem.FillTo(endpoint); err != nil {
		return err
	}

	if endpoint.AuthType == "" {
		if provider, _ := e.ProviderDAL.Get(ctx, endpoint.ProviderID); provider != nil {
			endpoint.AuthType = provider.AuthType
		}
	}
	if endpoint.AuthType == "" {
		endpoint.AuthType = "api_key"
	}

	endpoint.Modifier = util.FromUsername(ctx)
	endpoint.UpdatedAt = time.Now()

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := rejectSmartEndpoint(ctx, e.ModelDAL, endpoint.ModelID); err != nil {
			return err
		}
		return e.EndpointDAL.Update(ctx, endpoint)
	})
	if err == nil {
		if oldModelCode != "" {
			_ = e.ConfigRedisSync.SyncModelByCode(ctx, oldModelCode)
		}
		if newModel, _ := e.ModelDAL.Get(ctx, endpoint.ModelID); newModel != nil && newModel.ModelCode != oldModelCode {
			_ = e.ConfigRedisSync.SyncModelByCode(ctx, newModel.ModelCode)
		}
		e.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeEndpoint, endpoint.ID, endpoint.Code, beforeEndpoint, endpoint)
	}
	return err
}

// ToggleEnabled updates only the enabled status of an endpoint and re-syncs the routing config to Redis.
func (e *Endpoint) ToggleEnabled(ctx context.Context, id string, formItem *schema.EndpointEnabledForm) error {
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if endpoint == nil {
		return errors.NotFound("", "Endpoint not found")
	}

	// No-op if the status is unchanged.
	if endpoint.Enabled == formItem.Enabled {
		return nil
	}

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
		return e.EndpointDAL.UpdateEnabled(ctx, id, formItem.Enabled, util.FromUsername(ctx))
	})
	if err == nil {
		if model, _ := e.ModelDAL.Get(ctx, endpoint.ModelID); model != nil {
			_ = e.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
		}
		beforeData := map[string]int{"enabled": endpoint.Enabled}
		afterData := map[string]int{"enabled": formItem.Enabled}
		e.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeEndpoint, endpoint.ID, endpoint.Code, beforeData, afterData)
	}
	return err
}

// Delete the specified endpoint.
func (e *Endpoint) Delete(ctx context.Context, id string) error {
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if endpoint == nil {
		return errors.NotFound("", "Endpoint not found")
	}

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := e.EndpointDAL.Delete(ctx, id); err != nil {
			return err
		}
		return e.DataPermissionBIZ.DeleteByTypeAndDataId(ctx, schema.DataPermissionTypeModel, id)
	})
	if err == nil {
		if model, _ := e.ModelDAL.Get(ctx, endpoint.ModelID); model != nil {
			_ = e.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
		}
		e.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypeEndpoint, endpoint.ID, endpoint.Code, endpoint, nil)
	}
	return err
}

// QueryEndpointsByModelID queries endpoints associated with a given Model ID (only enabled endpoints).
func (e *Endpoint) QueryEndpointsByModelID(ctx context.Context, modelID string) (schema.Endpoints, error) {
	endpoints, err := e.EndpointDAL.QueryEndpointsByModelID(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if len(endpoints) > 0 {
		e.fillEndpointsStatusPoints(ctx, endpoints)
	}
	return endpoints, nil
}

// QueryEndpointsByProviderID queries endpoints associated with a given Provider ID (only enabled endpoints).
func (e *Endpoint) QueryEndpointsByProviderID(ctx context.Context, providerID string) (schema.Endpoints, error) {
	endpoints, err := e.EndpointDAL.QueryEndpointsByProviderID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if len(endpoints) > 0 {
		e.fillEndpointsStatusPoints(ctx, endpoints)
	}
	return endpoints, nil
}

// QueryEndpointsByModelCode queries enabled endpoints by model code (for routing).
// Joins endpoint -> model -> provider and filters all three to enabled + not deleted.
func (e *Endpoint) QueryEndpointsByModelCode(ctx context.Context, modelCode string) (schema.Endpoints, error) {
	return e.EndpointDAL.QueryEndpointsByModelCode(ctx, modelCode)
}

// SelectEndpoint selects the best enabled endpoint for a given model code,
// applying priority-based failover and weighted load balancing within the same priority group.
func (e *Endpoint) SelectEndpoint(ctx context.Context, modelCode string) (*schema.Endpoint, error) {
	endpoints, err := e.QueryEndpointsByModelCode(ctx, modelCode)
	if err != nil {
		return nil, err
	}

	if len(endpoints) == 0 {
		return nil, errors.NotFound("", "No available endpoint for model: %s", modelCode)
	}

	// Group endpoints by priority
	priorityGroups := make(map[int]schema.Endpoints)
	for _, ep := range endpoints {
		priorityGroups[ep.Priority] = append(priorityGroups[ep.Priority], ep)
	}

	// Collect and sort priorities (ascending: lower value = higher priority)
	priorities := make([]int, 0, len(priorityGroups))
	for p := range priorityGroups {
		priorities = append(priorities, p)
	}
	sort.Ints(priorities)

	// Select from the highest-priority (lowest value) group using weighted random
	for _, priority := range priorities {
		group := priorityGroups[priority]

		totalWeight := 0
		for _, ep := range group {
			totalWeight += ep.Weight
		}
		if totalWeight == 0 {
			continue
		}

		randWeight := rand.Intn(totalWeight)
		currentWeight := 0
		for _, ep := range group {
			currentWeight += ep.Weight
			if randWeight < currentWeight {
				return ep, nil
			}
		}
	}

	return nil, errors.InternalServerError("", "Failed to select endpoint for model: %s", modelCode)
}

// Test 临时测试草稿端点配置
func (e *Endpoint) Test(ctx context.Context, formItem *schema.EndpointForm) (*schema.EndpointTestResult, error) {
	// 1. 获取关联的 Model 和 Provider 数据以继承缺省参数
	model, err := e.ModelDAL.Get(ctx, formItem.ModelID)
	if err != nil {
		return nil, errors.BadRequest("", "加载关联模型失败: %s", err.Error())
	} else if model == nil {
		return nil, errors.BadRequest("", "关联模型不存在")
	}

	provider, err := e.ProviderDAL.Get(ctx, formItem.ProviderID)
	if err != nil {
		return nil, errors.BadRequest("", "加载关联供应商失败: %s", err.Error())
	} else if provider == nil {
		return nil, errors.BadRequest("", "关联供应商不存在")
	}

	// 2. 继承缺省参数
	// 认证类型：端点未指定时回退到 provider 级别
	authType := formItem.AuthType
	if authType == "" {
		authType = provider.AuthType
	}

	// OAuth 类型：token 即将过期时先同步刷新，与运行时/同步路径保持一致
	if provider.AuthType == "oauth_token" {
		cred := provider.GetOAuth()
		if cred != nil && cred.RefreshToken != "" {
			now := time.Now()
			if cred.ExpiresAt == nil || cred.ExpiresAt.Before(now.Add(5*time.Minute)) {
				refresher := NewTokenRefresher(e.ProviderDAL.DB, e.ConfigRedisSync)
				refresher.lockAndRefreshProvider(ctx, *provider)
				if refreshed, err := e.ProviderDAL.Get(ctx, formItem.ProviderID); err == nil && refreshed != nil {
					provider = refreshed
				}
			}
		}
	}

	// 探活只用第一把 key。解析函数返回全部，取哪一把由探活自己决定。
	customHeaders := map[string]string{}
	if len(formItem.Headers) > 0 && string(formItem.Headers) != "null" {
		_ = json.Unmarshal(formItem.Headers, &customHeaders)
	}
	probe := ResolveCall(CallInput{
		EndpointRealModel: formItem.RealModel,
		ModelCode:         model.ModelCode,
		AuthType:          authType,
		EndpointAPIKey:    formItem.ApiKey,
		ProviderAPIKeys:   providerAPIKeyValues(provider),
		Headers:           customHeaders,
		OAuthAccountID:    providerOAuthAccountID(provider),
	})
	apiKey := ""
	if len(probe.APIKeys) > 0 {
		apiKey = probe.APIKeys[0]
	}
	realModel := probe.RealModel
	customHeaders = probe.Headers

	protocol := formItem.Protocol
	if protocol == "" {
		protocol = provider.Protocol
	}

	url := strings.TrimSpace(formItem.URL)
	if url == "" {
		return nil, errors.BadRequest("", "端点 URL 不能为空")
	}

	// 3. 根据 RequestTypes 决定发送的请求协议
	var apis []string
	if model.RequestTypes != "" {
		_ = json.Unmarshal([]byte(model.RequestTypes), &apis)
	}

	isCodex := isCodexEndpoint(provider, url, apis)
	probeKind := selectEndpointProbeKind(protocol, apis, isCodex)

	// 4. 构造 HTTP 请求
	var reqURL string
	var reqBody []byte

	switch probeKind {
	case endpointProbeImageGeneration:
		reqURL, reqBody = buildImageGenerationProbe(url, realModel)
	case endpointProbeEmbedding:
		if strings.Contains(url, "/embeddings") {
			reqURL = url
		} else {
			reqURL = strings.TrimRight(url, "/") + "/embeddings"
		}
		reqBody, _ = json.Marshal(map[string]interface{}{
			"model": realModel,
			"input": "ping",
		})
	case endpointProbeJoyCode:
		var err error
		reqURL, reqBody, err = buildJoyCodeProbe(url, realModel, apis)
		if err != nil {
			return nil, err
		}
	case endpointProbeResponses:
		// OpenAI Responses / Codex backend 探测
		if strings.Contains(url, "/responses") {
			reqURL = url
		} else {
			reqURL = strings.TrimRight(url, "/") + "/responses"
		}
		// Minimal Responses payload. Codex backend requires:
		// 1) input as a list (not a plain string)
		// 2) stream=true ("Stream must be set to true")
		// 3) store=false ("Store must be set to false")
		// and rejects some OpenAI Responses params such as max_output_tokens.
		bodyMap := map[string]interface{}{
			"model": realModel,
			"input": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]string{
						{"type": "input_text", "text": "ping"},
					},
				},
			},
			"stream": isCodex, // Codex backend rejects non-stream probes
			"store":  false,
		}
		if !isCodex {
			// Keep a tighter non-Codex Responses probe when supported.
			bodyMap["max_output_tokens"] = 16
		}
		reqBody, _ = json.Marshal(bodyMap)
	case endpointProbeAnthropic:
		if strings.Contains(url, "/messages") {
			reqURL = url
		} else {
			reqURL = strings.TrimRight(url, "/") + "/messages"
		}
		reqBody, _ = json.Marshal(map[string]interface{}{
			"model":      realModel,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
			"max_tokens": 1,
			"stream":     false,
		})
	default:
		if strings.Contains(url, "/chat/completions") {
			reqURL = url
		} else {
			reqURL = strings.TrimRight(url, "/") + "/chat/completions"
		}
		reqBody, _ = json.Marshal(map[string]interface{}{
			"model":      realModel,
			"messages":   []map[string]string{{"role": "user", "content": "ping"}},
			"max_tokens": 1,
			"stream":     false,
		})
	}

	// 图片生成通常显著慢于文本/向量探测，给予更长的首个结果等待时间。
	testCtx, cancel := context.WithTimeout(ctx, endpointProbeTimeout(probeKind == endpointProbeImageGeneration))
	defer cancel()

	req, err := http.NewRequestWithContext(testCtx, http.MethodPost, reqURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return &schema.EndpointTestResult{
			Success: false,
			Message: fmt.Sprintf("构建请求失败: %s", err.Error()),
		}, nil
	}

	// 5. 设置 Header 头部信息
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 处理不同协议的 API Key Auth 头部
	if apiKey != "" {
		if protocol == "anthropic" && !isCodex {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
			// 如果是非官方 Anthropic 域名且 apiKey 不为空，则自动补充 Authorization: Bearer <key>
			// 以兼容类似于商汤(Sensenova)等使用 Anthropic 协议但采用 OpenAI 鉴权机制的第三方提供商
			if !strings.Contains(url, "anthropic.com") {
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
		} else if protocol == "joycode" && !isCodex {
			req.Header.Set("ptKey", apiKey)
			req.Header.Set("loginType", getLoginTypeForPtKey(apiKey))
			req.Header.Set("x-ms-client-request-id", uuid.NewString())
			req.Header.Set("client", "JoyCodeIDE")
			req.Header.Set("clientVersion", "3.8.61")
			req.Header.Set("Content-Type", "application/json; charset=UTF-8")
		} else {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	// Codex backend expects CLI-like headers in addition to Bearer + account id.
	if isCodex {
		req.Header.Set("User-Agent", codexModelsUserAgent)
		req.Header.Set("Originator", codexModelsOriginator)
		req.Header.Set("Session_id", uuid.NewString())
		req.Header.Set("Connection", "Keep-Alive")
		// Stream probes return SSE.
		req.Header.Set("Accept", "text/event-stream")
	}

	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}

	// 6. 执行请求并测量耗时
	client := &http.Client{}
	startTime := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(startTime).Milliseconds()

	if err != nil {
		errMsg := err.Error()
		if testCtx.Err() == context.DeadlineExceeded {
			errMsg = "请求超时 (超过 10 秒)"
		}
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("发送请求失败: %s", errMsg),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	rawDetail := string(bodyBytes)
	if len(rawDetail) > 2048 {
		rawDetail = rawDetail[:2048] + "... (响应过长截断)"
	}

	// 7. 处理响应结果
	if probeKind == endpointProbeImageGeneration {
		if resp.StatusCode != http.StatusOK {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("上游返回错误状态码: %d", resp.StatusCode),
				Detail:    rawDetail,
			}, nil
		}
		if err := validateImageGenerationProbeResponse(bodyBytes); err != nil {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   err.Error(),
				Detail:    rawDetail,
			}, nil
		}
		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    "图片生成成功",
		}, nil

	} else if probeKind == endpointProbeEmbedding {
		// 校验 Embedding
		var embResp struct {
			Data []struct {
				Embedding []float64 `json:"embedding"`
			} `json:"data"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		_ = json.Unmarshal(bodyBytes, &embResp)

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("上游返回错误状态码: %d", resp.StatusCode)
			if embResp.Error != nil && embResp.Error.Message != "" {
				errMsg = fmt.Sprintf("上游返回错误状态码: %d (%s)", resp.StatusCode, embResp.Error.Message)
			}
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   errMsg,
				Detail:    rawDetail,
			}, nil
		}

		if embResp.Error != nil {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("上游返回业务报错: %s", embResp.Error.Message),
				Detail:    rawDetail,
			}, nil
		}

		if len(embResp.Data) == 0 {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   "上游响应不符合 Embedding 规范 (未获取到 data 数组)",
				Detail:    rawDetail,
			}, nil
		}

		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    "Embedding 向量获取成功",
		}, nil

	} else if probeKind == endpointProbeJoyCode {
		return evaluateJoyCodeTestResult(resp.StatusCode, latency, bodyBytes, rawDetail, realModel)

	} else if probeKind == endpointProbeResponses || isCodex {
		return evaluateResponsesTestResult(resp.StatusCode, latency, bodyBytes, rawDetail, isCodex)

	} else if protocol == "anthropic" {
		// 校验 Anthropic Chat
		var antResp struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Error *struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}

		_ = json.Unmarshal(bodyBytes, &antResp)

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("上游返回错误状态码: %d", resp.StatusCode)
			if antResp.Error != nil && antResp.Error.Message != "" {
				errMsg = fmt.Sprintf("上游返回错误状态码: %d (%s)", resp.StatusCode, antResp.Error.Message)
			}
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   errMsg,
				Detail:    rawDetail,
			}, nil
		}

		if antResp.Error != nil {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("Anthropic 上游返回业务报错: %s", antResp.Error.Message),
				Detail:    rawDetail,
			}, nil
		}

		detailText := "连接成功 (未返回文本内容)"
		if len(antResp.Content) > 0 {
			detailText = antResp.Content[0].Text
		}

		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    detailText,
		}, nil

	} else {
		// 校验 OpenAI Chat
		var oaResp struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}

		_ = json.Unmarshal(bodyBytes, &oaResp)

		if resp.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("上游返回错误状态码: %d", resp.StatusCode)
			if oaResp.Error != nil && oaResp.Error.Message != "" {
				errMsg = fmt.Sprintf("上游返回错误状态码: %d (%s)", resp.StatusCode, oaResp.Error.Message)
			}
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   errMsg,
				Detail:    rawDetail,
			}, nil
		}

		if oaResp.Error != nil {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("OpenAI 上游返回业务报错: %s", oaResp.Error.Message),
				Detail:    rawDetail,
			}, nil
		}

		if len(oaResp.Choices) == 0 {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   "上游响应不符合 OpenAI 规范 (未获取到 Choices 数组)",
				Detail:    rawDetail,
			}, nil
		}

		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    oaResp.Choices[0].Message.Content,
		}, nil
	}
}

type endpointProbeKind int

const (
	endpointProbeChat endpointProbeKind = iota
	endpointProbeResponses
	endpointProbeAnthropic
	endpointProbeJoyCode
	endpointProbeEmbedding
	endpointProbeImageGeneration
)

func selectEndpointProbeKind(protocol string, requestTypes []string, isCodex bool) endpointProbeKind {
	if isCodex {
		return endpointProbeResponses
	}
	if protocol == "joycode" {
		return endpointProbeJoyCode
	}

	hasEmbedding := false
	hasResponses := false
	hasMessages := false
	hasChat := false
	for _, cap := range requestTypes {
		switch strings.TrimSpace(cap) {
		case "image_generation":
			return endpointProbeImageGeneration
		case "embedding":
			hasEmbedding = true
		case "responses":
			hasResponses = true
		case "messages":
			hasMessages = true
		case "chat_completion":
			hasChat = true
		}
	}
	if hasEmbedding && !hasResponses && !hasMessages && !hasChat {
		return endpointProbeEmbedding
	}
	if protocol == "anthropic" || (hasMessages && !hasChat && !hasResponses) {
		return endpointProbeAnthropic
	}
	if hasResponses && !hasChat {
		return endpointProbeResponses
	}
	return endpointProbeChat
}

func buildJoyCodeProbe(baseURL, model string, requestTypes []string) (string, []byte, error) {
	base := strings.TrimSpace(baseURL)
	if base == "" || base == "http://joycode-api-saas.jd.com" {
		base = "https://api-ai.jd.com"
	}
	base = strings.TrimRight(base, "/")

	hasResponses := false
	hasChat := false
	for _, cap := range requestTypes {
		switch strings.TrimSpace(cap) {
		case "responses":
			hasResponses = true
		case "chat_completion":
			hasChat = true
		}
	}

	// Match the gateway JoyCode invoker: Claude models always hit anthropic_completions.
	// Responses-only non-Claude endpoints hit responses_completions; messages-only ones
	// still authenticate through chat_completions after protocol translation.
	functionID := "chat_completions"
	path := "/api/saas/openai/v2/chat/completions"
	useAnthropic := strings.Contains(strings.ToLower(model), "claude")
	if useAnthropic {
		functionID = "anthropic_completions"
		path = "/api/saas/anthropic/v1/messages"
	} else if hasResponses && !hasChat {
		functionID = "responses_completions"
		path = "/api/saas/openai/v2/responses/completions"
	}

	var reqURL string
	if strings.HasPrefix(base, "https://") {
		signed, err := signJoyCodeGatewayURL(base, functionID)
		if err != nil {
			return "", nil, err
		}
		reqURL = signed
	} else {
		reqURL = base + path
	}

	var body map[string]interface{}
	if functionID == "responses_completions" {
		body = map[string]interface{}{
			"model": model,
			"input": []map[string]interface{}{
				{
					"role": "user",
					"content": []map[string]string{
						{"type": "input_text", "text": "ping"},
					},
				},
			},
			"stream":            false,
			"store":             false,
			"max_output_tokens": 16,
			"client":            "JoyCodeIDE",
			"clientVersion":     "3.8.61",
		}
	} else {
		body = map[string]interface{}{
			"model":         model,
			"messages":      []map[string]string{{"role": "user", "content": "ping"}},
			"max_tokens":    1,
			"stream":        false,
			"client":        "JoyCodeIDE",
			"clientVersion": "3.8.61",
		}
	}
	reqBody, _ := json.Marshal(body)
	return reqURL, reqBody, nil
}

func evaluateJoyCodeTestResult(statusCode int, latency int64, bodyBytes []byte, rawDetail, model string) (*schema.EndpointTestResult, error) {
	if msg := joyCodeLoginFailureMessage(bodyBytes); msg != "" {
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   msg,
			Detail:    rawDetail,
		}, nil
	}

	isAnt := strings.Contains(strings.ToLower(model), "claude")
	if isAnt {
		var antResp struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			Error *struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.Unmarshal(bodyBytes, &antResp)
		if statusCode != http.StatusOK {
			errMsg := fmt.Sprintf("JoyCode 上游返回错误状态码: %d", statusCode)
			if antResp.Error != nil && antResp.Error.Message != "" {
				errMsg = fmt.Sprintf("JoyCode 上游返回错误状态码: %d (%s)", statusCode, antResp.Error.Message)
			}
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   errMsg,
				Detail:    rawDetail,
			}, nil
		}
		if antResp.Error != nil {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("JoyCode Anthropic 上游返回业务报错: %s", antResp.Error.Message),
				Detail:    rawDetail,
			}, nil
		}
		detailText := "连接成功 (未返回文本内容)"
		if len(antResp.Content) > 0 {
			detailText = antResp.Content[0].Text
		}
		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    detailText,
		}, nil
	}

	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Output []struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Status string `json:"status"`
		Error  *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
		Code    interface{} `json:"code"`
		Msg     string      `json:"msg"`
		Message string      `json:"message"`
	}
	_ = json.Unmarshal(bodyBytes, &envelope)

	if statusCode != http.StatusOK {
		errMsg := fmt.Sprintf("JoyCode 上游返回错误状态码: %d", statusCode)
		if envelope.Error != nil && envelope.Error.Message != "" {
			errMsg = fmt.Sprintf("JoyCode 上游返回错误状态码: %d (%s)", statusCode, envelope.Error.Message)
		} else if envelope.Msg != "" {
			errMsg = fmt.Sprintf("JoyCode 上游返回错误状态码: %d (%s)", statusCode, envelope.Msg)
		} else if envelope.Message != "" {
			errMsg = fmt.Sprintf("JoyCode 上游返回错误状态码: %d (%s)", statusCode, envelope.Message)
		}
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   errMsg,
			Detail:    rawDetail,
		}, nil
	}
	if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("JoyCode 上游返回业务报错: %s", envelope.Error.Message),
			Detail:    rawDetail,
		}, nil
	}
	if !joyCodeBusinessCodeOK(envelope.Code) {
		errMsg := envelope.Msg
		if errMsg == "" {
			errMsg = envelope.Message
		}
		if errMsg == "" {
			errMsg = "JoyCode 登录状态无效"
		}
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("JoyCode 上游返回业务报错 (%v): %s", envelope.Code, errMsg),
			Detail:    rawDetail,
		}, nil
	}
	if len(envelope.Choices) > 0 {
		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    envelope.Choices[0].Message.Content,
		}, nil
	}
	if text := firstResponsesTextFromRaw(bodyBytes); text != "" || strings.EqualFold(envelope.Status, "completed") || len(envelope.Output) > 0 {
		detail := text
		if detail == "" {
			detail = "Responses 连接成功"
		}
		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    detail,
		}, nil
	}
	return &schema.EndpointTestResult{
		Success:   false,
		LatencyMs: latency,
		Message:   "JoyCode 上游响应不符合预期 (未获取到模型输出，可能已失去登录状态)",
		Detail:    rawDetail,
	}, nil
}

func joyCodeLoginFailureMessage(body []byte) string {
	text := strings.ToLower(string(body))
	markers := []string{
		"未登录",
		"登录失效",
		"登录已失效",
		"登录过期",
		"登陆失效",
		"login expired",
		"not login",
		"not logged",
		"invalid ptkey",
		"ptkey expired",
		"ptkey invalid",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return "JoyCode 登录状态已失效"
		}
	}
	return ""
}

func joyCodeBusinessCodeOK(code interface{}) bool {
	if code == nil {
		return true
	}
	switch val := code.(type) {
	case float64:
		return val == 0 || val == 200
	case string:
		return val == "" || val == "0" || val == "200"
	case json.Number:
		return val == "0" || val == "200"
	default:
		return false
	}
}

func firstResponsesTextFromRaw(body []byte) string {
	var respAPI responsesAPIProbe
	if err := json.Unmarshal(body, &respAPI); err != nil {
		return ""
	}
	return firstResponsesText(&respAPI)
}

func buildImageGenerationProbe(baseURL, model string) (string, []byte) {
	reqURL := strings.TrimRight(baseURL, "/")
	if !strings.Contains(reqURL, "/images/generations") {
		reqURL += "/images/generations"
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"model":           model,
		"prompt":          "A simple red circle on a white background",
		"response_format": "url",
	})
	return reqURL, reqBody
}

func endpointProbeTimeout(isImageGeneration bool) time.Duration {
	if isImageGeneration {
		return 120 * time.Second
	}
	return 10 * time.Second
}

func validateImageGenerationProbeResponse(body []byte) error {
	var imageResp struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &imageResp); err != nil {
		return fmt.Errorf("上游响应不是有效 JSON: %w", err)
	}
	if imageResp.Error != nil {
		return fmt.Errorf("上游返回业务报错: %s", imageResp.Error.Message)
	}
	if len(imageResp.Data) == 0 {
		return fmt.Errorf("上游响应不符合图片生成规范 (未获取到 data 数组)")
	}
	for _, item := range imageResp.Data {
		if item.URL != "" || item.B64JSON != "" {
			return nil
		}
	}
	return fmt.Errorf("上游响应不符合图片生成规范 (data 中没有图片)")
}

// TestByID 测试已保存端点的连通性
func (e *Endpoint) TestByID(ctx context.Context, id string) (*schema.EndpointTestResult, error) {
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if endpoint == nil {
		return nil, errors.NotFound("", "端点不存在")
	}

	// 将 schema.Endpoint 转为 EndpointForm 进行复用
	form := &schema.EndpointForm{
		ProviderID:  endpoint.ProviderID,
		ModelID:     endpoint.ModelID,
		URL:         endpoint.URL,
		ApiKey:      endpoint.ApiKey,
		AuthType:    endpoint.AuthType,
		Protocol:    endpoint.Protocol,
		RealModel:   endpoint.RealModel,
		Priority:    endpoint.Priority,
		Weight:      endpoint.Weight,
		Enabled:     endpoint.Enabled,
		Headers:     json.RawMessage(endpoint.Headers),
		Metadata:    json.RawMessage(endpoint.Metadata),
		Description: endpoint.Description,
	}

	return e.Test(ctx, form)
}

func signJoyCodeGatewayURL(baseURL, functionID string) (string, error) {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	t := time.Now().UnixNano() / int64(time.Millisecond)

	appID := os.Getenv("JOYCODE_APPID")
	signKey := os.Getenv("JOYCODE_SIGN_KEY")

	if appID == "" && signKey == "" {
		if strings.Contains(baseURL, "api-ai.jd.com") || strings.Contains(baseURL, "jd.com") {
			appID = "joycode_ide"
			signKey = "0691a3f0b37b4a85aeb63ad0fc7db3ed"
		} else {
			return "", fmt.Errorf("JOYCODE_APPID environment variable is missing")
		}
	} else {
		if appID == "" {
			return "", fmt.Errorf("JOYCODE_APPID environment variable is missing")
		}
		if signKey == "" {
			return "", fmt.Errorf("JOYCODE_SIGN_KEY environment variable is missing")
		}
	}

	stringToSign := fmt.Sprintf("%s&%s&%d", appID, functionID, t)
	key := []byte(signKey)

	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(stringToSign))
	sign := hex.EncodeToString(mac.Sum(nil))

	return fmt.Sprintf("%s/api?appid=%s&functionId=%s&t=%d&sign=%s", baseURL, appID, functionID, t, sign), nil
}

func getLoginTypeForPtKey(ptKey string) string {
	if strings.HasPrefix(ptKey, "BJ.") {
		return "ERP"
	}
	return "N_PIN_PC"
}

func injectJoyCodePayload(rawBody []byte) []byte {
	var m map[string]interface{}
	if err := json.Unmarshal(rawBody, &m); err != nil {
		return rawBody
	}
	m["client"] = "JoyCodeIDE"
	m["clientVersion"] = "3.8.61"

	if out, err := json.Marshal(m); err == nil {
		return out
	}
	return rawBody
}

// isCodexEndpoint reports whether endpoint connectivity should use the Codex
// ChatGPT backend Responses API instead of OpenAI chat.completions.
func isCodexEndpoint(provider *schema.Provider, endpointURL string, requestTypes []string) bool {
	if isCodexModelsBaseURL(endpointURL) {
		return true
	}
	if isCodexOAuthProvider(provider) {
		return true
	}
	for _, rt := range requestTypes {
		if rt == "responses" && strings.Contains(strings.ToLower(endpointURL), "chatgpt.com") {
			return true
		}
	}
	return false
}

type responsesAPIProbe struct {
	Type     string `json:"type"`
	Status   string `json:"status"`
	Response *struct {
		Status string `json:"status"`
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	} `json:"response"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func evaluateResponsesTestResult(statusCode int, latency int64, bodyBytes []byte, rawDetail string, isCodex bool) (*schema.EndpointTestResult, error) {
	// Non-SSE JSON error/success first.
	var respAPI responsesAPIProbe
	_ = json.Unmarshal(bodyBytes, &respAPI)

	if statusCode != http.StatusOK {
		errMsg := fmt.Sprintf("上游返回错误状态码: %d", statusCode)
		if msg := firstResponsesErrorMessage(bodyBytes, &respAPI); msg != "" {
			errMsg = fmt.Sprintf("上游返回错误状态码: %d (%s)", statusCode, msg)
		}
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   errMsg,
			Detail:    rawDetail,
		}, nil
	}

	// Codex stream=true returns text/event-stream.
	if isCodex || looksLikeSSE(bodyBytes) {
		if errMsg := extractResponsesSSEError(bodyBytes); errMsg != "" {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   fmt.Sprintf("Responses 上游返回业务报错: %s", errMsg),
				Detail:    rawDetail,
			}, nil
		}
		detail := extractResponsesSSEDetail(bodyBytes)
		if detail == "" {
			detail = "连接成功 (Responses SSE)"
		}
		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    detail,
		}, nil
	}

	if respAPI.Error != nil && strings.TrimSpace(respAPI.Error.Message) != "" {
		return &schema.EndpointTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("Responses 上游返回业务报错: %s", respAPI.Error.Message),
			Detail:    rawDetail,
		}, nil
	}

	detailText := "连接成功"
	if respAPI.Status != "" {
		detailText = "status=" + respAPI.Status
	}
	if respAPI.Response != nil && respAPI.Response.Status != "" {
		detailText = "status=" + respAPI.Response.Status
	}
	if text := firstResponsesText(&respAPI); text != "" {
		detailText = text
	} else if len(respAPI.Output) == 0 && respAPI.Status == "" && (respAPI.Response == nil || respAPI.Response.Status == "") {
		detailText = "连接成功 (Responses HTTP 200)"
	}

	return &schema.EndpointTestResult{
		Success:   true,
		LatencyMs: latency,
		Message:   "测试连接成功",
		Detail:    detailText,
	}, nil
}

func looksLikeSSE(body []byte) bool {
	s := strings.TrimSpace(string(body))
	return strings.HasPrefix(s, "data:") || strings.Contains(s, "\ndata:")
}

func firstResponsesErrorMessage(body []byte, fallback *responsesAPIProbe) string {
	if fallback != nil && fallback.Error != nil && strings.TrimSpace(fallback.Error.Message) != "" {
		return strings.TrimSpace(fallback.Error.Message)
	}
	if msg := extractResponsesSSEError(body); msg != "" {
		return msg
	}
	return ""
}

func firstResponsesText(respAPI *responsesAPIProbe) string {
	if respAPI == nil {
		return ""
	}
	scan := func(items []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}) string {
		for _, item := range items {
			for _, c := range item.Content {
				if strings.TrimSpace(c.Text) != "" {
					return strings.TrimSpace(c.Text)
				}
			}
		}
		return ""
	}
	if text := scan(respAPI.Output); text != "" {
		return text
	}
	if respAPI.Response != nil {
		return scan(respAPI.Response.Output)
	}
	return ""
}

func extractResponsesSSEError(body []byte) string {
	for _, data := range iterSSEDataPayloads(body) {
		var evt responsesAPIProbe
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}
		if evt.Error != nil && strings.TrimSpace(evt.Error.Message) != "" {
			return strings.TrimSpace(evt.Error.Message)
		}
		// Some streams put error under top-level message fields.
		if strings.EqualFold(evt.Type, "error") || strings.Contains(strings.ToLower(evt.Type), "failed") {
			if evt.Error != nil && strings.TrimSpace(evt.Error.Message) != "" {
				return strings.TrimSpace(evt.Error.Message)
			}
		}
	}
	return ""
}

func extractResponsesSSEDetail(body []byte) string {
	var lastStatus string
	var lastText string
	sawEvent := false
	for _, data := range iterSSEDataPayloads(body) {
		sawEvent = true
		var evt responsesAPIProbe
		if err := json.Unmarshal(data, &evt); err != nil {
			continue
		}
		if evt.Status != "" {
			lastStatus = evt.Status
		}
		if evt.Response != nil && evt.Response.Status != "" {
			lastStatus = evt.Response.Status
		}
		if text := firstResponsesText(&evt); text != "" {
			lastText = text
		}
		// Prefer completed event markers.
		if strings.Contains(strings.ToLower(evt.Type), "completed") && lastStatus == "" {
			lastStatus = "completed"
		}
	}
	if lastText != "" {
		return lastText
	}
	if lastStatus != "" {
		return "status=" + lastStatus
	}
	if sawEvent {
		return "连接成功 (收到 Responses 事件流)"
	}
	return ""
}

func iterSSEDataPayloads(body []byte) [][]byte {
	lines := strings.Split(string(body), "\n")
	out := make([][]byte, 0, 8)
	var dataBuf strings.Builder
	flush := func() {
		if dataBuf.Len() == 0 {
			return
		}
		payload := strings.TrimSpace(dataBuf.String())
		dataBuf.Reset()
		if payload == "" || payload == "[DONE]" {
			return
		}
		out = append(out, []byte(payload))
	}
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "data:") {
			chunk := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(chunk)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flush()
		}
	}
	flush()
	return out
}
