# 模型别名（Model Alias）Gateway 生效实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Model Alias 从纯管理端概念变为 Gateway 运行时实际生效的功能 — 客户端通过别名访问与通过原始 model_code 访问效果一致。

**Architecture:** 分两阶段实施。Phase 1（admin 端）：修改 ModelAlias schema、增强 biz 层校验（model_code 冲突 + 数量上限）、在 `ConfigRedisSync` 中新增别名同步方法并扩展 Model 的禁用/启用/code变更/全量同步联动。Phase 2（gateway 端）：新建 `AliasService`（`ExpirableCache` + Redis 查询），在 Engine 的 `HandleRequest` 中插入别名解析。

**Tech Stack:** Go, Gin, GORM, Redis (go-redis/v9), hashicorp/golang-lru/v2/expirable, Google Wire

---

## 文件结构

### Admin 端（`/Users/chenzhiguo/Projects/tokenlive-admin`）

| 操作 | 文件 | 职责 |
|------|------|------|
| Modify | `internal/mods/resource/schema/model_alias.go` | alias 字段长度改为 64，新增全局唯一索引 |
| Modify | `internal/mods/resource/dal/model_alias.dal.go` | 新增 `CountByModelId` 和 `ListByModelId` 方法 |
| Modify | `internal/mods/resource/biz/model_alias.biz.go` | 注入 `ConfigRedisSync`，Create 校验 model_code 冲突 + 数量上限，CRUD 后触发 Redis 同步 |
| Modify | `internal/mods/resource/biz/redis_sync.go` | 新增 `SyncAlias`/`DeleteAlias`/`SyncAliasesByModelId`，扩展 `SyncModelDisable`/`SyncModelEnable`/`SyncModelCodeChange`/`SyncAllToRedis` |
| Modify | `internal/mods/resource/wire.go` | `biz.ModelAlias` 注入 `ConfigRedisSync` |
| Modify | `internal/mods/resource/biz/model.biz.go` | Delete 时同步清理别名 Redis key |

### Gateway 端（`/Users/chenzhiguo/Projects/ai-gateway`）

| 操作 | 文件 | 职责 |
|------|------|------|
| Modify | `pkg/store/keys.go` | 新增 `RedisKeyAlias` 函数 |
| Create | `internal/service/alias.go` | `AliasService`（ExpirableCache + Redis 查询 + Resolve 方法） |
| Modify | `pkg/core/engine.go` | 新增 `aliasService` 字段和 `SetAliasService` setter |
| Modify | `pkg/core/engine.go:HandleRequest` | 在 parseRequest 和 matchPipeline 之间插入别名解析 |
| Modify | `cmd/server/wire/engine.go` | 构造 `AliasService` 并注入 Engine |

---

## Phase 1: Admin 端

### Task 1: 修改 ModelAlias Schema — alias 长度和索引

**Files:**

- Modify: `internal/mods/resource/schema/model_alias.go`

- [ ] **Step 1: 修改 ModelAlias struct 的 Alias 字段**

将 `Alias` 字段的 GORM tag 中 `size:255` 改为 `size:64`，并新增一个独立的全局唯一索引 `uk_alias_deleted`：

```go
// 修改前 (line 16):
Alias       string          `json:"alias" gorm:"size:255;not null;uniqueIndex:uk_model_alias;"`

// 修改后:
Alias       string          `json:"alias" gorm:"size:64;not null;uniqueIndex:uk_model_alias;uniqueIndex:uk_alias_deleted;"`
```

同时修改 `Deleted` 字段，新增对应的唯一索引：

```go
// 修改前 (line 21):
Deleted     string          `json:"-" gorm:"uniqueIndex:uk_model_alias;size:20;default:0;"`

// 修改后:
Deleted     string          `json:"-" gorm:"uniqueIndex:uk_model_alias;uniqueIndex:uk_alias_deleted;size:20;default:0;"`
```

- [ ] **Step 2: 修改 ModelAliasForm 的 Alias 字段校验**

