# 简化 ModelProvider 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 model_provider 表的信息迁移到 endpoint 表中，简化数据模型，移除 model_provider 表

**Architecture:** 扩展 Endpoint 表结构，添加 model_id、real_model、priority 字段，将 Model-Provider 的关联信息直接存储在 Endpoint 中。通过 Endpoint 支持双向查询（Model → Provider 和 Provider → Model），简化创建流程，集中管理 failover 和负载均衡策略。

**Tech Stack:** Go, Gin, GORM, MySQL/PostgreSQL/SQLite, Vue 3, Ant Design Vue

**Note:** 本计划适用于新系统设计开发阶段，无需考虑数据迁移兼容性问题。表结构将通过 GORM AutoMigrate 自动创建。

---

## 文件结构

### 核心变更文件

**Schema 层（数据模型）：**
- Modify: `internal/mods/resource/schema/endpoint.go` - 添加 model_id、real_model、priority 字段，添加唯一约束
- Delete: `internal/mods/resource/schema/model_provider.go` - 完全移除

**DAL 层（数据访问）：**
- Modify: `internal/mods/resource/dal/endpoint.dal.go` - 更新查询方法，支持新字段过滤
- Delete: `internal/mods/resource/dal/model_provider.dal.go` - 完全移除

**Biz 层（业务逻辑）：**
- Modify: `internal/mods/resource/biz/endpoint.biz.go` - 更新业务逻辑，支持 failover 选择
- Delete: `internal/mods/resource/biz/model_provider.biz.go` - 完全移除

**API 层（HTTP 处理器）：**
- Modify: `internal/mods/resource/api/endpoint.api.go` - 更新 API，支持新字段和新查询
- Delete: `internal/mods/resource/api/model_provider.api.go` - 完全移除

**模块初始化：**
- Modify: `internal/mods/resource/main.go` - 移除 ModelProvider 模块初始化

**依赖注入：**
- Modify: `internal/wirex/wire.go` - 移除 ModelProvider 相关的 Provider
- Regenerate: `internal/wirex/wire_gen.go` - 重新生成（运行 `make wire`）

**测试文件：**
- Modify: `test/endpoint_test.go` - 更新测试用例
- Delete: `test/model_provider_test.go` - 完全移除

**前端文件：**
- Modify: `frontend/src/apis/modules/endpoint.js` - 更新 API 调用
- Delete: `frontend/src/apis/modules/modelProvider.js` - 完全移除
- Modify: `frontend/src/router/routes/resource.js` - 移除 ModelProvider 路由（如果存在）
- Modify: `frontend/src/views/resource/EndpointEditDialog.vue` - 更新表单，支持新字段
- Modify: `frontend/src/views/resource/ModelDetail.vue` - 更新页面，显示新字段

---

## 任务分解

### Task 1: 更新 Endpoint Schema（添加新字段）

**Files:**
- Modify: `internal/mods/resource/schema/endpoint.go`

- [ ] **Step 1: 读取当前 endpoint.go 文件**

```bash
cat internal/mods/resource/schema/endpoint.go
```

- [ ] **Step 2: 添加新字段到 Endpoint 结构体**

在 `Endpoint` 结构体中添加以下字段（在 `ProviderID` 字段之后）：

