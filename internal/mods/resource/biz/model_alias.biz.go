package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelAlias business logic layer
type ModelAlias struct {
	Trans           *util.Trans
	ModelAliasDAL   *dal.ModelAlias
	ModelDAL        *dal.Model
	ConfigRedisSync *ConfigRedisSync
}

// Query model aliases.
func (m *ModelAlias) Query(ctx context.Context, params schema.ModelAliasQueryParam) (*schema.ModelAliasQueryResult, error) {
	params.Pagination = false

	result, err := m.ModelAliasDAL.Query(ctx, params, schema.ModelAliasQueryOptions{
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

// Get the specified model alias.
func (m *ModelAlias) Get(ctx context.Context, id string) (*schema.ModelAlias, error) {
	alias, err := m.ModelAliasDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if alias == nil {
		return nil, errors.NotFound("", "Model alias not found")
	}
	return alias, nil
}

// Create a new model alias.
func (m *ModelAlias) Create(ctx context.Context, formItem *schema.ModelAliasForm) (*schema.ModelAlias, error) {
	// 1. 检查 alias 是否与已有 model_code 冲突
	if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.Alias); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Alias conflicts with an existing model code")
	}

	// 2. 检查全局是否已有同名别名
	if exists, err := m.ModelAliasDAL.ExistsByAlias(ctx, formItem.Alias); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Alias already exists")
	}

	// 3. 检查该 model 的别名数量是否已达上限（10）
	const maxAliasesPerModel = 10
	count, err := m.ModelAliasDAL.CountByModelId(ctx, formItem.ModelId)
	if err != nil {
		return nil, err
	}
	if count >= maxAliasesPerModel {
		return nil, errors.BadRequest("", "Maximum number of aliases (10) per model reached")
	}

	alias := &schema.ModelAlias{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(alias); err != nil {
		return nil, err
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelAliasDAL.Create(ctx, alias)
	})
	if err != nil {
		return nil, err
	}

	// 4. 同步别名到 Redis（fire-and-forget）
	if m.ConfigRedisSync != nil {
		if model, err := m.ModelDAL.Get(ctx, formItem.ModelId); err == nil && model != nil && model.Enabled == 1 {
			_ = m.ConfigRedisSync.SyncAlias(ctx, formItem.Alias, model.ModelCode)
		}
	}

	return alias, nil
}

// Update the specified model alias.
func (m *ModelAlias) Update(ctx context.Context, id string, formItem *schema.ModelAliasForm) error {
	alias, err := m.ModelAliasDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if alias == nil {
		return errors.NotFound("", "Model alias not found")
	}

	// 如果 alias 名称变更，需要检查冲突
	if alias.Alias != formItem.Alias {
		// 检查是否与已有 model_code 冲突
		if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.Alias); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Alias conflicts with an existing model code")
		}
		// 检查全局是否已有同名别名
		if exists, err := m.ModelAliasDAL.ExistsByAlias(ctx, formItem.Alias); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Alias already exists")
		}
	}

	oldAlias := alias.Alias

	if err := formItem.FillTo(alias); err != nil {
		return err
	}
	alias.UpdatedAt = time.Now()

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelAliasDAL.Update(ctx, alias)
	})
	if err != nil {
		return err
	}

	// 同步别名到 Redis（fire-and-forget）
	if m.ConfigRedisSync != nil {
		if oldAlias != formItem.Alias {
			_ = m.ConfigRedisSync.DeleteAlias(ctx, oldAlias)
		}
		if model, err := m.ModelDAL.Get(ctx, alias.ModelId); err == nil && model != nil && model.Enabled == 1 {
			_ = m.ConfigRedisSync.SyncAlias(ctx, formItem.Alias, model.ModelCode)
		}
	}

	return nil
}

// Delete the specified model alias.
func (m *ModelAlias) Delete(ctx context.Context, id string) error {
	alias, err := m.ModelAliasDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if alias == nil {
		return errors.NotFound("", "Model alias not found")
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelAliasDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 同步删除 Redis 中的别名映射（fire-and-forget）
	if m.ConfigRedisSync != nil {
		_ = m.ConfigRedisSync.DeleteAlias(ctx, alias.Alias)
	}

	return nil
}
