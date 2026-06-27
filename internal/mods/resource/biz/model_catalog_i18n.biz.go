package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalogI18n business logic layer
type ModelCatalogI18n struct {
	Trans               *util.Trans
	ModelCatalogI18nDAL *dal.ModelCatalogI18n
	ModelCatalogDAL     *dal.ModelCatalog
}

// Query model catalog i18n entries.
func (m *ModelCatalogI18n) Query(ctx context.Context, params schema.ModelCatalogI18nQueryParam) (*schema.ModelCatalogI18nQueryResult, error) {
	params.Pagination = true
	return m.ModelCatalogI18nDAL.Query(ctx, params)
}

// Get gets a specific i18n entry.
func (m *ModelCatalogI18n) Get(ctx context.Context, modelID, locale string) (*schema.ModelCatalogI18n, error) {
	item, err := m.ModelCatalogI18nDAL.Get(ctx, modelID, locale)
	if err != nil {
		return nil, err
	} else if item == nil {
		return nil, errors.NotFound("", "I18n entry not found")
	}
	return item, nil
}

// QueryByModelID gets all i18n entries for a model.
func (m *ModelCatalogI18n) QueryByModelID(ctx context.Context, modelID string) (schema.ModelCatalogI18ns, error) {
	return m.ModelCatalogI18nDAL.QueryByModelID(ctx, modelID)
}

// Create creates a new i18n entry.
func (m *ModelCatalogI18n) Create(ctx context.Context, formItem *schema.ModelCatalogI18nForm) (*schema.ModelCatalogI18n, error) {
	// Verify model catalog exists
	exists, err := m.ModelCatalogDAL.Exists(ctx, formItem.ModelID)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.NotFound("", "Model catalog not found: %s", formItem.ModelID)
	}

	item := &schema.ModelCatalogI18n{
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := formItem.FillTo(item); err != nil {
		return nil, err
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogI18nDAL.Create(ctx, item)
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// Update updates an i18n entry.
func (m *ModelCatalogI18n) Update(ctx context.Context, modelID, locale string, formItem *schema.ModelCatalogI18nForm) error {
	item, err := m.ModelCatalogI18nDAL.Get(ctx, modelID, locale)
	if err != nil {
		return err
	} else if item == nil {
		return errors.NotFound("", "I18n entry not found")
	}

	if err := formItem.FillTo(item); err != nil {
		return err
	}
	item.Modifier = util.FromUsername(ctx)
	item.UpdatedAt = time.Now()

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogI18nDAL.Update(ctx, item)
	})
}

// BatchUpsert batch upserts i18n entries for a model.
func (m *ModelCatalogI18n) BatchUpsert(ctx context.Context, formItem *schema.ModelCatalogI18nBatchForm) error {
	// Verify model catalog exists
	exists, err := m.ModelCatalogDAL.Exists(ctx, formItem.ModelID)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Model catalog not found: %s", formItem.ModelID)
	}

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		for _, entry := range formItem.Entries {
			item := &schema.ModelCatalogI18n{
				Creator:   util.FromUsername(ctx),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := entry.FillTo(item); err != nil {
				return err
			}
			item.ModelID = formItem.ModelID
			if err := m.ModelCatalogI18nDAL.Upsert(ctx, item); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete deletes a specific i18n entry.
func (m *ModelCatalogI18n) Delete(ctx context.Context, modelID, locale string) error {
	item, err := m.ModelCatalogI18nDAL.Get(ctx, modelID, locale)
	if err != nil {
		return err
	} else if item == nil {
		return errors.NotFound("", "I18n entry not found")
	}

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogI18nDAL.Delete(ctx, modelID, locale)
	})
}
