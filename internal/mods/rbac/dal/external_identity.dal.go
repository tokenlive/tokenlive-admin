package dal

import (
	"context"

	"github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/errors"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm"
)

func GetExternalIdentityDB(ctx context.Context, defDB *gorm.DB) *gorm.DB {
	return util.GetDB(ctx, defDB).Model(new(schema.ExternalIdentity))
}

type ExternalIdentity struct {
	DB *gorm.DB
}

func (a *ExternalIdentity) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*schema.ExternalIdentity, error) {
	item := new(schema.ExternalIdentity)
	ok, err := util.FindOne(ctx, GetExternalIdentityDB(ctx, a.DB).Where("provider=? AND provider_user_id=?", provider, providerUserID), util.QueryOptions{}, item)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if !ok {
		return nil, nil
	}
	return item, nil
}

func (a *ExternalIdentity) Create(ctx context.Context, item *schema.ExternalIdentity) error {
	result := GetExternalIdentityDB(ctx, a.DB).Create(item)
	return errors.WithStack(result.Error)
}

func (a *ExternalIdentity) UpdateSnapshot(ctx context.Context, item *schema.ExternalIdentity) error {
	result := GetExternalIdentityDB(ctx, a.DB).Where("id=?", item.ID).Select("email", "email_verified", "display_name", "avatar_url", "updated_at").Updates(item)
	return errors.WithStack(result.Error)
}
