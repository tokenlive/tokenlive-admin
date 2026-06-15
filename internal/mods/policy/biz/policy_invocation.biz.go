package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Invocation policy management
type PolicyInvocation struct {
	Trans               *util.Trans
	PolicyInvocationDAL *dal.PolicyInvocation
	PolicyBindingDAL    *dal.PolicyBinding
	PolicyRedisSync     *PolicyRedisSync
}

// Query policy invocations from the data access object based on the provided parameters and options.
func (a *PolicyInvocation) Query(ctx context.Context, params schema.PolicyInvocationQueryParam) (*schema.PolicyInvocationQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyInvocationDAL.Query(ctx, params, schema.PolicyInvocationQueryOptions{
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

// Get the specified policy invocation from the data access object.
func (a *PolicyInvocation) Get(ctx context.Context, id string) (*schema.PolicyInvocationForm, error) {
	policyInvocation, err := a.PolicyInvocationDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyInvocation == nil {
		return nil, errors.NotFound("", "Policy invocation not found")
	}
	var form schema.PolicyInvocationForm
	if err := policyInvocation.ConvertTo(&form); err != nil {
		return nil, err
	}
	return &form, nil
}

// Create a new policy invocation in the data access object.
func (a *PolicyInvocation) Create(ctx context.Context, formItem *schema.PolicyInvocationForm) (*schema.PolicyInvocation, error) {
	// Check unique key before creating.
	if exists, err := a.PolicyInvocationDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy invocation with the same name already exists")
	}

	policyInvocation := &schema.PolicyInvocation{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyInvocation); err != nil {
		return nil, err
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyInvocation.Creator = &username
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyInvocationDAL.Create(ctx, policyInvocation)
	})
	if err != nil {
		return nil, err
	}
	return policyInvocation, nil
}

// Update the specified policy invocation in the data access object.
func (a *PolicyInvocation) Update(ctx context.Context, id string, formItem *schema.PolicyInvocationForm) error {
	policyInvocation, err := a.PolicyInvocationDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyInvocation == nil {
		return errors.NotFound("", "Policy invocation not found")
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyInvocation.Name != formItem.Name {
		if exists, err := a.PolicyInvocationDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy invocation with the same name already exists")
		}
	}

	if err := formItem.FillTo(policyInvocation); err != nil {
		return err
	}
	policyInvocation.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyInvocation.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyInvocationDAL.Update(ctx, policyInvocation)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "invocation", id); err != nil {
		return err
	}

	return nil
}

// Delete the specified policy invocation from the data access object.
func (a *PolicyInvocation) Delete(ctx context.Context, id string) error {
	exists, err := a.PolicyInvocationDAL.Exists(ctx, id)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Policy invocation not found")
	}

	// Check if the policy is bound to any dimension
	isBound, err := a.PolicyBindingDAL.ExistsByPolicyID(ctx, "invocation", id)
	if err != nil {
		return err
	}
	if isBound {
		return errors.BadRequest("", "Cannot delete policy: it is currently bound to one or more dimensions. Please unbind it first.")
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyInvocationDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "invocation", id); err != nil {
		return err
	}

	return nil
}