```go
// 修改前 (line 55):
Alias       string  `json:"alias" binding:"required,max=255"`

// 修改后:
Alias       string  `json:"alias" binding:"required,max=64"`
```

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功

---

### Task 2: 扩展 ModelAlias DAL 层 — 新增 CountByModelId 和 ListByModelId

**Files:**

- Modify: `internal/mods/resource/dal/model_alias.dal.go`

- [ ] **Step 1: 在 dal/model_alias.dal.go 末尾新增两个方法**

在文件末尾（`Delete` 方法之后）添加：

```go
// CountByModelId counts active aliases for a given model ID.
func (m *ModelAlias) CountByModelId(ctx context.Context, modelId string) (int64, error) {
 var count int64
 err := GetModelAliasDB(ctx, m.DB).Where("model_id = ?", modelId).Count(&count).Error
 return count, errors.WithStack(err)
}

// ListByModelId returns all active aliases for a given model ID.
func (m *ModelAlias) ListByModelId(ctx context.Context, modelId string) (schema.ModelAliases, error) {
 var list schema.ModelAliases
 err := GetModelAliasDB(ctx, m.DB).Where("model_id = ?", modelId).Find(&list).Error
 return list, errors.WithStack(err)
}

// ExistsByAlias checks if a model alias with the given alias name exists (global scope).
func (m *ModelAlias) ExistsByAlias(ctx context.Context, alias string) (bool, error) {
 ok, err := util.Exists(ctx, GetModelAliasDB(ctx, m.DB).Where("alias = ?", alias))
 return ok, errors.WithStack(err)
}
```

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功

---

### Task 3: 在 ConfigRedisSync 中新增别名同步方法

**Files:**

- Modify: `internal/mods/resource/biz/redis_sync.go`

- [ ] **Step 1: 在 ConfigRedisSync struct 中新增 ModelAliasDAL 字段**

```go
// 修改前 (lines 36-40):
type ConfigRedisSync struct {
 RedisClient *redis.Client
 EndpointDAL *dal.Endpoint
 ModelDAL    *dal.Model
}

// 修改后:
type ConfigRedisSync struct {
 RedisClient   *redis.Client
 EndpointDAL   *dal.Endpoint
 ModelDAL      *dal.Model
 ModelAliasDAL *dal.ModelAlias
}
```

- [ ] **Step 2: 在 redis_sync.go 中添加别名 Redis key 常量和同步方法**

在文件顶部的 `const` 块中（line 16-18）添加别名 key 前缀：

```go
const (
 RedisKeyConfigModelVersions = "aigw:config:model_versions"
 RedisKeyConfigAliasPrefix   = "aigw:config:alias:"
)
```

在 `SyncProviderID` 方法之前（约 line 175）添加以下三个方法：

```go
// SyncAlias synchronizes a single alias mapping to Redis: aigw:config:alias:{alias} → modelCode.
func (s *ConfigRedisSync) SyncAlias(ctx context.Context, alias string, modelCode string) error {
 if s.RedisClient == nil || alias == "" || modelCode == "" {
  return nil
 }
 key := RedisKeyConfigAliasPrefix + alias
 return s.RedisClient.Set(ctx, key, modelCode, 0).Err()
}

// DeleteAlias removes a single alias mapping from Redis.
func (s *ConfigRedisSync) DeleteAlias(ctx context.Context, alias string) error {
 if s.RedisClient == nil || alias == "" {
  return nil
 }
 key := RedisKeyConfigAliasPrefix + alias
 return s.RedisClient.Del(ctx, key).Err()
}

// SyncAliasesByModelId re-syncs all aliases for a given model ID to Redis.
// If the model is disabled (enabled=0), it deletes all alias keys instead.
func (s *ConfigRedisSync) SyncAliasesByModelId(ctx context.Context, modelId string, modelCode string, enabled int) error {
 if s.RedisClient == nil || modelId == "" {
  return nil
 }
 aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelId)
 if err != nil {
  return err
 }
 for _, a := range aliases {
  if enabled == 1 {
   _ = s.SyncAlias(ctx, a.Alias, modelCode)
  } else {
   _ = s.DeleteAlias(ctx, a.Alias)
  }
 }
 return nil
}

// deleteAliasesByModelId deletes all alias Redis keys for a given model ID.
func (s *ConfigRedisSync) deleteAliasesByModelId(ctx context.Context, modelId string) error {
 if s.RedisClient == nil || modelId == "" {
  return nil
 }
 aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelId)
 if err != nil {
  return err
 }
 for _, a := range aliases {
  _ = s.DeleteAlias(ctx, a.Alias)
 }
 return nil
}
```

