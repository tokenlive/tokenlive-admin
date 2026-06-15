# Design Spec: Fix Empty Endpoint Metadata/Headers, Validate RequestTypes Inherited from Model, Remove Obsolete RequestType, and Log Composite Discovery Failures

## Problem 1: Metadata and Headers Storing "null" String

When updating or creating an endpoint with empty/cleared headers or metadata in the admin frontend, the frontend sends `"headers": null` and/or `"metadata": null`.
In Go backend, the fields are modeled as `json.RawMessage`. Unmarshaling a JSON `null` value into `json.RawMessage` yields `[]byte("null")`.
When GORM updates or creates the record, GORM serializes this slice directly into the database, leading to a literal string `"null"` stored in the database.

### Proposed Solution for Problem 1

Intercept the data mapping inside `EndpointForm.FillTo` in `tokenlive-admin` before GORM persistence.
If the unmarshaled raw bytes for `Headers` or `Metadata` equals `"null"` or is empty, we force the value to `nil` before it is copied to the GORM struct `schema.Endpoint`.

---

## Problem 2: Hardcoded/Missing RequestTypes on Endpoints

RequestTypes were hardcoded inside `ai-gateway` to `chat_completion` and `embedding`. Endpoints must inherit apis from model configs and we should fail early with clear errors if apis are not configured.

### Proposed Solution for Problem 2 (Strict RequestTypes Verification)

1. **`tokenlive-admin` Sync (redis_sync.go)**:
   - Read `ep.Model.RequestTypes`. Parse the JSON string.
   - If empty, return a hard error in `SyncModelByCode` to prevent synchronization of misconfigured models.

2. **`ai-gateway` Config Loading (loader.go / types.go)**:
   - In `config.Validate`, assert that every model config has `len(m.RequestTypes) > 0`. If empty, fail to load config and block server boot.
   - Map `m.RequestTypes` in `Resolve` function.

3. **`ai-gateway` Static Registry (engine.go)**:
   - Change `registerEndpointsFromResolvedEndpoints` to return an `error`.
   - If any `ResolvedEndpoint` has zero apis, return a configuration error. Callers (like `NewGatewayEngine`) must catch and propagate this error to halt initialization.

4. **`ai-gateway` Dynamic Discovery (discovery_dynamic.go)**:
   - In `DynamicDiscovery.List`, if `len(de.RequestTypes) == 0`, immediately return an error instead of fallback.

---

## Problem 3: Redundant RequestType in ModelConfig

The `RequestType` field inside `ModelConfig` is obsolete and unused by core routing logic. It conflicts with `RequestTypes` and causes configuration complexity.

### Proposed Solution for Problem 3

Remove `RequestType` field entirely from `ModelConfig` and cleanup all mock configurations / assertions in unit tests.

---

## Problem 4: Obscured Errors in CompositeDiscovery Chain

When `CompositeDiscovery` iterates through discoveries in `List`, an error returned by one discovery (such as `DynamicDiscovery` failing apis validation) can be easily overridden by succeeding elements or fallbacks. Developers cannot see the real cause of discovery failures in logs.

### Proposed Solution for Problem 4

Inside `CompositeDiscovery.List`, retrieve `zapLogger` from `ctx` (if present) and output a `Warn` log with full details (model, provider/discovery type, and err) whenever any discovery element fails.

---

## Proposed Changes File by File

### Project: tokenlive-admin

#### [MODIFY] [endpoint.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/schema/endpoint.go)

```go
 if len(e.Headers) == 0 || string(e.Headers) == "null" {
  endpoint.Headers = nil
 } else {
  endpoint.Headers = e.Headers
 }

 if len(e.Metadata) == 0 || string(e.Metadata) == "null" {
  endpoint.Metadata = nil
 } else {
  endpoint.Metadata = e.Metadata
 }
```

#### [MODIFY] [redis_sync.go](file:///Users/chenzhiguo/Projects/tokenlive-admin/internal/mods/resource/biz/redis_sync.go)

Map `ep.Model.RequestTypes` in `SyncModelByCode`. Raise an error if resolved apis list is empty.

---

### Project: ai-gateway

#### [MODIFY] [types.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/types.go)

- Add `RequestTypes []string`mapstructure:"apis" json:"apis,omitempty"`` to `ResolvedEndpoint`.
- Remove `RequestType` from `ModelConfig`.

#### [MODIFY] [loader.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/loader.go)

- Assert `len(m.RequestTypes) > 0` in `config.Validate`.
- Assign `RequestTypes: m.RequestTypes` in `Resolve`.

#### [MODIFY] [discovery_dynamic.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_dynamic.go)

Add `RequestTypes []string` to `DynamicEndpoint`. In `List`, return an error if `len(de.RequestTypes) == 0`.

#### [MODIFY] [discovery_composite.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/core/discovery_composite.go)

- Retrieve `zapLogger` from `ctx` and print `Warn` log for failed discoveries inside `List`.

#### [MODIFY] [engine.go](file:///Users/chenzhiguo/Projects/ai-gateway/cmd/server/wire/engine.go)

- Change `registerEndpointsFromResolvedEndpoints` signature to return `error`.
- Return error in `NewGatewayEngine` if static service registration fails.
- Map apis in adapter.

#### [MODIFY] [loader_test.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/loader_test.go)

- Remove `RequestType` properties and assertions.

#### [MODIFY] [config_manager_test.go](file:///Users/chenzhiguo/Projects/ai-gateway/pkg/config/config_manager_test.go)

- Remove `RequestType` properties from mock configuration setups.

---

## Verification Plan

### Automated Tests

1. Unit tests: run `go test ./...` in both repositories.
