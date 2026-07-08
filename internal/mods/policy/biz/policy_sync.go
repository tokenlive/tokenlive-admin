package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const policyRedisSyncLogTag = "policy_redis_sync"

type PolicyRedisSync struct {
	RedisClient           *redis.Client
	PolicyLoadbalanceDAL  *dal.PolicyLoadbalance
	PolicyRouteDAL        *dal.PolicyRoute
	PolicyLimitDAL        *dal.PolicyLimit
	PolicyCircuitBreakDAL *dal.PolicyCircuitBreak
	PolicyInvocationDAL   *dal.PolicyInvocation
	PolicyTaggingDAL      *dal.PolicyTagging
	ModelDAL              *resourceDal.Model
}

// SyncDimension 同步一个维度下的所有策略到 Redis
func (s *PolicyRedisSync) SyncDimension(ctx context.Context, tenantCode, userID, modelCode string) (err error) {
	start := time.Now()
	action := "init"
	scopeType := "global"
	scopeCode := ""
	redisKey := ""
	redisField := ""
	modelID := ""
	counts := policyRedisSyncCounts{}
	defer func() {
		fields := policyRedisSyncFields(tenantCode, userID, modelCode, modelID, scopeType, scopeCode, redisKey, redisField, action, counts, time.Since(start))
		if err != nil {
			fields = append(fields, zap.Error(err))
			policyRedisSyncLogger(ctx).Error("Policy Redis sync failed", fields...)
			return
		}
		policyRedisSyncLogger(ctx).Info("Policy Redis sync completed", fields...)
	}()

	if s.RedisClient == nil {
		action = "skip_nil_redis_client"
		return nil
	}
	if !config.C.Sync.Policies {
		action = "skip_policy_sync_disabled"
		return nil
	}

	// 根据 modelCode 查 modelID
	if modelCode != "*" && modelCode != "" {
		modelID, err = s.lookupModelIDByCode(ctx, modelCode)
		if err != nil {
			return err
		}
	}

	if userID != "" {
		scopeType = "user"
		scopeCode = userID
	} else if tenantCode != "" {
		scopeType = "tenant"
		scopeCode = tenantCode
	}

	var ok bool
	redisKey, redisField, ok = resolveRedisKeyAndField(tenantCode, userID, modelCode)
	if !ok {
		action = "skip_no_dimension"
		return nil
	}

	// 多表级联聚合策略
	policyAgg := &schema.Policy{}

	// 1. loadbalance
	var lbs []schema.PolicyLoadbalance
	dbLbs := util.GetDB(ctx, s.PolicyLoadbalanceDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbLbs = buildModelIDWhere(dbLbs, modelID)
	if err := dbLbs.Order("priority ASC, created_at DESC").Find(&lbs).Error; err != nil {
		return err
	}
	counts.LoadBalance = len(lbs)
	if len(lbs) > 0 {
		var form schema.PolicyLoadbalanceForm
		if err := lbs[0].ConvertTo(&form); err == nil {
			policyAgg.LoadBalancePolicy = &form
		}
	}

	// 2. invocation
	var invs []schema.PolicyInvocation
	dbInvs := util.GetDB(ctx, s.PolicyInvocationDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbInvs = buildModelIDWhere(dbInvs, modelID)
	if err := dbInvs.Order("priority ASC, created_at DESC").Find(&invs).Error; err != nil {
		return err
	}
	counts.Invocation = len(invs)
	if len(invs) > 0 {
		var form schema.PolicyInvocationForm
		if err := invs[0].ConvertTo(&form); err == nil {
			policyAgg.InvocationPolicy = &form
		}
	}

	// 3. limit
	var limits []schema.PolicyLimit
	dbLimits := util.GetDB(ctx, s.PolicyLimitDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbLimits = buildModelIDWhere(dbLimits, modelID)
	if err := dbLimits.Order("priority ASC, created_at DESC").Find(&limits).Error; err != nil {
		return err
	}
	counts.Limit = len(limits)
	for _, lim := range limits {
		var form schema.PolicyLimitForm
		if err := lim.ConvertTo(&form); err == nil {
			policyAgg.LimitPolicies = append(policyAgg.LimitPolicies, &form)
		}
	}

	// 4. circuit_break
	var cbs []schema.PolicyCircuitBreak
	dbCbs := util.GetDB(ctx, s.PolicyCircuitBreakDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbCbs = buildModelIDWhere(dbCbs, modelID)
	if err := dbCbs.Order("priority ASC, created_at DESC").Find(&cbs).Error; err != nil {
		return err
	}
	counts.CircuitBreak = len(cbs)
	for _, cb := range cbs {
		var form schema.PolicyCircuitBreakForm
		if err := cb.ConvertTo(&form); err == nil {
			policyAgg.CircuitBreakPolicies = append(policyAgg.CircuitBreakPolicies, &form)
		}
	}

	// 5. tagging
	var taggings []schema.PolicyTagging
	dbTaggings := util.GetDB(ctx, s.PolicyTaggingDAL.DB).
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbTaggings = buildModelIDWhere(dbTaggings, modelID)
	if err := dbTaggings.Order("priority ASC, created_at DESC").Find(&taggings).Error; err != nil {
		return err
	}
	counts.Tagging = len(taggings)
	for _, tag := range taggings {
		var form schema.PolicyTaggingForm
		if err := tag.ConvertTo(&form); err == nil {
			policyAgg.TaggingPolicies = append(policyAgg.TaggingPolicies, &form)
		}
	}

	// 6. route
	var routes []schema.PolicyRoute
	dbRoutes := util.GetDB(ctx, s.PolicyRouteDAL.DB).
		Preload("Details").
		Where("scope_type = ? AND scope_code = ? AND enabled = 1 AND deleted = '0'", scopeType, scopeCode)
	dbRoutes = buildModelIDWhere(dbRoutes, modelID)
	if err := dbRoutes.Order("priority ASC, created_at DESC").Find(&routes).Error; err != nil {
		return err
	}
	counts.Route = len(routes)
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

		action = "hdel_empty_policy"
		return s.RedisClient.HDel(ctx, redisKey, redisField).Err()
	}

	// 序列化为 JSON
	jsonData, err := json.Marshal(policyAgg)
	if err != nil {
		return err
	}

	action = "hset_policy"
	return s.RedisClient.HSet(ctx, redisKey, redisField, string(jsonData)).Err()
}

// SyncPolicyChange 当某个具体的策略配置变更时，同步该策略对应的维度
func (s *PolicyRedisSync) SyncPolicyChange(ctx context.Context, scopeType, scopeCode, modelID string) (err error) {
	start := time.Now()
	modelCode := ""
	action := "sync_dimension"
	defer func() {
		fields := []zap.Field{
			zap.String("action", action),
			zap.String("scope_type", scopeType),
			zap.String("scope_code", scopeCode),
			zap.String("model_id", modelID),
			zap.String("model_code", modelCode),
			zap.Duration("cost", time.Since(start)),
		}
		if err != nil {
			fields = append(fields, zap.Error(err))
			policyRedisSyncLogger(ctx).Error("Policy Redis change sync failed", fields...)
			return
		}
		policyRedisSyncLogger(ctx).Info("Policy Redis change sync completed", fields...)
	}()

	if s.RedisClient == nil {
		action = "skip_nil_redis_client"
		return nil
	}
	if !config.C.Sync.Policies {
		action = "skip_policy_sync_disabled"
		return nil
	}

	if modelID != "" {
		modelCode, err = s.lookupModelCodeByID(ctx, modelID)
		if err != nil {
			return err
		}
	}

	var tenantCode, userID string
	if scopeType == "user" {
		userID = scopeCode
	} else if scopeType == "tenant" {
		tenantCode = scopeCode
	}

	return s.SyncDimension(ctx, tenantCode, userID, modelCode)
}

func resolveRedisKeyAndField(tenantCode, userID, modelCode string) (string, string, bool) {
	if userID != "" {
		if modelCode != "" && modelCode != "*" {
			return fmt.Sprintf("aigw:policies:user:%s", userID), modelCode, true
		}
		return fmt.Sprintf("aigw:policies:user:%s", userID), "*", true
	}
	if tenantCode != "" {
		if modelCode != "" && modelCode != "*" {
			return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), modelCode, true
		}
		return fmt.Sprintf("aigw:policies:tenant:%s", tenantCode), "*", true
	}
	if modelCode != "" && modelCode != "*" {
		return fmt.Sprintf("aigw:policies:model:%s", modelCode), "*", true
	}
	return "", "", false
}

func buildModelIDWhere(db *gorm.DB, modelID string) *gorm.DB {
	if modelID == "" {
		return db.Where("(model_id = '' OR model_id IS NULL)")
	}
	return db.Where("model_id = ?", modelID)
}

func (s *PolicyRedisSync) lookupModelCodeByID(ctx context.Context, modelID string) (string, error) {
	if s.ModelDAL == nil {
		return "", errors.InternalServerError("", "policy redis sync requires model dal")
	}
	return s.ModelDAL.GetModelCodeByID(ctx, modelID)
}

func (s *PolicyRedisSync) lookupModelIDByCode(ctx context.Context, modelCode string) (string, error) {
	if s.ModelDAL == nil {
		return "", errors.InternalServerError("", "policy redis sync requires model dal")
	}
	return s.ModelDAL.GetIDByModelCode(ctx, modelCode)
}

type policyRedisSyncCounts struct {
	LoadBalance  int
	Invocation   int
	Limit        int
	CircuitBreak int
	Tagging      int
	Route        int
}

func policyRedisSyncLogger(ctx context.Context) *zap.Logger {
	return logging.Context(logging.NewTag(ctx, policyRedisSyncLogTag))
}

func policyRedisSyncFields(tenantCode, userID, modelCode, modelID, scopeType, scopeCode, redisKey, redisField, action string, counts policyRedisSyncCounts, cost time.Duration) []zap.Field {
	return []zap.Field{
		zap.String("action", action),
		zap.String("tenant_code", tenantCode),
		zap.String("user_id", userID),
		zap.String("model_code", modelCode),
		zap.String("model_id", modelID),
		zap.String("scope_type", scopeType),
		zap.String("scope_code", scopeCode),
		zap.String("redis_key", redisKey),
		zap.String("redis_field", redisField),
		zap.Int("load_balance_count", counts.LoadBalance),
		zap.Int("invocation_count", counts.Invocation),
		zap.Int("limit_count", counts.Limit),
		zap.Int("circuit_break_count", counts.CircuitBreak),
		zap.Int("tagging_count", counts.Tagging),
		zap.Int("route_count", counts.Route),
		zap.Duration("cost", cost),
	}
}
