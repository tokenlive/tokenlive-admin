package test

import (
	"context"
	"net/http"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/tokenlive/tokenlive-admin/internal/config"
	policySchema "github.com/tokenlive/tokenlive-admin/internal/mods/policy/schema"
	rbacSchema "github.com/tokenlive/tokenlive-admin/internal/mods/rbac/schema"
	modelSchema "github.com/tokenlive/tokenlive-admin/internal/mods/resource/schema"
	"github.com/tokenlive/tokenlive-admin/pkg/util"
)

func TestModelSync_OnCodeChange(t *testing.T) {
	if config.C.Storage.Cache.Type != "redis" {
		t.Skip("skip test because cache type is not redis")
	}

	e := tester(t)
	assert := assert.New(t)
	ctx := context.Background()

	// 1. Create a tenant
	tenantFormItem := rbacSchema.TenantForm{
		Code:        "t-sync-test",
		Name:        "Test Sync Tenant",
		Status:      rbacSchema.TenantStatusActivated,
		Description: "For testing sync",
	}
	var tenant rbacSchema.Tenant
	e.POST(baseAPI + "/tenants").WithJSON(tenantFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &tenant})
	assert.NotEmpty(tenant.ID)

	// 2. Create a model
	modelFormItem := modelSchema.ModelForm{
		ModelName:     "m-sync-test-name",
		ModelCode:     "m-sync-old",
		SpaceCode:     "default",
		RequestTypes:  `["chat_completion"]`,
		ContextLength: 128000,
		Enabled:       1,
		Description:   "Model for sync test",
	}
	var model modelSchema.Model
	e.POST(baseAPI + "/models").WithJSON(modelFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &model})
	assert.NotEmpty(model.ID)

	// 3. Bind model to tenant
	bindingFormItem := rbacSchema.TenantModelForm{
		TenantCode: tenant.Code,
		ModelIDs:   []string{model.ID},
	}
	e.POST(baseAPI + "/tenant-models/bindings").WithJSON(bindingFormItem).
		Expect().Status(http.StatusOK)

	// 4. Initialize Redis Client to verify
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.C.Storage.Cache.Redis.Addr,
		DB:       config.C.Storage.Cache.Redis.DB,
		Username: config.C.Storage.Cache.Redis.Username,
		Password: config.C.Storage.Cache.Redis.Password,
	})
	defer rdb.Close()

	// Check if old model code is in tenant models set
	tenantModelsKey := "aigw:tenant:" + tenant.Code + ":models"
	isMember, err := rdb.SIsMember(ctx, tenantModelsKey, "m-sync-old").Result()
	assert.NoError(err)
	assert.True(isMember)

	// Write mock providers to tenant-model providers whitelist in Redis
	oldProvidersKey := "aigw:tenant:" + tenant.Code + ":model:m-sync-old:providers"
	err = rdb.SAdd(ctx, oldProvidersKey, "openai-official", "anthropic-official").Err()
	assert.NoError(err)

	// 5. Update Model Code from m-sync-old to m-sync-new
	modelFormItem.ModelCode = "m-sync-new"
	e.PUT(baseAPI + "/models/" + model.ID).WithJSON(modelFormItem).
		Expect().Status(http.StatusOK)

	// Verify Redis Sync
	// - aigw:tenant:{tenant}:models should now contain m-sync-new and not m-sync-old
	isMemberNew, err := rdb.SIsMember(ctx, tenantModelsKey, "m-sync-new").Result()
	assert.NoError(err)
	assert.True(isMemberNew)

	isMemberOld, err := rdb.SIsMember(ctx, tenantModelsKey, "m-sync-old").Result()
	assert.NoError(err)
	assert.False(isMemberOld)

	// - oldProvidersKey should be deleted
	existsOld, err := rdb.Exists(ctx, oldProvidersKey).Result()
	assert.NoError(err)
	assert.Equal(int64(0), existsOld)

	// - newProvidersKey should be created with the migrated members
	newProvidersKey := "aigw:tenant:" + tenant.Code + ":model:m-sync-new:providers"
	members, err := rdb.SMembers(ctx, newProvidersKey).Result()
	assert.NoError(err)
	assert.Len(members, 2)
	assert.Contains(members, "openai-official")
	assert.Contains(members, "anthropic-official")

	// 6. Cleanup
	_ = rdb.Del(ctx, newProvidersKey).Err()
	_ = rdb.Del(ctx, tenantModelsKey).Err()
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/tenants/" + tenant.ID).Expect().Status(http.StatusOK)
}

