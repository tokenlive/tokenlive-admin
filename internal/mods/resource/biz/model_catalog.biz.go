package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelCatalog business logic layer
type ModelCatalog struct {
	Trans              *util.Trans
	ModelCatalogDAL    *dal.ModelCatalog
	ModelCatalogI18nDAL *dal.ModelCatalogI18n
}

// Query model catalogs.
func (m *ModelCatalog) Query(ctx context.Context, params schema.ModelCatalogQueryParam) (*schema.ModelCatalogQueryResult, error) {
	params.Pagination = true

	return m.ModelCatalogDAL.Query(ctx, params, schema.ModelCatalogQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "featured", Direction: util.DESC},
				{Field: "sort_weight", Direction: util.DESC},
				{Field: "created_at", Direction: util.DESC},
			},
		},
	})
}

// Get the specified model catalog.
func (m *ModelCatalog) Get(ctx context.Context, modelID string) (*schema.ModelCatalog, error) {
	catalog, err := m.ModelCatalogDAL.Get(ctx, modelID)
	if err != nil {
		return nil, err
	} else if catalog == nil {
		return nil, errors.NotFound("", "Model catalog not found")
	}
	return catalog, nil
}

// GetBySlug gets a model catalog by slug.
func (m *ModelCatalog) GetBySlug(ctx context.Context, slug string) (*schema.ModelCatalog, error) {
	catalog, err := m.ModelCatalogDAL.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	} else if catalog == nil {
		return nil, errors.NotFound("", "Model catalog not found")
	}
	return catalog, nil
}

// Create a new model catalog.
func (m *ModelCatalog) Create(ctx context.Context, formItem *schema.ModelCatalogForm) (*schema.ModelCatalog, error) {
	// Check slug uniqueness
	exists, err := m.ModelCatalogDAL.ExistsBySlug(ctx, formItem.Slug, "")
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errors.Conflict("", "Slug already exists: %s", formItem.Slug)
	}

	catalog := &schema.ModelCatalog{
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}
	if err := formItem.FillTo(catalog); err != nil {
		return nil, err
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogDAL.Create(ctx, catalog)
	})
	if err != nil {
		return nil, err
	}
	return catalog, nil
}

// Update the specified model catalog.
func (m *ModelCatalog) Update(ctx context.Context, modelID string, formItem *schema.ModelCatalogForm) error {
	catalog, err := m.ModelCatalogDAL.Get(ctx, modelID)
	if err != nil {
		return err
	} else if catalog == nil {
		return errors.NotFound("", "Model catalog not found")
	}

	// Check slug uniqueness (excluding self)
	exists, err := m.ModelCatalogDAL.ExistsBySlug(ctx, formItem.Slug, modelID)
	if err != nil {
		return err
	} else if exists {
		return errors.Conflict("", "Slug already exists: %s", formItem.Slug)
	}

	if err := formItem.FillTo(catalog); err != nil {
		return err
	}
	catalog.Modifier = util.FromUsername(ctx)
	catalog.UpdatedAt = time.Now()

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogDAL.Update(ctx, catalog)
	})
}

// Publish publishes a model catalog (sets visibility and published_at).
func (m *ModelCatalog) Publish(ctx context.Context, modelID string, formItem *schema.ModelCatalogPublishForm) error {
	catalog, err := m.ModelCatalogDAL.Get(ctx, modelID)
	if err != nil {
		return err
	} else if catalog == nil {
		return errors.NotFound("", "Model catalog not found")
	}

	catalog.Visibility = formItem.Visibility
	if formItem.PublishedAt != nil {
		catalog.PublishedAt = formItem.PublishedAt
	} else if catalog.PublishedAt == nil {
		now := time.Now()
		catalog.PublishedAt = &now
	}
	catalog.Modifier = util.FromUsername(ctx)
	catalog.UpdatedAt = time.Now()

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelCatalogDAL.Update(ctx, catalog)
	})
}

// Delete the specified model catalog.
func (m *ModelCatalog) Delete(ctx context.Context, modelID string) error {
	catalog, err := m.ModelCatalogDAL.Get(ctx, modelID)
	if err != nil {
		return err
	} else if catalog == nil {
		return errors.NotFound("", "Model catalog not found")
	}

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		// Delete i18n entries first
		if err := m.ModelCatalogI18nDAL.DeleteByModelID(ctx, modelID); err != nil {
			return err
		}
		return m.ModelCatalogDAL.Delete(ctx, modelID)
	})
}

// QueryPublic queries public available model catalogs.
func (m *ModelCatalog) QueryPublic(ctx context.Context, limit int) (schema.ModelCatalogs, error) {
	return m.ModelCatalogDAL.QueryPublic(ctx, limit)
}
