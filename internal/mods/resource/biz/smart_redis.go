package biz

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
	"gorm.io/gorm/clause"
)

const RedisKeySmartRoutingPrefix = "aigw:config:smart_routing:"

// The version increment is first: Redis rejects a bad version hash before
// changing any routing keys. Lua keeps readers from seeing a half publication.
var publishRoutingScript = redis.NewScript(`
if ARGV[3] ~= '' then
  local old = redis.call('GET', KEYS[1])
  if old then
    local ok, routing = pcall(cjson.decode, old)
    if ok and routing.version and tonumber(routing.version) > tonumber(ARGV[3]) then
      return redis.error_reply('STALE smart routing version')
    end
  end
end
local generation = redis.call('HINCRBY', KEYS[3], ARGV[1], 1)
redis.call('SET', KEYS[1], ARGV[2])
redis.call('DEL', KEYS[2])
return generation
`)

var deleteRoutingScript = redis.NewScript(`
redis.call('HDEL', KEYS[3], ARGV[1])
redis.call('DEL', KEYS[1], KEYS[2])
return 1
`)

func (s *ConfigRedisSync) publishSmartRouting(ctx context.Context, model *schema.Model) error {
	return (&util.Trans{DB: s.ModelDAL.DB}).Exec(ctx, func(ctx context.Context) error {
		var current schema.Model
		if err := util.GetDB(ctx, s.ModelDAL.DB).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted = '0'", model.ID).First(&current).Error; err != nil {
			return err
		}
		if current.Enabled != 1 || current.ModelType != schema.ModelTypeSmart || current.ModelCode != model.ModelCode ||
			current.SmartRouting == nil || model.SmartRouting == nil || current.SmartRouting.Version != model.SmartRouting.Version {
			return fmt.Errorf("smart routing publication superseded by a newer model state")
		}
		runtime, err := resolveSmartRuntime(ctx, s.ModelDAL.DB, &current)
		if err != nil {
			return err
		}
		data, err := json.Marshal(runtime)
		if err != nil {
			return err
		}
		return s.publishRoutingVersion(ctx, model.ModelCode, RedisKeySmartRoutingPrefix+model.ModelCode, "aigw:config:endpoints:"+model.ModelCode, data, runtime.Version)
	})
}

func (s *ConfigRedisSync) publishRouting(ctx context.Context, code, activeKey, obsoleteKey string, data []byte) error {
	return s.publishRoutingVersion(ctx, code, activeKey, obsoleteKey, data, "")
}

func (s *ConfigRedisSync) publishRoutingVersion(ctx context.Context, code, activeKey, obsoleteKey string, data []byte, version any) error {
	if _, err := publishRoutingScript.Run(ctx, s.RedisClient, []string{activeKey, obsoleteKey, RedisKeyConfigModelVersions}, code, string(data), version).Result(); err != nil {
		return err
	}
	ClearGatewayConfigCache()
	util.NotifyConfigChanged(ctx, util.ConfigChangeAll, code)
	return nil
}

func (s *ConfigRedisSync) deleteRouting(ctx context.Context, code string) error {
	if _, err := deleteRoutingScript.Run(ctx, s.RedisClient, []string{RedisKeySmartRoutingPrefix + code, "aigw:config:endpoints:" + code, RedisKeyConfigModelVersions}, code).Result(); err != nil {
		return err
	}
	ClearGatewayConfigCache()
	util.NotifyConfigChanged(ctx, util.ConfigChangeAll, code)
	return nil
}
