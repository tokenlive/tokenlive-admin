package biz

import (
	"context"
	"time"

	opsBiz "github.com/tokenlive/tokenlive-admin/internal/mods/ops/biz"
	opsSchema "github.com/tokenlive/tokenlive-admin/internal/mods/ops/schema"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Tagging policy management
type PolicyTagging struct {
	Trans            *util.Trans
	PolicyTaggingDAL *dal.PolicyTagging
	PolicyBindingDAL *dal.PolicyBinding
	PolicyRedisSync  *PolicyRedisSync
	AuditLogBIZ      *opsBiz.AuditLog
}

// Query policy taggings from the data access object based on the provided parameters and options.
func (a *PolicyTagging) Query(ctx context.Context, params schema.PolicyTaggingQueryParam) (*schema.PolicyTaggingQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyTaggingDAL.Query(ctx, params, schema.PolicyTaggingQueryOptions{
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

// Get the specified policy tagging from the data access object.
func (a *PolicyTagging) Get(ctx context.Context, id string) (*schema.PolicyTaggingForm, error) {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyTagging == nil {
		return nil, errors.NotFound("", "Policy tagging not found")
	}
	var form schema.PolicyTaggingForm
	if err := policyTagging.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy tagging in the data access object.
func (a *PolicyTagging) Create(ctx context.Context, formItem *schema.PolicyTaggingForm) (*schema.PolicyTagging, error) {
	// Check unique key (name) before creating.
	if exists, err := a.PolicyTaggingDAL.ExistsByName(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy tagging with the same name already exists")
	}

	creator := util.FromUsername(ctx)
	policyTagging := &schema.PolicyTagging{
		ID:        util.NewXID(),
		Deleted:   "0",
		Creator:   &creator,
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyTagging); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Create(ctx, policyTagging)
	})
	if err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, nil, policyTagging)

	return policyTagging, nil
}

// Update the specified policy tagging in the data access object.
func (a *PolicyTagging) Update(ctx context.Context, id string, formItem *schema.PolicyTaggingForm) error {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyTagging == nil {
		return errors.NotFound("", "Policy tagging not found")
	}

	// If name changed, ensure the new name is not occupied.
	if policyTagging.Name != formItem.Name {
		if exists, err := a.PolicyTaggingDAL.ExistsByName(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy tagging with the same name already exists")
		}
	}

	beforePolicy := *policyTagging

	if err := formItem.FillTo(policyTagging); err != nil {
		return err
	}
	modifier := util.FromUsername(ctx)
	policyTagging.Modifier = &modifier
	policyTagging.UpdatedAt = time.Now()

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Update(ctx, policyTagging)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "tagging", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, beforePolicy, policyTagging)

	return nil
}

// Delete the specified policy tagging from the data access object.
func (a *PolicyTagging) Delete(ctx context.Context, id string) error {
	policyTagging, err := a.PolicyTaggingDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyTagging == nil {
		return errors.NotFound("", "Policy tagging not found")
	}

	if err := ensurePolicyUnbound(ctx, a.PolicyBindingDAL, "tagging", id); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyTaggingDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "tagging", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyTagging.ID, policyTagging.Name, policyTagging, nil)

	return nil
}
