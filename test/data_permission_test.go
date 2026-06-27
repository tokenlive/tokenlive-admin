package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestDataPermissionRejectsCrossTenantWriteForNonRootUser(t *testing.T) {
	biz := testInjector.M.Resource.DataPermissionAPI.DataPermissionBIZ
	ctx := util.NewTenant(util.NewUsername(context.Background(), "tenant-user"), "tenant-a")

	form := &resourceSchema.DataPermissionForm{
		Type:       resourceSchema.DataPermissionTypeModel,
		DataId:     "model-cross-tenant",
		User:       "other-user",
		Tenant:     "tenant-b",
		Role:       "viewer",
		Permission: 1,
	}

	created, err := biz.Create(ctx, form)
	require.Error(t, err)
	require.Nil(t, created)
}
