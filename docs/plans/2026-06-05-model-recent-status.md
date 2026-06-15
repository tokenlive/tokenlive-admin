# 模型列表最近状态显示功能实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 AI 网关模型列表中展示各模型最近 100 分钟的状态，以 10 个颜色方块切片（10分钟维度）直观显现，网关进行分钟级数据记录，管理台批量汇总并展现。

**Architecture:** 网关在 Outbound 阶段通过 `StatusCollectorFilter` 分钟级对 Redis Key (String) 进行递增记录并设 2 小时 TTL；管理台在获取模型列表时利用 `MGET` 批量拉取 100 分钟数据并快速在内存中按 10 分钟切片累加，最后在前端列表渲染带有 Tooltip 的彩色方块。

**Tech Stack:** Go, Gin, Ant Design Vue 4 (Vue 3), Redis (go-redis/v9).

---

### Task 1: 网关新增 status_collector 过滤器

**Files:**

- Create: `pkg/filters/outbound/status_collector.go`

- [ ] **Step 1: 创建 status_collector.go 并实现过滤器**
  
  在 `/Users/chenzhiguo/Projects/ai-gateway/pkg/filters/outbound/status_collector.go` 中写入以下代码：

  ```go
  package outbound

  import (
   "fmt"
   "time"

   "ai-gateway/pkg/core"
   "github.com/redis/go-redis/v9"
  )

  // StatusCollectorFilter 收集模型成功/失败的过滤器，用于最近状态显示
  type StatusCollectorFilter struct {
   rdb *redis.Client
  }

  // NewStatusCollectorFilter 创建 StatusCollectorFilter
  func NewStatusCollectorFilter(rdb *redis.Client) *StatusCollectorFilter {
   return &StatusCollectorFilter{
    rdb: rdb,
   }
  }

  func (f *StatusCollectorFilter) Name() string                        { return "status_collector" }
  func (f *StatusCollectorFilter) Order() int                          { return 31 }
  func (f *StatusCollectorFilter) Criticality() core.FilterCriticality { return core.BestEffort }

  func (f *StatusCollectorFilter) OnResponse(gctx *core.GatewayContext) error {
   if f.rdb == nil || gctx.Model == "" {
    return nil
   }

   minute := time.Now().Unix() / 60
   var statusKey string
   if gctx.Err == nil {
    statusKey = fmt.Sprintf("aigw:status:model:%s:%d:s", gctx.Model, minute)
   } else {
    statusKey = fmt.Sprintf("aigw:status:model:%s:%d:f", gctx.Model, minute)
   }

   // 使用 Pipeline 高效递增并设置过期时间为 2 小时
   pipe := f.rdb.Pipeline()
   pipe.Incr(gctx.Ctx, statusKey)
   pipe.Expire(gctx.Ctx, statusKey, 2*time.Hour)
   _, _ = pipe.Exec(gctx.Ctx)

   return nil
  }
  ```

- [ ] **Step 2: 验证编译**
  
  在 `/Users/chenzhiguo/Projects/ai-gateway` 目录下执行编译，验证无语法错误：
  `go build ./pkg/filters/outbound/...`

---

### Task 2: 网关注册与装配 status_collector

**Files:**

- Modify: `cmd/server/wire/engine.go`

- [ ] **Step 1: 修改 cmd/server/wire/engine.go 注册 StatusCollectorFilter 并装配至各 pipelines**

  在 `/Users/chenzhiguo/Projects/ai-gateway/cmd/server/wire/engine.go` 中定位到 `engine.RegisterFilter("access_log", ...)`，并在其后注册新过滤器，同时更新内置 pipelines 的 `OutboundFilters`：
  
  ```go
   engine.RegisterFilter("metrics", outbound.NewMetricsFilter(prometheus.DefaultRegisterer))
   engine.RegisterFilter("access_log", outbound.NewAccessLogFilter(logger.Logger))
   engine.RegisterFilter("status_collector", outbound.NewStatusCollectorFilter(rdb)) // 新增这行
  ```

  并且，更新 `buildFromRelationalConfig` 中 `model_list` 之外的所有 pipeline 的 OutboundFilters，如：

  ```go
   // 2. 创建通用的 chat_completion pipeline
   if _, exists := engineConfig.Pipelines["chat_completion"]; !exists {
    engineConfig.Pipelines["chat_completion"] = &core.PipelineConfig{
     Name:         "chat_completion",
     RequestTypes: []core.RequestType{core.RequestTypeChatCompletion},
     Invoker: core.InvokerConfig{
      Type: "cluster",
     },
     InboundFilters:          inboundFilters,
     OutboundFilters:         []string{"token_settlement", "sticky_session", "metrics", "status_collector", "access_log"}, // 加上 status_collector
     CriticalOutboundFilters: []string{"token_settlement", "sticky_session"},
    }
   }

   // 3. 创建通用的 embedding pipeline
   if _, exists := engineConfig.Pipelines["embedding"]; !exists {
    engineConfig.Pipelines["embedding"] = &core.PipelineConfig{
     Name:         "embedding",
     RequestTypes: []core.RequestType{core.RequestTypeEmbedding},
     Invoker: core.InvokerConfig{
      Type: "cluster",
     },
     InboundFilters:          inboundFilters,
     OutboundFilters:         []string{"token_settlement", "sticky_session", "metrics", "status_collector", "access_log"}, // 加上 status_collector
     CriticalOutboundFilters: []string{"token_settlement", "sticky_session"},
    }
   }

   // 4. 创建通用的 messages pipeline (Anthropic 原生协议)
   if _, exists := engineConfig.Pipelines["messages"]; !exists {
    engineConfig.Pipelines["messages"] = &core.PipelineConfig{
     Name:         "messages",
     RequestTypes: []core.RequestType{core.RequestTypeMessages},
     Invoker: core.InvokerConfig{
      Type: "cluster",
     },
     InboundFilters:          inboundFilters,
     OutboundFilters:         []string{"token_settlement", "sticky_session", "metrics", "status_collector", "access_log"}, // 加上 status_collector
     CriticalOutboundFilters: []string{"token_settlement", "sticky_session"},
    }
   }
  ```