func TestTenantSync_OnCodeChange(t *testing.T) {
	if config.C.Storage.Cache.Type != "redis" {
		t.Skip("skip test because cache type is not redis")
	}

	e := tester(t)
	assert := assert.New(t)
	ctx := context.Background()

	// 1. Create a tenant
	tenantFormItem := rbacSchema.TenantForm{
		Code:        "t-code-old",
		Name:        "Test Code Change Tenant",
		Status:      rbacSchema.TenantStatusActivated,
		Description: "For testing tenant code change sync",
	}
	var tenant rbacSchema.Tenant
	e.POST(baseAPI + "/tenants").WithJSON(tenantFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &tenant})
	assert.NotEmpty(tenant.ID)

	// 2. Create a model
	modelFormItem := modelSchema.ModelForm{
		ModelName:     "m-tenant-sync-name",
		ModelCode:     "m-tenant-sync-code",
		SpaceCode:     "default",
		RequestTypes:  `["chat_completion"]`,
		ContextLength: 128000,
		Enabled:       1,
		Description:   "Model for tenant sync test",
	}
	var model modelSchema.Model
	e.POST(baseAPI + "/models").WithJSON(modelFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &model})
	assert.NotEmpty(model.ID)

	// 3. Bind model to tenant
	bindingFormItem := rbacSchema.TenantModelForm{
		TenantCode: tenant.Code,
		ModelIDs:   []string{model.ID},
	}
	e.POST(baseAPI + "/tenant-models/bindings").WithJSON(bindingFormItem).
		Expect().Status(http.StatusOK)

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.C.Storage.Cache.Redis.Addr,
		DB:       config.C.Storage.Cache.Redis.DB,
		Username: config.C.Storage.Cache.Redis.Username,
		Password: config.C.Storage.Cache.Redis.Password,
	})
	defer rdb.Close()

	// Verify old tenant models key exists
	oldModelsKey := "aigw:tenant:t-code-old:models"
	newModelsKey := "aigw:tenant:t-code-new:models"

	isMemberOld, err := rdb.SIsMember(ctx, oldModelsKey, "m-tenant-sync-code").Result()
	assert.NoError(err)
	assert.True(isMemberOld)

	// Setup mock provider whitelist for old tenant
	oldProvidersKey := "aigw:tenant:t-code-old:model:m-tenant-sync-code:providers"
	newProvidersKey := "aigw:tenant:t-code-new:model:m-tenant-sync-code:providers"
	err = rdb.SAdd(ctx, oldProvidersKey, "mock-provider").Err()
	assert.NoError(err)

	// 4. Update tenant code from t-code-old to t-code-new
	tenantFormItem.APIKey = tenant.APIKey
	tenantFormItem.Code = "t-code-new"
	e.PUT(baseAPI + "/tenants/" + tenant.ID).WithJSON(tenantFormItem).
		Expect().Status(http.StatusOK)

	// 5. Verify Redis Sync after tenant code change
	// - oldModelsKey should be deleted/renamed
	existsOldModels, err := rdb.Exists(ctx, oldModelsKey).Result()
	assert.NoError(err)
	assert.Equal(int64(0), existsOldModels)

	// - newModelsKey should contain the model
	isMemberNew, err := rdb.SIsMember(ctx, newModelsKey, "m-tenant-sync-code").Result()
	assert.NoError(err)
	assert.True(isMemberNew)

	// - oldProvidersKey should be deleted/renamed
	existsOldProviders, err := rdb.Exists(ctx, oldProvidersKey).Result()
	assert.NoError(err)
	assert.Equal(int64(0), existsOldProviders)

	// - newProvidersKey should contain the provider
	isMemberProviderNew, err := rdb.SIsMember(ctx, newProvidersKey, "mock-provider").Result()
	assert.NoError(err)
	assert.True(isMemberProviderNew)

	// - API key Redis sync should use new tenant code
	apiKeyKey := "aigw:apikey:" + tenant.APIKey
	tenantVal, err := rdb.HGet(ctx, apiKeyKey, "tenant").Result()
	assert.NoError(err)
	assert.Equal("t-code-new", tenantVal)

	// 6. Cleanup
	_ = rdb.Del(ctx, newProvidersKey).Err()
	_ = rdb.Del(ctx, newModelsKey).Err()
	_ = rdb.Del(ctx, apiKeyKey).Err()
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/tenants/" + tenant.ID).Expect().Status(http.StatusOK)
}

