package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
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
	numKeys := numEndpoints * numMinutes * 2
	keys := make([]string, numKeys)

	idx := 0
	for _, ep := range endpoints {
		for i := 0; i < numMinutes; i++ {
			minute := currentMin - int64(numMinutes-1-i)
			keys[idx] = fmt.Sprintf("aigw:status:endpoint:%s:%d:s", ep.ID, minute)
			keys[idx+1] = fmt.Sprintf("aigw:status:endpoint:%s:%d:f", ep.ID, minute)
			idx += 2
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
		err = fmt.Errorf("redis client is nil")
	}

	idx = 0
	for _, ep := range endpoints {
		minSuccess := make([]int64, numMinutes)
		minFail := make([]int64, numMinutes)

		if err == nil && len(values) == numKeys {
			for i := 0; i < numMinutes; i++ {
				sVal := values[idx]
				fVal := values[idx+1]
				idx += 2

				if sVal != nil {
					if sStr, ok := sVal.(string); ok {
						if val, parseErr := strconv.ParseInt(sStr, 10, 64); parseErr == nil {
							minSuccess[i] = val
						}
					}
				}
				if fVal != nil {
					if fStr, ok := fVal.(string); ok {
						if val, parseErr := strconv.ParseInt(fStr, 10, 64); parseErr == nil {
							minFail[i] = val
						}
					}
				}
			}
		}

		points := make([]schema.StatusPoint, 10)
		for pIdx := 0; pIdx < 10; pIdx++ {
			var successSum int64
			var failSum int64
			if err == nil {
				for mOffset := 0; mOffset < 10; mOffset++ {
					mIdx := pIdx*10 + mOffset
					if mIdx < len(minSuccess) {
						successSum += minSuccess[mIdx]
						failSum += minFail[mIdx]
					}
				}
			}
			startSec := (currentMin - int64(numMinutes-1-pIdx*10)) * 60
			endSec := (currentMin - int64(numMinutes-1-(pIdx*10+9)) + 1) * 60
			startTimeStr := time.Unix(startSec, 0).Format("15:04")
			endTimeStr := time.Unix(endSec, 0).Format("15:04")

			points[pIdx] = schema.StatusPoint{
				SuccessCount: successSum,
				FailCount:    failSum,
				StartTime:    startTimeStr,
				EndTime:      endTimeStr,
			}
		}
		ep.StatusPoints = points
	}
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
	// Exists check for duplicate endpoint
	exists, err := e.EndpointDAL.ExistsDuplicate(ctx, formItem.ModelID, formItem.ProviderID, formItem.URL, formItem.ApiKey, formItem.RealModel, "")
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errors.Conflict("", "已经存在相同模型、相同供应商、相同 URL、相同 API Key 和相同真实模型的 Endpoint 记录")
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

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
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
	endpoint, err := e.EndpointDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if endpoint == nil {
		return errors.NotFound("", "Endpoint not found")
	}

	// Exists check (excluding self)
	exists, err := e.EndpointDAL.ExistsDuplicate(ctx, formItem.ModelID, formItem.ProviderID, formItem.URL, formItem.ApiKey, formItem.RealModel, id)
	if err != nil {
		return err
	} else if exists {
		return errors.Conflict("", "已经存在相同模型、相同供应商、相同 URL、相同 API Key 和相同真实模型的 Endpoint 记录")
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
	endpoint.Modifier = util.FromUsername(ctx)
	endpoint.UpdatedAt = time.Now()

	err = e.Trans.Exec(ctx, func(ctx context.Context) error {
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
	apiKey := formItem.ApiKey
	if apiKey == "" {
		keys := provider.GetApiKeys()
		if len(keys) > 0 {
			apiKey = keys[0]
		}
	}

	realModel := formItem.RealModel
	if realModel == "" {
		realModel = model.ModelCode
	}

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

	isEmbedding := false
	for _, cap := range apis {
		if cap == "embedding" {
			isEmbedding = true
		}
		if cap == "chat_completion" {
			isEmbedding = false // 如果同时拥有，优先采用 chat_completion 进行通用测试
			break
		}
	}

	// 4. 构造 HTTP 请求
	var reqURL string
	var reqBody []byte

	if isEmbedding {
		// Embedding 探测
		if strings.Contains(url, "/embeddings") {
			reqURL = url
		} else {
			reqURL = strings.TrimRight(url, "/") + "/embeddings"
		}
		bodyMap := map[string]interface{}{
			"model": realModel,
			"input": "ping",
		}
		reqBody, _ = json.Marshal(bodyMap)
	} else {
		// Chat Completions 探测
		if protocol == "anthropic" {
			if strings.Contains(url, "/messages") {
				reqURL = url
			} else {
				reqURL = strings.TrimRight(url, "/") + "/messages"
			}
			bodyMap := map[string]interface{}{
				"model":      realModel,
				"messages":   []map[string]string{{"role": "user", "content": "ping"}},
				"max_tokens": 1,
				"stream":     false,
			}
			reqBody, _ = json.Marshal(bodyMap)
		} else {
			// 默认 openai 兼容协议
			if strings.Contains(url, "/chat/completions") {
				reqURL = url
			} else {
				reqURL = strings.TrimRight(url, "/") + "/chat/completions"
			}
			bodyMap := map[string]interface{}{
				"model":      realModel,
				"messages":   []map[string]string{{"role": "user", "content": "ping"}},
				"max_tokens": 1,
				"stream":     false,
			}
			reqBody, _ = json.Marshal(bodyMap)
		}
	}

	// 创建带 10s 超时的 Context
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
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
		if protocol == "anthropic" {
			req.Header.Set("x-api-key", apiKey)
			req.Header.Set("anthropic-version", "2023-06-01")
		} else {
			req.Header.Set("Authorization", "Bearer "+apiKey)
		}
	}

	// 解析自定义 Header 并注入
	if len(formItem.Headers) > 0 && string(formItem.Headers) != "null" {
		var customHeaders map[string]string
		if err := json.Unmarshal(formItem.Headers, &customHeaders); err == nil {
			for k, v := range customHeaders {
				req.Header.Set(k, v)
			}
		}
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
	if isEmbedding {
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

		if len(antResp.Content) == 0 {
			return &schema.EndpointTestResult{
				Success:   false,
				LatencyMs: latency,
				Message:   "上游响应不符合 Anthropic 规范 (未获取到 Content)",
				Detail:    rawDetail,
			}, nil
		}

		return &schema.EndpointTestResult{
			Success:   true,
			LatencyMs: latency,
			Message:   "测试连接成功",
			Detail:    antResp.Content[0].Text,
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
		Protocol:    endpoint.Protocol,
		RealModel:   endpoint.RealModel,
		Priority:    endpoint.Priority,
		Weight:      endpoint.Weight,
		Enabled:     endpoint.Enabled,
		Headers:     endpoint.Headers,
		Metadata:    endpoint.Metadata,
		Description: endpoint.Description,
	}

	return e.Test(ctx, form)
}
