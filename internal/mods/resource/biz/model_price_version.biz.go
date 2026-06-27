package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// ModelPriceVersion business logic layer
type ModelPriceVersion struct {
	Trans                *util.Trans
	ModelPriceVersionDAL *dal.ModelPriceVersion
	ModelCatalogDAL      *dal.ModelCatalog
	AuditLogBIZ          *opsBiz.AuditLog
}

// Query model price versions.
func (m *ModelPriceVersion) Query(ctx context.Context, params schema.ModelPriceVersionQueryParam) (*schema.ModelPriceVersionQueryResult, error) {
	params.Pagination = true
	return m.ModelPriceVersionDAL.Query(ctx, params, schema.ModelPriceVersionQueryOptions{
		QueryOptions: util.QueryOptions{
			OrderFields: []util.OrderByParam{
				{Field: "effective_from", Direction: util.DESC},
			},
		},
	})
}

// Get the specified model price version.
func (m *ModelPriceVersion) Get(ctx context.Context, id string) (*schema.ModelPriceVersion, error) {
	version, err := m.ModelPriceVersionDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if version == nil {
		return nil, errors.NotFound("", "Price version not found")
	}
	return version, nil
}

// GetCurrentPrice gets the currently effective price for a model.
func (m *ModelPriceVersion) GetCurrentPrice(ctx context.Context, modelID, currency string) (*schema.ModelPriceVersion, error) {
	return m.ModelPriceVersionDAL.GetCurrentPrice(ctx, modelID, currency)
}

// Create a new model price version.
func (m *ModelPriceVersion) Create(ctx context.Context, formItem *schema.ModelPriceVersionForm) (*schema.ModelPriceVersion, error) {
	// Verify model catalog exists
	exists, err := m.ModelCatalogDAL.Exists(ctx, formItem.ModelID)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, errors.NotFound("", "Model catalog not found: %s", formItem.ModelID)
	}

	version := &schema.ModelPriceVersion{
		Creator:         util.FromUsername(ctx),
		CreatedAt:       time.Now(),
		PublishedByUser: util.FromUsername(ctx),
	}
	if err := formItem.FillTo(version); err != nil {
		return nil, err
	}

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelPriceVersionDAL.Create(ctx, version)
	})
	if err != nil {
		return nil, err
	}
	m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePriceVersion, version.ID, version.ID, nil, version)
	return version, nil
}

// Update the specified model price version.
func (m *ModelPriceVersion) Update(ctx context.Context, id string, formItem *schema.ModelPriceVersionForm) error {
	version, err := m.ModelPriceVersionDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if version == nil {
		return errors.NotFound("", "Price version not found")
	}

	beforeVersion := *version

	if err := formItem.FillTo(version); err != nil {
		return err
	}
	version.Modifier = util.FromUsername(ctx)
	version.UpdatedAt = time.Now()

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelPriceVersionDAL.Update(ctx, version)
	})
	if err == nil {
		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePriceVersion, version.ID, version.ID, beforeVersion, version)
	}
	return err
}

// Deactivate deactivates a price version.
func (m *ModelPriceVersion) Deactivate(ctx context.Context, id string) error {
	version, err := m.ModelPriceVersionDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if version == nil {
		return errors.NotFound("", "Price version not found")
	}

	beforeVersion := *version

	version.Status = schema.ModelPriceStatusInactive
	version.Modifier = util.FromUsername(ctx)
	version.UpdatedAt = time.Now()

	err = m.Trans.Exec(ctx, func(ctx context.Context) error {
		return m.ModelPriceVersionDAL.Update(ctx, version)
	})
	if err == nil {
		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDisable, opsSchema.AuditResourceTypePriceVersion, version.ID, version.ID, beforeVersion, version)
	}
	return err
}

// Delete the specified model price version.
func (m *ModelPriceVersion) Delete(ctx context.Context, id string) error {
	version, err := m.ModelPriceVersionDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if version == nil {
		return errors.NotFound("", "Price version not found")
	}

	return m.Trans.Exec(ctx, func(ctx context.Context) error {
		if err := m.ModelPriceVersionDAL.Delete(ctx, id); err != nil {
			return err
		}
		m.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePriceVersion, version.ID, version.ID, version, nil)
		return nil
	})
}

// QueryByModelID queries all price versions for a model.
func (m *ModelPriceVersion) QueryByModelID(ctx context.Context, modelID string) (schema.ModelPriceVersions, error) {
	return m.ModelPriceVersionDAL.QueryByModelID(ctx, modelID)
}