- [ ] **Step 2: 验证网关编译与测试**
  
  运行测试确保装配工作正常：
  `go test ./pkg/filters/outbound/...`

---

### Task 3: 管理台后端 Schema 扩展与 Redis 注入

**Files:**

- Modify: `internal/mods/resource/schema/model.go`
- Modify: `internal/mods/resource/biz/model.biz.go`

- [ ] **Step 1: 修改 internal/mods/resource/schema/model.go**

  在 `/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/schema/model.go` 文件尾部增加以下数据结构，并在 `Model` 结构体中添加 `StatusPoints` 字段：
  
  ```go
  // ModelStatusPoint 模型最近状态时间点的成功失败数
  type ModelStatusPoint struct {
   SuccessCount int64 `json:"success_count"`
   FailCount    int64 `json:"fail_count"`
  }
  ```

  同时，在 `Model` 结构体中（例如 `UpdatedAt` 字段下方）新增：

  ```go
   StatusPoints  []ModelStatusPoint `json:"status_points" gorm:"-"`
  ```

- [ ] **Step 2: 修改 internal/mods/resource/biz/model.biz.go 注入 RedisClient**

  在 `/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/model.biz.go` 的 `Model` 结构体中添加 `RedisClient` 字段：
  
  ```go
  // Model business logic layer
  type Model struct {
   Trans             *util.Trans
   ModelDAL          *dal.Model
   DataPermissionBIZ *DataPermission
   ConfigRedisSync   *ConfigRedisSync
   RedisClient       *redis.Client // 加上这行
  }
  ```

  并在文件头部增加导入 `"github.com/redis/go-redis/v9"`。

---

### Task 4: 实现最近状态数据 MGET 拉取与切片计算算法

**Files:**

- Modify: `internal/mods/resource/biz/model.biz.go`

- [ ] **Step 1: 实现 fillModelsStatusPoints 处理逻辑**

  在 `/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/model.biz.go` 文件末尾添加状态填充函数：
  
  ```go
  func (m *Model) fillModelsStatusPoints(ctx context.Context, models []*schema.Model) {
   if len(models) == 0 || m.RedisClient == nil {
    return
   }

   currentMin := time.Now().Unix() / 60
   numModels := len(models)
   numMinutes := 100
   numKeys := numModels * numMinutes * 2
   keys := make([]string, numKeys)

   idx := 0
   for _, model := range models {
    for i := 0; i < numMinutes; i++ {
     minute := currentMin - int64(numMinutes-1-i)
     keys[idx] = fmt.Sprintf("aigw:status:model:%s:%d:s", model.ModelCode, minute)
     keys[idx+1] = fmt.Sprintf("aigw:status:model:%s:%d:f", model.ModelCode, minute)
     idx += 2
    }
   }

   values, err := m.RedisClient.MGet(ctx, keys...).Result()
   if err != nil {
    return
   }

   idx = 0
   for _, model := range models {
    minSuccess := make([]int64, numMinutes)
    minFail := make([]int64, numMinutes)

    for i := 0; i < numMinutes; i++ {
     sVal := values[idx]
     fVal := values[idx+1]
     idx += 2

     if sVal != nil {
      if sStr, ok := sVal.(string); ok {
       if val, parseErr := strconv.ParseInt(sStr, 10, 64); parseErr == nil {
        minSuccess[i] = val
       }
      }
     }
     if fVal != nil {
      if fStr, ok := fVal.(string); ok {
       if val, parseErr := strconv.ParseInt(fStr, 10, 64); parseErr == nil {
        minFail[i] = val
       }
      }
     }
    }

    points := make([]schema.ModelStatusPoint, 10)
    for pIdx := 0; pIdx < 10; pIdx++ {
     var successSum int64
     var failSum int64
     for mOffset := 0; mOffset < 10; mOffset++ {
      mIdx := pIdx*10 + mOffset
      successSum += minSuccess[mIdx]
      failSum += minFail[mIdx]
     }
     points[pIdx] = schema.ModelStatusPoint{
      SuccessCount: successSum,
      FailCount:    failSum,
     }
    }
    model.StatusPoints = points
   }
  }
  ```