- [ ] **Step 3: 扩展 SyncModelDisable — 禁用时清理别名 Redis key**

在 `SyncModelDisable` 方法的末尾（`return nil` 之前，约 line 335）添加：

```go
 // 5. 清理该模型所有别名的 Redis key
 _ = s.deleteAliasesByModelId(ctx, modelID)
```

- [ ] **Step 4: 扩展 SyncModelEnable — 启用时恢复别名 Redis key**

在 `SyncModelEnable` 方法的末尾（`return nil` 之前，约 line 408）添加：

```go
 // 5. 重新同步该模型所有别名的 Redis key
 _ = s.SyncAliasesByModelId(ctx, modelID, modelCode, 1)
```

- [ ] **Step 5: 扩展 SyncModelCodeChange — code 变更时更新别名映射**

在 `SyncModelCodeChange` 方法的末尾（`return nil` 之前，约 line 298）添加：

```go
 // 5. 更新该模型所有别名的 Redis value 为新 modelCode
 if s.ModelAliasDAL != nil {
  aliases, err := s.ModelAliasDAL.ListByModelId(ctx, modelID)
  if err == nil {
   for _, a := range aliases {
    _ = s.SyncAlias(ctx, a.Alias, newModelCode)
   }
  }
 }
```

- [ ] **Step 6: 扩展 SyncAllToRedis — 全量同步时纳入别名**

在 `SyncAllToRedis` 方法中，在 `// 6. 遍历模型` 那段循环（约 line 560）的末尾，`return nil` 之前，添加别名全量同步：

```go
 // 7. 全量同步所有别名映射到 Redis
 if s.ModelAliasDAL != nil {
  var allAliases schema.ModelAliases
  aliasTable := config.C.FormatTableName("model_alias")
  err = db.Table(aliasTable).
   Where("deleted = '0'").
   Find(&allAliases).Error
  if err != nil {
   return err
  }
  // 构建 modelID → (modelCode, enabled) 映射
  modelMap := make(map[string]schema.Model)
  for _, m := range models {
   modelMap[m.ID] = m
  }
  for _, a := range allAliases {
   if m, ok := modelMap[a.ModelId]; ok && m.Enabled == 1 {
    _ = s.SyncAlias(ctx, a.Alias, m.ModelCode)
   } else {
    _ = s.DeleteAlias(ctx, a.Alias)
   }
  }
 }
```

- [ ] **Step 7: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功（如果 `schema.Model` 在循环中需要解引用 `*allAliases`，调整为指针即可）

---

### Task 4: 增强 ModelAlias Biz 层 — Redis 同步 + 冲突校验 + 数量上限

**Files:**

- Modify: `internal/mods/resource/biz/model_alias.biz.go`

- [ ] **Step 1: 在 ModelAlias struct 中注入 ConfigRedisSync 和 ModelDAL**

```go
// 修改前 (lines 14-17):
type ModelAlias struct {
 Trans         *util.Trans
 ModelAliasDAL *dal.ModelAlias
}

// 修改后:
type ModelAlias struct {
 Trans           *util.Trans
 ModelAliasDAL   *dal.ModelAlias
 ModelDAL        *dal.Model
 ConfigRedisSync *ConfigRedisSync
}
```

- [ ] **Step 2: 在 Create 方法中增加 model_code 冲突校验、数量上限校验和 Redis 同步**

替换整个 `Create` 方法（lines 48-73）：

