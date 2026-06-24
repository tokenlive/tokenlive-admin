package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

const (
	RedisKeyConfigModelVersions = "aigw:config:model_versions"
	RedisKeyConfigAliasPrefix   = "aigw:config:alias:"
)

type ResolvedEndpoint struct {
	ID                 string            `json:"id,omitempty"`
	Description        string            `json:"description,omitempty"`
	RealModel          string            `json:"real_model"`
	ProviderName       string            `json:"provider_name"`
	ProviderProtocol   string            `json:"provider_protocol"`
	APIKey             string            `json:"api_key"`
	URL                string            `json:"url"`
	Timeout            int64             `json:"timeout"` // 毫秒
	MaxRetries         int               `json:"max_retries"`
	Priority           int               `json:"priority"`
	Weight             int               `json:"weight"`
	Headers            map[string]string `json:"headers,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	RequestTypes       []string          `json:"request_types,omitempty"`
	InputPrice         *float64          `json:"input_price,omitempty"`
	OutputPrice        *float64          `json:"output_price,omitempty"`
	CachedPrice        *float64          `json:"cached_price,omitempty"`
	CacheCreationPrice *float64          `json:"cache_creation_price,omitempty"`
}

type ConfigRedisSync struct {
	RedisClient   *redis.Client
	EndpointDAL   *dal.Endpoint
	ModelDAL      *dal.Model
	ModelAliasDAL *dal.ModelAlias
}

func (s *ConfigRedisSync) SyncModelByCode(ctx context.Context, modelCode string) error {
	if s.RedisClient == nil || modelCode == "" {
		return nil
	}

	// 同步模型默认费率策略到 Redis (aigw:policies:model:<model_code> -> "*")
	var model schema.Model
	db := util.GetDB(ctx, s.ModelDAL.DB)
	err := db.Where("model_code = ? AND deleted = '0'", modelCode).First(&model).Error
	policyKey := "aigw:policies:model:" + modelCode
	if err == nil {
		if model.Enabled == 1 {
			// 先读取已有的配置，防止冲掉 policy_binding 绑定的其它策略
			var existingPolicy map[string]interface{}
			if oldData, err := s.RedisClient.HGet(ctx, policyKey, "*").Result(); err == nil && oldData != "" {
				_ = json.Unmarshal([]byte(oldData), &existingPolicy)
			}
			if existingPolicy == nil {
				existingPolicy = make(map[string]interface{})
			}

			// 更新计费价格策略
			existingPolicy["billing"] = map[string]interface{}{
				"input_price":          model.InputPrice,
				"output_price":         model.OutputPrice,
				"cached_price":         model.CachedPrice,
				"cache_creation_price": model.CacheCreationPrice,
			}

			policyData, err := json.Marshal(existingPolicy)
			if err == nil {
				_ = s.RedisClient.HSet(ctx, policyKey, "*", string(policyData)).Err()
			}
		} else {
			_ = s.RedisClient.Del(ctx, policyKey).Err()
		}
	}

	// 1. Query endpoints associated with the model code, preloading Model and Provider relations.
	endpoints, err := s.queryResolvedEndpointsByCode(ctx, modelCode)
	if err != nil {
		return err
	}

	redisKey := "aigw:config:endpoints:" + modelCode

	// 2. If no active endpoints exist, remove the key and its Hash version
	// 注意：不调用 incrementVersion，避免重新创建已删除的 version 记录
	if len(endpoints) == 0 {
		_ = s.RedisClient.Del(ctx, redisKey).Err()
		_ = s.RedisClient.HDel(ctx, RedisKeyConfigModelVersions, modelCode).Err()
		return nil
	}

	// 3. Map to ResolvedEndpoint structures applying inheritance rules
	var resolvedList []ResolvedEndpoint
	for _, ep := range endpoints {
		if ep.Model == nil || ep.Provider == nil {
			continue
		}

		// Inheritance: API Keys (endpoint > provider)
		var apiKeys []string
		if ep.ApiKey != "" {
			apiKeys = []string{ep.ApiKey}
		} else {
			apiKeys = ep.Provider.GetApiKeys()
		}
		if len(apiKeys) == 0 {
			apiKeys = []string{""}
		}

		// Inheritance: Protocol (endpoint > provider)
		protocol := ep.Protocol
		if protocol == "" {
			protocol = ep.Provider.Protocol
		}

		// Defaults in milliseconds/counts
		var timeout int64 = 60000 // default 60s
		var maxRetries int = 3    // default 3

		// Parse metadata overrides
		var metadataMap map[string]interface{}
		var stringMetadataMap map[string]string
		if len(ep.Metadata) > 0 {
			_ = json.Unmarshal(ep.Metadata, &metadataMap)
			_ = json.Unmarshal(ep.Metadata, &stringMetadataMap)
		}

		if metadataMap != nil {
			if v, ok := metadataMap["timeout"]; ok {
				switch val := v.(type) {
				case float64:
					timeout = int64(val)
				case string:
					if parsed, err := strconv.ParseInt(val, 10, 64); err == nil {
						timeout = parsed
					}
				}
			}
			if v, ok := metadataMap["max_retries"]; ok {
				switch val := v.(type) {
				case float64:
					maxRetries = int(val)
				case string:
					if parsed, err := strconv.Atoi(val); err == nil {
						maxRetries = parsed
					}
				}
			}
		}

		realModel := ep.RealModel
		if realModel == "" {
			realModel = ep.Model.ModelCode
		}

		var headersMap map[string]string
		if len(ep.Headers) > 0 {
			_ = json.Unmarshal(ep.Headers, &headersMap)
		}

		var apis []string
		if ep.Model != nil && ep.Model.RequestTypes != "" {
			_ = json.Unmarshal([]byte(ep.Model.RequestTypes), &apis)
		}
		if len(apis) == 0 {
			return fmt.Errorf("model %s has no request_types configured", modelCode)
		}
		apis = normalizeRequestTypesForProtocol(protocol, apis)
		if len(apis) == 0 {
			return fmt.Errorf("model %s has no request_types compatible with protocol %s", modelCode, protocol)
		}

		// 价格继承前置 (Admin 写入 Redis 缓存时完成继承)
		var (
			inputPriceVal         = ep.Model.InputPrice
			outputPriceVal        = ep.Model.OutputPrice
			cachedPriceVal        = ep.Model.CachedPrice
			cacheCreationPriceVal = ep.Model.CacheCreationPrice
		)

		if ep.InputPrice != nil {
			inputPriceVal = *ep.InputPrice
		}
		if ep.OutputPrice != nil {
			outputPriceVal = *ep.OutputPrice
		}
		if ep.CachedPrice != nil {
			cachedPriceVal = *ep.CachedPrice
		} else if ep.InputPrice != nil {
			cachedPriceVal = *ep.InputPrice
		}
		if ep.CacheCreationPrice != nil {
			cacheCreationPriceVal = *ep.CacheCreationPrice
		} else if ep.InputPrice != nil {
			cacheCreationPriceVal = *ep.InputPrice
		}

		for _, apiKey := range apiKeys {
			resolvedList = append(resolvedList, ResolvedEndpoint{
				ID:                 ep.ID,
				Description:        ep.Description,
				RealModel:          realModel,
				ProviderName:       ep.Provider.Name,
				ProviderProtocol:   protocol,
				APIKey:             apiKey,
				URL:                ep.URL,
				Timeout:            timeout,
				MaxRetries:         maxRetries,
				Priority:           ep.Priority,
				Weight:             ep.Weight,
				Headers:            headersMap,
				Metadata:           stringMetadataMap,
				RequestTypes:       apis,
				InputPrice:         &inputPriceVal,
				OutputPrice:        &outputPriceVal,
				CachedPrice:        &cachedPriceVal,
				CacheCreationPrice: &cacheCreationPriceVal,
			})
		}
	}

	// 4. Serialize to JSON and write to Redis
	jsonData, err := json.Marshal(resolvedList)
	if err != nil {
		return err
	}

	if err := s.RedisClient.Set(ctx, redisKey, string(jsonData), 0).Err(); err != nil {
		return err
	}

	// 5. Increment version
	return s.incrementVersion(ctx, modelCode)
}

func normalizeRequestTypesForProtocol(protocol string, requestTypes []string) []string {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	var result []string
	seen := make(map[string]bool, len(requestTypes))

	add := func(rt string) {
		rt = strings.TrimSpace(rt)
		if rt == "" || seen[rt] {
			return
		}
		seen[rt] = true
		result = append(result, rt)
	}

	for _, rt := range requestTypes {
		switch protocol {
		case "anthropic":
			switch rt {
			case "messages", "chat_completion":
				add("messages")
			}
		case "joycode":
			switch rt {
			case "chat_completion", "responses":
				add(rt)
			}
		case "openai":
			switch rt {
			case "chat_completion", "embedding", "responses", "messages":
				add(rt)
			}
		default:
			add(rt)
		}
	}

	return result
}

// SyncAlias synchronizes a single alias mapping to Redis: aigw:config:alias:{alias} → modelCode.
func (s *ConfigRedisSync) SyncAlias(ctx context.Context, alias string, modelCode string) error {
	if s.RedisClient == nil || alias == "" || modelCode == "" {
		return nil
	}
	key := RedisKeyConfigAliasPrefix + alias
	return s.RedisClient.Set(ctx, key, modelCode, 0).Err()
}

// DeleteAlias removes a single alias mapping from Redis.
func (s *ConfigRedisSync) DeleteAlias(ctx context.Context, alias string) error {
	if s.RedisClient == nil || alias == "" {
		return nil
	}
	key := RedisKeyConfigAliasPrefix + alias
	return s.RedisClient.Del(ctx, key).Err()
}

// SyncAliasesByModelId re-syncs all aliases for a given model ID to Redis.
// If the model is disabled (enabled=0), it deletes all alias keys instead.
func (s *ConfigRedisSync) SyncAliasesByModelId(ctx context.Context, modelId string, modelCode string, enabled int) error {
	if s.RedisClient == nil || modelId == "" || s.ModelAliasDAL == nil {
		return nil
	}
	aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelId)
	if err != nil {
		return err
	}
	for _, a := range aliases {
		if enabled == 1 {
			_ = s.SyncAlias(ctx, a.Alias, modelCode)
		} else {
			_ = s.DeleteAlias(ctx, a.Alias)
		}
	}
	return nil
}

// deleteAliasesByModelId deletes all alias Redis keys for a given model ID.
func (s *ConfigRedisSync) deleteAliasesByModelId(ctx context.Context, modelId string) error {
	if s.RedisClient == nil || modelId == "" || s.ModelAliasDAL == nil {
		return nil
	}
	aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelId)
	if err != nil {
		return err
	}
	for _, a := range aliases {
		_ = s.DeleteAlias(ctx, a.Alias)
	}
	return nil
}

// SyncProviderID synchronizes all models affected by the provider ID.
func (s *ConfigRedisSync) SyncProviderID(ctx context.Context, providerID string) error {
	if s.RedisClient == nil || providerID == "" {
		return nil
	}

	modelCodes, err := s.GetModelCodesByProvider(ctx, providerID)
	if err != nil {
		return err
	}

	for _, code := range modelCodes {
		_ = s.SyncModelByCode(ctx, code)
	}
	return nil
}

// GetModelCodesByProvider queries model codes referencing the given provider ID.
func (s *ConfigRedisSync) GetModelCodesByProvider(ctx context.Context, providerID string) ([]string, error) {
	endpointTable := config.C.FormatTableName("endpoint")
	modelTable := config.C.FormatTableName("model")

	var modelCodes []string
	err := util.GetDB(ctx, s.EndpointDAL.DB).
		Table(endpointTable).
		Joins("JOIN "+modelTable+" ON "+endpointTable+".model_id = "+modelTable+".id").
		Where(endpointTable+".provider_id = ?", providerID).
		Where(endpointTable+".deleted = '0'").
		Where(modelTable+".deleted = '0'").
		Pluck(modelTable+".model_code", &modelCodes).Error

	return modelCodes, err
}

func (s *ConfigRedisSync) incrementVersion(ctx context.Context, modelCode string) error {
	if s.RedisClient == nil {
		return nil
	}
	return s.RedisClient.HIncrBy(ctx, RedisKeyConfigModelVersions, modelCode, 1).Err()
}

func (s *ConfigRedisSync) queryResolvedEndpointsByCode(ctx context.Context, modelCode string) (schema.Endpoints, error) {
	endpointTable := config.C.FormatTableName("endpoint")
	modelTable := config.C.FormatTableName("model")
	providerTable := config.C.FormatTableName("provider")

	var list schema.Endpoints
	db := util.GetDB(ctx, s.EndpointDAL.DB).
		Table(endpointTable).
		Preload("Model").
		Preload("Provider").
		Joins("JOIN "+modelTable+" ON "+endpointTable+".model_id = "+modelTable+".id").
		Joins("JOIN "+providerTable+" ON "+endpointTable+".provider_id = "+providerTable+".id").
		Where(modelTable+".model_code = ?", modelCode).
		Where(endpointTable + ".enabled = 1").
		Where(modelTable + ".enabled = 1").
		Where(providerTable + ".enabled = 1").
		Where(endpointTable + ".deleted = '0'").
		Where(modelTable + ".deleted = '0'").
		Where(providerTable + ".deleted = '0'").
		Order(endpointTable + ".priority ASC, " + endpointTable + ".weight DESC")

	if err := db.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// SyncModelCodeChange handles updating tenant-related cache keys in Redis when a model's code changes.
func (s *ConfigRedisSync) SyncModelCodeChange(ctx context.Context, modelID, oldModelCode, newModelCode string) error {
	if s.RedisClient == nil || modelID == "" || oldModelCode == "" || newModelCode == "" || oldModelCode == newModelCode {
		return nil
	}

	// 1. 查询所有关联此 modelID 的租户编码
	var tenantCodes []string
	tenantModelTable := config.C.FormatTableName("tenant_model")
	db := util.GetDB(ctx, s.EndpointDAL.DB)
	err := db.Table(tenantModelTable).
		Where("model_id = ?", modelID).
		Pluck("tenant_code", &tenantCodes).Error
	if err != nil {
		return err
	}

	for _, tenantCode := range tenantCodes {
		// 2. 更新 aigw:tenant:{tenantCode}:models 集合，移除旧 code，写入新 code
		oldModelsKey := "aigw:tenant:" + tenantCode + ":models"
		_ = s.RedisClient.SRem(ctx, oldModelsKey, oldModelCode).Err()
		_ = s.RedisClient.SAdd(ctx, oldModelsKey, newModelCode).Err()


		// 4. 迁移 aigw:tenant:{tenantCode}:model:{modelCode}:endpoints 集合（新）
		oldEndpointsKey := "aigw:tenant:" + tenantCode + ":model:" + oldModelCode + ":endpoints"
		newEndpointsKey := "aigw:tenant:" + tenantCode + ":model:" + newModelCode + ":endpoints"

		// 获取并迁移端点白名单
		epMembers, err := s.RedisClient.SMembers(ctx, oldEndpointsKey).Result()
		if err == nil && len(epMembers) > 0 {
			var interfaces []interface{}
			for _, m := range epMembers {
				interfaces = append(interfaces, m)
			}
			_ = s.RedisClient.SAdd(ctx, newEndpointsKey, interfaces...).Err()
		}
		// 删除低端点白名单缓存
		_ = s.RedisClient.Del(ctx, oldEndpointsKey).Err()
	}

	// 5. 更新该模型所有别名的 Redis value 为新 modelCode
	if s.ModelAliasDAL != nil {
		aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelID)
		if err == nil {
			for _, a := range aliases {
				_ = s.SyncAlias(ctx, a.Alias, newModelCode)
			}
		}
	}

	return nil
}

// SyncModelDisable handles removing model code from associated tenants' allowed model sets and deleting provider whitelist caches.
func (s *ConfigRedisSync) SyncModelDisable(ctx context.Context, modelID, modelCode string, tenantCodes ...string) error {
	if s.RedisClient == nil || modelID == "" || modelCode == "" {
		return nil
	}

	var resolvedTenants []string
	if len(tenantCodes) > 0 {
		resolvedTenants = tenantCodes
	} else {
		// 1. 查询所有关联此 modelID 的租户编码
		tenantModelTable := config.C.FormatTableName("tenant_model")
		db := util.GetDB(ctx, s.EndpointDAL.DB)
		err := db.Table(tenantModelTable).
			Where("model_id = ?", modelID).
			Pluck("tenant_code", &resolvedTenants).Error
		if err != nil {
			return err
		}
	}

	for _, tenantCode := range resolvedTenants {
		// 2. 从 aigw:tenant:{tenantCode}:models 集合中移除该 modelCode
		modelsKey := "aigw:tenant:" + tenantCode + ":models"
		_ = s.RedisClient.SRem(ctx, modelsKey, modelCode).Err()


		// 4. 删除 endpoints 白名单缓存（新）
		endpointsKey := "aigw:tenant:" + tenantCode + ":model:" + modelCode + ":endpoints"
		_ = s.RedisClient.Del(ctx, endpointsKey).Err()
	}

	// 5. 清理该模型所有别名的 Redis key
	_ = s.deleteAliasesByModelId(ctx, modelID)

	return nil
}

// SyncModelEnable handles adding model code back to associated tenants' allowed model sets and rebuilding endpoint whitelist caches.
func (s *ConfigRedisSync) SyncModelEnable(ctx context.Context, modelID, modelCode string) error {
	if s.RedisClient == nil || modelID == "" || modelCode == "" {
		return nil
	}

	// 1. 查询所有关联此 modelID 的租户编码
	var tenantCodes []string
	tenantModelTable := config.C.FormatTableName("tenant_model")
	db := util.GetDB(ctx, s.EndpointDAL.DB)
	err := db.Table(tenantModelTable).
		Where("model_id = ?", modelID).
		Pluck("tenant_code", &tenantCodes).Error
	if err != nil {
		return err
	}

	tenantEndpointTable := config.C.FormatTableName("tenant_endpoint")
	endpointTable := config.C.FormatTableName("endpoint")

	for _, tenantCode := range tenantCodes {
		// 2. 将 modelCode 重新加回到 aigw:tenant:{tenantCode}:models 集合中
		modelsKey := "aigw:tenant:" + tenantCode + ":models"
		_ = s.RedisClient.SAdd(ctx, modelsKey, modelCode).Err()

		// 3. 重新同步该租户此模型的 endpoints 限制白名单（新）
		endpointsKey := "aigw:tenant:" + tenantCode + ":model:" + modelCode + ":endpoints"

		var endpointIDs []string
		err = db.Table(tenantEndpointTable+" AS te").
			Select("te.endpoint_id").
			Joins("JOIN "+endpointTable+" AS ep ON te.endpoint_id = ep.id AND ep.deleted = '0'").
			Where("te.tenant_code = ? AND ep.model_id = ?", tenantCode, modelID).
			Pluck("te.endpoint_id", &endpointIDs).Error

		if err == nil {
			_ = s.RedisClient.Del(ctx, endpointsKey).Err()
			if len(endpointIDs) > 0 {
				var members []interface{}
				for _, id := range endpointIDs {
					members = append(members, id)
				}
				_ = s.RedisClient.SAdd(ctx, endpointsKey, members...).Err()
			}
		}
	}

	// 5. 重新同步该模型所有别名的 Redis key
	_ = s.SyncAliasesByModelId(ctx, modelID, modelCode, 1)

	return nil
}

// SyncAllToRedis synchronizes all active endpoints, model versions, and tenant bindings to Redis.
func (s *ConfigRedisSync) SyncAllToRedis(ctx context.Context) error {
	if s.RedisClient == nil {
		return nil
	}

	db := util.GetDB(ctx, s.ModelDAL.DB)

	// 1. 查询所有未删除的模型 (deleted = '0')
	var models []schema.Model
	err := db.Table(config.C.FormatTableName("model")).
		Where("deleted = '0'").
		Find(&models).Error
	if err != nil {
		return err
	}

	// 2. 查询所有未删除的租户编码列表
	var tenantCodes []string
	err = db.Table(config.C.FormatTableName("tenant")).
		Where("deleted = '0'").
		Pluck("code", &tenantCodes).Error
	if err != nil {
		return err
	}

	// 3. 查询当前所有启用的租户-模型绑定关系
	type tenantBinding struct {
		TenantCode string `gorm:"column:tenant_code"`
		ModelID    string `gorm:"column:model_id"`
		ModelCode  string `gorm:"column:model_code"`
	}
	var bindings []tenantBinding
	tenantModelTable := config.C.FormatTableName("tenant_model")
	modelTable := config.C.FormatTableName("model")
	err = db.Table(tenantModelTable).
		Joins("JOIN " + modelTable + " m ON " + tenantModelTable + ".model_id = m.id").
		Where("m.enabled = 1 AND m.deleted = '0'").
		Select(tenantModelTable + ".tenant_code, " + tenantModelTable + ".model_id, m.model_code").
		Find(&bindings).Error
	if err != nil {
		return err
	}

	// 4. 将绑定关系根据 tenant_code 进行内存分组
	tenantToModels := make(map[string][]string)
	for _, b := range bindings {
		tenantToModels[b.TenantCode] = append(tenantToModels[b.TenantCode], b.ModelCode)
	}

	// 5. 对每一个租户进行原子覆盖 models 缓存，以及 endpoints 限制白名单
	tenantEndpointTable := config.C.FormatTableName("tenant_endpoint")
	endpointTable := config.C.FormatTableName("endpoint")

	for _, tenantCode := range tenantCodes {
		modelsKey := "aigw:tenant:" + tenantCode + ":models"
		allowedModels := tenantToModels[tenantCode]

		if len(allowedModels) > 0 {
			// 原子替换 models 集合：先写入 tmp，再 Rename 覆盖
			tmpModelsKey := modelsKey + ":tmp"
			_ = s.RedisClient.Del(ctx, tmpModelsKey).Err()

			var members []interface{}
			for _, code := range allowedModels {
				members = append(members, code)
			}
			if err := s.RedisClient.SAdd(ctx, tmpModelsKey, members...).Err(); err != nil {
				return err
			}
			if err := s.RedisClient.Rename(ctx, tmpModelsKey, modelsKey).Err(); err != nil {
				return err
			}
		} else {
			// 若当前租户没有任何可用模型绑定，则直接 Del 清理
			_ = s.RedisClient.Del(ctx, modelsKey).Err()
		}

		// 为租户所绑定的每一个模型，同步其 endpoints 白名单
		for _, m := range models {
			if m.Enabled == 1 {
				// 同步 endpoints 白名单（新）
				endpointsKey := "aigw:tenant:" + tenantCode + ":model:" + m.ModelCode + ":endpoints"

				var endpointIDs []string
				err = db.Table(tenantEndpointTable+" AS te").
					Select("te.endpoint_id").
					Joins("JOIN "+endpointTable+" AS ep ON te.endpoint_id = ep.id AND ep.deleted = '0'").
					Where("te.tenant_code = ? AND ep.model_id = ?", tenantCode, m.ID).
					Pluck("te.endpoint_id", &endpointIDs).Error

				if err == nil {
					if len(endpointIDs) > 0 {
						// 原子替换 endpoints 集合
						tmpEndpointsKey := endpointsKey + ":tmp"
						_ = s.RedisClient.Del(ctx, tmpEndpointsKey).Err()

						var members []interface{}
						for _, id := range endpointIDs {
							members = append(members, id)
						}
						if err := s.RedisClient.SAdd(ctx, tmpEndpointsKey, members...).Err(); err != nil {
							return err
						}
						if err := s.RedisClient.Rename(ctx, tmpEndpointsKey, endpointsKey).Err(); err != nil {
							return err
						}
					} else {
						// 若无限制，物理 Del 白名单
						_ = s.RedisClient.Del(ctx, endpointsKey).Err()
					}
				}
			}
		}
	}

	// 6. 遍历模型：启用的模型执行 SyncModelByCode 刷新端点，禁用的模型执行清理
	for _, m := range models {
		if m.Enabled == 1 {
			if err := s.SyncModelByCode(ctx, m.ModelCode); err != nil {
				return err
			}
		} else {
			redisKey := "aigw:config:endpoints:" + m.ModelCode
			_ = s.RedisClient.Del(ctx, redisKey).Err()
			_ = s.RedisClient.HDel(ctx, RedisKeyConfigModelVersions, m.ModelCode).Err()
			_ = s.RedisClient.Del(ctx, "aigw:policies:model:"+m.ModelCode).Err()

			// 物理清理各租户中被禁用模型的授权与 endpoints 缓存
			for _, tenantCode := range tenantCodes {
				modelsKey := "aigw:tenant:" + tenantCode + ":models"
				_ = s.RedisClient.SRem(ctx, modelsKey, m.ModelCode).Err()

				endpointsKey := "aigw:tenant:" + tenantCode + ":model:" + m.ModelCode + ":endpoints"
				_ = s.RedisClient.Del(ctx, endpointsKey).Err()
			}
		}
	}

	// 7. 全量同步所有别名映射到 Redis
	if s.ModelAliasDAL != nil {
		var allAliases schema.ModelAliases
		aliasTable := config.C.FormatTableName("model_alias")
		err = db.Table(aliasTable).
			Where("deleted = '0'").
			Find(&allAliases).Error
		if err != nil {
			return err
		}
		// 构建 modelID → (modelCode, enabled) 映射
		modelMap := make(map[string]schema.Model)
		for _, m := range models {
			modelMap[m.ID] = m
		}
		for _, a := range allAliases {
			if m, ok := modelMap[a.ModelId]; ok && m.Enabled == 1 {
				_ = s.SyncAlias(ctx, a.Alias, m.ModelCode)
			} else {
				_ = s.DeleteAlias(ctx, a.Alias)
			}
		}
	}

	// 8. Sync all active user API keys to Redis
	// 8a. Pre-load user→tenant mapping
	userTable := config.C.FormatTableName("user")
	var userTenants []struct {
		ID     string `gorm:"column:id"`
		Tenant string `gorm:"column:tenant"`
	}
	_ = db.Table(userTable).Find(&userTenants).Error
	userTenantMap := make(map[string]string, len(userTenants))
	for _, u := range userTenants {
		userTenantMap[u.ID] = u.Tenant
	}

	var userKeys []struct {
		UserID    string     `gorm:"column:user_id"`
		APIKey    string     `gorm:"column:api_key"`
		Status    int        `gorm:"column:status"`
		Quota     int64      `gorm:"column:quota"`
		ExpiresAt *time.Time `gorm:"column:expires_at"`
	}
	userApiKeyTable := config.C.FormatTableName("user_api_key")
	err = db.Table(userApiKeyTable).Where("deleted = '0'").Find(&userKeys).Error
	if err == nil {
		for _, key := range userKeys {
			if key.APIKey == "" {
				continue
			}
			redisKey := "aigw:apikey:" + key.APIKey
			var expiresAtVal int64 = 0
			if key.ExpiresAt != nil {
				expiresAtVal = key.ExpiresAt.Unix()
			}
			fields := map[string]interface{}{
				"user_id":     key.UserID,
				"user_tenant": userTenantMap[key.UserID],
				"status":      key.Status,
				"quota":       key.Quota,
				"expires_at":  expiresAtVal,
			}
			_ = s.RedisClient.HSet(ctx, redisKey, fields).Err()
		}
	}

	// 9. Sync all active tenant API keys to Redis
	var tenantKeys []struct {
		Code   string `gorm:"column:code"`
		APIKey string `gorm:"column:api_key"`
		Status string `gorm:"column:status"`
	}
	tenantTable := config.C.FormatTableName("tenant")
	err = db.Table(tenantTable).Where("deleted = '0' AND api_key IS NOT NULL AND api_key != ''").Find(&tenantKeys).Error
	if err == nil {
		for _, t := range tenantKeys {
			redisKey := "aigw:apikey:" + t.APIKey
			if t.Status == "activated" {
				fields := map[string]interface{}{
					"tenant":     t.Code,
					"status":     1,
					"quota":      -1,
					"expires_at": 0,
				}
				_ = s.RedisClient.HSet(ctx, redisKey, fields).Err()
			} else {
				_ = s.RedisClient.Del(ctx, redisKey).Err()
			}
		}
	}

	return nil
}
