# 简化 ModelProvider 设计：扩展 Endpoint 表

**日期**：2026-05-30
**状态**：已批准
**作者**：Claude Code
**审阅者**：用户

---

## 概述

本文档描述了简化 `model_provider` 表的设计方案。核心思路是将 Model 和 Provider 的关联信息迁移到 `Endpoint` 表中，从而减少数据模型的复杂度，提高查询效率，并简化创建流程。

**注意**：本设计适用于新系统设计开发阶段，无需考虑数据迁移兼容性问题。

### 问题陈述

当前设计中存在以下问题：
- `model_provider` 表增加了数据模型的复杂度
- 创建流程需要多步操作（Model → Provider → ModelProvider → Endpoint）
- 查询路由时需要多表关联（Model → ModelProvider → Provider）
- 数据可能不一致（ModelProvider 和 Endpoint 中的信息不同步）

### 解决方案

**完全移除 `model_provider` 表**，将所有关联信息迁移到 `Endpoint` 表中：
- 在 `Endpoint` 表中添加 `model_id`、`real_model`、`priority` 字段
- 所有 failover 和负载均衡策略集中在 Endpoint 层管理
- 提供便捷的查询 API 支持 Model → Provider 和 Provider → Model 的双向查询

---

## 数据模型设计

### 核心变更：扩展 Endpoint 表

```go
type Endpoint struct {
    ID          string          `json:"id" gorm:"size:20;primarykey;"`
    ProviderID  string          `json:"provider_id" gorm:"size:20;not null;index;"`
    ModelID     string          `json:"model_id" gorm:"size:20;not null;index;"`
    URL         string          `json:"url" gorm:"size:512;not null;"`
    ApiKey      string          `json:"api_key,omitempty" gorm:"size:512;"`
    RealModel   string          `json:"real_model,omitempty" gorm:"size:128;"`
    Priority    int             `json:"priority" gorm:"not null;default:0;"`
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

func (e *Endpoint) TableName() string {
    return config.C.FormatTableName("endpoint")
}
```

### 删除的表

**ModelProvider 表**：完全移除，所有信息迁移到 Endpoint 表。

### 关键字段说明

| 字段 | 来源 | 说明 |
|------|------|------|
| `ModelID` | 新增 | 关联 Model 表（必填） |
| `RealModel` | 从 ModelProvider 迁移 | 覆盖 Model 级别的 real_model（可选） |
| `Priority` | 从 ModelProvider 迁移 | failover 优先级，值越小优先级越高 |
| `ApiKey` | 保留 | 覆盖 Provider 级别的 api_key（可选） |
| `Weight` | 保留 | 同优先级内的负载均衡权重 |

### 数据完整性约束

1. **唯一性约束**：同一 Model + Provider + Priority 的组合必须唯一
2. **外键约束**：ModelID 和 ProviderID 必须关联到有效的记录
3. **逻辑删除**：支持逻辑删除，不影响唯一性约束

---

## API 设计

### Endpoint API 扩展

现有的 Endpoint CRUD API 保持不变，但需要支持新字段：

#### 创建 Endpoint

```http
POST /api/v1/endpoints
```

**请求体**：
```json
{
  "provider_id": "provider_abc123",
  "model_id": "model_xyz789",
  "url": "https://api.openai.com/v1/chat/completions",
  "api_key": "sk-xxx",
  "real_model": "gpt-4-turbo",
  "priority": 0,
  "weight": 1,
  "enabled": 1,
  "description": "OpenAI 主 endpoint"
}
```

**响应**：
```json
{
  "id": "endpoint_def456",
  "provider_id": "provider_abc123",
  "model_id": "model_xyz789",
  "url": "https://api.openai.com/v1/chat/completions",
  "api_key": "sk-xxx",
  "real_model": "gpt-4-turbo",
  "priority": 0,
  "weight": 1,
  "enabled": 1,
  "created_at": "2026-05-30T10:00:00Z"
}
```

#### 查询 Endpoint 列表

```http
GET /api/v1/endpoints?model_id=model_xyz789&provider_id=provider_abc123
```

