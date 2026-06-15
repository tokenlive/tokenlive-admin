package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Route policy management
type PolicyRoute struct {
	Trans            *util.Trans
	PolicyRouteDAL   *dal.PolicyRoute
	PolicyBindingDAL *dal.PolicyBinding
	PolicyRedisSync  *PolicyRedisSync
}

// Query policy routes from the data access object based on the provided parameters and options.
func (a *PolicyRoute) Query(ctx context.Context, params schema.PolicyRouteQueryParam) (*schema.PolicyRouteQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyRouteDAL.Query(ctx, params, schema.PolicyRouteQueryOptions{
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

// Get the specified policy route from the data access object.
func (a *PolicyRoute) Get(ctx context.Context, id string) (*schema.PolicyRouteForm, error) {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyRoute == nil {
		return nil, errors.NotFound("", "Policy route not found")
	}
	var policyRouteForm schema.PolicyRouteForm
	if err := policyRoute.ConvertTo(&policyRouteForm); err != nil {
		return nil, err
	}
	return &policyRouteForm, nil
}

// Create a new policy route in the data access object.
func (a *PolicyRoute) Create(ctx context.Context, formItem *schema.PolicyRouteForm) (*schema.PolicyRoute, error) {
	// Check unique key before creating.
	if exists, err := a.PolicyRouteDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy route with the same name already exists")
	}

	policyRoute := &schema.PolicyRoute{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyRoute.Creator = &username
	}

	if err := formItem.FillTo(policyRoute); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Create(ctx, policyRoute)
	})
	if err != nil {
		return nil, err
	}
	return policyRoute, nil
}

// Update the specified policy route in the data access object.
func (a *PolicyRoute) Update(ctx context.Context, id string, formItem *schema.PolicyRouteForm) error {
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRoute == nil {
		return errors.NotFound("", "Policy route not found")
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyRoute.Name != formItem.Name {
		if exists, err := a.PolicyRouteDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy route with the same name already exists")
		}
	}

	if err := formItem.FillTo(policyRoute); err != nil {
		return err
	}
	policyRoute.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyRoute.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Update(ctx, policyRoute)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "route", id); err != nil {
		return err
	}

	return nil
}

// Delete the specified policy route from the data access object.
func (a *PolicyRoute) Delete(ctx context.Context, id string) error {
	exists, err := a.PolicyRouteDAL.Exists(ctx, id)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Policy route not found")
	}

	// Check if the policy is bound to any dimension
	isBound, err := a.PolicyBindingDAL.ExistsByPolicyID(ctx, "route", id)
	if err != nil {
		return err
	}
	if isBound {
		return errors.BadRequest("", "Cannot delete policy: it is currently bound to one or more dimensions. Please unbind it first.")
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "route", id); err != nil {
		return err
	}

	return nil
}
