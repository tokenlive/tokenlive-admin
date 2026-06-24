package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	policyBiz "github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Model business logic layer
type Model struct {
	Trans             *util.Trans
	ModelDAL          *dal.Model
	DataPermissionBIZ *DataPermission
	ConfigRedisSync   *ConfigRedisSync
	PolicyRedisSync   *policyBiz.PolicyRedisSync
	RedisClient       *redis.Client
}

// Query models.
func (m *Model) Query(ctx context.Context, params schema.ModelQueryParam) (*schema.ModelQueryResult, error) {
	params.Pagination = true

	result, err := m.ModelDAL.Query(ctx, params, schema.ModelQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(result.Data) > 0 && m.RedisClient != nil {
		m.fillModelsStatusPoints(ctx, result.Data)
	}

	return result, nil
}

// Get the specified model.
func (m *Model) Get(ctx context.Context, id string) (*schema.Model, error) {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if model == nil {
		return nil, errors.NotFound("", "Model not found")
	}

	if !util.FromIsRootUser(ctx) {
		ok, err := m.DataPermissionBIZ.HasReadPermission(ctx, schema.DataPermissionTypeModel, id)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, errors.NotFound("", "Model not found")
		}
	}

	return model, nil
}

// Create a new model.
func (m *Model) Create(ctx context.Context, formItem *schema.ModelForm) (*schema.Model, error) {
	if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.ModelCode); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Model code already exists")
	}

	if exists, err := m.ModelDAL.ExistsByModelName(ctx, formItem.ModelName); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Model name already exists")
	}

	model := &schema.Model{
		ID:        util.NewXID(),
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(model); err != nil {
		return nil, err
	}

	err := m.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := m.ModelDAL.Create(ctx, model); err != nil {
			return err
		}
		return m.DataPermissionBIZ.CreateByOwner(ctx, schema.DataPermissionTypeModel, model.ID, util.FromTenant(ctx))
	})
	if err != nil {
		return nil, err
	}
	_ = m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
	return model, nil
}

// Update the specified model.
func (m *Model) Update(ctx context.Context, id string, formItem *schema.ModelForm) error {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if model == nil {
		return errors.NotFound("", "Model not found")
	}

	// Check model_code uniqueness if changed
	if model.ModelCode != formItem.ModelCode {
		if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.ModelCode); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Model code already exists")
		}
	}

	// Check model_name uniqueness if changed
	if model.ModelName != formItem.ModelName {
		if exists, err := m.ModelDAL.ExistsByModelName(ctx, formItem.ModelName); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Model name already exists")
		}
	}

	originalModelCode := model.ModelCode
	originalEnabled := model.Enabled

	if err := formItem.FillTo(model); err != nil {
		return err
	}
	model.Modifier = util.FromUsername(ctx)
	model.UpdatedAt = time.Now()

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelDAL.Update(ctx, model)
	})
	if err == nil {
		_ = m.ConfigRedisSync.SyncModelByCode(ctx, originalModelCode)
		if originalModelCode != model.ModelCode {
			_ = m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
			_ = m.ConfigRedisSync.SyncModelCodeChange(ctx, model.ID, originalModelCode, model.ModelCode)
		}

		// 检查启用状态是否变化
		if originalEnabled != model.Enabled {
			if model.Enabled == 0 {
				_ = m.ConfigRedisSync.SyncModelDisable(ctx, model.ID, model.ModelCode)
			} else if model.Enabled == 1 {
				_ = m.ConfigRedisSync.SyncModelEnable(ctx, model.ID, model.ModelCode)
			}
		}
	}
	return err
}

// ToggleEnabled updates only the enabled status of a model and re-syncs Redis.
// It replicates the enabled-change side effects of Update: SyncModelByCode plus
// SyncModelEnable/SyncModelDisable (which handle tenant binding relationships).
// model_code is not changed by a toggle, so no SyncModelCodeChange is needed.
func (m *Model) ToggleEnabled(ctx context.Context, id string, formItem *schema.ModelEnabledForm) error {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if model == nil {
		return errors.NotFound("", "Model not found")
	}

	// No-op if the status is unchanged.
	if model.Enabled == formItem.Enabled {
		return nil
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelDAL.UpdateEnabled(ctx, id, formItem.Enabled, util.FromUsername(ctx))
	})
	if err == nil {
		_ = m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
		if formItem.Enabled == 0 {
			_ = m.ConfigRedisSync.SyncModelDisable(ctx, model.ID, model.ModelCode)
		} else {
			_ = m.ConfigRedisSync.SyncModelEnable(ctx, model.ID, model.ModelCode)
		}
	}
	return err
}