func TestModelSync_OnDelete(t *testing.T) {
	if config.C.Storage.Cache.Type != "redis" {
		t.Skip("skip test because cache type is not redis")
	}

	e := tester(t)
	assert := assert.New(t)
	ctx := context.Background()

	// 1. Create tenant
	tenantFormItem := rbacSchema.TenantForm{
		Code:   "t-del-test",
		Name:   "Test Model Delete Tenant",
		Status: rbacSchema.TenantStatusActivated,
	}
	var tenant rbacSchema.Tenant
	e.POST(baseAPI + "/tenants").WithJSON(tenantFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &tenant})

	// 2. Create model
	modelFormItem := modelSchema.ModelForm{
		ModelName:     "m-del-test-name",
		ModelCode:     "m-del-test-code",
		SpaceCode:     "default",
		RequestTypes:  `["chat_completion"]`,
		ContextLength: 128000,
		Enabled:       1,
	}
	var model modelSchema.Model
	e.POST(baseAPI + "/models").WithJSON(modelFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &model})

	// 3. Bind model
	bindingFormItem := rbacSchema.TenantModelForm{
		TenantCode: tenant.Code,
		ModelIDs:   []string{model.ID},
	}
	e.POST(baseAPI + "/tenant-models/bindings").WithJSON(bindingFormItem).
		Expect().Status(http.StatusOK)

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.C.Storage.Cache.Redis.Addr,
		DB:       config.C.Storage.Cache.Redis.DB,
		Username: config.C.Storage.Cache.Redis.Username,
		Password: config.C.Storage.Cache.Redis.Password,
	})
	defer rdb.Close()

	tenantModelsKey := "aigw:tenant:" + tenant.Code + ":models"
	isMember, err := rdb.SIsMember(ctx, tenantModelsKey, "m-del-test-code").Result()
	assert.NoError(err)
	assert.True(isMember)

	providersKey := "aigw:tenant:" + tenant.Code + ":model:m-del-test-code:providers"
	err = rdb.SAdd(ctx, providersKey, "mock").Err()
	assert.NoError(err)

	// 4. Delete Model
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)

	// 5. Verify Redis Sync
	// - modelCode should be removed from tenantModelsKey
	isMemberAfter, err := rdb.SIsMember(ctx, tenantModelsKey, "m-del-test-code").Result()
	assert.NoError(err)
	assert.False(isMemberAfter)

	// - providersKey should be deleted
	existsProviders, err := rdb.Exists(ctx, providersKey).Result()
	assert.NoError(err)
	assert.Equal(int64(0), existsProviders)

	// 6. Cleanup
	_ = rdb.Del(ctx, tenantModelsKey)
	_ = rdb.Del(ctx, "aigw:apikey:"+tenant.APIKey)
	e.DELETE(baseAPI + "/tenants/" + tenant.ID).Expect().Status(http.StatusOK)
}