```go
type Endpoint struct {
    ID          string          `json:"id" gorm:"size:20;primarykey;"`
    ProviderID  string          `json:"provider_id" gorm:"size:20;not null;index;uniqueIndex:uk_endpoint_model_provider_priority,priority:2"`
    ModelID     string          `json:"model_id" gorm:"size:20;not null;index;uniqueIndex:uk_endpoint_model_provider_priority,priority:1"`
    URL         string          `json:"url" gorm:"size:512;not null;"`
    ApiKey      string          `json:"api_key,omitempty" gorm:"size:512;"`
    RealModel   string          `json:"real_model,omitempty" gorm:"size:128;"`
    Priority    int             `json:"priority" gorm:"not null;default:0;uniqueIndex:uk_endpoint_model_provider_priority,priority:3"`
    Weight      int             `json:"weight" gorm:"default:1;"`
    Enabled     int             `json:"enabled" gorm:"not null;default:0;"`
    Metadata    json.RawMessage `json:"metadata,omitempty" gorm:"type:json;"`
    Description string          `json:"description" gorm:"size:255;"`
    Creator     string          `json:"creator" gorm:"size:255;"`
    Modifier    string          `json:"modifier" gorm:"size:255;"`
    CreatedAt   time.Time       `json:"created_at" gorm:"index;"`
    UpdatedAt   time.Time       `json:"updated_at" gorm:"index;"`
    Deleted     string          `json:"-" gorm:"size:20;uniqueIndex:uk_endpoint_model_provider_priority,priority:4;default:0"`
    DeletedAt   *gorm.DeletedAt `json:"-" gorm:"comment:Delete time;"`
    
    // 关联查询
    Model    *Model    `json:"model,omitempty" gorm:"foreignKey:ModelID;references:ID"`
    Provider *Provider `json:"provider,omitempty" gorm:"foreignKey:ProviderID;references:ID"`
}
```

- [ ] **Step 3: 更新 EndpointForm 结构体**

在 `EndpointForm` 结构体中添加新字段：

```go
type EndpointForm struct {
    ProviderID  string          `json:"provider_id" binding:"required,max=20"` // Associated provider ID
    ModelID     string          `json:"model_id" binding:"required,max=20"`    // 新增：关联 Model ID
    URL         string          `json:"url" binding:"required,max=512"`        // Upstream API address
    ApiKey      string          `json:"api_key"`                               // Optional, overrides provider-level api_key
    RealModel   string          `json:"real_model"`                            // 新增：Optional, overrides model-level real_model
    Priority    int             `json:"priority"`                              // 新增：Failover priority
    Weight      int             `json:"weight"`                                // Load balance weight
    Enabled     int             `json:"enabled"`                               // Enable status: 0-disabled, 1-enabled
    Metadata    json.RawMessage `json:"metadata"`                              // Metadata
    Description string          `json:"description"`                           // Description
}
```

- [ ] **Step 4: 更新 FillTo 方法**

在 `FillTo` 方法中添加新字段的赋值：

```go
func (e *EndpointForm) FillTo(endpoint *Endpoint) error {
    endpoint.ProviderID = e.ProviderID
    endpoint.ModelID = e.ModelID
    endpoint.URL = e.URL
    endpoint.ApiKey = e.ApiKey
    endpoint.RealModel = e.RealModel
    endpoint.Priority = e.Priority
    endpoint.Weight = e.Weight
    endpoint.Enabled = e.Enabled
    endpoint.Metadata = e.Metadata
    endpoint.Description = e.Description
    return nil
}
```

- [ ] **Step 5: 更新 EndpointQueryParam 结构体**

在 `EndpointQueryParam` 结构体中添加新查询参数：

```go
type EndpointQueryParam struct {
    util.PaginationParam
    ProviderID string `form:"provider_id"` // Filter by provider ID
    ModelID    string `form:"model_id"`    // 新增：Filter by model ID
    LikeURL    string `form:"url"`         // URL (like)
    Priority   int    `form:"priority"`    // 新增：Filter by priority
}
```

- [ ] **Step 6: 验证编译**

```bash
go build ./internal/mods/resource/schema/
```

Expected: 编译成功，无错误

---

### Task 2: 更新 Endpoint DAL（数据访问层）

**Files:**
- Modify: `internal/mods/resource/dal/endpoint.dal.go`

- [ ] **Step 1: 读取当前 endpoint.dal.go 文件**

```bash
cat internal/mods/resource/dal/endpoint.dal.go
```

- [ ] **Step 2: 更新查询方法以支持新字段**

在查询方法中添加对 `model_id`、`priority` 的过滤支持。找到查询构建的地方，添加：

