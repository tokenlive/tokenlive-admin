# Fix Endpoint Empty Fields and Inherit RequestTypes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Resolve the issue of `"null"` string database persistence for empty headers/metadata in `tokenlive-admin` AND enable dynamic `RequestTypes` inheritance from Model to Endpoint across `tokenlive-admin` and `ai-gateway`.

**Architecture:**

1. Map `"null"` raw bytes to `nil` in Go backend data mapping step for endpoints.
2. Extend `ResolvedEndpoint` struct to include `RequestTypes []string`.
3. Read Model apis from GORM database in admin panel, serialize them to the Redis payload, and parse them at the gateway end to configure the discovery endpoints dynamically.

**Tech Stack:** Go 1.21+, GORM, Redis, Viper, test-driven-development.

---

### Task 1: Admin - Sync RequestTypes to Redis

**Files:**

- Modify: `internal/mods/resource/biz/redis_sync.go`
- Test: `internal/mods/resource/biz/redis_sync_test.go` (if exists, or verify through test)

- [ ] **Step 1: Update ResolvedEndpoint Struct in admin**
  
  Add `RequestTypes []string`json:"apis,omitempty"`` to `ResolvedEndpoint` struct inside [/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/redis_sync.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/redis_sync.go):

  ```go
  type ResolvedEndpoint struct {
   RealModel        string            `json:"real_model"`
   ProviderName     string            `json:"provider_name"`
   ProviderProtocol string            `json:"provider_protocol"`
   APIKey           string            `json:"api_key"`
   URL              string            `json:"url"`
   Timeout          int64             `json:"timeout"`      // 毫秒
   MaxRetries       int               `json:"max_retries"`
   Priority         int               `json:"priority"`
   Weight           int               `json:"weight"`
   Headers          map[string]string `json:"headers,omitempty"`
   RequestTypes     []string          `json:"apis,omitempty"`
  }
  ```

- [ ] **Step 2: Map Model RequestTypes during Sync**
  
  In the `SyncModelByCode` function in `/Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/redis_sync.go`, parse `ep.Model.RequestTypes` (which is a JSON array string in database) and append it to `ResolvedEndpoint`:

  ```go
    var apis []string
    if ep.Model.RequestTypes != "" {
     _ = json.Unmarshal([]byte(ep.Model.RequestTypes), &apis)
    }
  ```

  And then assign it to the `ResolvedEndpoint`:

  ```go
    resolvedList = append(resolvedList, ResolvedEndpoint{
     RealModel:        realModel,
     ProviderName:     ep.Provider.Name,
     ProviderProtocol: ep.Provider.Protocol,
     APIKey:           apiKey,
     URL:              ep.URL,
     Timeout:          timeout,
     MaxRetries:       maxRetries,
     Priority:         ep.Priority,
     Weight:           ep.Weight,
     Headers:          headersMap,
     RequestTypes:     apis,
    })
  ```

- [ ] **Step 3: Run admin build to ensure code compilability**
  
  Run: `go build ./...` inside `tokenlive-admin`.
  Expected: Success.

---

### Task 2: Gateway - Update ResolvedEndpoint Struct and Resolver

**Files:**

- Modify: `pkg/config/types.go`
- Modify: `pkg/config/loader.go`
- Test: `pkg/config/loader_test.go` (if exists, or verify by test)

- [ ] **Step 1: Update ResolvedEndpoint Struct in gateway**
  
  Add `RequestTypes []string`mapstructure:"apis" json:"apis,omitempty"`` to `ResolvedEndpoint` struct inside [/Users/chenzhiguo/Projects/ai-gateway/pkg/config/types.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/types.go):

  ```go
  type ResolvedEndpoint struct {
   RealModel        string            `json:"real_model"`
   ProviderName     string            `json:"provider_name"`
   ProviderProtocol string            `json:"provider_protocol"`
   APIKey           string            `json:"api_key"`
   URL              string            `json:"url"`
   Timeout          int64             `json:"timeout"`      // 毫秒
   MaxRetries       int               `json:"max_retries"`
   Priority         int               `json:"priority"`
   Weight           int               `json:"weight"`
   Headers          map[string]string `json:"headers,omitempty"`
   RequestTypes     []string          `mapstructure:"apis" json:"apis,omitempty"`
  }
  ```

- [ ] **Step 2: Map Model Config RequestTypes in Resolve**
  
  Update `Resolve` function in [/Users/chenzhiguo/Projects/ai-gateway/pkg/config/loader.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/loader.go) to pass apis to `ResolvedEndpoint`:

  ```go
     re := ResolvedEndpoint{
      ProviderName:     ep.Provider,
      ProviderProtocol: p.Protocol,
      URL:              ep.URL,
      Priority:         ep.Priority,
      RequestTypes:     m.RequestTypes,
     }
  ```