func TestModelSync_OnDisable(t *testing.T) {
	if config.C.Storage.Cache.Type != "redis" {
		t.Skip("skip test because cache type is not redis")
	}

	e := tester(t)
	assert := assert.New(t)
	ctx := context.Background()

	// 1. Create tenant
	tenantFormItem := rbacSchema.TenantForm{
		Code:   "t-dis-test",
		Name:   "Test Model Disable Tenant",
		Status: rbacSchema.TenantStatusActivated,
	}
	var tenant rbacSchema.Tenant
	e.POST(baseAPI + "/tenants").WithJSON(tenantFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &tenant})

	// 2. Create model
	modelFormItem := modelSchema.ModelForm{
		ModelName:     "m-dis-test-name",
		ModelCode:     "m-dis-test-code",
		SpaceCode:     "default",
		RequestTypes:  `["chat_completion"]`,
		ContextLength: 128000,
		Enabled:       1,
	}
	var model modelSchema.Model
	e.POST(baseAPI + "/models").WithJSON(modelFormItem).
		Expect().Status(http.StatusOK).JSON().Decode(&util.ResponseResult{Data: &model})

	// 3. Bind model
	bindingFormItem := rbacSchema.TenantModelForm{
		TenantCode: tenant.Code,
		ModelIDs:   []string{model.ID},
	}
	e.POST(baseAPI + "/tenant-models/bindings").WithJSON(bindingFormItem).
		Expect().Status(http.StatusOK)

	rdb := redis.NewClient(&redis.Options{
		Addr:     config.C.Storage.Cache.Redis.Addr,
		DB:       config.C.Storage.Cache.Redis.DB,
		Username: config.C.Storage.Cache.Redis.Username,
		Password: config.C.Storage.Cache.Redis.Password,
	})
	defer rdb.Close()

	tenantModelsKey := "aigw:tenant:" + tenant.Code + ":models"
	isMember, err := rdb.SIsMember(ctx, tenantModelsKey, "m-dis-test-code").Result()
	assert.NoError(err)
	assert.True(isMember)

	providersKey := "aigw:tenant:" + tenant.Code + ":model:m-dis-test-code:providers"
	err = rdb.SAdd(ctx, providersKey, "mock").Err()
	assert.NoError(err)

	// 4. Disable Model (Enabled = 0)
	modelFormItem.Enabled = 0
	e.PUT(baseAPI + "/models/" + model.ID).WithJSON(modelFormItem).
		Expect().Status(http.StatusOK)

	// Verify Disabled:
	// - modelCode should be removed from tenantModelsKey
	isMemberDis, err := rdb.SIsMember(ctx, tenantModelsKey, "m-dis-test-code").Result()
	assert.NoError(err)
	assert.False(isMemberDis)

	// - providersKey should be deleted
	existsProviders, err := rdb.Exists(ctx, providersKey).Result()
	assert.NoError(err)
	assert.Equal(int64(0), existsProviders)

	// 5. Enable Model back (Enabled = 1)
	modelFormItem.Enabled = 1
	e.PUT(baseAPI + "/models/" + model.ID).WithJSON(modelFormItem).
		Expect().Status(http.StatusOK)

	// Verify Re-enabled:
	// - modelCode should be back in tenantModelsKey
	isMemberEn, err := rdb.SIsMember(ctx, tenantModelsKey, "m-dis-test-code").Result()
	assert.NoError(err)
	assert.True(isMemberEn)

	// 6. Cleanup
	_ = rdb.Del(ctx, tenantModelsKey)
	_ = rdb.Del(ctx, "aigw:apikey:"+tenant.APIKey)
	e.DELETE(baseAPI + "/models/" + model.ID).Expect().Status(http.StatusOK)
	e.DELETE(baseAPI + "/tenants/" + tenant.ID).Expect().Status(http.StatusOK)
}