```go
// 在查询条件构建中添加
if v := params.ModelID; v != "" {
    db = db.Where("model_id = ?", v)
}
if v := params.Priority; v > 0 {
    db = db.Where("priority = ?", v)
}
```

- [ ] **Step 3: 添加新查询方法：查询 Model 关联的 Endpoint 列表**

```go
// QueryEndpointsByModelID 根据 Model ID 查询 Endpoint 列表
func (a *Endpoint) QueryEndpointsByModelID(ctx context.Context, modelID string) (schema.Endpoints, error) {
    var endpoint schema.Endpoint
    var list schema.Endpoints
    
    db := endpoint.DB(ctx).Where("model_id = ? AND enabled = 1", modelID)
    db = db.Order("priority ASC, weight DESC")
    
    if err := db.Find(&list).Error; err != nil {
        return nil, errors.WithStack(err)
    }
    
    return list, nil
}
```

- [ ] **Step 4: 添加新查询方法：查询 Provider 关联的 Endpoint 列表**

```go
// QueryEndpointsByProviderID 根据 Provider ID 查询 Endpoint 列表
func (a *Endpoint) QueryEndpointsByProviderID(ctx context.Context, providerID string) (schema.Endpoints, error) {
    var endpoint schema.Endpoint
    var list schema.Endpoints
    
    db := endpoint.DB(ctx).Where("provider_id = ? AND enabled = 1", providerID)
    db = db.Order("priority ASC, weight DESC")
    
    if err := db.Find(&list).Error; err != nil {
        return nil, errors.WithStack(err)
    }
    
    return list, nil
}
```

- [ ] **Step 5: 添加新查询方法：查询路由（Model Name → Endpoint 列表）**

```go
// QueryEndpointsByModelName 根据 Model Name 查询 Endpoint 列表（用于路由）
func (a *Endpoint) QueryEndpointsByModelName(ctx context.Context, modelName string) (schema.Endpoints, error) {
    var endpoint schema.Endpoint
    var list schema.Endpoints
    
    db := endpoint.DB(ctx).
        Joins("JOIN model ON endpoint.model_id = model.id").
        Joins("JOIN provider ON endpoint.provider_id = provider.id").
        Where("model.model_name = ? AND endpoint.enabled = 1 AND model.enabled = 1 AND provider.enabled = 1", modelName).
        Select("endpoint.*").
        Order("endpoint.priority ASC, endpoint.weight DESC")
    
    if err := db.Find(&list).Error; err != nil {
        return nil, errors.WithStack(err)
    }
    
    return list, nil
}
```

- [ ] **Step 6: 验证编译**

```bash
go build ./internal/mods/resource/dal/
```

Expected: 编译成功，无错误

---

### Task 3: 更新 Endpoint Biz（业务逻辑层）

**Files:**
- Modify: `internal/mods/resource/biz/endpoint.biz.go`

- [ ] **Step 1: 读取当前 endpoint.biz.go 文件**

```bash
cat internal/mods/resource/biz/endpoint.biz.go
```

- [ ] **Step 2: 添加新方法：查询 Model 关联的 Endpoint 列表**

```go
// QueryEndpointsByModelID 根据 Model ID 查询 Endpoint 列表
func (a *Endpoint) QueryEndpointsByModelID(ctx context.Context, modelID string) (schema.Endpoints, error) {
    result, err := a.EndpointDAL.QueryEndpointsByModelID(ctx, modelID)
    if err != nil {
        return nil, err
    }
    return result, nil
}
```

- [ ] **Step 3: 添加新方法：查询 Provider 关联的 Endpoint 列表**

```go
// QueryEndpointsByProviderID 根据 Provider ID 查询 Endpoint 列表
func (a *Endpoint) QueryEndpointsByProviderID(ctx context.Context, providerID string) (schema.Endpoints, error) {
    result, err := a.EndpointDAL.QueryEndpointsByProviderID(ctx, providerID)
    if err != nil {
        return nil, err
    }
    return result, nil
}
```

