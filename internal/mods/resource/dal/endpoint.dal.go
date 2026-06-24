package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// GetEndpointDB returns the database instance for Endpoint (only active records).
func GetEndpointDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.Endpoint)).Where("deleted = '0'")
}

// Endpoint data access layer
type Endpoint struct {
	DB *gorm.DB
}

// Query endpoints from the database.
func (e *Endpoint) Query(ctx context.Context, params schema.EndpointQueryParam, opts ...schema.EndpointQueryOptions) (*schema.EndpointQueryResult, error) {
	var opt schema.EndpointQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetEndpointDB(ctx, e.DB)
	if v := params.ProviderID; len(v) > 0 {
		db = db.Where("provider_id = ?", v)
	}
	if v := params.ModelID; len(v) > 0 {
		db = db.Where("model_id = ?", v)
	}
	if v := params.Priority; v > 0 {
		db = db.Where("priority = ?", v)
	}
	if v := params.LikeURL; len(v) > 0 {
		db = db.Where("url LIKE ?", "%"+v+"%")
	}

	var list schema.Endpoints
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &schema.EndpointQueryResult{
		PageResult: pageResult,
		Data:       list,
	}, nil
}

// Get the specified endpoint from the database.
func (e *Endpoint) Get(ctx context.Context, id string, opts ...schema.EndpointQueryOptions) (*schema.Endpoint, error) {
	var opt schema.EndpointQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.Endpoint)
	ok, err := util.FindOne(ctx, GetEndpointDB(ctx, e.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified endpoint exists.
func (e *Endpoint) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetEndpointDB(ctx, e.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsDuplicate checks if an active endpoint with the same model, provider, url, api_key, and real_model already exists.
func (e *Endpoint) ExistsDuplicate(ctx context.Context, modelID, providerID, url, apiKey, realModel string, excludeID string) (bool, error) {
	db := GetEndpointDB(ctx, e.DB).
		Where("model_id = ? AND provider_id = ? AND url = ? AND api_key = ? AND real_model = ?", modelID, providerID, url, apiKey, realModel)
	if len(excludeID) > 0 {
		db = db.Where("id != ?", excludeID)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return false, errors.WithStack(err)
	}
	return count > 0, nil
}

// Create a new endpoint.
func (e *Endpoint) Create(ctx context.Context, item *schema.Endpoint) error {
	result := GetEndpointDB(ctx, e.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified endpoint.
func (e *Endpoint) Update(ctx context.Context, item *schema.Endpoint) error {
	result := GetEndpointDB(ctx, e.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// UpdateEnabled updates only the enabled status (and modifier) of the specified endpoint.
func (e *Endpoint) UpdateEnabled(ctx context.Context, id string, enabled int, modifier string) error {
	result := GetEndpointDB(ctx, e.DB).Where("id=?", id).Updates(map[string]interface{}{
		"enabled":  enabled,
		"modifier": modifier,
	})
	return errors.WithStack(result.Error)
}

// Delete the specified endpoint logically.
func (e *Endpoint) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetEndpointDB(ctx, e.DB), id))
}

// QueryEndpointsByModelID queries endpoints by Model ID (only enabled endpoints).
func (e *Endpoint) QueryEndpointsByModelID(ctx context.Context, modelID string) (schema.Endpoints, error) {
	var list schema.Endpoints
	db := GetEndpointDB(ctx, e.DB).
		Preload("Provider").
		Where("model_id = ? AND enabled = 1", modelID).
		Order("priority ASC, weight DESC")
	if err := db.Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}

// QueryEndpointsByProviderID queries endpoints by Provider ID (only enabled endpoints).
func (e *Endpoint) QueryEndpointsByProviderID(ctx context.Context, providerID string) (schema.Endpoints, error) {
	var list schema.Endpoints
	db := GetEndpointDB(ctx, e.DB).
		Where("provider_id = ? AND enabled = 1", providerID).
		Order("priority ASC, weight DESC")
	if err := db.Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}

// QueryEndpointsByModelCode queries enabled endpoints by model_code (for routing).
// Joins endpoint → model → provider and filters all three to enabled + not deleted.
func (e *Endpoint) QueryEndpointsByModelCode(ctx context.Context, modelCode string) (schema.Endpoints, error) {
	endpointTable := config.C.FormatTableName("endpoint")
	modelTable := config.C.FormatTableName("model")
	providerTable := config.C.FormatTableName("provider")

	var list schema.Endpoints
	db := util.GetDB(ctx, e.DB).
		Table(endpointTable).
		Joins("JOIN "+modelTable+" ON "+endpointTable+".model_id = "+modelTable+".id").
		Joins("JOIN "+providerTable+" ON "+endpointTable+".provider_id = "+providerTable+".id").
		Where(modelTable+".model_code = ?", modelCode).
		Where(endpointTable + ".enabled = 1").
		Where(modelTable + ".enabled = 1").
		Where(providerTable + ".enabled = 1").
		Where(endpointTable + ".deleted = '0'").
		Where(modelTable + ".deleted = '0'").
		Where(providerTable + ".deleted = '0'").
		Select(endpointTable + ".*").
		Order(endpointTable + ".priority ASC, " + endpointTable + ".weight DESC")

	if err := db.Find(&list).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return list, nil
}
