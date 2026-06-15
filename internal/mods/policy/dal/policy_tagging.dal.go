package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

// Get policy tagging storage instance (only active records)
func GetPolicyTaggingDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.PolicyTagging)).Where("deleted = '0'")
}

// Tagging policy management
type PolicyTagging struct {
	DB *gorm.DB
}

// Query policy taggings from the database based on the provided parameters and options.
func (a *PolicyTagging) Query(ctx context.Context, params schema.PolicyTaggingQueryParam, opts ...schema.PolicyTaggingQueryOptions) (*schema.PolicyTaggingQueryResult, error) {
	var opt schema.PolicyTaggingQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	db := GetPolicyTaggingDB(ctx, a.DB)
	if v := params.Name; v != "" {
		db = db.Where("name LIKE ?", "%"+v+"%")
	}

	var list schema.PolicyTaggings
	pageResult, err := util.WrapPageQuery(ctx, db, params.PaginationParam, opt.QueryOptions, &list)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	queryResult := &schema.PolicyTaggingQueryResult{
		PageResult: pageResult,
		Data:       list,
	}
	return queryResult, nil
}

// Get the specified policy tagging from the database.
func (a *PolicyTagging) Get(ctx context.Context, id string, opts ...schema.PolicyTaggingQueryOptions) (*schema.PolicyTagging, error) {
	var opt schema.PolicyTaggingQueryOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	item := new(schema.PolicyTagging)
	ok, err := util.FindOne(ctx, GetPolicyTaggingDB(ctx, a.DB).Where("id=?", id), opt.QueryOptions, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

// Exists checks if the specified policy tagging exists in the database.
func (a *PolicyTagging) Exists(ctx context.Context, id string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyTaggingDB(ctx, a.DB).Where("id=?", id))
	return ok, errors.WithStack(err)
}

// ExistsByName checks whether a policy tagging with the given name already exists.
func (a *PolicyTagging) ExistsByName(ctx context.Context, name string) (bool, error) {
	ok, err := util.Exists(ctx, GetPolicyTaggingDB(ctx, a.DB).Where("name = ?", name))
	return ok, errors.WithStack(err)
}

// Create a new policy tagging.
func (a *PolicyTagging) Create(ctx context.Context, item *schema.PolicyTagging) error {
	result := GetPolicyTaggingDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

// Update the specified policy tagging in the database.
func (a *PolicyTagging) Update(ctx context.Context, item *schema.PolicyTagging) error {
	result := GetPolicyTaggingDB(ctx, a.DB).Where("id=?", item.ID).Select("*").Omit("created_at").Updates(item)
	return errors.WithStack(result.Error)
}

// Delete the specified policy tagging from the database using logical deletion.
func (a *PolicyTagging) Delete(ctx context.Context, id string) error {
	return errors.WithStack(util.SoftDelete(ctx, GetPolicyTaggingDB(ctx, a.DB), id))
}
