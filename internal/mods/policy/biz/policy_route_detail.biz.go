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

	return a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDetailDAL.Update(ctx, policyRouteDetail)
	})
}

// Delete the specified policy route detail from the data access object.
func (a *PolicyRouteDetail) Delete(ctx context.Context, id string) error {
	exists, err := a.PolicyRouteDetailDAL.Exists(ctx, id)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Policy route detail not found")
	}

	return a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyRouteDetailDAL.Delete(ctx, id)
	})
}