```go
// Create a new model alias.
func (m *ModelAlias) Create(ctx context.Context, formItem *schema.ModelAliasForm) (*schema.ModelAlias, error) {
 // 1. 检查 alias 是否与已有 model_code 冲突
 if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.Alias); err != nil {
  return nil, err
 } else if exists {
  return nil, errors.BadRequest("", "Alias conflicts with an existing model code")
 }

 // 2. 检查全局是否已有同名别名
 if exists, err := m.ModelAliasDAL.ExistsByAlias(ctx, formItem.Alias); err != nil {
  return nil, err
 } else if exists {
  return nil, errors.BadRequest("", "Alias already exists")
 }

 // 3. 检查该 model 的别名数量是否已达上限（10）
 const maxAliasesPerModel = 10
 count, err := m.ModelAliasDAL.CountByModelId(ctx, formItem.ModelId)
 if err != nil {
  return nil, err
 }
 if count >= maxAliasesPerModel {
  return nil, errors.BadRequest("", "Maximum number of aliases (10) per model reached")
 }

 alias := &schema.ModelAlias{
  ID:        util.NewXID(),
  Deleted:   "0",
  CreatedAt: time.Now(),
 }

 if err := formItem.FillTo(alias); err != nil {
  return nil, err
 }

 err = m.Trans.Exec(ctx, func(ctx context.Context) error {
  return m.ModelAliasDAL.Create(ctx, alias)
 })
 if err != nil {
  return nil, err
 }

 // 4. 同步别名到 Redis（fire-and-forget）
 if m.ConfigRedisSync != nil {
  // 查询目标 model 的 model_code
  if model, err := m.ModelDAL.Get(ctx, formItem.ModelId); err == nil && model != nil && model.Enabled == 1 {
   _ = m.ConfigRedisSync.SyncAlias(ctx, formItem.Alias, model.ModelCode)
  }
 }

 return alias, nil
}
```

- [ ] **Step 3: 在 Update 方法中增加冲突校验和 Redis 同步**

替换整个 `Update` 方法（lines 76-101）：

```go
// Update the specified model alias.
func (m *ModelAlias) Update(ctx context.Context, id string, formItem *schema.ModelAliasForm) error {
 alias, err := m.ModelAliasDAL.Get(ctx, id)
 if err != nil {
  return err
 } else if alias == nil {
  return errors.NotFound("", "Model alias not found")
 }

 // 如果 alias 名称变更，需要检查冲突
 if alias.Alias != formItem.Alias {
  // 检查是否与已有 model_code 冲突
  if exists, err := m.ModelDAL.ExistsByModelCode(ctx, formItem.Alias); err != nil {
   return err
  } else if exists {
   return errors.BadRequest("", "Alias conflicts with an existing model code")
  }
  // 检查全局是否已有同名别名
  if exists, err := m.ModelAliasDAL.ExistsByAlias(ctx, formItem.Alias); err != nil {
   return err
  } else if exists {
   return errors.BadRequest("", "Alias already exists")
  }
 }

 oldAlias := alias.Alias

 if err := formItem.FillTo(alias); err != nil {
  return err
 }
 alias.UpdatedAt = time.Now()

 err = m.Trans.Exec(ctx, func(ctx context.Context) error {
  return m.ModelAliasDAL.Update(ctx, alias)
 })
 if err != nil {
  return err
 }

 // 同步别名到 Redis（fire-and-forget）
 if m.ConfigRedisSync != nil {
  // 如果 alias 名称变更，先删旧的
  if oldAlias != formItem.Alias {
   _ = m.ConfigRedisSync.DeleteAlias(ctx, oldAlias)
  }
  // 查询目标 model 的 model_code
  if model, err := m.ModelDAL.Get(ctx, alias.ModelId); err == nil && model != nil && model.Enabled == 1 {
   _ = m.ConfigRedisSync.SyncAlias(ctx, formItem.Alias, model.ModelCode)
  }
 }

 return nil
}
```

- [ ] **Step 4: 在 Delete 方法中增加 Redis 同步**