// Delete the specified model.
func (m *Model) Delete(ctx context.Context, id string) error {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if model == nil {
		return errors.NotFound("", "Model not found")
	}

	var tenantCodes []string
	type Dimension struct {
		TenantCode string
		UserID     string
		ModelCode  string
	}
	var affectedDimensions []Dimension

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		tx := util.GetDB(ctx, m.ModelDAL.DB)

		// 提前查出绑定该模型的租户编码，备于后续清理缓存
		tenantModelTable := config.C.FormatTableName("tenant_model")
		err := tx.Table(tenantModelTable).Where("model_id = ?", id).Pluck("tenant_code", &tenantCodes).Error
		if err != nil {
			return err
		}

		if err := m.ModelDAL.Delete(ctx, id); err != nil {
			return err
		}
		if err := m.DataPermissionBIZ.DeleteByTypeAndDataId(ctx, schema.DataPermissionTypeModel, id); err != nil {
			return err
		}

		// 级联删除 tenant_model 表中绑定关系
		if err := tx.Table(tenantModelTable).Where("model_id = ?", id).Delete(nil).Error; err != nil {
			return err
		}

		// 1. 级联逻辑删除相关的模型别名
		modelAliasTable := config.C.FormatTableName("model_alias")
		if err := tx.Table(modelAliasTable).Where("model_id = ? AND deleted = '0'", id).Update("deleted", gorm.Expr("id")).Error; err != nil {
			return err
		}

		// 2. 级联逻辑删除关联的治理策略
		var bindings []*policySchema.PolicyBinding
		policyBindingTable := config.C.FormatTableName("policy_binding")
		if err := tx.Table(policyBindingTable).Where("model_code = ? AND deleted = '0'", model.ModelCode).Find(&bindings).Error; err != nil {
			return err
		}

		for _, b := range bindings {
			affectedDimensions = append(affectedDimensions, Dimension{
				TenantCode: b.TenantCode,
				UserID:     b.UserID,
				ModelCode:  b.ModelCode,
			})

			// 检查策略是否被其它模型绑定 (排除当前 model.ModelCode)
			var otherCount int64
			err := tx.Table(policyBindingTable).
				Where("policy_type = ? AND policy_id = ? AND model_code != ? AND deleted = '0'", b.PolicyType, b.PolicyID, model.ModelCode).
				Count(&otherCount).Error
			if err != nil {
				return err
			}

			// 若没有其它绑定关系，说明该具体策略记录由该模型独占，可以将其逻辑删除
			if otherCount == 0 {
				tableName := config.C.FormatTableName("policy_" + b.PolicyType)
				// 特殊处理 route, 级联逻辑删除 policy_route_detail 子表记录
				if b.PolicyType == "route" {
					routeDetailTable := config.C.FormatTableName("policy_route_detail")
					if err := tx.Table(routeDetailTable).Where("route_id = ? AND deleted = '0'", b.PolicyID).Update("deleted", gorm.Expr("id")).Error; err != nil {
						return err
					}
				}
				if err := tx.Table(tableName).Where("id = ? AND deleted = '0'", b.PolicyID).Update("deleted", gorm.Expr("id")).Error; err != nil {
					return err
				}
			}
		}

		// 逻辑删除 policy_binding 记录本身
		if err := tx.Table(policyBindingTable).Where("model_code = ? AND deleted = '0'", model.ModelCode).Update("deleted", gorm.Expr("id")).Error; err != nil {
			return err
		}

		return nil
	})
	if err == nil {
		_ = m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
		// 删除时，同步清理 Redis 相关租户的缓存
		_ = m.ConfigRedisSync.SyncModelDisable(ctx, model.ID, model.ModelCode, tenantCodes...)

		// 重新同步/清理受影响维度的策略缓存
		if m.PolicyRedisSync != nil && len(affectedDimensions) > 0 {
			// 对维度去重
			seen := make(map[string]bool)
			for _, dim := range affectedDimensions {
				dimKey := fmt.Sprintf("%s:%s:%s", dim.TenantCode, dim.UserID, dim.ModelCode)
				if seen[dimKey] {
					continue
				}
				seen[dimKey] = true
				_ = m.PolicyRedisSync.SyncDimension(ctx, dim.TenantCode, dim.UserID, dim.ModelCode)
			}
		}
	}
	return err
}

