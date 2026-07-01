package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Tenant management for RBAC
type Tenant struct {
	Trans       *util.Trans
	TenantDAL   *dal.Tenant
	UserDAL     *dal.User
	RedisClient *redis.Client
	AuditLogBIZ *opsBiz.AuditLog
}

// Query tenants from the data access object based on the provided parameters and options.
func (a *Tenant) Query(ctx context.Context, params schema.TenantQueryParam) (*schema.TenantQueryResult, error) {
	params.Pagination = true

	result, err := a.TenantDAL.Query(ctx, params, schema.TenantQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Get the specified tenant from the data access object.
func (a *Tenant) Get(ctx context.Context, id string) (*schema.Tenant, error) {
	tenant, err := a.TenantDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if tenant == nil {
		return nil, errors.NotFound("", "Tenant not found")
	}

	return tenant, nil
}

// Create a new tenant in the data access object.
func (a *Tenant) Create(ctx context.Context, formItem *schema.TenantForm) (*schema.Tenant, error) {
	existsCode, err := a.TenantDAL.ExistsCode(ctx, formItem.Code)
	if err != nil {
		return nil, err
	} else if existsCode {
		return nil, errors.BadRequest("", "Tenant code already exists")
	}

	// 自动生成高熵随机 API Key，若表单为空
	if formItem.APIKey == "" {
		apiKey, err := a.generateRandomAPIKey(ctx)
		if err != nil {
			return nil, err
		}
		formItem.APIKey = apiKey
	}

	tenant := &schema.Tenant{
		ID:        util.NewXID(),
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(tenant); err != nil {
		return nil, err
	}

	username := util.FromUsername(ctx)
	tenant.Creator = username

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.TenantDAL.Create(ctx, tenant)
	})
	if err != nil {
		return nil, err
	}

	// 同步 API Key 到 Redis
	_ = a.syncToRedis(ctx, tenant)

	a.AuditLogBIZ.RecordActionWithTenant(ctx, tenant.Code, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypeTenant, tenant.ID, tenant.Code, nil, tenant)
	return tenant, nil
}

// Update the specified tenant in the data access object.
func (a *Tenant) Update(ctx context.Context, id string, formItem *schema.TenantForm) error {
	tenant, err := a.TenantDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if tenant == nil {
		return errors.NotFound("", "Tenant not found")
	}

	if tenant.Code != formItem.Code {
		existsCode, err := a.TenantDAL.ExistsCode(ctx, formItem.Code)
		if err != nil {
			return err
		} else if existsCode {
			return errors.BadRequest("", "Tenant code already exists")
		}

		// Check if there are users associated with the old tenant code
		hasUsers, err := util.Exists(ctx, dal.GetUserDB(ctx, a.UserDAL.DB).Where("tenant=?", tenant.Code))
		if err != nil {
			return err
		} else if hasUsers {
			return errors.BadRequest("", "Cannot change tenant code because there are users associated with it")
		}
	}

	oldCode := tenant.Code
	oldAPIKey := tenant.APIKey

	beforeTenant := *tenant

	if err := formItem.FillTo(tenant); err != nil {
		return err
	}
	tenant.UpdatedAt = time.Now()
	tenant.Modifier = util.FromUsername(ctx)

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		if oldCode != tenant.Code {
			tx := util.GetDB(ctx, a.TenantDAL.DB)

			// 级联更新 tenant_model 表
			tenantModelTable := config.C.FormatTableName("tenant_model")
			err := tx.Table(tenantModelTable).Where("tenant_code = ?", oldCode).Update("tenant_code", tenant.Code).Error
			if err != nil {
				return err
			}
		}

		return a.TenantDAL.Update(ctx, tenant)
	})
	if err != nil {
		return err
	}

	// 若 API Key 发生变更，删除旧的缓存
	if a.RedisClient != nil && oldAPIKey != "" && oldAPIKey != tenant.APIKey {
		_ = a.RedisClient.Del(ctx, apiKeyRuntimeRedisKeys(oldAPIKey)...).Err()
	}

	// 同步新缓存到 Redis
	_ = a.syncToRedis(ctx, tenant)

	a.AuditLogBIZ.RecordActionWithTenant(ctx, tenant.Code, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeTenant, tenant.ID, tenant.Code, beforeTenant, tenant)

	// 如果租户 Code 发生变更，且 Redis 可用，则迁移 models、providers 和 endpoints 的 Redis 键
	if a.RedisClient != nil && oldCode != tenant.Code {
		oldModelsKey := "aigw:tenant:" + oldCode + ":models"
		newModelsKey := "aigw:tenant:" + tenant.Code + ":models"

		// 1. 迁移 aigw:tenant:{oldCode}:models
		exists, err := a.RedisClient.Exists(ctx, oldModelsKey).Result()
		if err == nil && exists > 0 {
			_ = a.RedisClient.Rename(ctx, oldModelsKey, newModelsKey).Err()
		}

		// 2. 迁移 aigw:tenant:{oldCode}:model:{modelCode}:endpoints
		if config.C.Sync.Endpoints {
			tx := util.GetDB(ctx, a.TenantDAL.DB)
			var modelCodes []string
			tenantModelTable := config.C.FormatTableName("tenant_model")
			modelTable := config.C.FormatTableName("model")

			err = tx.Table(tenantModelTable).
				Joins("JOIN "+modelTable+" ON "+tenantModelTable+".model_id = "+modelTable+".id").
				Where(tenantModelTable+".tenant_code = ? AND "+modelTable+".deleted = '0'", tenant.Code).
				Pluck(modelTable+".model_code", &modelCodes).Error

			if err == nil {
				for _, modelCode := range modelCodes {
					// 迁移 endpoints key（新）
					oldEndpointsKey := "aigw:tenant:" + oldCode + ":model:" + modelCode + ":endpoints"
					newEndpointsKey := "aigw:tenant:" + tenant.Code + ":model:" + modelCode + ":endpoints"

					existsEndpoints, err := a.RedisClient.Exists(ctx, oldEndpointsKey).Result()
					if err == nil && existsEndpoints > 0 {
						_ = a.RedisClient.Rename(ctx, oldEndpointsKey, newEndpointsKey).Err()
					}
				}
			}
		}
	}

	return nil
}