替换整个 `Delete` 方法（lines 104-115）：

```go
// Delete the specified model alias.
func (m *ModelAlias) Delete(ctx context.Context, id string) error {
 alias, err := m.ModelAliasDAL.Get(ctx, id)
 if err != nil {
  return err
 } else if alias == nil {
  return errors.NotFound("", "Model alias not found")
 }

 err = m.Trans.Exec(ctx, func(ctx context.Context) error {
  return m.ModelAliasDAL.Delete(ctx, id)
 })
 if err != nil {
  return err
 }

 // 同步删除 Redis 中的别名映射（fire-and-forget）
 if m.ConfigRedisSync != nil {
  _ = m.ConfigRedisSync.DeleteAlias(ctx, alias.Alias)
 }

 return nil
}
```

- [ ] **Step 5: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功。需要确认 `dal.Model` 有 `ExistsByModelCode` 方法，如果没有则需要在 Task 2 中补充。

---

### Task 5: DAL 层补充 Model 的 ExistsByModelCode 方法

**Files:**

- Modify: `internal/mods/resource/dal/model.dal.go`

- [ ] **Step 1: 在 model.dal.go 中确认或新增 ExistsByModelCode 方法**

先检查 `dal/model.dal.go` 是否已有 `ExistsByModelCode`。如果没有，在文件中添加：

```go
// ExistsByModelCode checks if a model with the given model_code exists (not deleted).
func (m *Model) ExistsByModelCode(ctx context.Context, modelCode string) (bool, error) {
 ok, err := util.Exists(ctx, GetModelDB(ctx, m.DB).Where("model_code = ?", modelCode))
 return ok, errors.WithStack(err)
}
```

注意：如果 `GetModelDB` 不存在，需要先定义（和 `GetModelAliasDB` 模式一致）。

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功

---

### Task 6: 更新 Wire 依赖注入

**Files:**

- Modify: `internal/mods/resource/wire.go`

- [ ] **Step 1: 确保 ConfigRedisSync 包含 ModelAliasDAL**

Wire 使用 `wire.Struct(new(biz.ConfigRedisSync), "*")` 自动注入所有字段。由于 Task 3 中我们给 `ConfigRedisSync` 新增了 `ModelAliasDAL *dal.ModelAlias` 字段，而 `dal.ModelAlias` 已经在 Wire Set 中注册，Wire 会自动注入。

无需手动修改 wire.go 中 ConfigRedisSync 的定义。

- [ ] **Step 2: 验证 Wire 代码生成**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && make wire`
Expected: `wire_gen.go` 重新生成，编译通过

- [ ] **Step 3: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功

---

### Task 7: Model Delete 时同步清理别名 Redis key

**Files:**

- Modify: `internal/mods/resource/biz/model.biz.go`

- [ ] **Step 1: 在 Model.Delete 方法中，级联软删别名之后，添加 Redis 清理**

在 `model.biz.go` 的 Delete 方法中，找到级联软删别名的代码（约 line 214）之后，添加 Redis 别名清理：

```go
 // 在 line 214 级联软删别名之后添加:
 // 2b. 清理该模型所有别名的 Redis key
 if m.ConfigRedisSync != nil {
  _ = m.ConfigRedisSync.DeleteAliasesByModelId(ctx, id)
 }
```

注意：需要确认 `ConfigRedisSync` 的方法名。如果 Task 3 中我们定义的是 `deleteAliasesByModelId`（小写开头，私有方法），则需要改为公开方法 `DeleteAliasesByModelId`，或者在 `SyncModelDisable` 的调用链中已经覆盖（因为 Delete 后面调用了 `SyncModelDisable`）。

实际上，查看 `model.biz.go` 的 Delete 方法（line 265-267），在事务成功后已经调用了 `SyncModelByCode` 和 `SyncModelDisable`。而 `SyncModelDisable` 在 Task 3 Step 3 中已经添加了 `deleteAliasesByModelId`。**因此这个 Task 可能不需要额外改动，只需验证 `SyncModelDisable` 能正确清理别名。**

- [ ] **Step 2: 验证确认**

检查 `model.biz.go` Delete 方法中 `SyncModelDisable` 的调用是否在级联软删别名之后执行。如果是，则别名 Redis 清理已经由 `SyncModelDisable` 覆盖。

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go build ./...`
Expected: 编译成功

