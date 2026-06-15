package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Circuit break policy management
type PolicyCircuitBreak struct {
	Trans                 *util.Trans
	PolicyCircuitBreakDAL *dal.PolicyCircuitBreak
	PolicyBindingDAL      *dal.PolicyBinding
	PolicyRedisSync       *PolicyRedisSync
}

// Query policy circuit breaks from the data access object based on the provided parameters and options.
func (a *PolicyCircuitBreak) Query(ctx context.Context, params schema.PolicyCircuitBreakQueryParam) (*schema.PolicyCircuitBreakQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyCircuitBreakDAL.Query(ctx, params, schema.PolicyCircuitBreakQueryOptions{
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

// Get the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Get(ctx context.Context, id string) (*schema.PolicyCircuitBreakForm, error) {
	policyCircuitBreak, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyCircuitBreak == nil {
		return nil, errors.NotFound("", "Policy circuit break not found")
	}
	var form schema.PolicyCircuitBreakForm
	if err := policyCircuitBreak.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Create(ctx context.Context, formItem *schema.PolicyCircuitBreakForm) (*schema.PolicyCircuitBreak, error) {
	// Check unique key before creating.
	if exists, err := a.PolicyCircuitBreakDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy circuit break with the same name already exists")
	}

	policyCircuitBreak := &schema.PolicyCircuitBreak{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyCircuitBreak); err != nil {
		return nil, err
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyCircuitBreak.Creator = &username
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyCircuitBreakDAL.Create(ctx, policyCircuitBreak)
	})
	if err != nil {
		return nil, err
	}
	return policyCircuitBreak, nil
}

// Update the specified policy circuit break in the data access object.
func (a *PolicyCircuitBreak) Update(ctx context.Context, id string, formItem *schema.PolicyCircuitBreakForm) error {
	policyCircuitBreak, err := a.PolicyCircuitBreakDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyCircuitBreak == nil {
		return errors.NotFound("", "Policy circuit break not found")
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyCircuitBreak.Name != formItem.Name {
		if exists, err := a.PolicyCircuitBreakDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy circuit break with the same name already exists")
		}
	}

	if err := formItem.FillTo(policyCircuitBreak); err != nil {
		return err
	}
	policyCircuitBreak.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyCircuitBreak.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyCircuitBreakDAL.Update(ctx, policyCircuitBreak)
	})
	if err != nil {
		return err
	}

	// 级联更新引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "circuit_break", id); err != nil {
		return err
	}

	return nil
}

// Delete the specified policy circuit break from the data access object.
func (a *PolicyCircuitBreak) Delete(ctx context.Context, id string) error {
	exists, err := a.PolicyCircuitBreakDAL.Exists(ctx, id)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Policy circuit break not found")
	}

	// Check if the policy is bound to any dimension
	isBound, err := a.PolicyBindingDAL.ExistsByPolicyID(ctx, "circuit_break", id)
	if err != nil {
		return err
	}
	if isBound {
		return errors.BadRequest("", "Cannot delete policy: it is currently bound to one or more dimensions. Please unbind it first.")
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyCircuitBreakDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联更新引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "circuit_break", id); err != nil {
		return err
	}

	return nil
}
