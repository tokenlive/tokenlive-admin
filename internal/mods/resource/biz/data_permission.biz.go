package biz

import (
	"context"
	"time"

	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

// Data permission management
type DataPermission struct {
	Trans             *util.Trans
	DataPermissionDAL *dal.DataPermission
}

// Query data permissions from the data access object based on the provided parameters and options.
func (a *DataPermission) Query(ctx context.Context, params schema.DataPermissionQueryParam) (*schema.DataPermissionQueryResult, error) {
	result, err := a.DataPermissionDAL.Query(ctx, params, schema.DataPermissionQueryOptions{
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

// Get the specified data permission from the data access object.
func (a *DataPermission) Get(ctx context.Context, id string) (*schema.DataPermission, error) {
	dataPermission, err := a.DataPermissionDAL.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if dataPermission == nil {
		return nil, errors.NotFound("", "Data permission not found")
	}
	return dataPermission, nil
}

// Create a new data permission in the data access object.
func (a *DataPermission) Create(ctx context.Context, formItem *schema.DataPermissionForm) (*schema.DataPermission, error) {
	dataPermission := &schema.DataPermission{
		ID:        util.NewXID(),
		Creator:   util.FromUsername(ctx),
		CreatedAt: time.Now(),
	}

	if err := formItem.FillTo(dataPermission); err != nil {
		return nil, err
	}

	err := a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.DataPermissionDAL.Create(ctx, dataPermission)
	})
	if err != nil {
		return nil, err
	}
	return dataPermission, nil
}

// Update the specified data permission in the data access object.
func (a *DataPermission) Update(ctx context.Context, id string, formItem *schema.DataPermissionForm) error {
	dataPermission, err := a.DataPermissionDAL.Get(ctx, id)
	if err != nil {
		return err
	} else if dataPermission == nil {
		return errors.NotFound("", "Data permission not found")
	}

	if err := formItem.FillTo(dataPermission); err != nil {
		return err
	}
	dataPermission.UpdatedAt = time.Now()

	return a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.DataPermissionDAL.Update(ctx, dataPermission)
	})
}

// Delete the specified data permission from the data access object.
func (a *DataPermission) Delete(ctx context.Context, id string) error {
	exists, err := a.DataPermissionDAL.Exists(ctx, id)
	if err != nil {
		return err
	} else if !exists {
		return errors.NotFound("", "Data permission not found")
	}

	return a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.DataPermissionDAL.Delete(ctx, id)
	})
}

// DeleteByTypeAndDataId deletes data permissions by type and data ID.
func (a *DataPermission) DeleteByTypeAndDataId(ctx context.Context, dataType, dataId string) error {
	return a.Trans.Exec(ctx, func(ctx context.Context) error {
		return a.DataPermissionDAL.DeleteByTypeAndDataId(ctx, dataType, dataId)
	})
}

// CreateByOwner creates a data permission record for the resource owner.
func (a *DataPermission) CreateByOwner(ctx context.Context, dataType, dataId, tenant string) error {
	perm := &schema.DataPermission{
		ID:         util.NewXID(),
		Type:       dataType,
		DataId:     dataId,
		User:       util.FromUsername(ctx),
		Tenant:     tenant,
		Role:       "owner",
		Permission: 0b111, // read + write + delete
		Creator:    util.FromUsername(ctx),
		CreatedAt:  time.Now(),
	}
	return a.DataPermissionDAL.Create(ctx, perm)
}

// HasReadPermission checks if the current user has read permission for a specific data item.
func (a *DataPermission) HasReadPermission(ctx context.Context, dataType, dataId string) (bool, error) {
	return a.DataPermissionDAL.HasReadPermission(ctx, dataType, dataId, util.FromUsername(ctx), util.FromTenant(ctx))
}