---

### Task 8: Admin 端集成测试

**Files:**

- Test: `test/model_alias_test.go`（新建或修改现有测试）

- [ ] **Step 1: 编写别名 CRUD + Redis 同步的集成测试**

创建或修改测试文件，验证以下场景：

```go
func TestModelAlias_Create_WithRedisSync(t *testing.T) {
 // 1. 创建一个 enabled model
 // 2. 为该 model 创建别名
 // 3. 验证 Redis 中 aigw:config:alias:{alias} 存在且值为 model_code
}

func TestModelAlias_Create_ConflictWithModelCode(t *testing.T) {
 // 1. 创建一个 model_code = "gpt-4o" 的 model
 // 2. 尝试创建 alias = "gpt-4o" 的别名
 // 3. 验证返回冲突错误
}

func TestModelAlias_Create_MaxLimit(t *testing.T) {
 // 1. 为同一个 model 创建 10 个别名
 // 2. 尝试创建第 11 个
 // 3. 验证返回上限错误
}

func TestModelAlias_Delete_ClearsRedis(t *testing.T) {
 // 1. 创建别名并验证 Redis 存在
 // 2. 删除别名
 // 3. 验证 Redis key 被清除
}

func TestModel_Disable_ClearsAliasRedis(t *testing.T) {
 // 1. 创建 model + 别名，验证 Redis 存在
 // 2. 禁用 model
 // 3. 验证别名 Redis key 被清除
}
```

- [ ] **Step 2: 运行测试**

Run: `cd /Users/chenzhiguo/Projects/tokenlive-admin && go test ./test/ -run TestModelAlias -v`
Expected: 所有测试通过

---

## Phase 2: Gateway 端

### Task 9: 新增 Redis Key 函数

**Files:**

- Modify: `pkg/store/keys.go`

- [ ] **Step 1: 在 keys.go 末尾新增 RedisKeyAlias 函数**

```go
// RedisKeyAlias 返回别名映射的 Redis key
func RedisKeyAlias(alias string) string {
 return "aigw:config:alias:" + alias
}
```

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/ai-gateway && go build ./...`
Expected: 编译成功

---

### Task 10: 新建 AliasService

**Files:**

- Create: `internal/service/alias.go`

- [ ] **Step 1: 创建 alias.go**

```go
package service

import (
 "context"
 "errors"
 "time"

 "ai-gateway/pkg/log"
 "ai-gateway/pkg/store"

 "github.com/redis/go-redis/v9"
 "go.uber.org/zap"
)

// AliasService 负责将客户端请求中的 model alias 解析为真实的 model_code
type AliasService struct {
 rdb    *redis.Client
 logger *log.Logger
 cache  *store.ExpirableCache[string, string] // alias → model_code
}

// NewAliasService 创建 AliasService 实例
func NewAliasService(rdb *redis.Client, logger *log.Logger) *AliasService {
 cache := store.NewExpirableCache[string, string](
  5000, 30*time.Second,  // valid cache: 5k 条, 30s TTL
  2000, 10*time.Second,  // invalid cache: 2k 条, 10s TTL
 )
 return &AliasService{
  rdb:    rdb,
  logger: logger,
  cache:  cache,
 }
}

