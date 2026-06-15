package biz

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantModel management for RBAC
type TenantModel struct {
	Trans          *util.Trans
	TenantModelDAL *dal.TenantModel
	RedisClient    *redis.Client
}

// GetAuthorizedModelIDs returns all model IDs authorized for the tenant.
func (a *TenantModel) GetAuthorizedModelIDs(ctx context.Context, tenantCode string) ([]string, error) {
	list, err := a.TenantModelDAL.QueryByTenantCode(ctx, tenantCode)
	if err != nil {
		return nil, err
	}
	modelIDs := make([]string, 0, len(list))
	for _, item := range list {
		modelIDs = append(modelIDs, item.ModelID)
	}
	return modelIDs, nil
}

// SaveBindings deletes old bindings of the tenant and inserts new ones inside a transaction.
func (a *TenantModel) SaveBindings(ctx context.Context, tenantCode string, modelIDs []string, creator string) error {
	var modelCodes []string
	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		tx := util.GetDB(ctx, a.TenantModelDAL.DB)

		// 1. Delete all old bindings for the tenant
		err := a.TenantModelDAL.DeleteByTenantCode(ctx, tx, tenantCode)
		if err != nil {
			return err
		}

		// 2. Prepare new records
		items := make([]*schema.TenantModel, 0, len(modelIDs))
		for _, modelID := range modelIDs {
			items = append(items, &schema.TenantModel{
				ID:         util.NewXID(),
				TenantCode: tenantCode,
				ModelID:    modelID,
				Creator:    creator,
			})
		}

		// 3. Batch create new records
		err = a.TenantModelDAL.CreateInBatches(ctx, tx, items)
		if err != nil {
			return err
		}

		// 4. 查询模型编码，备于后续同步 Redis
		if len(modelIDs) > 0 {
			modelTable := config.C.FormatTableName("model")
			err = tx.Table(modelTable).Where("id IN ? AND deleted = '0'", modelIDs).Pluck("model_code", &modelCodes).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 5. 同步至 Redis: aigw:tenant:{tenantCode}:models
	if a.RedisClient != nil {
		redisKey := "aigw:tenant:" + tenantCode + ":models"

		if len(modelIDs) == 0 {
			// 若为空，代表该租户未绑定任何模型，直接清理
			return a.RedisClient.Del(ctx, redisKey).Err()
		} else {
			// 先清理旧白名单
			err = a.RedisClient.Del(ctx, redisKey).Err()
			if err != nil {
				return err
			}

			// 写入新绑定的模型编码
			if len(modelCodes) > 0 {
				var members []interface{}
				for _, code := range modelCodes {
					members = append(members, code)
				}
				err = a.RedisClient.SAdd(ctx, redisKey, members...).Err()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// Transaction helper to allow transactional execution
func (a *TenantModel) ExecTrans(ctx context.Context, fn func(context.Context) error) error {
	return a.Trans.Exec(ctx, fn)
}
