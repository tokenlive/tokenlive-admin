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

// Loadbalance policy management
type PolicyLoadbalance struct {
	Trans                *util.Trans
	PolicyLoadbalanceDAL *dal.PolicyLoadbalance
	PolicyBindingDAL     *dal.PolicyBinding
	PolicyRedisSync      *PolicyRedisSync
	AuditLogBIZ          *opsBiz.AuditLog
}

// Query policy loadbalances from the data access object based on the provided parameters and options.
func (a *PolicyLoadbalance) Query(ctx context.Context, params schema.PolicyLoadbalanceQueryParam) (*schema.PolicyLoadbalanceQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyLoadbalanceDAL.Query(ctx, params, schema.PolicyLoadbalanceQueryOptions{
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

// Get the specified policy loadbalance from the data access object.
func (a *PolicyLoadbalance) Get(ctx context.Context, id string) (*schema.PolicyLoadbalanceForm, error) {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyLoadbalance == nil {
		return nil, errors.NotFound("", "Policy loadbalance not found")
	}
	var form schema.PolicyLoadbalanceForm
	if err := policyLoadbalance.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy loadbalance in the data access object.
func (a *PolicyLoadbalance) Create(ctx context.Context, formItem *schema.PolicyLoadbalanceForm) (*schema.PolicyLoadbalance, error) {
	// Check unique key (name) before creating.
	if exists, err := a.PolicyLoadbalanceDAL.ExistsByName(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy loadbalance with the same name already exists")
	}

	creator := util.FromUsername(ctx)
	policyLoadbalance := &schema.PolicyLoadbalance{
		ID:        util.NewXID(),
		Deleted:   "0",
		Creator:   &creator,
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyLoadbalance); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Create(ctx, policyLoadbalance)
	})
	if err != nil {
		return nil, err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, nil, policyLoadbalance)

	return policyLoadbalance, nil
}

// Update the specified policy loadbalance in the data access object.
func (a *PolicyLoadbalance) Update(ctx context.Context, id string, formItem *schema.PolicyLoadbalanceForm) error {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLoadbalance == nil {
		return errors.NotFound("", "Policy loadbalance not found")
	}

	// If name changed, ensure the new name is not occupied.
	if policyLoadbalance.Name != formItem.Name {
		if exists, err := a.PolicyLoadbalanceDAL.ExistsByName(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy loadbalance with the same name already exists")
		}
	}

	beforePolicy := *policyLoadbalance

	if err := formItem.FillTo(policyLoadbalance); err != nil {
		return err
	}
	modifier := util.FromUsername(ctx)
	policyLoadbalance.Modifier = &modifier
	policyLoadbalance.UpdatedAt = time.Now()

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Update(ctx, policyLoadbalance)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "loadbalance", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, beforePolicy, policyLoadbalance)

	return nil
}

// Delete the specified policy loadbalance from the data access object.
func (a *PolicyLoadbalance) Delete(ctx context.Context, id string) error {
	policyLoadbalance, err := a.PolicyLoadbalanceDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLoadbalance == nil {
		return errors.NotFound("", "Policy loadbalance not found")
	}

	if err := ensurePolicyUnbound(ctx, a.PolicyBindingDAL, "loadbalance", id); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLoadbalanceDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "loadbalance", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyLoadbalance.ID, policyLoadbalance.Name, policyLoadbalance, nil)

	return nil
}
