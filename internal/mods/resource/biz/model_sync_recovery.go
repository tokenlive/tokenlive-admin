package biz

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (m *Model) reconcileModelPublication(ctx context.Context, modelID string, tenantCodes ...string) error {
	if m.ConfigRedisSync == nil {
		ClearGatewayConfigCache()
		util.NotifyConfigChanged(ctx, util.ConfigChangeAll)
		return nil
	}
	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		db := util.GetDB(ctx, m.ModelDAL.DB)
		var current schema.Model
		if err := db.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", modelID).First(&current).Error; err != nil {
			return err
		}
		if err := m.ConfigRedisSync.syncModelAvailability(ctx, current.ID, current.ModelCode, tenantCodes...); err != nil {
			return err
		}
		refs, err := smartReferences(ctx, m.ModelDAL.DB, current.ID)
		if err != nil {
			return err
		}
		for _, ref := range refs {
			if err := m.ConfigRedisSync.SyncModelByCode(ctx, ref.ModelCode); err != nil {
				return err
			}
		}
		return nil
	})
}

// Availability is a projection of current committed state, not a command to
// replay an earlier toggle. Keep the row locked through routing and bindings.
func (s *ConfigRedisSync) syncModelAvailability(ctx context.Context, modelID, modelCode string, tenantCodes ...string) error {
	if modelID == "" || modelCode == "" {
		return nil
	}
	if s.RedisClient == nil {
		ClearGatewayConfigCache()
		util.NotifyConfigChanged(ctx, util.ConfigChangeAll, modelCode)
		return nil
	}
	return (&util.Trans{DB: s.ModelDAL.DB}).Exec(ctx, func(ctx context.Context) error {
		db := util.GetDB(ctx, s.ModelDAL.DB)
		var current schema.Model
		err := db.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", modelID).First(&current).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == nil {
			modelCode = current.ModelCode
		}
		var owner schema.Model
		err = db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("model_code = ? AND deleted = '0'", modelCode).First(&owner).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == nil && owner.ID != modelID {
			// A deleted model's code may already belong to another model.
			return s.reconcileReusedModelAccess(ctx, &owner)
		}
		if err := s.SyncModelByCode(ctx, modelCode); err != nil {
			return err
		}
		if current.ID != "" && current.Deleted == "0" && current.Enabled == 1 {
			return s.applyModelEnable(ctx, modelID, modelCode)
		}
		return s.applyModelDisable(ctx, modelID, modelCode, tenantCodes...)
	})
}