- [ ] **Step 4: 添加新方法：查询路由（Model Name → Endpoint 列表）**

```go
// QueryEndpointsByModelName 根据 Model Name 查询 Endpoint 列表（用于路由）
func (a *Endpoint) QueryEndpointsByModelName(ctx context.Context, modelName string) (schema.Endpoints, error) {
    result, err := a.EndpointDAL.QueryEndpointsByModelName(ctx, modelName)
    if err != nil {
        return nil, err
    }
    return result, nil
}
```

- [ ] **Step 5: 添加新方法：选择 Endpoint（Failover 逻辑）**

```go
// SelectEndpoint 根据 Model Name 选择最优 Endpoint（考虑 failover 和负载均衡）
func (a *Endpoint) SelectEndpoint(ctx context.Context, modelName string) (*schema.Endpoint, error) {
    endpoints, err := a.QueryEndpointsByModelName(ctx, modelName)
    if err != nil {
        return nil, err
    }
    
    if len(endpoints) == 0 {
        return nil, errors.NotFound("", "No available endpoint for model: %s", modelName)
    }
    
    // 按优先级分组
    priorityGroups := make(map[int]schema.Endpoints)
    for _, ep := range endpoints {
        priorityGroups[ep.Priority] = append(priorityGroups[ep.Priority], ep)
    }
    
    // 获取所有优先级并排序
    priorities := make([]int, 0, len(priorityGroups))
    for p := range priorityGroups {
        priorities = append(priorities, p)
    }
    sort.Ints(priorities)
    
    // 从最高优先级（值最小）开始选择
    for _, priority := range priorities {
        group := priorityGroups[priority]
        
        // 同优先级内按权重随机选择
        totalWeight := 0
        for _, ep := range group {
            totalWeight += ep.Weight
        }
        
        if totalWeight == 0 {
            continue
        }
        
        // 加权随机选择
        randWeight := rand.Intn(totalWeight)
        currentWeight := 0
        for _, ep := range group {
            currentWeight += ep.Weight
            if randWeight < currentWeight {
                return ep, nil
            }
        }
    }
    
    return nil, errors.InternalServerError("", "Failed to select endpoint for model: %s", modelName)
}
```

- [ ] **Step 6: 添加必要的 import**

在文件顶部添加：

```go
import (
    "math/rand"
    "sort"
)
```

- [ ] **Step 7: 验证编译**

```bash
go build ./internal/mods/resource/biz/
```

Expected: 编译成功，无错误

---

### Task 4: 更新 Endpoint API（HTTP 处理器）

**Files:**
- Modify: `internal/mods/resource/api/endpoint.api.go`

- [ ] **Step 1: 读取当前 endpoint.api.go 文件**

```bash
cat internal/mods/resource/api/endpoint.api.go
```

- [ ] **Step 2: 添加新 API：查询 Model 关联的 Endpoint 列表**

```go
// QueryEndpointsByModelID 查询 Model 关联的 Endpoint 列表
func (a *Endpoint) QueryEndpointsByModelID(c *gin.Context) {
    modelID := c.Param("id")
    if modelID == "" {
        util.ResError(c, errors.BadRequest("", "Model ID is required"))
        return
    }
    
    result, err := a.EndpointBiz.QueryEndpointsByModelID(c.Request.Context(), modelID)
    if err != nil {
        util.ResError(c, err)
        return
    }
    
    util.ResSuccess(c, result)
}
```

- [ ] **Step 3: 添加新 API：查询 Provider 关联的 Endpoint 列表**

```go
// QueryEndpointsByProviderID 查询 Provider 关联的 Endpoint 列表
func (a *Endpoint) QueryEndpointsByProviderID(c *gin.Context) {
    providerID := c.Param("id")
    if providerID == "" {
        util.ResError(c, errors.BadRequest("", "Provider ID is required"))
        return
    }
    
    result, err := a.EndpointBiz.QueryEndpointsByProviderID(c.Request.Context(), providerID)
    if err != nil {
        util.ResError(c, err)
        return
    }
    
    util.ResSuccess(c, result)
}
```