- [ ] **Step 3: Run gateway build**
  
  Run: `go build ./...` inside `ai-gateway`.
  Expected: Success.

---

### Task 3: Gateway - Update Discovery and Adapter

**Files:**

- Modify: `pkg/core/discovery_dynamic.go`
- Modify: `cmd/server/wire/engine.go`

- [ ] **Step 1: Update DynamicEndpoint Struct**
  
  Add `RequestTypes []string` to `DynamicEndpoint` inside [/Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_dynamic.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_dynamic.go):

  ```go
  type DynamicEndpoint struct {
   ProviderName     string
   ProviderProtocol string
   URL              string
   APIKey           string
   RealModel        string
   Weight           int
   Headers          map[string]string
   RequestTypes     []string
  }
  ```

- [ ] **Step 2: Map apis in DynamicDiscovery.List**
  
  In the `List` method of `DynamicDiscovery` in [/Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_dynamic.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_dynamic.go), load apis dynamically:

  ```go
   result := make([]*Endpoint, 0, len(endpoints))
   for i, de := range endpoints {
    var apis []RequestType
    if len(de.RequestTypes) > 0 {
     for _, capStr := range de.RequestTypes {
      apis = append(apis, RequestType(capStr))
     }
    } else {
     // fallback
     apis = []RequestType{
      RequestTypeChatCompletion,
      RequestTypeEmbedding,
     }
    }
  
    ep := &Endpoint{
     ID:               fmt.Sprintf("%s-%s-%d", de.ProviderName, model, i),
     URL:              de.URL,
     Provider:         de.ProviderName,
     ProviderProtocol: de.ProviderProtocol,
     APIKey:           de.APIKey,
     Model:            model,
     UpstreamModel:    de.RealModel,
     Weight:           de.Weight,
     Headers:          de.Headers,
     Healthy:          true,
     RequestTypes:     apis,
    }
    result = append(result, ep)
   }
  ```

- [ ] **Step 3: Update Wire Engine Adapter and Static Register**
  
  In [/Users/chenzhiguo/Projects/ai-gateway/cmd/server/wire/engine.go](file:///Users/chenzhiguo/Projects/ai-gateway/cmd/server/wire/engine.go):
  
  Update `dynamicEndpointAdapter.GetEndpoints`:

  ```go
  func (a *dynamicEndpointAdapter) GetEndpoints(ctx context.Context, model string) []core.DynamicEndpoint {
   eps := a.mgr.GetEndpoints(ctx, model)
   res := make([]core.DynamicEndpoint, len(eps))
   for i, ep := range eps {
    res[i] = core.DynamicEndpoint{
     ProviderName:     ep.ProviderName,
     ProviderProtocol: ep.ProviderProtocol,
     URL:              ep.URL,
     APIKey:           ep.APIKey,
     RealModel:        ep.RealModel,
     Weight:           ep.Weight,
     Headers:          ep.Headers,
     RequestTypes:     ep.RequestTypes,
    }
   }
   return res
  }
  ```
  
  Update `registerEndpointsFromResolvedEndpoints`:

  ```go
  func registerEndpointsFromResolvedEndpoints(sd *core.StaticDiscovery, resolved map[string][]config.ResolvedEndpoint) {
   for modelName, eps := range resolved {
    endpoints := make([]*core.Endpoint, 0, len(eps))
    for i, re := range eps {
     var apis []core.RequestType
     if len(re.RequestTypes) > 0 {
      for _, capStr := range re.RequestTypes {
       apis = append(apis, core.RequestType(capStr))
      }
     } else {
      apis = []core.RequestType{
       core.RequestTypeChatCompletion,
       core.RequestTypeEmbedding,
      }
     }
     endpoint := &core.Endpoint{
      ID:               fmt.Sprintf("%s-%s-%d", re.ProviderName, modelName, i),
      URL:              re.URL,
      Provider:         re.ProviderName,
      ProviderProtocol: re.ProviderProtocol,
      APIKey:           re.APIKey,
      Model:            modelName,
      UpstreamModel:    re.RealModel,
      Weight:           re.Weight,
      Headers:          re.Headers,
      Healthy:          true,
      RequestTypes:     apis,
     }
     endpoints = append(endpoints, endpoint)
    }
    sd.RegisterService(modelName, endpoints)
   }
  }
  ```

- [ ] **Step 4: Run wire generation and compile check**
  
  Run: `wire ./cmd/server/wire/` and `go test ./...` inside `ai-gateway`.
  Expected: Success.
