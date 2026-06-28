package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type GatewaySync struct {
	DB            *gorm.DB
	EndpointDAL   *dal.Endpoint
	ModelDAL      *dal.Model
	ModelAliasDAL *dal.ModelAlias
}

var (
	gatewayConfigCache     *cache.Cache
	gatewayConfigCacheOnce sync.Once
)

func getGatewayConfigCache() *cache.Cache {
	gatewayConfigCacheOnce.Do(func() {
		// 缓存 10 秒，每 1 分钟清理一次过期项
		gatewayConfigCache = cache.New(10*time.Second, 1*time.Minute)
	})
	return gatewayConfigCache
}

// ClearGatewayConfigCache 主动清除所有网关缓存项
func ClearGatewayConfigCache() {
	getGatewayConfigCache().Flush()
}

type GatewayConfig struct {
	Models    map[string]ModelConfig `json:"models"`
	Providers map[string]ProviderConfig `json:"providers"`
	Fallbacks map[string][]string    `json:"fallbacks"`
}

type ModelConfig struct {
	RequestTypes []string         `json:"request_types"`
	Endpoints    []EndpointConfig `json:"endpoints"`
}

type ProviderConfig struct {
	Protocol   string `json:"protocol"`
	APIKey     string `json:"api_key"`
	Timeout    string `json:"timeout"`
	MaxRetries int    `json:"max_retries"`
}

