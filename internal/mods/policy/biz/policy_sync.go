package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

type PolicyRedisSync struct {
	RedisClient           *redis.Client
	PolicyBindingDAL      *dal.PolicyBinding
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

	// 1. 获取该维度下所有启用的有效绑定 (未删除)
	var bindings []schema.PolicyBinding
	db := util.GetDB(ctx, s.PolicyBindingDAL.DB).
		Model(new(schema.PolicyBinding)).
		Where("tenant_code = ? AND user_id = ? AND model_code = ?", tenantCode, userID, modelCode).
		Where("enabled = 1 AND deleted = '0'")
	if err := db.Find(&bindings).Error; err != nil {
		return err
	}

	redisKey, redisField := resolveRedisKeyAndField(tenantCode, userID, modelCode)

	// 2. 如果无任何有效绑定，清理旧策略，但保留计费 billing
	if len(bindings) == 0 {
		var existingPolicy map[string]interface{}
		if oldData, err := s.RedisClient.HGet(ctx, redisKey, redisField).Result(); err == nil && oldData != "" {
			_ = json.Unmarshal([]byte(oldData), &existingPolicy)
		}
		if existingPolicy != nil && existingPolicy["billing"] != nil {
			// 只保留 billing 并返回
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

	// 3. 多表级联聚合策略
	policyAgg := &schema.Policy{}

	for _, b := range bindings {
		switch b.PolicyType {
		case "loadbalance":
			var lb schema.PolicyLoadbalance
			err := util.GetDB(ctx, s.PolicyLoadbalanceDAL.DB).
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&lb).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyLoadbalanceForm
				if err := lb.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.LoadBalancePolicy = &form
			}

		case "invocation":
			var inv schema.PolicyInvocation
			err := util.GetDB(ctx, s.PolicyInvocationDAL.DB).
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&inv).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyInvocationForm
				if err := inv.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.InvocationPolicy = &form
			}

		case "limit":
			var lim schema.PolicyLimit
			err := util.GetDB(ctx, s.PolicyLimitDAL.DB).
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&lim).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyLimitForm
				if err := lim.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.LimitPolicies = append(policyAgg.LimitPolicies, &form)
			}

		case "circuit_break":
			var cb schema.PolicyCircuitBreak
			err := util.GetDB(ctx, s.PolicyCircuitBreakDAL.DB).
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&cb).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyCircuitBreakForm
				if err := cb.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.CircuitBreakPolicies = append(policyAgg.CircuitBreakPolicies, &form)
			}

		case "tagging":
			var tag schema.PolicyTagging
			err := util.GetDB(ctx, s.PolicyTaggingDAL.DB).
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&tag).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyTaggingForm
				if err := tag.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.TaggingPolicies = append(policyAgg.TaggingPolicies, &form)
			}

		case "route":
			var r schema.PolicyRoute
			err := util.GetDB(ctx, s.PolicyRouteDAL.DB).
				Preload("Details").
				Where("id = ? AND enabled = 1 AND deleted = '0'", b.PolicyID).
				First(&r).Error
			if err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
			} else {
				var form schema.PolicyRouteForm
				if err := r.ConvertTo(&form); err != nil {
					return err
				}
				policyAgg.RoutePolicies = append(policyAgg.RoutePolicies, &form)
			}
		}
	}

	// 再次校验是否为空
	if policyAgg.LoadBalancePolicy == nil &&
		policyAgg.InvocationPolicy == nil &&
		len(policyAgg.LimitPolicies) == 0 &&
		len(policyAgg.RoutePolicies) == 0 &&
		len(policyAgg.CircuitBreakPolicies) == 0 &&
		len(policyAgg.TaggingPolicies) == 0 {
		return s.RedisClient.HDel(ctx, redisKey, redisField).Err()
	}

	// 4. 序列化为 JSON
	jsonData, err := json.Marshal(policyAgg)
	if err != nil {
		return err
	}

	// 5. 先读取原有数据，防止将 model 默认计费等非 policy_binding 管理的信息冲掉
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

// SyncPolicyChange 当某个具体的策略配置变更时，反查所有关联维度并同步
func (s *PolicyRedisSync) SyncPolicyChange(ctx context.Context, policyType, policyID string) error {
	if s.RedisClient == nil || policyID == "" {
		return nil
	}

	// 1. 从绑定表中查出所有关联了该 policyID 且未被软删除的维度记录
	var bindings []schema.PolicyBinding
	db := util.GetDB(ctx, s.PolicyBindingDAL.DB).
		Model(new(schema.PolicyBinding)).
		Where("policy_type = ? AND policy_id = ?", policyType, policyID).
		Where("deleted = '0'")
	if err := db.Find(&bindings).Error; err != nil {
		return err
	}

	// 2. 去重并依次同步各个关联的维度
	seen := make(map[string]bool)
	var errs []error
	for _, b := range bindings {
		dimKey := fmt.Sprintf("%s:%s:%s", b.TenantCode, b.UserID, b.ModelCode)
		if seen[dimKey] {
			continue
		}
		seen[dimKey] = true

		if err := s.SyncDimension(ctx, b.TenantCode, b.UserID, b.ModelCode); err != nil {
			errs = append(errs, fmt.Errorf("sync dimension %s failed: %w", dimKey, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("sync policy change failed: %v", errs)
	}

	return nil
}

func resolveRedisKeyAndField(tenantCode, userID, modelCode string) (string, string) {
	if userID != "" {
		if modelCode != "" {
			return fmt.Sprintf("aigw:policies:user:%s", userID), modelCode
		}
		return fmt.Sprintf("aigw:policies:user:%s", userID), "*"
	}
	if tenantCode != "" {
		if modelCode != "" {
			return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), modelCode
		}
		return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), "*"
	}
	if modelCode != "" {
		return fmt.Sprintf("aigw:policies:model:%s", modelCode), "*"
	}
	return "aigw:policies:global", "*"
}