func TestModelSync_CascadeDelete(t *testing.T) {
	ctx := context.Background()
	assert := assert.New(t)

	// 1. 获取 Injector 和 GORM DB
	injector := GetTestInjector()
	db := injector.DB

	// 2. 准备测试数据
	// A. 创建模型 M1 和 M2
	model1 := &modelSchema.Model{
		ID:           "cascade-model-1",
		ModelName:    "cascade-model-1-name",
		ModelCode:    "cascade-model-1-code",
		SpaceCode:    "default",
		RequestTypes: `["chat_completion"]`,
		Enabled:      1,
		Deleted:      "0",
	}
	model2 := &modelSchema.Model{
		ID:           "cascade-model-2",
		ModelName:    "cascade-model-2-name",
		ModelCode:    "cascade-model-2-code",
		SpaceCode:    "default",
		RequestTypes: `["chat_completion"]`,
		Enabled:      1,
		Deleted:      "0",
	}
	assert.NoError(db.Create(model1).Error)
	assert.NoError(db.Create(model2).Error)

	// B. 为 M1 创建模型别名
	alias1 := &modelSchema.ModelAlias{
		ID:        "cascade-alias-1",
		SpaceCode: "default",
		Alias:     "cascade-alias-1-name",
		ModelId:   model1.ID,
		Deleted:   "0",
	}
	assert.NoError(db.Create(alias1).Error)

	// C. 创建策略：
	slidingWindowsJSON := "[]"
	polExclusive := &policySchema.PolicyLimit{
		ID:             "pol-exclusive-1",
		ModelID:        model1.ID,
		Name:           "cascade-exclusive-limit",
		Type:           "request",
		SlidingWindows: &slidingWindowsJSON,
		Enabled:        1,
		Deleted:        "0",
	}
	polShared := &policySchema.PolicyLimit{
		ID:             "pol-shared-1",
		ModelID:        model2.ID,
		Name:           "cascade-shared-limit",
		Type:           "request",
		SlidingWindows: &slidingWindowsJSON,
		Enabled:        1,
		Deleted:        "0",
	}
	assert.NoError(db.Create(polExclusive).Error)
	assert.NoError(db.Create(polShared).Error)

	// D. 创建租户白名单授权 tenant_model
	tenantModel1 := &rbacSchema.TenantModel{
		ID:         "tm-cascade-1",
		TenantCode: "t-cascade",
		ModelID:    model1.ID,
	}
	assert.NoError(db.Create(tenantModel1).Error)

	// 3. 执行模型 1 的删除操作，由于有别名和策略，应该被拦截
	err := injector.M.Resource.ModelAPI.ModelBIZ.Delete(ctx, model1.ID)
	assert.Error(err)
	assert.Contains(err.Error(), "模型存在关联别名")

	// 4. 手动清理关联的别名和独占策略
	assert.NoError(db.Model(&modelSchema.ModelAlias{}).Where("id = ?", alias1.ID).Update("deleted", "1").Error)
	assert.NoError(db.Model(&policySchema.PolicyLimit{}).Where("id = ?", polExclusive.ID).Update("deleted", "1").Error)

	// 5. 再次执行删除操作，应当成功
	err = injector.M.Resource.ModelAPI.ModelBIZ.Delete(ctx, model1.ID)
	assert.NoError(err)

	// 6. 验证删除状态
	// A. 验证模型 1 被软删除
	var deletedModel modelSchema.Model
	assert.NoError(db.Unscoped().First(&deletedModel, "id = ?", model1.ID).Error)
	assert.NotEqual("0", deletedModel.Deleted)

	// B. 验证共享策略 (关联 model2) 未被删除
	var activePol policySchema.PolicyLimit
	assert.NoError(db.First(&activePol, "id = ?", polShared.ID).Error)
	assert.Equal("0", activePol.Deleted)

	// C. 验证数据权限（tenant_model）被物理删除
	var tmCount int64
	assert.NoError(db.Table(tenantModel1.TableName()).Where("model_id = ?", model1.ID).Count(&tmCount).Error)
	assert.Equal(int64(0), tmCount)

	// 7. 数据清理
	db.Unscoped().Delete(model1)
	db.Unscoped().Delete(model2)
	db.Unscoped().Delete(alias1)
	db.Unscoped().Delete(polExclusive)
	db.Unscoped().Delete(polShared)
	db.Unscoped().Delete(tenantModel1)
}

