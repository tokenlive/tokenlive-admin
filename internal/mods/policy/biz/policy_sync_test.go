package biz

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	policyDal "github.com/tokenlive/tokenlive-admin/internal/mods/policy/dal"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	resourceDal "github.com/tokenlive/tokenlive-admin/internal/mods/resource/dal"
	resourceSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"gorm.io/gorm"
)

func TestResolveRedisKeyAndField_NoGlobalPolicyKey(t *testing.T) {
	key, field, ok := resolveRedisKeyAndField("", "", "")

	require.False(t, ok)
	require.Empty(t, key)
	require.Empty(t, field)
}

func TestResolveRedisKeyAndField_ModelAndScopedKeys(t *testing.T) {
	key, field, ok := resolveRedisKeyAndField("", "", "gpt-5")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:model:gpt-5", key)
	require.Equal(t, "*", field)

	key, field, ok = resolveRedisKeyAndField("tenant-a", "", "gpt-5")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:tenant:tenant-a", key)
	require.Equal(t, "gpt-5", field)

	key, field, ok = resolveRedisKeyAndField("tenant-a", "", "")
	require.True(t, ok)
	require.Equal(t, "aigw:policies:tenant:tenant-a", key)
	require.Equal(t, "*", field)
}

func TestSyncDimensionDeletesRedisFieldWhenModelNoLongerExists(t *testing.T) {
	oldSyncPolicies := config.C.Sync.Policies
	config.C.Sync.Policies = true
	t.Cleanup(func() { config.C.Sync.Policies = oldSyncPolicies })

	db := newPolicySyncTestDB(t)
	require.NoError(t, db.Create(&resourceSchema.Model{
		ID:           "model-1",
		ModelName:    "Deleted Model",
		ModelCode:    "deleted-model",
		SpaceCode:    "default",
		RequestTypes: "[]",
		Deleted:      "model-1",
		CreatedAt:    time.Now(),
	}).Error)
	require.NoError(t, db.Create(&policySchema.PolicyLoadbalance{
		ID:        "template-1",
		Name:      "Template Loadbalance",
		Type:      "ROUND_ROBIN",
		ScopeType: "global",
		Enabled:   1,
		Deleted:   "0",
		CreatedAt: time.Now(),
	}).Error)

	hook := &recordingRedisHook{}
	redisClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	redisClient.AddHook(hook)
	syncer := &PolicyRedisSync{
		RedisClient:           redisClient,
		PolicyLoadbalanceDAL:  &policyDal.PolicyLoadbalance{DB: db},
		PolicyRouteDAL:        &policyDal.PolicyRoute{DB: db},
		PolicyLimitDAL:        &policyDal.PolicyLimit{DB: db},
		PolicyCircuitBreakDAL: &policyDal.PolicyCircuitBreak{DB: db},
		PolicyInvocationDAL:   &policyDal.PolicyInvocation{DB: db},
		PolicyTaggingDAL:      &policyDal.PolicyTagging{DB: db},
		ModelDAL:              &resourceDal.Model{DB: db},
	}

	err := syncer.SyncDimension(context.Background(), "", "", "deleted-model")

	require.NoError(t, err)
	require.Contains(t, hook.commands, "hdel")
	require.NotContains(t, hook.commands, "hset")
}

type recordingRedisHook struct {
	commands []string
}

func (h *recordingRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *recordingRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		h.commands = append(h.commands, cmd.Name())
		return nil
	}
}

func (h *recordingRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func newPolicySyncTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbName := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", dbName)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&policySchema.PolicyLoadbalance{},
		&policySchema.PolicyRoute{},
		&policySchema.PolicyRouteDetail{},
		&policySchema.PolicyLimit{},
		&policySchema.PolicyCircuitBreak{},
		&policySchema.PolicyInvocation{},
		&policySchema.PolicyTagging{},
		&resourceSchema.Model{},
	))
	return db
}
