package biz

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm/clause"
)

// Change the model membership and its endpoint restriction together. The
// membership operation is first so a corrupt Redis set cannot erase a valid
// endpoint restriction before returning an error.
var reconcileReusedModelAccessScript = redis.NewScript(`
if ARGV[2] == '1' then
  redis.call('SADD', KEYS[1], ARGV[1])
else
  redis.call('SREM', KEYS[1], ARGV[1])
end
redis.call('DEL', KEYS[2])
if ARGV[2] == '1' then
  for i = 3, #ARGV do
    redis.call('SADD', KEYS[2], ARGV[i])
  end
end
return 1
`)

// The caller holds the current owner's row lock. Reusing a code must preserve
// that owner's routing, but cannot inherit the former owner's tenant grants.
func (s *ConfigRedisSync) reconcileReusedModelAccess(ctx context.Context, owner *schema.Model) error {
	db := util.GetDB(ctx, s.ModelDAL.DB)
	var ownerTenants []string
	if err := db.Table(config.C.FormatTableName("tenant_model")).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("model_id = ?", owner.ID).Pluck("tenant_code", &ownerTenants).Error; err != nil {
		return err
	}
	allowed := make(map[string]bool, len(ownerTenants))
	tenants := make(map[string]bool, len(ownerTenants))
	for _, tenant := range ownerTenants {
		tenants[tenant] = true
		allowed[tenant] = owner.Enabled == 1 && owner.Deleted == "0"
	}
	// Include cached grants with no remaining DB binding as well as endpoint
	// restrictions whose membership key is already absent.
	const prefix = "aigw:tenant:"
	for _, suffix := range []string{":models", ":model:" + owner.ModelCode + ":endpoints"} {
		iter := s.RedisClient.Scan(ctx, 0, prefix+"*"+suffix, 100).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			tenant := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
			if tenant != "" {
				tenants[tenant] = true
			}
		}
		if err := iter.Err(); err != nil {
			return err
		}
	}
	for tenant := range tenants {
		var endpointIDs []string
		isAllowed := 0
		if allowed[tenant] {
			isAllowed = 1
			if err := db.Table(config.C.FormatTableName("tenant_endpoint")+" AS te").Clauses(clause.Locking{Strength: "UPDATE"}).
				Select("te.endpoint_id").
				Joins("JOIN "+config.C.FormatTableName("endpoint")+" AS ep ON te.endpoint_id = ep.id AND ep.deleted = '0'").
				Where("te.tenant_code = ? AND ep.model_id = ?", tenant, owner.ID).
				Pluck("te.endpoint_id", &endpointIDs).Error; err != nil {
				return err
			}
		}
		args := []interface{}{owner.ModelCode, isAllowed}
		for _, id := range endpointIDs {
			args = append(args, id)
		}
		if _, err := reconcileReusedModelAccessScript.Run(ctx, s.RedisClient, []string{
			prefix + tenant + ":models",
			prefix + tenant + ":model:" + owner.ModelCode + ":endpoints",
		}, args...).Result(); err != nil {
			return err
		}
	}
	ClearGatewayConfigCache()
	util.NotifyConfigChanged(ctx, util.ConfigChangeAll, owner.ModelCode)
	return nil
}
