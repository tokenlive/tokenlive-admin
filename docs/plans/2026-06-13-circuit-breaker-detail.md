# Circuit Breaker Detail Enrichment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 丰富首页熔断告警细节，由纯端点 ID 升级为显示具体的“大模型 (供应商)”的红色 Tag，并在悬浮时提示详情（端点 ID 和 URL）。

**Architecture:** 后端联查 GORM 获取关联的外键表，接口返回结构化数据；前端使用 `<a-tag>` 与 `<a-tooltip>` 进行组合渲染。

**Tech Stack:** Go (Gin / GORM), Redis, Vue 3 (Ant Design Vue)

---

### Task 1: 后端 API 返回字段重构与数据库级联联查

**Files:**

- Modify: `internal/mods/dashboard/api/dashboard.api.go`

- [ ] **Step 1: 在 API 文件中声明 CircuitBreakerInfo 结构体**

在 [/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go) 的 package 导入块下方（例如第 22 行附近）声明此结构体：

```go
type CircuitBreakerInfo struct {
 ID           string `json:"id"`
 Type         string `json:"type"`          // "endpoint" 或 "service"
 Name         string `json:"name"`          // 显示名称
 ModelID      string `json:"model_id"`      // 关联模型 ID
 ModelName    string `json:"model_name"`    // 关联模型名称
 ProviderID   string `json:"provider_id"`   // 关联供应商 ID
 ProviderName string `json:"provider_name"` // 关联供应商名称
 URL          string `json:"url"`           // 关联的 URL 地址
}
```

- [ ] **Step 2: 修改 QueryCircuitBreakers API 方法**

修改 [/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go) 中的 `QueryCircuitBreakers` 方法（大约在第 110-125 行左右）。
原代码：

```go
func (a *Dashboard) QueryCircuitBreakers(c *gin.Context) {
 if a.RedisClient == nil {
  util.ResSuccess(c, []string{})
  return
 }

 ctx := c.Request.Context()
 endpoints, _ := a.RedisClient.SMembers(ctx, "aigw:cb:open_endpoints").Result()
 services, _ := a.RedisClient.SMembers(ctx, "aigw:cb:open_services").Result()

 all := make([]string, 0, len(endpoints)+len(services))
 all = append(all, endpoints...)
 all = append(all, services...)

 util.ResSuccess(c, all)
}
```

修改为：

```go
func (a *Dashboard) QueryCircuitBreakers(c *gin.Context) {
 if a.RedisClient == nil {
  util.ResSuccess(c, []CircuitBreakerInfo{})
  return
 }

 ctx := c.Request.Context()
 endpoints, _ := a.RedisClient.SMembers(ctx, "aigw:cb:open_endpoints").Result()
 services, _ := a.RedisClient.SMembers(ctx, "aigw:cb:open_services").Result()

 var endpointInfos []CircuitBreakerInfo
 if len(endpoints) > 0 {
  var dbEndpoints []rschema.Endpoint
  err := a.DB.WithContext(ctx).
   Preload("Model").
   Preload("Provider").
   Where("id IN ?", endpoints).
   Find(&dbEndpoints).Error
  if err == nil {
   for _, ep := range dbEndpoints {
    info := CircuitBreakerInfo{
     ID:   ep.ID,
     Type: "endpoint",
     URL:  ep.URL,
    }
    if ep.Model != nil {
     info.ModelID = ep.ModelID
     info.ModelName = ep.Model.ModelName
    }
    if ep.Provider != nil {
     info.ProviderID = ep.ProviderID
     info.ProviderName = ep.Provider.Name
    }
    // 组合 Name
    if info.ModelName != "" && info.ProviderName != "" {
     info.Name = fmt.Sprintf("%s (%s)", info.ModelName, info.ProviderName)
    } else if info.ModelName != "" {
     info.Name = info.ModelName
    } else {
     info.Name = ep.URL
    }
    endpointInfos = append(endpointInfos, info)
   }
  }

  // 兜底填充：如果某些端点被逻辑删除（但在 Redis 缓存状态里仍然存活），则使用 ID 填补
  foundMap := make(map[string]bool)
  for _, info := range endpointInfos {
   foundMap[info.ID] = true
  }
  for _, id := range endpoints {
   if !foundMap[id] {
    endpointInfos = append(endpointInfos, CircuitBreakerInfo{
     ID:   id,
     Type: "endpoint",
     Name: id,
    })
   }
  }
 }

 var serviceInfos []CircuitBreakerInfo
 for _, s := range services {
  serviceInfos = append(serviceInfos, CircuitBreakerInfo{
   ID:   s,
   Type: "service",
   Name: s,
  })
 }

 allInfos := make([]CircuitBreakerInfo, 0, len(endpointInfos)+len(serviceInfos))
 allInfos = append(allInfos, endpointInfos...)
 allInfos = append(allInfos, serviceInfos...)

 util.ResSuccess(c, allInfos)
}
```

- [ ] **Step 3: 编译验证**

在 `tokenlive-admin` 项目根目录下运行编译，确认后端能够正常构建：

```bash
go build -o bin/server main.go
```

### Task 2: 前端控制台首页告警组件重构

**Files:**

- Modify: `frontend/src/views/home/index.vue`

- [ ] **Step 1: 修改模板，使用 a-tooltip 和 a-tag 进行数据渲染**

修改 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/home/index.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/home/index.vue) 中的第 20 行。
原代码：

```html
                        <strong style="margin-left: 8px; color: #595959">[{{ circuitBreakers.join(', ') }}]</strong>
```

修改为：

```html
                        <span style="margin-left: 8px">
                            <a-tooltip
                                v-for="cb in circuitBreakers"
                                :key="cb.id || cb">
                                <template #title>
                                    <div style="font-size: 12px; line-height: 1.6; padding: 4px">
                                        <div><strong>Type:</strong> {{ cb.type || 'endpoint' }}</div>
                                        <div><strong>ID:</strong> {{ cb.id || cb }}</div>
                                        <div
                                            v-if="cb.url"
                                            style="word-break: break-all">
                                            <strong>URL:</strong> {{ cb.url }}
                                        </div>
                                    </div>
                                </template>
                                <a-tag
                                    color="error"
                                    style="cursor: help; margin-right: 4px; border-radius: 4px; font-weight: 500">
                                    <alert-outlined style="margin-right: 4px" />{{ cb.name || cb.id || cb }}
                                </a-tag>
                            </a-tooltip>
                        </span>
```

- [ ] **Step 2: 运行 Prettier 格式化前端文件**

在 `frontend` 目录下运行：

```bash
npx prettier --config .prettierrc --write src/views/home/index.vue
```

### Task 3: 界面集成手动验证

- [ ] **Step 1: 启动服务并进入首页**

启动 `tokenlive-admin` 并观察首页。

- [ ] **Step 2: 注入模拟熔断数据进行检验**

1. 选择一个数据库里存在的大模型端点 ID。
2. 使用 redis-cli 向 `aigw:cb:open_endpoints` 添加此端点 ID：

   ```bash
   redis-cli -h 127.0.0.1 -p 6087 -a 12345 SADD aigw:cb:open_endpoints <VALID_ENDPOINT_ID>
   ```

3. 观察首页的告警是否正确渲染为红色 `ModelName (ProviderName)` 标签。
4. 鼠标悬停标签，验证 Tooltip 能否拉出 ID 详情和上游 API 访问 URL。
5. 验证完清理测试数据：

   ```bash
   redis-cli -h 127.0.0.1 -p 6087 -a 12345 SREM aigw:cb:open_endpoints <VALID_ENDPOINT_ID>
   ```