// Resolve 尝试将 model 解析为真实的 model_code。
// 如果 model 不是别名，返回原始 model（静默降级）。
// 如果 Redis 不可用，返回错误（fail-close）。
func (s *AliasService) Resolve(ctx context.Context, model string) (string, error) {
 if model == "" {
  return model, nil
 }

 // 1. 查本地缓存（正向 + 负向）
 if modelCode, errMsg, ok := s.cache.Get(model); ok {
  if errMsg != "" {
   // 负向缓存命中：这个 model 不是已知别名，原样返回
   return model, nil
  }
  return modelCode, nil
 }

 // 2. 查 Redis
 if s.rdb == nil {
  return model, nil
 }

 key := store.RedisKeyAlias(model)
 modelCode, err := s.rdb.Get(ctx, key).Result()
 if err != nil {
  if errors.Is(err, redis.Nil) {
   // Redis 中不存在此别名，写入负向缓存
   s.cache.AddInvalid(model, "not an alias")
   return model, nil
  }
  // Redis 不可用，fail-close
  s.logger.Logger.Error("failed to query alias from redis", zap.Error(err), zap.String("key", key))
  return "", err
 }

 // 3. 命中，写入正向缓存
 s.cache.AddValid(model, modelCode)
 return modelCode, nil
}
```

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/ai-gateway && go build ./...`
Expected: 编译成功

---

### Task 11: 在 Engine 中注入 AliasService 并插入别名解析

**Files:**

- Modify: `pkg/core/engine.go`

- [ ] **Step 1: 在 Engine struct 中新增 aliasService 字段**

在 Engine struct 的 optional components 注释下方（约 line 53 附近）添加：

```go
 aliasService AliasResolver
```

- [ ] **Step 2: 定义 AliasResolver 接口**

在 engine.go 文件顶部（import 之后）或一个新的 interface 文件中定义：

```go
// AliasResolver 将 model alias 解析为真实 model_code
type AliasResolver interface {
 Resolve(ctx context.Context, model string) (string, error)
}
```

注意：为了避免在 `pkg/core` 中 import `internal/service`，使用接口解耦。`service.AliasService` 会自动实现此接口（因为它有匹配的 `Resolve` 方法签名）。

- [ ] **Step 3: 新增 SetAliasService setter**

在其他 setter 方法附近（约 line 116 之后）添加：

```go
// SetAliasService 注入别名解析服务（可选，Init 之前调用）
func (e *Engine) SetAliasService(as AliasResolver) {
 e.aliasService = as
}
```

- [ ] **Step 4: 在 HandleRequest 中插入别名解析**

在 `HandleRequest` 方法中，`parseRequest` 之后、`matchPipeline` 之前（约 lines 229-231 之间）插入：

```go
 // 1.5 别名解析：将 model alias 解析为真实 model_code
 if e.aliasService != nil && gctx.Model != "" {
  resolved, err := e.aliasService.Resolve(gctx.Ctx, gctx.Model)
  if err != nil {
   e.writeError(w, http.StatusBadGateway, fmt.Errorf("alias resolution error: %w", err), gctx)
   return
  }
  gctx.Model = resolved
 }
```

注意：`gctx.OriginalModel` 已经在 `parseRequest` 中被设置为原始值，所以这里只替换 `gctx.Model`，`gctx.OriginalModel` 保持不变用于日志和 metrics。

- [ ] **Step 5: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/ai-gateway && go build ./...`
Expected: 编译成功

---

### Task 12: 在 Wire 中装配 AliasService

**Files:**

- Modify: `cmd/server/wire/engine.go`

- [ ] **Step 1: 在 NewGatewayEngine 中构造 AliasService 并注入 Engine**

在 `NewGatewayEngine` 函数中，找到创建 `modelService` 的位置附近，添加 `aliasService` 的构造：

```go
 // 在 modelService 创建之后添加:
 aliasService := service.NewAliasService(rdb, logger)
```

在 `engine.SetInvokerBuilder` 之后（约 line 165）添加注入：

```go
 engine.SetAliasService(aliasService)
```

- [ ] **Step 2: 验证编译通过**

Run: `cd /Users/chenzhiguo/Projects/ai-gateway && go build ./...`
Expected: 编译成功

---

### Task 13: Gateway 端单元测试

**Files:**

- Create: `internal/service/alias_test.go`

- [ ] **Step 1: 编写 AliasService 单元测试**

```go
package service