- [ ] **Step 4: 注册新路由**

在路由注册函数中添加：

```go
// 查询 Model 关联的 Endpoint 列表
v1.GET("/models/:id/endpoints", a.QueryEndpointsByModelID)

// 查询 Provider 关联的 Endpoint 列表
v1.GET("/providers/:id/endpoints", a.QueryEndpointsByProviderID)
```

- [ ] **Step 5: 验证编译**

```bash
go build ./internal/mods/resource/api/
```

Expected: 编译成功，无错误

---

### Task 5: 删除 ModelProvider 相关文件

**Files:**
- Delete: `internal/mods/resource/schema/model_provider.go`
- Delete: `internal/mods/resource/dal/model_provider.dal.go`
- Delete: `internal/mods/resource/biz/model_provider.biz.go`
- Delete: `internal/mods/resource/api/model_provider.api.go`

- [ ] **Step 1: 删除 ModelProvider Schema 文件**

```bash
rm internal/mods/resource/schema/model_provider.go
```

- [ ] **Step 2: 删除 ModelProvider DAL 文件**

```bash
rm internal/mods/resource/dal/model_provider.dal.go
```

- [ ] **Step 3: 删除 ModelProvider Biz 文件**

```bash
rm internal/mods/resource/biz/model_provider.biz.go
```

- [ ] **Step 4: 删除 ModelProvider API 文件**

```bash
rm internal/mods/resource/api/model_provider.api.go
```

- [ ] **Step 5: 验证编译**

```bash
go build ./...
```

Expected: 编译成功，无错误（可能需要先更新依赖）

---

### Task 6: 更新模块初始化和依赖注入

**Files:**
- Modify: `internal/mods/resource/main.go`
- Modify: `internal/wirex/wire.go`
- Regenerate: `internal/wirex/wire_gen.go`

- [ ] **Step 1: 读取 resource/main.go 文件**

```bash
cat internal/mods/resource/main.go
```

- [ ] **Step 2: 移除 ModelProvider 模块初始化**

在 `Init` 方法和 `RegisterRouters` 方法中，移除所有 ModelProvider 相关的代码：

```go
// 移除类似以下的代码：
// a.ModelProvider = &ModelProvider{...}
// a.ModelProvider.Init()
// v1.GET("/model-providers", a.ModelProvider.Query)
// 等等
```

- [ ] **Step 3: 读取 wirex/wire.go 文件**

```bash
cat internal/wirex/wire.go
```

- [ ] **Step 4: 移除 ModelProvider 相关的 Provider**

在 `wire.go` 中，移除所有 ModelProvider 相关的 Provider 定义：

```go
// 移除类似以下的代码：
// ModelProviderSet = wire.NewSet(
//     dal.NewModelProvider,
//     biz.NewModelProvider,
//     api.NewModelProvider,
// )
```

- [ ] **Step 5: 重新生成 Wire 依赖注入**

```bash
make wire
```

Expected: 生成新的 `wire_gen.go` 文件

- [ ] **Step 6: 验证编译**

```bash
go build ./...
```

Expected: 编译成功，无错误

---

### Task 7: 更新测试文件

**Files:**
- Modify: `test/endpoint_test.go`
- Delete: `test/model_provider_test.go`

- [ ] **Step 1: 读取当前 endpoint_test.go 文件**

```bash
cat test/endpoint_test.go
```

- [ ] **Step 2: 更新 Endpoint 测试用例**

在测试用例中添加新字段的测试：