func (m *Model) fillModelsStatusPoints(ctx context.Context, models []*schema.Model) {
	if len(models) == 0 || m.RedisClient == nil {
		return
	}

	currentMin := time.Now().Unix() / 60
	numModels := len(models)
	numMinutes := 100
	numKeys := numModels * numMinutes * 2
	keys := make([]string, numKeys)

	idx := 0
	for _, model := range models {
		for i := 0; i < numMinutes; i++ {
			minute := currentMin - int64(numMinutes-1-i)
			keys[idx] = fmt.Sprintf("aigw:status:model:%s:%d:s", model.ModelCode, minute)
			keys[idx+1] = fmt.Sprintf("aigw:status:model:%s:%d:f", model.ModelCode, minute)
			idx += 2
		}
	}

	values, err := m.RedisClient.MGet(ctx, keys...).Result()
	if err != nil {
		return
	}

	idx = 0
	for _, model := range models {
		minSuccess := make([]int64, numMinutes)
		minFail := make([]int64, numMinutes)

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

		points := make([]schema.StatusPoint, 10)
		for pIdx := 0; pIdx < 10; pIdx++ {
			var successSum int64
			var failSum int64
			for mOffset := 0; mOffset < 10; mOffset++ {
				mIdx := pIdx*10 + mOffset
				successSum += minSuccess[mIdx]
				failSum += minFail[mIdx]
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
		model.StatusPoints = points
	}
}

// Sync model's Redis cache data.
func (m *Model) Sync(ctx context.Context, id string) error {
	model, err := m.ModelDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if model == nil {
		return errors.NotFound("", "Model not found")
	}

	if model.Deleted != "0" {
		return errors.BadRequest("", "Cannot sync a deleted model")
	}

	// 1. 同步端点配置及默认费率策略
	if err := m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode); err != nil {
		return err
	}

	// 2. 根据启用状态同步租户绑定关系
	if model.Enabled == 1 {
		if err := m.ConfigRedisSync.SyncModelEnable(ctx, model.ID, model.ModelCode); err != nil {
			return err
		}
	} else {
		if err := m.ConfigRedisSync.SyncModelDisable(ctx, model.ID, model.ModelCode); err != nil {
			return err
		}
	}

	// 3. 同步该模型关联的所有治理策略缓存 (包括公共策略和其它限定维度的策略)
	if m.PolicyRedisSync != nil {
		// (a) 先同步模型公共策略缓存 (tenantCode = "", userID = "")
		if err := m.PolicyRedisSync.SyncDimension(ctx, "", "", model.ModelCode); err != nil {
			return err
		}

		// (b) 再同步其它跟此 model 绑定的维度策略缓存
		var bindings []*policySchema.PolicyBinding
		policyBindingTable := config.C.FormatTableName("policy_binding")
		db := util.GetDB(ctx, m.ModelDAL.DB)
		err := db.Table(policyBindingTable).
			Where("model_code = ? AND deleted = '0'", model.ModelCode).
			Find(&bindings).Error
		if err == nil && len(bindings) > 0 {
			seen := make(map[string]bool)
			seen["::"+model.ModelCode] = true // 过滤掉前面已经刷过的公共维度
			for _, b := range bindings {
				dimKey := fmt.Sprintf("%s:%s:%s", b.TenantCode, b.UserID, b.ModelCode)
				if seen[dimKey] {
					continue
				}
				seen[dimKey] = true
				_ = m.PolicyRedisSync.SyncDimension(ctx, b.TenantCode, b.UserID, b.ModelCode)
			}
		}
	}

	return nil
}