import (
 "context"
 "testing"
 "time"

 "ai-gateway/pkg/log"
 "ai-gateway/pkg/store"

 "github.com/alicebob/miniredis/v2"
 "github.com/redis/go-redis/v9"
)

func TestAliasService_Resolve_AliasExists(t *testing.T) {
 s := miniredis.RunT(t)
 rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
 defer rdb.Close()

 // 预置别名
 rdb.Set(context.Background(), "aigw:config:alias:fast-gpt", "gpt-4o", 0)

 logger := log.NewLogger()
 svc := NewAliasService(rdb, logger)

 modelCode, err := svc.Resolve(context.Background(), "fast-gpt")
 if err != nil {
  t.Fatalf("unexpected error: %v", err)
 }
 if modelCode != "gpt-4o" {
  t.Fatalf("expected gpt-4o, got %s", modelCode)
 }
}

func TestAliasService_Resolve_NotAnAlias(t *testing.T) {
 s := miniredis.RunT(t)
 rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
 defer rdb.Close()

 logger := log.NewLogger()
 svc := NewAliasService(rdb, logger)

 modelCode, err := svc.Resolve(context.Background(), "gpt-4o")
 if err != nil {
  t.Fatalf("unexpected error: %v", err)
 }
 if modelCode != "gpt-4o" {
  t.Fatalf("expected gpt-4o (passthrough), got %s", modelCode)
 }
}

func TestAliasService_Resolve_CacheHit(t *testing.T) {
 s := miniredis.RunT(t)
 rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
 defer rdb.Close()

 rdb.Set(context.Background(), "aigw:config:alias:fast-gpt", "gpt-4o", 0)

 logger := log.NewLogger()
 svc := NewAliasService(rdb, logger)

 // 第一次查询，填充缓存
 _, _ = svc.Resolve(context.Background(), "fast-gpt")

 // 删除 Redis key，验证缓存命中
 s.Del("aigw:config:alias:fast-gpt")

 modelCode, err := svc.Resolve(context.Background(), "fast-gpt")
 if err != nil {
  t.Fatalf("unexpected error: %v", err)
 }
 if modelCode != "gpt-4o" {
  t.Fatalf("expected gpt-4o from cache, got %s", modelCode)
 }
}

func TestAliasService_Resolve_RedisDown_FailClose(t *testing.T) {
 // 使用一个不存在的 Redis 地址
 rdb := redis.NewClient(&redis.Options{Addr: "localhost:19999"})
 defer rdb.Close()

 logger := log.NewLogger()
 svc := NewAliasService(rdb, logger)

 _, err := svc.Resolve(context.Background(), "fast-gpt")
 if err == nil {
  t.Fatal("expected error when redis is down, got nil")
 }
}
```

- [ ] **Step 2: 运行测试**

Run: `cd /Users/chenzhiguo/Projects/ai-gateway && go test ./internal/service/ -run TestAliasService -v`
Expected: 所有测试通过

---

## 验证清单

完成所有 Task 后，进行端到端验证：

- [ ] Admin 端：创建 Model（enabled=1）→ 创建 Alias → 验证 Redis 中 `aigw:config:alias:{alias}` 存在
- [ ] Admin 端：更新 Alias 名称 → 验证 Redis 中旧 key 消失、新 key 存在
- [ ] Admin 端：删除 Alias → 验证 Redis key 消失
- [ ] Admin 端：禁用 Model → 验证关联 Alias 的 Redis key 被清理
- [ ] Admin 端：启用 Model → 验证关联 Alias 的 Redis key 恢复
- [ ] Admin 端：修改 Model Code → 验证 Alias Redis value 更新为新 code
- [ ] Admin 端：手动触发全量同步 → 验证所有 Alias 写入 Redis
- [ ] Gateway 端：用别名发送请求 → 验证 `gctx.Model` 被替换为真实 code，请求正常路由
- [ ] Gateway 端：用不存在的别名发送请求 → 验证当作普通 model_code 处理（静默降级）
- [ ] Gateway 端：用真实 model_code 发送请求 → 验证行为不变（零影响）
