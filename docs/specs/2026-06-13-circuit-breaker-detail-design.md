# 熔断告警详情丰富化技术设计方案

## 1. 目标与背景

首页（Dashboard）中，当有大模型服务节点熔断时，顶部会展示一个横幅告警。但由于熔断的数据存储仅保留了 `endpoint` 的 ID，导致原有 API 返回的数据也仅是端点的 ID 列表。管理员在首页只看到冷冰冰的 ID，根本不知道是哪款模型以及哪家提供商发生了熔断，这极大地妨碍了排障效率。
本项目旨在通过后端级联查询与前端组件重构，将熔断提醒丰富为“模型名称 (供应商名称)”的精致红色 Tag 徽章形式，并在鼠标悬浮时展示具体的端点 ID 和上游 API 访问地址。

## 2. 详细技术方案

### 2.1 后端修改设计 (Go API 丰富化)

在 [/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/dashboard/api/dashboard.api.go) 中：

1. 声明熔断详情专用结构体 `CircuitBreakerInfo`：

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

2. 修改接口方法 `QueryCircuitBreakers(c *gin.Context)`：
   - 当 `endpoints` 列表非空时，通过 GORM `Preload("Model").Preload("Provider")` 去数据库查询这些端点并获取外键表信息。
   - 提取并格式化 `Name` 字段为 `ModelName (ProviderName)`，或在其数据不全时回退为 `URL` 或 `ID`。
   - 具有健壮性保护：若个别端点在数据库内已被逻辑删除（如不存在），对其进行填充兜底，避免返回数据缺失。
   - 接口返回 `[]CircuitBreakerInfo` 列表。

### 2.2 前端修改设计 (Vue 界面升级)

在 [/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/home/index.vue](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/home/index.vue) 中：

1. 替换告警提示部分的普通中括号包裹文本：
   原代码：

   ```html
   <strong style="margin-left: 8px; color: #595959">[{{ circuitBreakers.join(', ') }}]</strong>
   ```

   修改为使用精致的红色 `<a-tag>` 并且自带 `<a-tooltip>` 的高级悬浮徽章组件：

   ```html
   <span style="margin-left: 8px;">
       <a-tooltip v-for="cb in circuitBreakers" :key="cb.id || cb">
           <template #title>
               <div style="font-size: 12px; line-height: 1.6; padding: 4px;">
                   <div><strong>Type:</strong> {{ cb.type }}</div>
                   <div><strong>ID:</strong> {{ cb.id || cb }}</div>
                   <div v-if="cb.url" style="word-break: break-all;"><strong>URL:</strong> {{ cb.url }}</div>
               </div>
           </template>
           <a-tag color="error" style="cursor: help; margin-right: 4px; border-radius: 4px; font-weight: 500">
               <alert-outlined style="margin-right: 4px" />{{ cb.name || cb.id || cb }}
           </a-tag>
       </a-tooltip>
   </span>
   ```

2. 保持兼容性：在获取数据方法 `fetchTelemetryData` 中，API 获取返回的结果赋给 `circuitBreakers.value`。
   如果因为任何异常返回的是单纯的 ID，模板中的逻辑如 `cb.name || cb.id || cb` 可实现平滑向下兼容，不会导致页面崩溃。

## 3. 验证计划

1. **构建并启动网关与管理后台**。
2. **模拟熔断**：通过向 Redis 中添加模拟熔断数据（如向 `aigw:cb:open_endpoints` 注入特定端点 ID）。
3. **查看效果**：
   - 访问管理后台首页，检查红色告警条中是否不再直接展示 ID，而是以红色的标签组件展示模型名与厂商名。
   - 鼠标悬停在 Tag 上时，验证浮窗能正确展示具体的端点 ID 以及其对应的上游 API URL。