func TestModelSync_ObsoleteCacheCleanup(t *testing.T) {
	if config.C.Storage.Cache.Type != "redis" {
		t.Skip("skip test because cache type is not redis")
	}

	injector := GetTestInjector()
	ctx := context.Background()

	// 1. 初始化 Redis Client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.C.Storage.Cache.Redis.Addr,
		DB:       config.C.Storage.Cache.Redis.DB,
		Username: config.C.Storage.Cache.Redis.Username,
		Password: config.C.Storage.Cache.Redis.Password,
	})
	defer rdb.Close()

	// 2. 预置一个废弃模型数据到 Redis 缓存中
	obsoleteModelCode := "m-obsolete-test-code"

	// A. 写入 aigw:config:model_versions
	err := rdb.HSet(ctx, "aigw:config:model_versions", obsoleteModelCode, "100").Err()
	assert.NoError(t, err)

	// B. 写入独立端点与费率配置
	err = rdb.Set(ctx, "aigw:config:endpoints:"+obsoleteModelCode, "junk-endpoint-data", 0).Err()
	assert.NoError(t, err)
	err = rdb.Set(ctx, "aigw:policies:model:"+obsoleteModelCode, "junk-policy-data", 0).Err()
	assert.NoError(t, err)

	// C. 写入租户关联缓存
	testTenantCode := "t-obsolete-test-tenant"
	tenantModelsKey := "aigw:tenant:" + testTenantCode + ":models"
	err = rdb.SAdd(ctx, tenantModelsKey, obsoleteModelCode).Err()
	assert.NoError(t, err)
	tenantEndpointsKey := "aigw:tenant:" + testTenantCode + ":model:" + obsoleteModelCode + ":endpoints"
	err = rdb.Set(ctx, tenantEndpointsKey, "junk-endpoints", 0).Err()
	assert.NoError(t, err)

	// D. 写入反向别名缓存
	reverseAliasKey := "aigw:config:model_aliases:" + obsoleteModelCode
	err = rdb.Set(ctx, reverseAliasKey, "junk-alias", 0).Err()
	assert.NoError(t, err)

	// 3. 执行全量覆盖同步 SyncAllToRedis
	syncBiz := injector.M.Resource.ModelAPI.ModelBIZ.ConfigRedisSync
	err = syncBiz.SyncAllToRedis(ctx)
	assert.NoError(t, err)

	// 4. 验证废弃模型的缓存已被完全清除
	// A. model_versions 中不再包含该 field
	existsInVersions, err := rdb.HExists(ctx, "aigw:config:model_versions", obsoleteModelCode).Result()
	assert.NoError(t, err)
	assert.False(t, existsInVersions)

	// B. 独立端点和费率配置已删除
	existsEndpoints, err := rdb.Exists(ctx, "aigw:config:endpoints:"+obsoleteModelCode).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), existsEndpoints)

	existsPolicies, err := rdb.Exists(ctx, "aigw:policies:model:"+obsoleteModelCode).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), existsPolicies)

	// C. 租户内允许模型集合被移除，租户模型端点被删除
	isMember, err := rdb.SIsMember(ctx, tenantModelsKey, obsoleteModelCode).Result()
	assert.NoError(t, err)
	assert.False(t, isMember)

	existsTenantEndpoints, err := rdb.Exists(ctx, tenantEndpointsKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), existsTenantEndpoints)

	// D. 反向别名缓存已删除
	existsReverseAlias, err := rdb.Exists(ctx, reverseAliasKey).Result()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), existsReverseAlias)

	// 5. 清理租户 models key
	_ = rdb.Del(ctx, tenantModelsKey).Err()
}
