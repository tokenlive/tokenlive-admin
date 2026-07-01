package biz

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// TenantEndpoint 租户端点绑定业务逻辑类
type TenantEndpoint struct {
	Trans             *util.Trans
	TenantEndpointDAL *dal.TenantEndpoint
	RedisClient       *redis.Client
	AuditLogBIZ       *opsBiz.AuditLog
}

// GetAllowedEndpointIDs 查询租户指定模型下已经允许的端点 ID 列表
func (a *TenantEndpoint) GetAllowedEndpointIDs(ctx context.Context, tenantCode, modelID string) ([]string, error) {
	return a.TenantEndpointDAL.QueryEndpointIDsByTenantAndModel(ctx, tenantCode, modelID)
}

// SaveEndpoints 在事务中保存允许的端点白名单，并在成功后同步至 Redis
func (a *TenantEndpoint) SaveEndpoints(ctx context.Context, tenantCode, modelID string, endpointIDs []string, creator string) error {
	var modelCode string

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		tx := util.GetDB(ctx, a.TenantEndpointDAL.DB)

		// 1. 清理指定租户指定模型旧关联（通过子查询按 model_id 范围删除）
		err := a.TenantEndpointDAL.DeleteByTenantAndModel(ctx, tx, tenantCode, modelID)
		if err != nil {
			return err
		}

		// 2. 批量写入新关联关系
		items := make([]*schema.TenantEndpoint, 0, len(endpointIDs))
		for _, endpointID := range endpointIDs {
			items = append(items, &schema.TenantEndpoint{
				ID:         util.NewXID(),
				TenantCode: tenantCode,
				EndpointID: endpointID,
				Creator:    creator,
			})
		}
		err = a.TenantEndpointDAL.CreateInBatches(ctx, tx, items)
		if err != nil {
			return err
		}

		// 3. 提前查好 model_code，备于事务成功后同步 Redis
		modelTable := config.C.FormatTableName("model")
		err = tx.Table(modelTable).Where("id = ? AND deleted = '0'", modelID).Pluck("model_code", &modelCode).Error
		if err != nil {
			return err
		}
		if modelCode == "" {
			return errors.BadRequest("", "Model not found")
		}

		return nil
	})
	if err != nil {
		return err
	}

	a.AuditLogBIZ.RecordActionWithTenant(ctx, tenantCode, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeTenantEndpoint, modelID, tenantCode, nil, map[string]interface{}{"tenant_code": tenantCode, "model_id": modelID, "endpoint_ids": endpointIDs})

	// 4. 事务执行成功，同步至 Redis
	if a.RedisClient != nil && config.C.Sync.Endpoints {
		endpointsKey := "aigw:tenant:" + tenantCode + ":model:" + modelCode + ":endpoints"
		providersKey := "aigw:tenant:" + tenantCode + ":model:" + modelCode + ":providers"

		if len(endpointIDs) == 0 {
			// 若为空，代表该模型的所有端点皆允许使用，清理白名单以支持"全放通"语义
			_ = a.RedisClient.Del(ctx, endpointsKey).Err()
			// 同时删除旧的 providers key（过渡期兼容）
			_ = a.RedisClient.Del(ctx, providersKey).Err()
			return nil
		}

		// 删除旧白名单
		_ = a.RedisClient.Del(ctx, endpointsKey).Err()
		// 同时删除旧的 providers key（过渡期兼容）
		_ = a.RedisClient.Del(ctx, providersKey).Err()

		// 写入 Redis 集合（endpoint IDs）
		if len(endpointIDs) > 0 {
			var members []interface{}
			for _, id := range endpointIDs {
				members = append(members, id)
			}
			err = a.RedisClient.SAdd(ctx, endpointsKey, members...).Err()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
