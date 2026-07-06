package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy loadbalance storage instance (only active records)
func GetPolicyLoadbalanceDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyLoadbalance)).Where("deleted = '0'")
}

// Loadbalance policy management
type PolicyLoadbalance struct {
	DB *gorm.DB
}

// Query policy loadbalances from the database based on the provided parameters and options.
func (a *PolicyLoadbalance) Query(ctx context.Context, params schema.PolicyLoadbalanceQueryParam, opts ...schema.PolicyLoadbalanceQueryOptions) (*schema.PolicyLoadbalanceQueryResult, error) {
	var opt schema.PolicyLoadbalanceQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyLoadbalanceDB(ctx, a.DB)
	if v := params.ModelID; v != "" {
		db = db.Where("model_id = ?", v)
	} else {
		db = db.Where("model_id = '' OR model_id IS NULL")
	}
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}
	if v := params.Type; v != "" {
		db = db.Where("type = ?", v)
	}
	if params.ModelID != "" && !util.FromIsRootUser(ctx) {
		permTable := config.C.FormatTableName("data_permission")
		db = db.Where("model_id IN (SELECT data_id FROM "+permTable+" WHERE type = ? AND user = ? AND tenant = ? AND permission & 1 = 1)",
			"model", util.FromUsername(ctx), util.FromTenant(ctx))
	}

	var list schema.PolicyLoadbalances
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyLoadbalanceQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy loadbalance from the database.
func (a *PolicyLoadbalance) Get(ctx context.Context, id string, opts ...schema.PolicyLoadbalanceQueryOptions) (*schema.PolicyLoadbalance, error) {
	var opt schema.PolicyLoadbalanceQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyLoadbalance)
	ok, err := util.FindOne(ctx, GetPolicyLoadbalanceDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy loadbalance exists in the database.
func (a *PolicyLoadbalance) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyLoadbalanceDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByName checks whether a policy loadbalance with the given scope and name already exists.
func (a *PolicyLoadbalance) ExistsByName(ctx context.Context, scopeType, scopeCode, modelID, name string) (bool, error) {
	db := GetPolicyLoadbalanceDB(ctx, a.DB).Where("name = ?", name)
	if modelID == "" {
		db = db.Where("model_id = '' OR model_id IS NULL")
	} else {
		db = db.Where("model_id = ?", modelID)
	}
	ok, err := util.Exists(ctx, db)
	return ok, errors.WithStack(err)
}

// Create a new policy loadbalance.
func (a *PolicyLoadbalance) Create(ctx context.Context, item *schema.PolicyLoadbalance) error {
	result := GetPolicyLoadbalanceDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy loadbalance in the database.
func (a *PolicyLoadbalance) Update(ctx context.Context, item *schema.PolicyLoadbalance) error {
	result := GetPolicyLoadbalanceDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy loadbalance from the database using logical deletion.
func (a *PolicyLoadbalance) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyLoadbalanceDB(ctx, a.DB), id))
}