type EndpointConfig struct {
	ID        string            `json:"id"`
	Code      string            `json:"code"`
	Provider  string            `json:"provider"`
	URL       string            `json:"url"`
	RealModel string            `json:"real_model"`
	APIKey    string            `json:"api_key,omitempty"`
	Protocol  string            `json:"protocol,omitempty"`
	Timeout   string            `json:"timeout,omitempty"`
	Priority  int               `json:"priority"`
	Weight    int               `json:"weight"`
	Headers   map[string]string `json:"headers,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type HTTPPolicyValue struct {
	LoadBalancePolicy    *policySchema.PolicyLoadbalanceForm     `json:"load_balance_policy,omitempty"`
	InvocationPolicy     *policySchema.PolicyInvocationForm      `json:"invocation_policy,omitempty"`
	LimitPolicies        []*policySchema.PolicyLimitForm         `json:"limit_policies,omitempty"`
	RoutePolicies        []*policySchema.PolicyRouteForm         `json:"route_policies,omitempty"`
	CircuitBreakPolicies []*policySchema.PolicyCircuitBreakForm  `json:"circuit_break_policies,omitempty"`
	TaggingPolicies      []*policySchema.PolicyTaggingForm       `json:"tagging_policies,omitempty"`
	Billing              *BillingPolicy                          `json:"billing,omitempty"`
}

type BillingPolicy struct {
	InputPrice         float64 `json:"input_price"`
	OutputPrice        float64 `json:"output_price"`
	CachedPrice        float64 `json:"cached_price"`
	CacheCreationPrice float64 `json:"cache_creation_price"`
}

type HTTPPolicyItem struct {
	Scope string           `json:"scope"` // "user:userID", "tenant:tenantCode", "model:modelCode", "global"
	Model string           `json:"model"` // model_code or "*"
	Value *HTTPPolicyValue `json:"value"`
}

type HTTPApiKeyItem struct {
	APIKey      string `json:"api_key"`
	UserID      string `json:"user_id"`
	Tenant      string `json:"tenant"`
	WorkspaceID string `json:"workspace_id"`
	UserTenant  string `json:"user_tenant"`
	Status      int    `json:"status"`
	Quota       int64  `json:"quota"`
	ExpiresAt   int64  `json:"expires_at"`
}

// GetGatewayConfig 获取大模型、端点及 Provider 路由配置
func (s *GatewaySync) GetGatewayConfig(ctx context.Context, modelCode string) (*GatewayConfig, error) {
	c := getGatewayConfigCache()
	cacheKey := "config:all"
	if modelCode != "" {
		cacheKey = "config:model:" + modelCode
	}

	if val, found := c.Get(cacheKey); found {
		if cached, ok := val.(*GatewayConfig); ok {
			return cached, nil
		}
	}

	db := util.GetDB(ctx, s.DB)
	endpointTable := config.C.FormatTableName("endpoint")
	modelTable := config.C.FormatTableName("model")
	providerTable := config.C.FormatTableName("provider")

	query := db.Table(endpointTable).
		Preload("Model").
		Preload("Provider").
		Joins("JOIN "+modelTable+" ON "+endpointTable+".model_id = "+modelTable+".id").
		Joins("JOIN "+providerTable+" ON "+endpointTable+".provider_id = "+providerTable+".id").
		Where(endpointTable + ".enabled = 1").
		Where(modelTable + ".enabled = 1").
		Where(providerTable + ".enabled = 1").
		Where(endpointTable + ".deleted = '0'").
		Where(modelTable + ".deleted = '0'").
		Where(providerTable + ".deleted = '0'")

	if modelCode != "" {
		query = query.Where(modelTable+".model_code = ?", modelCode)
	}

	var dbEndpoints []schema.Endpoint
	err := query.Order(endpointTable + ".priority ASC, " + endpointTable + ".weight DESC").Find(&dbEndpoints).Error
	if err != nil {
		return nil, fmt.Errorf("query endpoints: %w", err)
	}

	modelsMap := make(map[string]ModelConfig)
	providersMap := make(map[string]ProviderConfig)

	for _, ep := range dbEndpoints {
		if ep.Model == nil || ep.Provider == nil {
			continue
		}

		mCode := ep.Model.ModelCode
		providerName := ep.Provider.Name

		if _, exists := providersMap[providerName]; !exists {
			apiKeys := ep.Provider.GetApiKeys()
			var primaryKey string
			if len(apiKeys) > 0 {
				primaryKey = apiKeys[0]
			}
			providersMap[providerName] = ProviderConfig{
				Protocol:   ep.Provider.Protocol,
				APIKey:     primaryKey,
				Timeout:    "60s",
				MaxRetries: 3,
			}
		}

		var headersMap map[string]string
		if len(ep.Headers) > 0 {
			_ = json.Unmarshal(ep.Headers, &headersMap)
		}
		var metadataMap map[string]string
		if len(ep.Metadata) > 0 {
			_ = json.Unmarshal(ep.Metadata, &metadataMap)
		}

		var apis []string
		if ep.Model.RequestTypes != "" {
			_ = json.Unmarshal([]byte(ep.Model.RequestTypes), &apis)
		}
		apis = normalizeRequestTypesForProtocol(ep.Protocol, apis)

		realModel := ep.RealModel
		if realModel == "" {
			realModel = mCode
		}

		epCfg := EndpointConfig{
			ID:        ep.ID,
			Code:      ep.Code,
			Provider:  providerName,
			URL:       ep.URL,
			RealModel: realModel,
			APIKey:    ep.ApiKey,
			Protocol:  ep.Protocol,
			Priority:  ep.Priority,
			Weight:    ep.Weight,
			Headers:   headersMap,
			Metadata:  metadataMap,
		}

		mCfg, exists := modelsMap[mCode]
		if !exists {
			mCfg = ModelConfig{
				RequestTypes: apis,
				Endpoints:    []EndpointConfig{},
			}
		}
		mCfg.Endpoints = append(mCfg.Endpoints, epCfg)
		modelsMap[mCode] = mCfg
	}

	fallbacks := make(map[string][]string)
	result := &GatewayConfig{
		Models:    modelsMap,
		Providers: providersMap,
		Fallbacks: fallbacks,
	}

	c.Set(cacheKey, result, cache.DefaultExpiration)
	return result, nil
}

// GetGatewayPolicies 获取治理策略及计费策略
func (s *GatewaySync) GetGatewayPolicies(ctx context.Context, modelCode string) ([]HTTPPolicyItem, error) {
	c := getGatewayConfigCache()
	cacheKey := "policies:all"
	if modelCode != "" {
		cacheKey = "policies:model:" + modelCode
	}

	if val, found := c.Get(cacheKey); found {
		if cached, ok := val.([]HTTPPolicyItem); ok {
			return cached, nil
		}
	}

	db := util.GetDB(ctx, s.DB)
	bindingQuery := db.Table(config.C.FormatTableName("policy_binding")).
		Where("enabled = 1 AND deleted = '0'")

	if modelCode != "" {
		bindingQuery = bindingQuery.Where("model_code = ? OR model_code = '' OR model_code IS NULL", modelCode)
	}

	var bindings []policySchema.PolicyBinding
	err := bindingQuery.Find(&bindings).Error
	if err != nil {
		return nil, fmt.Errorf("query policy bindings: %w", err)
	}

	var lbs []policySchema.PolicyLoadbalance
	_ = db.Table(config.C.FormatTableName("policy_loadbalance")).Where("enabled = 1 AND deleted = '0'").Find(&lbs)
	lbMap := make(map[string]*policySchema.PolicyLoadbalance)
	for i := range lbs {
		lbMap[lbs[i].ID] = &lbs[i]
	}

	var invocations []policySchema.PolicyInvocation
	_ = db.Table(config.C.FormatTableName("policy_invocation")).Where("enabled = 1 AND deleted = '0'").Find(&invocations)
	invMap := make(map[string]*policySchema.PolicyInvocation)
	for i := range invocations {
		invMap[invocations[i].ID] = &invocations[i]
	}

	var limits []policySchema.PolicyLimit
	_ = db.Table(config.C.FormatTableName("policy_limit")).Where("enabled = 1 AND deleted = '0'").Find(&limits)
	limitMap := make(map[string]*policySchema.PolicyLimit)
	for i := range limits {
		limitMap[limits[i].ID] = &limits[i]
	}

	var cbs []policySchema.PolicyCircuitBreak
	_ = db.Table(config.C.FormatTableName("policy_circuit_break")).Where("enabled = 1 AND deleted = '0'").Find(&cbs)
	cbMap := make(map[string]*policySchema.PolicyCircuitBreak)
	for i := range cbs {
		cbMap[cbs[i].ID] = &cbs[i]
	}

	var taggings []policySchema.PolicyTagging
	_ = db.Table(config.C.FormatTableName("policy_tagging")).Where("enabled = 1 AND deleted = '0'").Find(&taggings)
	tagMap := make(map[string]*policySchema.PolicyTagging)
	for i := range taggings {
		tagMap[taggings[i].ID] = &taggings[i]
	}

	var routes []policySchema.PolicyRoute
	_ = db.Table(config.C.FormatTableName("policy_route")).Preload("Details").Where("enabled = 1 AND deleted = '0'").Find(&routes)
	routeMap := make(map[string]*policySchema.PolicyRoute)
	for i := range routes {
		routeMap[routes[i].ID] = &routes[i]
	}

	type policyGroupKey struct {
		Scope string
		Model string
	}
	policyGroups := make(map[policyGroupKey]*HTTPPolicyValue)

	getOrCreateGroup := func(scope, model string) *HTTPPolicyValue {
		key := policyGroupKey{Scope: scope, Model: model}
		if p, ok := policyGroups[key]; ok {
			return p
		}
		p := &HTTPPolicyValue{}
		policyGroups[key] = p
		return p
	}

	for _, b := range bindings {
		scope, model := resolveScopeAndModel(b.TenantCode, b.UserID, b.ModelCode)
		policyAgg := getOrCreateGroup(scope, model)

		switch b.PolicyType {
		case "loadbalance":
			if lb, ok := lbMap[b.PolicyID]; ok {
				var form policySchema.PolicyLoadbalanceForm
				if err := lb.ConvertTo(&form); err == nil {
					policyAgg.LoadBalancePolicy = &form
				}
			}
		case "invocation":
			if inv, ok := invMap[b.PolicyID]; ok {
				var form policySchema.PolicyInvocationForm
				if err := inv.ConvertTo(&form); err == nil {
					policyAgg.InvocationPolicy = &form
				}
			}
		case "limit":
			if lim, ok := limitMap[b.PolicyID]; ok {
				var form policySchema.PolicyLimitForm
				if err := lim.ConvertTo(&form); err == nil {
					policyAgg.LimitPolicies = append(policyAgg.LimitPolicies, &form)
				}
			}
		case "circuit_break":
			if cb, ok := cbMap[b.PolicyID]; ok {
				var form policySchema.PolicyCircuitBreakForm
				if err := cb.ConvertTo(&form); err == nil {
					policyAgg.CircuitBreakPolicies = append(policyAgg.CircuitBreakPolicies, &form)
				}
			}
		case "tagging":
			if tag, ok := tagMap[b.PolicyID]; ok {
				var form policySchema.PolicyTaggingForm
				if err := tag.ConvertTo(&form); err == nil {
					policyAgg.TaggingPolicies = append(policyAgg.TaggingPolicies, &form)
				}
			}
		case "route":
			if r, ok := routeMap[b.PolicyID]; ok {
				var form policySchema.PolicyRouteForm
				if err := r.ConvertTo(&form); err == nil {
					policyAgg.RoutePolicies = append(policyAgg.RoutePolicies, &form)
				}
			}
		}
	}

	modelTable := config.C.FormatTableName("model")
	modelQuery := db.Table(modelTable).Where("enabled = 1 AND deleted = '0'")
	if modelCode != "" {
		modelQuery = modelQuery.Where("model_code = ?", modelCode)
	}

	var dbModels []schema.Model
	_ = modelQuery.Find(&dbModels)
	for _, m := range dbModels {
		scope := "model:" + m.ModelCode
		model := "*"
		policyAgg := getOrCreateGroup(scope, model)
		policyAgg.Billing = &BillingPolicy{
			InputPrice:         m.InputPrice,
			OutputPrice:        m.OutputPrice,
			CachedPrice:        m.CachedPrice,
			CacheCreationPrice: m.CacheCreationPrice,
		}
	}

	var policyItems []HTTPPolicyItem
	for key, p := range policyGroups {
		policyItems = append(policyItems, HTTPPolicyItem{
			Scope: key.Scope,
			Model: key.Model,
			Value: p,
		})
	}

	c.Set(cacheKey, policyItems, cache.DefaultExpiration)
	return policyItems, nil
}

// GetGatewayApiKeys 获取 API 密钥配置（支持单 key 查询）
func (s *GatewaySync) GetGatewayApiKeys(ctx context.Context, apiKey string) ([]HTTPApiKeyItem, error) {
	c := getGatewayConfigCache()
	cacheKey := "apikeys:all"
	if apiKey != "" {
		cacheKey = "apikeys:key:" + apiKey
	}

	if val, found := c.Get(cacheKey); found {
		if cached, ok := val.([]HTTPApiKeyItem); ok {
			return cached, nil
		}
	}

	db := util.GetDB(ctx, s.DB)
	var apiKeys []HTTPApiKeyItem

	// 1. 查询用户租户关联
	userTable := config.C.FormatTableName("user")
	var userTenants []struct {
		ID     string `gorm:"column:id"`
		Tenant string `gorm:"column:tenant"`
	}
	_ = db.Table(userTable).Find(&userTenants)
	userTenantMap := make(map[string]string, len(userTenants))
	for _, u := range userTenants {
		userTenantMap[u.ID] = u.Tenant
	}

	// 2. 查询用户 API Keys
	var userKeys []struct {
		UserID    string     `gorm:"column:user_id"`
		APIKey    string     `gorm:"column:api_key"`
		Status    int        `gorm:"column:status"`
		Quota     int64      `gorm:"column:quota"`
		ExpiresAt *time.Time `gorm:"column:expires_at"`
	}
	userApiKeyTable := config.C.FormatTableName("user_api_key")
	userKeyQuery := db.Table(userApiKeyTable).Where("deleted = '0' AND api_key != ''")
	if apiKey != "" {
		userKeyQuery = userKeyQuery.Where("api_key = ?", apiKey)
	}

	err := userKeyQuery.Find(&userKeys).Error
	if err == nil {
		for _, key := range userKeys {
			var expiresAtVal int64 = 0
			if key.ExpiresAt != nil {
				expiresAtVal = key.ExpiresAt.Unix()
			}
			apiKeys = append(apiKeys, HTTPApiKeyItem{
				APIKey:     key.APIKey,
				UserID:     key.UserID,
				UserTenant: userTenantMap[key.UserID],
				Status:     key.Status,
				Quota:      key.Quota,
				ExpiresAt:  expiresAtVal,
			})
		}
	}

	// 3. 查询租户 API Keys (如果传入了特定的 apiKey 且在用户 Key 中未找到，或者 apiKey 为空)
	if apiKey == "" || len(apiKeys) == 0 {
		var tenantKeys []struct {
			Code   string `gorm:"column:code"`
			APIKey string `gorm:"column:api_key"`
			Status string `gorm:"column:status"`
		}
		tenantTable := config.C.FormatTableName("tenant")
		tenantKeyQuery := db.Table(tenantTable).Where("deleted = '0' AND api_key IS NOT NULL AND api_key != ''")
		if apiKey != "" {
			tenantKeyQuery = tenantKeyQuery.Where("api_key = ?", apiKey)
		}

		err = tenantKeyQuery.Find(&tenantKeys).Error
		if err == nil {
			for _, t := range tenantKeys {
				statusVal := 2 // disabled
				if t.Status == "activated" {
					statusVal = 1 // enabled
				}
				apiKeys = append(apiKeys, HTTPApiKeyItem{
					APIKey:    t.APIKey,
					Tenant:    t.Code,
					Status:    statusVal,
					Quota:     -1, // unlimited
					ExpiresAt: 0,  // never expires
				})
			}
		}
	}

	c.Set(cacheKey, apiKeys, cache.DefaultExpiration)
	return apiKeys, nil
}

func resolveScopeAndModel(tenantCode, userID, modelCode string) (string, string) {
	if userID != "" {
		if modelCode != "" {
			return "user:" + userID, modelCode
		}
		return "user:" + userID, "*"
	}
	if tenantCode != "" {
		if modelCode != "" {
			return "tenant:" + tenantCode, modelCode
		}
		return "tenant:" + tenantCode, "*"
	}
	if modelCode != "" {
		return "model:" + modelCode, "*"
	}
	return "global", "*"
}