**新增查询参数**：
- `model_id`：按 Model 过滤
- `provider_id`：按 Provider 过滤（已有）
- `priority`：按优先级过滤

#### 更新 Endpoint

```http
PUT /api/v1/endpoints/:id
```

支持更新所有字段，包括 `model_id`、`real_model`、`priority` 等。

### 新增查询 API

#### 查询 Model 关联的所有 Provider（通过 Endpoint）

```http
GET /api/v1/models/:id/endpoints
```

**响应**：
```json
{
  "data": [
    {
      "id": "endpoint_def456",
      "provider_id": "provider_abc123",
      "model_id": "model_xyz789",
      "url": "https://api.openai.com/v1/chat/completions",
      "priority": 0,
      "weight": 1
    }
  ]
}
```

#### 查询 Provider 服务的所有 Model（通过 Endpoint）

```http
GET /api/v1/providers/:id/endpoints
```

**响应**：
```json
{
  "data": [
    {
      "id": "endpoint_def456",
      "provider_id": "provider_abc123",
      "model_id": "model_xyz789",
      "url": "https://api.openai.com/v1/chat/completions",
      "priority": 0,
      "weight": 1
    }
  ]
}
```

### 删除的 API

**ModelProvider 相关的所有 API**：
- `GET /api/v1/model-providers`
- `POST /api/v1/model-providers`
- `PUT /api/v1/model-providers/:id`
- `DELETE /api/v1/model-providers/:id`

这些 API 完全移除，功能由 Endpoint API 承担。

---

## 查询场景和数据流

### 核心查询场景

#### 场景 1：客户端请求路由

当客户端发送请求 `POST /v1/chat/completions` 并指定 `model=gpt-4` 时：

```sql
-- 查询路由的 SQL
SELECT e.*, p.protocol, p.api_key as provider_api_key
FROM endpoint e
JOIN provider p ON e.provider_id = p.id
JOIN model m ON e.model_id = m.id
WHERE m.model_name = 'gpt-4'
  AND e.enabled = 1
  AND p.enabled = 1
  AND m.enabled = 1
ORDER BY e.priority ASC, e.weight DESC;
```

**返回结果**：
- 所有服务 `gpt-4` 的 Endpoint
- 按优先级排序（值越小优先级越高）
- 同优先级内按权重负载均衡

#### 场景 2：Failover 逻辑

```go
// 伪代码：选择 Endpoint
func selectEndpoint(modelName string) (*Endpoint, error) {
    endpoints := queryEndpointsByModel(modelName)
    
    // 按优先级分组
    priorityGroups := groupByPriority(endpoints)
    
    for _, group := range priorityGroups {
        // 同优先级内按权重随机选择
        endpoint := weightedRandomSelect(group)
        
        // 尝试调用
        if tryEndpoint(endpoint) {
            return endpoint, nil
        }
        
        // 失败，尝试同优先级的其他 Endpoint
        for _, ep := range group {
            if ep != endpoint && tryEndpoint(ep) {
                return ep, nil
            }
        }
        
        // 同优先级都失败，降级到下一优先级
    }
    
    return nil, errors.New("all endpoints failed")
}
```

#### 场景 3：查看 Model 的 Endpoint 列表

```sql
-- 查询 Model 关联的所有 Endpoint
SELECT e.*
FROM endpoint e
WHERE e.model_id = 'model_xyz789'
  AND e.enabled = 1
ORDER BY e.priority ASC, e.weight DESC;
```

#### 场景 4：查看 Provider 的 Endpoint 列表

```sql
-- 查询 Provider 的所有 Endpoint
SELECT e.*
FROM endpoint e
WHERE e.provider_id = 'provider_abc123'
  AND e.enabled = 1
ORDER BY e.priority ASC, e.weight DESC;
```

### 数据流示例

#### 创建 Model + Provider + Endpoint 的完整流程