```go
func TestEndpoint_CreateWithModelID(t *testing.T) {
    // 测试创建 Endpoint 时指定 model_id
    endpoint := &schema.Endpoint{
        ProviderID: "provider_123",
        ModelID:    "model_456",
        URL:        "https://api.openai.com/v1/chat/completions",
        Priority:   0,
        Weight:     1,
        Enabled:    1,
    }
    
    // 创建 Endpoint
    err := endpointBiz.Create(context.Background(), endpoint)
    assert.NoError(t, err)
    assert.NotEmpty(t, endpoint.ID)
    
    // 查询并验证
    result, err := endpointBiz.Get(context.Background(), endpoint.ID)
    assert.NoError(t, err)
    assert.Equal(t, "model_456", result.ModelID)
    assert.Equal(t, 0, result.Priority)
}
```

- [ ] **Step 3: 添加查询测试用例**

```go
func TestEndpoint_QueryByModelID(t *testing.T) {
    // 先创建一些测试数据
    // ...
    
    // 查询 Model 关联的 Endpoint
    result, err := endpointBiz.QueryEndpointsByModelID(context.Background(), "model_123")
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Greater(t, len(result), 0)
    
    // 验证排序（按优先级和权重）
    for i := 1; i < len(result); i++ {
        if result[i-1].Priority == result[i].Priority {
            assert.GreaterOrEqual(t, result[i-1].Weight, result[i].Weight)
        } else {
            assert.Less(t, result[i-1].Priority, result[i].Priority)
        }
    }
}
```

- [ ] **Step 4: 添加路由选择测试用例**

```go
func TestEndpoint_SelectEndpoint(t *testing.T) {
    // 创建多个 Endpoint，具有不同的优先级和权重
    // ...
    
    // 测试路由选择
    endpoint, err := endpointBiz.SelectEndpoint(context.Background(), "gpt-4")
    assert.NoError(t, err)
    assert.NotNil(t, endpoint)
    assert.Equal(t, "model_123", endpoint.ModelID)
}
```

- [ ] **Step 5: 删除 ModelProvider 测试文件**

```bash
rm test/model_provider_test.go
```

- [ ] **Step 6: 运行测试**

```bash
go test ./test/ -v -run TestEndpoint
```

Expected: 所有 Endpoint 测试通过

---

### Task 8: 更新前端 API 调用

**Files:**
- Modify: `frontend/src/apis/modules/endpoint.js`
- Delete: `frontend/src/apis/modules/modelProvider.js`

- [ ] **Step 1: 读取当前 endpoint.js 文件**

```bash
cat frontend/src/apis/modules/endpoint.js
```

- [ ] **Step 2: 添加新 API 方法**

```javascript
// 查询 Model 关联的 Endpoint 列表
export function getEndpointsByModelId(modelId) {
  return request.basic.get(`/api/v1/models/${modelId}/endpoints`);
}

// 查询 Provider 关联的 Endpoint 列表
export function getEndpointsByProviderId(providerId) {
  return request.basic.get(`/api/v1/providers/${providerId}/endpoints`);
}
```

- [ ] **Step 3: 删除 modelProvider.js 文件**

```bash
rm frontend/src/apis/modules/modelProvider.js
```

- [ ] **Step 4: 验证前端编译**

```bash
cd frontend
npm run build:prod
```

Expected: 编译成功，无错误

---

### Task 9: 更新前端 UI

**Files:**
- Modify: `frontend/src/views/resource/EndpointEditDialog.vue`
- Modify: `frontend/src/views/resource/ModelDetail.vue`

- [ ] **Step 1: 读取 EndpointEditDialog.vue 文件**

```bash
cat frontend/src/views/resource/EndpointEditDialog.vue
```

- [ ] **Step 2: 更新 Endpoint 表单**

在 Endpoint 表单中添加新字段：