// Delete the specified tenant from the data access object.
func (a *Tenant) Delete(ctx context.Context, id string) error {
	tenant, err := a.TenantDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if tenant == nil {
		return errors.NotFound("", "Tenant not found")
	}

	// Check if there are users belonging to this tenant
	hasUsers, err := util.Exists(ctx, dal.GetUserDB(ctx, a.UserDAL.DB).Where("tenant=?", tenant.Code))
	if err != nil {
		return err
	} else if hasUsers {
		return errors.BadRequest("", "Cannot delete tenant because there are users associated with it")
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.TenantDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 清除 Redis 缓存
	if a.RedisClient != nil && tenant.APIKey != "" {
		_ = a.RedisClient.Del(ctx, apiKeyRuntimeRedisKeys(tenant.APIKey)...).Err()
	}

	a.AuditLogBIZ.RecordActionWithTenant(ctx, tenant.Code, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypeTenant, tenant.ID, tenant.Code, tenant, nil)
	return nil
}

// generateRandomAPIKey 生成高熵随机租户 API Key
func (a *Tenant) generateRandomAPIKey(ctx context.Context) (string, error) {
	bytes := make([]byte, 16) // 16 字节 = 32位 16进制字符
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.WithStack(err)
	}
	return "sk-t-" + hex.EncodeToString(bytes), nil
}

// syncToRedis 同步租户 API Key 至 Redis 中
func (a *Tenant) syncToRedis(ctx context.Context, tenant *schema.Tenant) error {
	if a.RedisClient == nil || tenant.APIKey == "" {
		return nil
	}

	redisKey := apiKeyRuntimeRedisKey(tenant.APIKey)

	// 逻辑删除或租户处于冻结状态，直接从 Redis 清除 Key
	if tenant.Deleted != "0" || tenant.Status != schema.TenantStatusActivated {
		return a.RedisClient.Del(ctx, apiKeyRuntimeRedisKeys(tenant.APIKey)...).Err()
	}

	fields := map[string]interface{}{
		"tenant":     tenant.Code,
		"status":     1,  // 1-启用
		"quota":      -1, // -1 代表无配额限制
		"expires_at": 0,  // 0 代表永不过期
	}

	if err := a.RedisClient.HSet(ctx, redisKey, fields).Err(); err != nil {
		return err
	}
	return nil
}
