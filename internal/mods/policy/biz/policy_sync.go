package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type PolicyRedisSync struct {
	RedisClient           *redis.Client
	PolicyLoadbalanceDAL  *dal.PolicyLoadbalance
	PolicyRouteDAL        *dal.PolicyRoute
	PolicyLimitDAL        *dal.PolicyLimit
	PolicyCircuitBreakDAL *dal.PolicyCircuitBreak
	PolicyInvocationDAL   *dal.PolicyInvocation
	PolicyTaggingDAL      *dal.PolicyTagging
}

// SyncDimension 同步一个维度下的所有策略到 Redis
func (s *PolicyRedisSync) SyncDimension(ctx context.Context, tenantCode, userID, modelCode string) error {
	if s.RedisClient == nil {
		return nil
	}
	if !config.C.Sync.Policies {
		return nil
	}

	// 根据 modelCode 查 modelID
	var modelID string
	if modelCode != "*" && modelCode != "" {
		var model struct {
			ID string
		}
		err := util.GetDB(ctx, s.PolicyLoadbalanceDAL.DB).
			Table("model").
			Select("id").
			Where("model_code = ? AND deleted = '0'", modelCode).
			First(&model).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		modelID = model.ID
	}

	var scopeType string = "global"
	var scopeCode string = ""
	if userID != "" {
		scopeType = "user"
		scopeCode = userID
	} else if tenantCode != "" {
		scopeType = "tenant"
		scopeCode = tenantCode
	}

	redisKey, redisField := resolveRedisKeyAndField(tenantCode, userID, modelCode)

	// 多表级联聚合策略
	policyAgg := &schema.Policy{}

	// 1. loadbalance
	var lbs []schema.PolicyLoadbalance
	if err := util.GetDB(ctx, s.PolicyLoadbalanceDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&lbs).Error; err != nil {
		return err
	}
	if len(lbs) > 0 {
		var form schema.PolicyLoadbalanceForm
		if err := lbs[0].ConvertTo(&form); err == nil {
			policyAgg.LoadBalancePolicy = &form
		}
	}

	// 2. invocation
	var invs []schema.PolicyInvocation
	if err := util.GetDB(ctx, s.PolicyInvocationDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&invs).Error; err != nil {
		return err
	}
	if len(invs) > 0 {
		var form schema.PolicyInvocationForm
		if err := invs[0].ConvertTo(&form); err == nil {
			policyAgg.InvocationPolicy = &form
		}
	}

	// 3. limit
	var limits []schema.PolicyLimit
	if err := util.GetDB(ctx, s.PolicyLimitDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&limits).Error; err != nil {
		return err
	}
	for _, lim := range limits {
		var form schema.PolicyLimitForm
		if err := lim.ConvertTo(&form); err == nil {
			policyAgg.LimitPolicies = append(policyAgg.LimitPolicies, &form)
		}
	}

	// 4. circuit_break
	var cbs []schema.PolicyCircuitBreak
	if err := util.GetDB(ctx, s.PolicyCircuitBreakDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&cbs).Error; err != nil {
		return err
	}
	for _, cb := range cbs {
		var form schema.PolicyCircuitBreakForm
		if err := cb.ConvertTo(&form); err == nil {
			policyAgg.CircuitBreakPolicies = append(policyAgg.CircuitBreakPolicies, &form)
		}
	}

	// 5. tagging
	var taggings []schema.PolicyTagging
	if err := util.GetDB(ctx, s.PolicyTaggingDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&taggings).Error; err != nil {
		return err
	}
	for _, tag := range taggings {
		var form schema.PolicyTaggingForm
		if err := tag.ConvertTo(&form); err == nil {
			policyAgg.TaggingPolicies = append(policyAgg.TaggingPolicies, &form)
		}
	}

	// 6. route
	var routes []schema.PolicyRoute
	if err := util.GetDB(ctx, s.PolicyRouteDAL.DB).
		Preload("Details").
		Where("scope_type = ? AND scope_code = ? AND model_id = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode, modelID).
		Order("priority ASC, created_at DESC").
		Find(&routes).Error; err != nil {
		return err
	}
	for _, r := range routes {
		var form schema.PolicyRouteForm
		if err := r.ConvertTo(&form); err == nil {
			policyAgg.RoutePolicies = append(policyAgg.RoutePolicies, &form)
		}
	}

	// 校验是否为空
	if policyAgg.LoadBalancePolicy == nil &&
		policyAgg.InvocationPolicy == nil &&
		len(policyAgg.LimitPolicies) == 0 &&
		len(policyAgg.RoutePolicies) == 0 &&
		len(policyAgg.CircuitBreakPolicies) == 0 &&
		len(policyAgg.TaggingPolicies) == 0 {

		var existingPolicy map[string]interface{}
		if oldData, err := s.RedisClient.HGet(ctx, redisKey, redisField).Result(); err == nil && oldData != "" {
			_ = json.Unmarshal([]byte(oldData), &existingPolicy)
		}
		if existingPolicy != nil && existingPolicy["billing"] != nil {
			finalMap := map[string]interface{}{
				"billing": existingPolicy["billing"],
			}
			finalJSON, err := json.Marshal(finalMap)
			if err == nil {
				return s.RedisClient.HSet(ctx, redisKey, redisField, string(finalJSON)).Err()
			}
		}
		return s.RedisClient.HDel(ctx, redisKey, redisField).Err()
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(policyAgg)
	if err != nil {
		return err
	}

	var finalMap map[string]interface{}
	if err := json.Unmarshal(jsonData, &finalMap); err != nil {
		return err
	}
	if finalMap == nil {
		finalMap = make(map[string]interface{})
	}

	var existingPolicy map[string]interface{}
	if oldData, err := s.RedisClient.HGet(ctx, redisKey, redisField).Result(); err == nil && oldData != "" {
		_ = json.Unmarshal([]byte(oldData), &existingPolicy)
	}

	if existingPolicy != nil {
		if billingVal, ok := existingPolicy["billing"]; ok {
			finalMap["billing"] = billingVal
		}
	}

	finalJSON, err := json.Marshal(finalMap)
	if err != nil {
		return err
	}

	return s.RedisClient.HSet(ctx, redisKey, redisField, string(finalJSON)).Err()
}

// SyncPolicyChange 当某个具体的策略配置变更时，同步该策略对应的维度
func (s *PolicyRedisSync) SyncPolicyChange(ctx context.Context, scopeType, scopeCode, modelID string) error {
	if s.RedisClient == nil {
		return nil
	}
	if !config.C.Sync.Policies {
		return nil
	}

	var modelCode string
	if modelID != "" {
		var model struct {
			ModelCode string
		}
		err := util.GetDB(ctx, s.PolicyLoadbalanceDAL.DB).
			Table("model").
			Select("model_code").
			Where("id = ? AND deleted = '0'", modelID).
			First(&model).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		modelCode = model.ModelCode
	}

	var tenantCode, userID string
	if scopeType == "user" {
		userID = scopeCode
	} else if scopeType == "tenant" {
		tenantCode = scopeCode
	}

	return s.SyncDimension(ctx, tenantCode, userID, modelCode)
}

func resolveRedisKeyAndField(tenantCode, userID, modelCode string) (string, string) {
	if userID != "" {
		if modelCode != "" && modelCode != "*" {
			return fmt.Sprintf("aigw:policies:user:%s", userID), modelCode
		}
		return fmt.Sprintf("aigw:policies:user:%s", userID), "*"
	}
	if tenantCode != "" {
		if modelCode != "" && modelCode != "*" {
			return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), modelCode
		}
		return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), "*"
	}
	if modelCode != "" && modelCode != "*" {
		return fmt.Sprintf("aigw:policies:model:%s", modelCode), "*"
	}
	return "aigw:policies:global", "*"
}