```bash
# 1. 创建 Model
POST /api/v1/models
{
  "model_name": "gpt-4",
  "model_code": "gpt4",
  "apis": '["chat_completion"]',
  "context_length": 128000,
  "owner": "OpenAI"
}
→ 返回 model_id: "model_xyz789"

# 2. 创建 Provider
POST /api/v1/providers
{
  "name": "openai-official",
  "protocol": "openai",
  "api_key": "sk-default"
}
→ 返回 provider_id: "provider_abc123"

# 3. 创建 Endpoint（关联 Model 和 Provider）
POST /api/v1/endpoints
{
  "provider_id": "provider_abc123",
  "model_id": "model_xyz789",
  "url": "https://api.openai.com/v1/chat/completions",
  "priority": 0,
  "weight": 1
}
→ 返回 endpoint_id: "endpoint_def456"

# 4. 添加 Failover Endpoint（相同 Model，不同 Provider）
POST /api/v1/providers
{
  "name": "azure-openai",
  "protocol": "openai",
  "api_key": "azure-key"
}
→ 返回 provider_id: "provider_ghi789"

POST /api/v1/endpoints
{
  "provider_id": "provider_ghi789",
  "model_id": "model_xyz789",
  "url": "https://azure.openai.com/v1/chat/completions",
  "priority": 1,  // 较低优先级
  "weight": 1
}
→ 返回 endpoint_id: "endpoint_jkl012"
```

#### 查询 Model 的 Endpoint 列表

```bash
GET /api/v1/models/model_xyz789/endpoints

响应：
{
  "data": [
    {
      "id": "endpoint_def456",
      "provider_id": "provider_abc123",
      "priority": 0,
      "weight": 1
    },
    {
      "id": "endpoint_jkl012",
      "provider_id": "provider_ghi789",
      "priority": 1,
      "weight": 1
    }
  ]
}
```

---

## 表结构设计

### Endpoint 表结构（新系统）

由于是新系统设计开发阶段，Endpoint 表结构将通过 GORM AutoMigrate 自动创建，无需手动迁移。

**核心字段：**
- `id` - 主键
- `provider_id` - 关联 Provider
- `model_id` - 关联 Model
- `url` - 上游 API 地址
- `api_key` - 可覆盖的 API Key
- `real_model` - 可覆盖的真实模型名
- `priority` - failover 优先级
- `weight` - 负载均衡权重
- `enabled` - 启用状态

**唯一约束：**
- `(model_id, provider_id, priority, deleted)` - 确保同一 Model-Provider-Priority 组合唯一

**外键约束：**
- `model_id` → `model.id`
- `provider_id` → `provider.id`

### ModelProvider 表（已移除）

原 `model_provider` 表的功能已完全迁移到 `endpoint` 表中，无需创建此表。

---

## 注意事项和权衡

### 关键权衡点

#### 1. 数据冗余 vs 查询效率

**权衡**：
- ❌ 如果一个 Model 有 3 个 Provider，需要创建 3 个 Endpoint（数据冗余）
- ✅ 但查询路由时直接查 Endpoint 表，性能更好

**影响**：
- Endpoint 表可能会比原来的 ModelProvider + Endpoint 表组合更大
- 但减少了 JOIN 操作，查询更简单

**建议**：
- 监控 Endpoint 表的增长
- 如果数据量过大，考虑添加缓存层（Redis）

#### 2. 数据一致性 vs 简化模型

**权衡**：
- ❌ 之前：ModelProvider 和 Endpoint 是独立的，可以分别管理
- ✅ 现在：所有信息集中在 Endpoint，简化了管理

**潜在问题**：
- 如果需要更新一个 Provider 的默认 api_key，需要更新所有关联的 Endpoint
- 如果需要查看一个 Model 被哪些 Provider 服务，需要通过 Endpoint 间接查询

**解决方案**：
- 提供批量更新 API
- 提供便捷的查询 API（已在第 2 节设计）

#### 3. 创建流程的复杂性

**之前**：
```bash
# 创建 Model
POST /models
# 创建 Provider
POST /providers
# 创建 ModelProvider 关联
POST /model-providers
# 创建 Endpoint
POST /endpoints
```

**现在**：
```bash
# 创建 Model
POST /models
# 创建 Provider
POST /providers
# 创建 Endpoint（同时关联 Model 和 Provider）
POST /endpoints
```

**改进**：
- 减少了一步操作
- 更直观：创建 Endpoint 时直接指定 Model 和 Provider

### 边界情况处理

