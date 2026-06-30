package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	policyBiz "github.com/tokenlive/tokenlive-admin/internal/mods/policy/biz"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/logging"
	"github.com/tokenlive/tokenlive-admin/pkg/metrics"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"go.uber.org/zap"
)

// Model business logic layer
type Model struct {
	Trans             *util.Trans
	ModelDAL          *dal.Model
	DataPermissionBIZ *DataPermission
	ConfigRedisSync   *ConfigRedisSync
	PolicyRedisSync   *policyBiz.PolicyRedisSync
	RedisClient       *redis.Client
	AuditLogBIZ       *opsBiz.AuditLog
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

	if len(result.Data) > 0 {
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
	m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypeModel, model.ID, model.ModelName, nil, model)
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

	// Capture before state for audit
	beforeModel := *model

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

		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeModel, model.ID, model.ModelName, beforeModel, model)
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
		beforeData := map[string]int{"enabled": model.Enabled}
		afterData := map[string]int{"enabled": formItem.Enabled}
		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypeModel, model.ID, model.ModelName, beforeData, afterData)
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

	if err := m.ensureModelCanDelete(ctx, model); err != nil {
		return err
	}

	var tenantCodes []string
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

		// 删除模型后清理租户绑定关系，避免留下无效 model_id。
		if err := tx.Table(tenantModelTable).Where("model_id = ?", id).Delete(nil).Error; err != nil {
			return err
		}

		return nil
	})
	if err == nil {
		_ = m.ConfigRedisSync.SyncModelByCode(ctx, model.ModelCode)
		// 删除时，同步清理 Redis 相关租户的缓存
		_ = m.ConfigRedisSync.SyncModelDisable(ctx, model.ID, model.ModelCode, tenantCodes...)

		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypeModel, model.ID, model.ModelName, model, nil)
	}
	return err
}

func (m *Model) ensureModelCanDelete(ctx context.Context, model *schema.Model) error {
	db := util.GetDB(ctx, m.ModelDAL.DB)

	checks := []struct {
		table   string
		where   string
		args    []interface{}
		message string
	}{
		{
			table:   config.C.FormatTableName("endpoint"),
			where:   "model_id = ? AND deleted = '0'",
			args:    []interface{}{model.ID},
			message: "模型存在关联端点，请先清理后再执行删除操作",
		},
		{
			table:   config.C.FormatTableName("model_alias"),
			where:   "model_id = ? AND deleted = '0'",
			args:    []interface{}{model.ID},
			message: "模型存在关联别名，请先清理后再执行删除操作",
		},
		{
			table:   config.C.FormatTableName("policy_binding"),
			where:   "model_code = ? AND deleted = '0'",
			args:    []interface{}{model.ModelCode},
			message: "模型存在关联策略，请先清理后再执行删除操作",
		},
	}

	for _, check := range checks {
		var count int64
		if err := db.Table(check.table).Where(check.where, check.args...).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.BadRequest("", "%s", check.message)
		}
	}
	return nil
}

func (m *Model) fillModelsStatusPoints(ctx context.Context, models []*schema.Model) {
	if len(models) == 0 {
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

	var values []interface{}
	var err error
	if m.RedisClient != nil {
		batchSize := 500
		values = make([]interface{}, 0, len(keys))
		for i := 0; i < len(keys); i += batchSize {
			end := i + batchSize
			if end > len(keys) {
				end = len(keys)
			}
			batchKeys := keys[i:end]
			batchValues, batchErr := m.RedisClient.MGet(ctx, batchKeys...).Result()
			if batchErr != nil {
				err = batchErr
				break
			}
			values = append(values, batchValues...)
		}
		if err != nil {
			logging.Context(ctx).Error("Failed to MGet model status points from Redis", zap.Error(err))
		} else {
			limit := 5
			if len(keys) < limit {
				limit = len(keys)
			}
			logging.Context(ctx).Info("Successfully MGet model status points from Redis",
				zap.Int("keysCount", len(keys)),
				zap.Int("valuesCount", len(values)),
				zap.Any("firstFewKeys", keys[0:limit]),
				zap.Any("firstFewValues", values[0:limit]))
		}
	} else {
		// 从内存获取
		values = make([]interface{}, len(keys))
		idx = 0
		for _, model := range models {
			for i := 0; i < numMinutes; i++ {
				minute := currentMin - int64(numMinutes-1-i)
				succ, fail := metrics.GlobalStore.GetModelStatus(model.ModelCode, minute)
				if succ > 0 {
					values[idx] = strconv.FormatInt(succ, 10)
				}
				if fail > 0 {
					values[idx+1] = strconv.FormatInt(fail, 10)
				}
				idx += 2
			}
		}
	}

	idx = 0
	for _, model := range models {
		minSuccess := make([]int64, numMinutes)
		minFail := make([]int64, numMinutes)

		if err == nil && len(values) == numKeys {
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
