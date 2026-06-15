package biz

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantModelProvider 租户模型供应商绑定业务逻辑类
type TenantModelProvider struct {
	Trans                  *util.Trans
	TenantModelProviderDAL *dal.TenantModelProvider
	RedisClient            *redis.Client
}

// GetAllowedProviderIDs 查询租户指定模型下已经允许的供应商 ID 列表
func (a *TenantModelProvider) GetAllowedProviderIDs(ctx context.Context, tenantCode, modelID string) ([]string, error) {
	list, err := a.TenantModelProviderDAL.QueryByTenantAndModel(ctx, tenantCode, modelID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ProviderID)
	}
	return ids, nil
}

// SaveProviders 在事务中保存允许的供应商白名单，并在成功后同步至 Redis
func (a *TenantModelProvider) SaveProviders(ctx context.Context, tenantCode, modelID string, providerIDs []string, creator string) error {
	var modelCode string
	var providerCodes []string

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		tx := util.GetDB(ctx, a.TenantModelProviderDAL.DB)

		// 1. 清理指定租户指定模型旧关联
		err := a.TenantModelProviderDAL.DeleteByTenantAndModel(ctx, tx, tenantCode, modelID)
		if err != nil {
			return err
		}

		// 2. 批量写入新关联关系
		items := make([]*schema.TenantModelProvider, 0, len(providerIDs))
		for _, providerID := range providerIDs {
			items = append(items, &schema.TenantModelProvider{
				ID:         util.NewXID(),
				TenantCode: tenantCode,
				ModelID:    modelID,
				ProviderID: providerID,
				Creator:    creator,
			})
		}
		err = a.TenantModelProviderDAL.CreateInBatches(ctx, tx, items)
		if err != nil {
			return err
		}

		// 3. 提前查好数据，备于事务成功后同步 Redis
		modelTable := config.C.FormatTableName("model")
		err = tx.Table(modelTable).Where("id = ? AND deleted = '0'", modelID).Pluck("model_code", &modelCode).Error
		if err != nil {
			return err
		}
		if modelCode == "" {
			return errors.BadRequest("", "Model not found")
		}

		if len(providerIDs) > 0 {
			providerTable := config.C.FormatTableName("provider")
			err = tx.Table(providerTable).Where("id IN ? AND deleted = '0'", providerIDs).Pluck("code", &providerCodes).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 4. 事务执行成功，我们才同步至 Redis 白名单 Set
	if a.RedisClient != nil {
		redisKey := "aigw:tenant:" + tenantCode + ":model:" + modelCode + ":providers"

		if len(providerIDs) == 0 {
			// 若为空，代表该模型的所有供应商皆允许使用，清理白名单以支持"全放通"语义
			return a.RedisClient.Del(ctx, redisKey).Err()
		}

		// 删除旧白名单
		err = a.RedisClient.Del(ctx, redisKey).Err()
		if err != nil {
			return err
		}

		// 写入 Redis 集合
		if len(providerCodes) > 0 {
			var members []interface{}
			for _, code := range providerCodes {
				members = append(members, code)
			}
			err = a.RedisClient.SAdd(ctx, redisKey, members...).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
