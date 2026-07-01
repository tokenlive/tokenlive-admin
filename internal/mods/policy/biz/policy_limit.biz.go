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

// Limit policy management
type PolicyLimit struct {
	Trans            *util.Trans
	PolicyLimitDAL   *dal.PolicyLimit
	PolicyBindingDAL *dal.PolicyBinding
	PolicyRedisSync  *PolicyRedisSync
	AuditLogBIZ      *opsBiz.AuditLog
}

// Query policy limits from the data access object based on the provided parameters and options.
func (a *PolicyLimit) Query(ctx context.Context, params schema.PolicyLimitQueryParam) (*schema.PolicyLimitQueryResult, error) {
	params.Pagination = false

	result, err := a.PolicyLimitDAL.Query(ctx, params, schema.PolicyLimitQueryOptions{
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

// Get the specified policy limit from the data access object.
func (a *PolicyLimit) Get(ctx context.Context, id string) (*schema.PolicyLimitForm, error) {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if policyLimit == nil {
		return nil, errors.NotFound("", "Policy limit not found")
	}
	var policyLimitForm schema.PolicyLimitForm
	if err := policyLimit.ConvertTo(&policyLimitForm); err != nil {
		return nil, err
	}
	return &policyLimitForm, nil
}

// Create a new policy limit in the data access object.
func (a *PolicyLimit) Create(ctx context.Context, formItem *schema.PolicyLimitForm) (*schema.PolicyLimit, error) {
	// Check unique key (name) before creating.
	if exists, err := a.PolicyLimitDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
		return nil, err
	} else if exists {
		return nil, errors.BadRequest("", "Policy limit with the same name already exists")
	}

	policyLimit := &schema.PolicyLimit{
		ID:        util.NewXID(),
		Deleted:   "0",
		CreatedAt: time.Now(),
	}

	username := util.FromUsername(ctx)
	if username != "" {
		policyLimit.Creator = &username
	}

	if err := formItem.FillTo(policyLimit); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLimitDAL.Create(ctx, policyLimit)
	})
	if err != nil {
		return nil, err
	}
	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionCreate, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, nil, policyLimit)
	return policyLimit, nil
}

// Update the specified policy limit in the data access object.
func (a *PolicyLimit) Update(ctx context.Context, id string, formItem *schema.PolicyLimitForm) error {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLimit == nil {
		return errors.NotFound("", "Policy limit not found")
	}

	// If unique key fields changed, ensure the new combination is not occupied.
	if policyLimit.Name != formItem.Name {
		if exists, err := a.PolicyLimitDAL.ExistsByUniqueKey(ctx, formItem.Name); err != nil {
			return err
		} else if exists {
			return errors.BadRequest("", "Policy limit with the same name already exists")
		}
	}

	beforePolicy := *policyLimit

	if err := formItem.FillTo(policyLimit); err != nil {
		return err
	}
	policyLimit.UpdatedAt = time.Now()

	username := util.FromUsername(ctx)
	if username != "" {
		policyLimit.Modifier = &username
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLimitDAL.Update(ctx, policyLimit)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "limit", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionUpdate, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, beforePolicy, policyLimit)
	return nil
}

// Delete the specified policy limit from the data access object.
func (a *PolicyLimit) Delete(ctx context.Context, id string) error {
	policyLimit, err := a.PolicyLimitDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if policyLimit == nil {
		return errors.NotFound("", "Policy limit not found")
	}

	if err := ensurePolicyUnbound(ctx, a.PolicyBindingDAL, "limit", id); err != nil {
		return err
	}

	err = a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.PolicyLimitDAL.Delete(ctx, id)
	})
	if err != nil {
		return err
	}

	// 级联同步引用此策略的维度到 Redis
	if err := a.PolicyRedisSync.SyncPolicyChange(ctx, "limit", id); err != nil {
		return err
	}

	a.AuditLogBIZ.RecordAction(ctx, opsSchema.AuditActionDelete, opsSchema.AuditResourceTypePolicy, policyLimit.ID, policyLimit.Name, policyLimit, nil)
	return nil
}