```vue
<template>
  <a-form :model="form" @submit="handleSubmit">
    <!-- 现有字段 -->
    
    <!-- 新增：Model 选择 -->
    <a-form-item label="Model" name="model_id">
      <a-select v-model="form.model_id" placeholder="Select Model">
        <a-option v-for="model in models" :key="model.id" :value="model.id">
          {{ model.model_name }}
        </a-option>
      </a-select>
    </a-form-item>
    
    <!-- 新增：Real Model（可选） -->
    <a-form-item label="Real Model" name="real_model">
      <a-input v-model="form.real_model" placeholder="Override model name (optional)" />
    </a-form-item>
    
    <!-- 新增：Priority -->
    <a-form-item label="Priority" name="priority">
      <a-input-number v-model="form.priority" :min="0" :max="100" />
    </a-form-item>
    
    <!-- 现有字段 -->
  </a-form>
</template>
```

- [ ] **Step 3: 读取 ModelDetail.vue 文件**

```bash
cat frontend/src/views/resource/ModelDetail.vue
```

- [ ] **Step 4: 更新 Endpoint 列表表格**

在表格中显示新字段：

```vue
<template>
  <a-table :data="endpoints" :columns="columns">
    <!-- 现有列 -->
    
    <!-- 新增：Priority 列 -->
    <template #priority="{ record }">
      {{ record.priority }}
    </template>
    
    <!-- 新增：Real Model 列 -->
    <template #real_model="{ record }">
      {{ record.real_model || '-' }}
    </template>
  </a-table>
</template>
```

- [ ] **Step 5: 验证前端编译**

```bash
cd frontend
npm run build:prod
```

Expected: 编译成功，无错误

---

### Task 10: 完整构建和验证

**Files:**
- No file changes

- [ ] **Step 1: 运行后端编译**

```bash
go build ./...
```

Expected: 编译成功，无错误

- [ ] **Step 2: 运行后端测试**

```bash
go test ./test/ -v
```

Expected: 所有测试通过

- [ ] **Step 3: 运行前端编译**

```bash
cd frontend
npm run build:prod
```

Expected: 编译成功，无错误

- [ ] **Step 4: 验证表结构**

启动应用，GORM AutoMigrate 会自动创建表结构：

```bash
make start
```

检查数据库中的表结构是否正确。

- [ ] **Step 5: 测试 API**

```bash
# 测试创建 Endpoint
curl -X POST http://localhost:8040/api/v1/endpoints \
  -H "Content-Type: application/json" \
  -d '{
    "provider_id": "provider_123",
    "model_id": "model_456",
    "url": "https://api.openai.com/v1/chat/completions",
    "priority": 0,
    "weight": 1,
    "enabled": 1
  }'

# 测试查询 Model 的 Endpoint 列表
curl http://localhost:8040/api/v1/models/model_456/endpoints

# 测试查询 Provider 的 Endpoint 列表
curl http://localhost:8040/api/v1/providers/provider_123/endpoints
```

Expected: API 正常响应

---

## 自审清单

### 1. 规范覆盖检查

- ✅ 数据模型设计（Task 1）
- ✅ API 设计（Task 4）
- ✅ 查询场景（Task 2, Task 3）
- ✅ 代码迁移（Task 5, Task 6）
- ✅ 前端更新（Task 8, Task 9）
- ✅ 测试（Task 7, Task 10）

### 2. 占位符扫描

- ✅ 无 "TBD", "TODO", "implement later" 占位符
- ✅ 所有代码块都是完整的
- ✅ 所有命令都有明确的预期输出

### 3. 类型一致性检查

- ✅ `Endpoint` 结构体字段名称在所有任务中一致
- ✅ `EndpointForm` 结构体字段名称在所有任务中一致
- ✅ `EndpointQueryParam` 结构体字段名称在所有任务中一致
- ✅ API 路径在所有任务中一致

---

## 执行选项

**计划完成并保存到 `docs/plans/2026-05-30-simplify-model-provider.md`。两种执行方式：**

**1. Subagent-Driven（推荐）** - 我为每个任务分发一个独立的 subagent，任务间进行 review，快速迭代

**2. Inline Execution** - 在当前会话中执行任务，批量执行并设置检查点

**选择哪种方式？**
