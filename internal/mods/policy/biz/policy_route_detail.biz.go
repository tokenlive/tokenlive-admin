package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Route policy detail management
type PolicyRouteDetail struct {
	Trans                *util.Trans
	PolicyRouteDetailDAL *dal.PolicyRouteDetail
	PolicyRouteDAL       *dal.PolicyRoute
	PolicyRedisSync      PolicyChangeSyncer
}

// Query policy route details from the data access object based on the provided parameters and options.
func (a *PolicyRouteDetail) Query(ctx context.Context, params schema.PolicyRouteDetailQueryParam) (*schema.PolicyRouteDetailQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyRouteDetailDAL.Query(ctx, params, schema.PolicyRouteDetailQueryOptions{
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

// Get the specified policy route detail from the data access object.
func (a *PolicyRouteDetail) Get(ctx context.Context, id string) (*schema.PolicyRouteDetailForm, error) {
	policyRouteDetail, err := a.PolicyRouteDetailDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyRouteDetail == nil {
		return nil, errors.NotFound("", "Policy route detail not found")
	}
	var policyRouteDetailForm schema.PolicyRouteDetailForm
	if err := policyRouteDetail.ConvertTo(&policyRouteDetailForm); err != nil {
		return nil, err
	}
	return &policyRouteDetailForm, nil
}

// Create a new policy route detail in the data access object.
func (a *PolicyRouteDetail) Create(ctx context.Context, formItem *schema.PolicyRouteDetailForm) (*schema.PolicyRouteDetail, error) {
	policyRouteDetail := &schema.PolicyRouteDetail{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(policyRouteDetail); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDetailDAL.Create(ctx, policyRouteDetail)
	})
	if err != nil {
		return nil, err
	}
	if err := a.syncParentRoutePolicy(ctx, policyRouteDetail.RouteId); err != nil {
		return nil, err
	}
	return policyRouteDetail, nil
}

// Update the specified policy route detail in the data access object.
func (a *PolicyRouteDetail) Update(ctx context.Context, id string, formItem *schema.PolicyRouteDetailForm) error {
	policyRouteDetail, err := a.PolicyRouteDetailDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRouteDetail == nil {
		return errors.NotFound("", "Policy route detail not found")
	}

	if err := formItem.FillTo(policyRouteDetail); err != nil {
		return err
	}
	policyRouteDetail.UpdatedAt = time.Now()

	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDetailDAL.Update(ctx, policyRouteDetail)
	}); err != nil {
		return err
	}
	return a.syncParentRoutePolicy(ctx, policyRouteDetail.RouteId)
}

// Delete the specified policy route detail from the data access object.
func (a *PolicyRouteDetail) Delete(ctx context.Context, id string) error {
	policyRouteDetail, err := a.PolicyRouteDetailDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyRouteDetail == nil {
		return errors.NotFound("", "Policy route detail not found")
	}

	if err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDetailDAL.Delete(ctx, id)
	}); err != nil {
		return err
	}
	return a.syncParentRoutePolicy(ctx, policyRouteDetail.RouteId)
}

func (a *PolicyRouteDetail) syncParentRoutePolicy(ctx context.Context, routeID string) error {
	if a.PolicyRedisSync == nil || a.PolicyRouteDAL == nil {
		return nil
	}
	policyRoute, err := a.PolicyRouteDAL.Get(ctx, routeID)
	if err != nil {
		return err
	}
	if policyRoute == nil {
		return errors.NotFound("", "Policy route not found")
	}
	return syncPolicyChangeAndLog(ctx, a.PolicyRedisSync, "route_detail", "sync_parent_route_policy", policyRoute.ScopeType, policyRoute.ScopeCode, policyRoute.ModelID)
}