#### 1. 重复的 Endpoint

**场景**：同一个 Model + Provider + Priority 可能有多个 Endpoint

**处理**：
- 添加唯一约束：`uk_endpoint_model_provider_priority (model_id, provider_id, priority, deleted)`
- 如果需要同优先级的多个 Endpoint，通过 weight 区分

#### 2. 删除 Model 或 Provider

**场景**：删除一个 Model 时，关联的 Endpoint 怎么处理？

**处理**：
- **软删除**：Model 标记为删除，Endpoint 保持不变（但查询时过滤掉）
- **硬删除**：同时删除所有关联的 Endpoint（级联删除）
- **建议**：使用软删除，避免数据丢失

#### 3. 更新 Provider 的 api_key

**场景**：Provider 的默认 api_key 更新了，需要同步更新所有 Endpoint？

**处理**：
- Endpoint 的 api_key 是可覆盖的，如果为空则使用 Provider 的 api_key
- 提供批量更新 Endpoint api_key 的 API
- 或者在查询时动态获取 Provider 的 api_key（如果 Endpoint 的 api_key 为空）

### 性能优化建议

#### 1. 索引优化

```sql
-- 核心查询索引
CREATE INDEX idx_endpoint_model_enabled ON endpoint(model_id, enabled) WHERE deleted = '0';
CREATE INDEX idx_endpoint_provider_enabled ON endpoint(provider_id, enabled) WHERE deleted = '0';
CREATE INDEX idx_endpoint_priority ON endpoint(model_id, priority, weight) WHERE deleted = '0' AND enabled = 1;
```

#### 2. 缓存策略

- **路由缓存**：Model Name → Endpoint 列表（缓存 5 分钟）
- **Provider 信息缓存**：Provider ID → Provider 详情（缓存 10 分钟）
- **失效策略**：Endpoint 或 Provider 更新时清除相关缓存

#### 3. 查询优化

- 使用 **预编译语句**（Prepared Statements）
- 使用 **连接池**（Connection Pooling）
- 避免 **N+1 查询**：使用 JOIN 或批量查询

### 测试策略

#### 单元测试

- [ ] Endpoint 创建/更新/删除
- [ ] Model → Endpoint 查询
- [ ] Provider → Endpoint 查询
- [ ] 路由选择算法（priority + weight）
- [ ] Failover 逻辑

#### 集成测试

- [ ] 完整的路由流程
- [ ] 多 Endpoint 的 failover 场景
- [ ] 并发访问场景

#### 性能测试

- [ ] 大量 Endpoint 的查询性能
- [ ] 并发路由请求的吞吐量
- [ ] 缓存命中率

### 监控和告警

#### 关键指标

- **Endpoint 数量**：监控增长趋势
- **路由查询延迟**：P95 < 100ms
- **缓存命中率**：> 80%
- **Failover 次数**：监控异常

#### 告警规则

- Endpoint 数量异常增长（可能是误操作）
- 路由查询延迟持续升高
- 缓存命中率突然下降
- Failover 频繁触发

---

## 总结

本设计通过扩展 Endpoint 表，移除 ModelProvider 表，实现了以下目标：

✅ **简化数据模型**：减少一个表，数据结构更清晰
✅ **提高查询效率**：减少多表关联，查询更直接
✅ **简化创建流程**：创建 Endpoint 时直接指定 Model 和 Provider
✅ **集中管理策略**：所有 failover 和负载均衡策略集中在 Endpoint 层
✅ **支持双向查询**：通过 Endpoint 支持 Model → Provider 和 Provider → Model 查询

### 适用场景

本设计适用于：
- ✅ 新系统设计开发阶段
- ✅ 简单的路由场景，Model 和 Provider 的对应关系不复杂
- ✅ 优先考虑数据模型简洁性和查询效率
- ✅ 愿意接受一定程度的数据冗余

### 不适用场景

本设计不适用于：
- ❌ 需要频繁查询 Model-Provider 对应关系（性能敏感）
- ❌ Model 和 Provider 的对应关系非常复杂（多对多）
- ❌ 需要独立管理 Model-Provider 关联（与 Endpoint 解耦）

---

**文档结束**