- [ ] **Step 2: 在 Query 方法中调用 fillModelsStatusPoints**

  修改 `Query` 方法以在结果列表非空时调用此函数：

  ```go
  // Query models.
  func (m *Model) Query(ctx context.Context, params schema.ModelQueryParam) (*schema.ModelQueryResult, error) {
   params.Pagination = true

   result, err := m.ModelDAL.Query(ctx, params, schema.ModelQueryOptions{
    QueryOptions: util.QueryOptions{
     OrderFields: []util.OrderByParam{
      {Field: "created_at", Direction: util.DESC},
     },
    },
   })
   if err != nil {
    return nil, err
   }

   if len(result.Data) > 0 && m.RedisClient != nil {
    m.fillModelsStatusPoints(ctx, result.Data)
   }

   return result, nil
  }
  ```

---

### Task 5: 重新编译并更新依赖注入

**Files:**

- Modify: `internal/wirex/wire_gen.go`

- [ ] **Step 1: 运行 wire 生成依赖注入代码**

  在 `/Users/chenzhiguo/Projects/tokenlive-admin` 目录下重新运行 wire 生成工具以更新 `wire_gen.go`：
  `wire gen ./internal/wirex`

- [ ] **Step 2: 验证编译**

  执行编译，确保所有模块被成功注入且能够正确编译：
  `go build ./cmd/server`

---

### Task 6: 管理台前端页面添加最近状态展示

**Files:**

- Modify: `frontend/src/views/resource/model.vue`

- [ ] **Step 1: 修改 model.vue 添加表格列和渲染模板**

  打开 `/Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/views/resource/model.vue`。
  首先在 `columns` 数组中的 `enabled` 列后增加一列配置：

  ```javascript
      { title: t('pages.model.recent_status'), key: 'recent_status', width: 220 },
  ```
  
  然后在 `<a-table>` 的 `<template #bodyCell>` 里增加如下对 `recent_status` 列的定制渲染模板（例如在 `'enabled' === column.key` 模板块下方）：
  
  ```html
                          <template v-if="'recent_status' === column.key">
                              <div style="display: flex; gap: 2px; align-items: center;">
                                  <a-tooltip v-for="(point, index) in (record.status_points || [])" :key="index">
                                      <template #title>
                                          成功: {{ point.success_count }} | 失败: {{ point.fail_count }}
                                      </template>
                                      <div :style="getPointStyle(point)"></div>
                                  </a-tooltip>
                              </div>
                          </template>
  ```

  在 `<script setup>` 中，增加工具方法 `getPointStyle`：
  
  ```javascript
  function getPointStyle(point) {
      let color = '#f5f5f5';
      let border = '1px solid #d9d9d9';
      if (point.success_count > 0 && point.fail_count === 0) {
          color = '#52c41a';
          border = '1px solid #52c41a';
      } else if (point.success_count === 0 && point.fail_count > 0) {
          color = '#f5222d';
          border = '1px solid #f5222d';
      } else if (point.success_count > 0 && point.fail_count > 0) {
          color = '#fa8c16';
          border = '1px solid #fa8c16';
      }
      return {
          width: '12px',
          height: '12px',
          backgroundColor: color,
          border: border,
          borderRadius: '2px',
          cursor: 'pointer'
      };
  }
  ```

- [ ] **Step 2: 配置中英文国际化语言包**

  修改中英文语言包文件（若无该词条，则添加）：
  - 简体中文 [zh-CN.json](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/zh-CN.json) / [zh-CN/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/zh-CN/pages.js) 中在 `model` 项下加上：
    `recent_status: '最近状态'`
  - 英文 [en-US.json](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/en-US.json) / [en-US/pages.js](file:///Users/chenzhiguo/Projects/tokenlive-admin/frontend/src/locales/en-US/pages.js) 中在 `model` 项下加上：
    `recent_status: 'Recent Status'`
