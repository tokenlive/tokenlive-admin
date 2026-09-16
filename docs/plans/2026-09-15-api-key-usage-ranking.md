# API Key 用量排行 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在首页模型排行下方增加仅 Root 可见、覆盖全部客户端 Key、每 30 秒刷新的只读用量榜单。

**Architecture:** ClickHouse 对请求日志去重并按候选身份预聚合；Admin 在服务端验证身份、合并候选、计算全量分母，再排序截取 Top N。名称和状态在截取后补充，失败不抹掉用量。独立前端组件、请求客户端和刷新控制器隔离本榜单的错误及刷新，不改变现有首页 WebSocket。

**Tech Stack:** 现有 Go 1.27.1、Gin、GORM、Wire、Vue 3、Pinia、Ant Design Vue、Node `node:test`；新增 ClickHouse Go v2 客户端，采用相邻 Gateway 当前锁定的 `v2.46.0`；复用已有 `shopspring/decimal v1.4.0` 与 `golang.org/x/sync v0.20.0`。

**Spec:** `docs/specs/2026-09-15-api-key-usage-ranking-design.md`，书面设计已于 2026-09-15 经用户确认。执行者必须同时阅读该设计、根目录 `AGENTS.md`、`CONTEXT.md` 和 `docs/adr/0001-separate-admin-portal-user-systems.md`。

## Global Constraints

- 仅 Root 超级管理员可见；后端独立校验。
- 覆盖 Admin 用户 Key、租户 Key、Portal 工作空间 Key，不包含供应商上游密钥。
- 输入 Token＋输出 Token 降序；请求数、Token 总量、费用，均由服务端全量排序后截取。
- Top 10，可选 Top 20、Top 50。
- 所选时段内存在调用记录，不以当前启用状态过滤。
- 所选时段全部 Key 的 Token 用量，包括无法归属的历史用量。
- 悬停显示输入、输出、缓存命中、缓存创建；缓存不再次叠加进总量。
- 不支持点击，不呈现链接、指针或可点击行样式。
- 已禁用、已删除的 Key 仍参与历史统计，并展示可确认的当前状态。
- 每 30 秒一次；筛选变化立即刷新；页面不可见或离开首页时暂停。
- 区分未启用、查询失败、查询成功但无调用，不将错误转换成零用量。
- 不修改 Gateway 的计费、鉴权或 Key 删除行为，不默认修改 Portal 接口。
- Admin 不自动建库、建表、迁移、写入或删除 ClickHouse 数据。
- API、日志和前端不得收到完整 Key、完整 `api_key_hash` 或连接凭据。
- 每次提交前对本任务新增或修改的 Go 文件运行 gofmt；前端 `.vue`、`.js` 文件按项目配置运行 Prettier。
- 用户已于 2026-09-16 授权在当前会话、当前目录和 `main` 原地实现，不使用子代理或独立 worktree。以下保留原实施步骤，实际结果见文末执行记录。

---

## 执行起点与文件边界

工作目录为 `/Users/chenzhiguo/Projects/tokenlive-admin`。开始实现前按执行技能检查隔离工作区与 Git 状态；不要在规划阶段创建工作区。

规划时已有两个不属于本功能的未跟踪文件：

- `frontend/tests/model-ranking-navigation.test.mjs`
- `frontend/tests/provider-endpoint-health.test.mjs`

保留原样，不能把它们计入本功能提交。代码定位先使用 CodeGraph，不重新索引仓库。所有提交只添加各任务明确列出的文件，禁止使用 `git add .`。

| 文件 | 职责 | 任务 |
| --- | --- | --- |
| `internal/config/clickhouse.go` | 可选只读连接配置与默认值 | 1 |
| `internal/config/config.go` | Storage 配置挂载、新增密码字段的安全打印 | 1 |
| `internal/mods/dashboard/schema/api_key_usage.go` | 查询参数、时间窗口、内部候选与公开响应契约 | 1 |
| `internal/mods/dashboard/dal/api_key_usage.go` | 单次 ClickHouse 候选预聚合读取与关闭连接 | 1 |
| `internal/mods/dashboard/biz/api_key_identity.go` | 验证 ID、Hash 别名与来源 | 2 |
| `internal/mods/dashboard/biz/api_key_metadata.go` | Admin／Portal 名称和当前状态补充 | 2 |
| `internal/mods/dashboard/biz/api_key_usage.go` | 最终归并、排序、分母、未知用量与响应组装 | 3 |
| `internal/mods/dashboard/biz/api_key_usage_cache.go` | 30 秒缓存、并发请求合并和取消隔离 | 3 |
| `internal/mods/dashboard/api/api_key_usage.api.go` | Root API 和 HTTP 状态映射 | 4 |
| `internal/mods/dashboard/providers.go` | 本功能依赖构造及 Reader 清理函数 | 4 |
| `internal/mods/dashboard/main.go`、`wire.go` | 独立 API 注册与 Wire 接线 | 4 |
| `internal/wirex/wire_gen.go` | 用固定版本 Wire 重新生成，不手写 | 4 |
| `internal/mods/rbac/schema/user.go`、`biz/login.biz.go` | 当前用户响应的非持久化 `is_root` 标志 | 4 |
| `frontend/src/utils/api-key-usage-request.js` | 可注入、无全局提示的读取与一次 401 恢复 | 5 |
| `frontend/src/apis/modules/api-key-usage.js`、`dashboard.js` | Axios／用户会话绑定与 API 导出 | 5 |
| `frontend/src/utils/api-key-usage-state.js` | 可独立测试的刷新与过期结果状态机 | 6 |
| `frontend/src/composables/useAPIKeyUsage.js` | Vue 生命周期、可见性与状态机桥接 | 6 |
| `frontend/src/store/modules/user.js` | 非持久化的用户身份已验证状态 | 6 |
| `frontend/src/views/home/components/APIKeyUsageRanking.vue` | 七列榜单、悬停、筛选与异常展示 | 7 |
| `frontend/src/views/home/index.vue` | 模型排行之后挂载组件、Root 门控 | 7 |
| 两种语言的 `frontend/src/locales/lang/*/pages.js` | 本功能所有文案 | 7 |
| `configs/dev/server.toml`、`docs/CONFIGURATION.md` | 默认关闭配置示例与只读接入说明 | 1、8 |

每个新 Go 源文件添加同目录 `_test.go`。前端新建下述任务指定的 `.test.mjs`，不得只用源码正则测试替代状态机和请求行为测试。

## 固定接口契约

以下类型名、方法名和字段名是任务之间的契约，不允许各任务自行改名。必要的实现内部辅助函数放在相应文件中，不导出到其他任务。

```go
// package schema
type Query struct {
    TimeRange string
    SortBy    string
    Limit     int
}
func ParseQuery(values url.Values) (Query, error)
func ResolveWindow(query Query, end time.Time) Window

type Window struct {
    Start    time.Time `json:"start"`
    End      time.Time `json:"end"`
    Timezone string    `json:"timezone"`
}
type KeyRef struct {
    Hash, KeyID, WorkspaceID, UserID, TenantID string
}
type Totals struct {
    Requests, Success, Input, Output, Cached, CacheCreation uint64
    Cost decimal.Decimal
}
func (t Totals) Tokens() uint64
func (t *Totals) Add(other Totals)

type Candidate struct {
    Ref     KeyRef
    Display string
    Totals  Totals
}
type Resolution struct {
    Keys     map[KeyRef]string
    Warnings []string
}
type Group struct {
    Canonical string
    Ref       KeyRef
    Display   string
    Totals    Totals
}
type Owner struct {
    Kind string `json:"kind"`
    ID   string `json:"id"`
    Name string `json:"name"`
}
type Metadata struct {
    KeyName, Display, Source, KeyStatus, MetadataStatus string
    Owner Owner
}
type Metrics struct {
    RequestCount        uint64   `json:"request_count"`
    SuccessCount        uint64   `json:"success_count"`
    SuccessRate         float64  `json:"success_rate"`
    InputTokens         uint64   `json:"input_tokens"`
    OutputTokens        uint64   `json:"output_tokens"`
    CachedTokens        uint64   `json:"cached_tokens"`
    CacheCreationTokens uint64   `json:"cache_creation_tokens"`
    TotalTokens         uint64   `json:"total_tokens"`
    TotalCost           string   `json:"total_cost"`
    TokenShare          *float64 `json:"token_share"`
}
type Item struct {
    RowID          string `json:"row_id"`
    KeyName        string `json:"key_name"`
    KeyDisplay     string `json:"key_display"`
    Source         string `json:"source"`
    Owner          Owner  `json:"owner"`
    KeyStatus      string `json:"key_status"`
    MetadataStatus string `json:"metadata_status"`
    Metrics
}
type Summary struct {
    KeyCount uint64 `json:"key_count"`
    Metrics
}
type Response struct {
    State        string     `json:"state"`
    DataSource   string     `json:"data_source"`
    Window       *Window    `json:"window"`
    GeneratedAt  *time.Time `json:"generated_at"`
    Summary      *Summary   `json:"summary"`
    Items        []Item     `json:"items"`
    Unattributed *Metrics   `json:"unattributed"`
    Warnings     []string   `json:"warnings"`
}

// package biz
type Reader interface {
    Enabled() bool
    Read(context.Context, schema.Window) ([]schema.Candidate, error)
}
type Resolver interface {
    Resolve(context.Context, []schema.Candidate) schema.Resolution
    Describe(context.Context, []schema.Group) map[string]schema.Metadata
}
type Aggregation struct {
    Groups       []schema.Group
    All          schema.Totals
    Unattributed schema.Totals
    KeyCount     uint64
}
func Aggregate([]schema.Candidate, schema.Resolution, schema.Query) Aggregation
func NewAPIKeyUsageService(Reader, Resolver, func() time.Time) *APIKeyUsageService
func (*APIKeyUsageService) Query(context.Context, schema.Query) (*schema.Response, error)

// package api
type UsageQuerier interface {
    Query(context.Context, schema.Query) (*schema.Response, error)
}
type APIKeyUsage struct {
    Service UsageQuerier
}
func (*APIKeyUsage) Query(c *gin.Context)
```

内部 `KeyRef`、`Candidate`、`Group` 和 `Metadata` 不得直接作为 HTTP 响应；仅转换为 `Response`。`Metadata.Source` 使用 `admin_user`、`tenant`、`portal_workspace`、`unknown`。状态使用 `enabled`、`disabled`、`revoked`、`expired`、`deleted`、`unknown`；元数据可用状态使用 `ready`、`partial`、`unavailable`。

---

### Task 1：可选的 ClickHouse 只读候选查询

**Files**

- Create: `internal/config/clickhouse.go`、`internal/config/clickhouse_test.go`
- Modify: `internal/config/config.go`、`configs/dev/server.toml`、`go.mod`、`go.sum`
- Create: `internal/mods/dashboard/schema/api_key_usage.go`、`internal/mods/dashboard/schema/api_key_usage_test.go`
- Create: `internal/mods/dashboard/dal/api_key_usage.go`、`internal/mods/dashboard/dal/api_key_usage_test.go`

**Interfaces**

- Consumes: 现有 `config.C.Storage`；Gateway 已有 `access_logs` 表结构。
- Produces: 上述 `schema` 契约；`dal.NewClickHouseReader(config.ClickHouseConfig) (*ClickHouseReader, func())`；Reader 的 `Enabled()`、`Read()` 方法。

- [x] **Step 1：先添加参数、窗口、SQL 及关闭状态的失败测试。**

```go
// package schema
func TestParseQueryRejectsInvalidValues(t *testing.T) {
    for _, values := range []url.Values{
        {"sort_by": {"key_hash"}},
        {"time_range": {"30d"}},
        {"limit": {"100"}},
        {"limit": {""}},
    } {
        _, err := ParseQuery(values)
        require.Error(t, err)
    }
    query, err := ParseQuery(url.Values{})
    require.NoError(t, err)
    require.Equal(t, Query{"today", "tokens", 10}, query)
}

func TestResolveWindowTodayPreservesOffset(t *testing.T) {
    zone := time.FixedZone("UTC+8", 8*3600)
    end := time.Date(2026, 9, 15, 12, 0, 0, 0, zone)
    window := ResolveWindow(Query{"today", "tokens", 10}, end)
    require.Equal(t, time.Date(2026, 9, 15, 0, 0, 0, 0, zone), window.Start)
    require.Equal(t, end, window.End)
}

// package dal
func TestDisabledReaderDoesNotOpenConnection(t *testing.T) {
    reader, closeReader := NewClickHouseReader(config.ClickHouseConfig{})
    defer closeReader()
    require.False(t, reader.Enabled())
    rows, err := reader.Read(context.Background(), schema.Window{})
    require.NoError(t, err)
    require.Empty(t, rows)
}

func TestCandidateSQLDoesNotTruncateBeforeIdentityResolution(t *testing.T) {
    require.Contains(t, candidateSQL, "FROM access_logs FINAL")
    require.Contains(t, candidateSQL, "time >= ? AND time < ?")
    require.NotContains(t, strings.ToUpper(candidateSQL), "LIMIT")
    require.NotContains(t, candidateSQL, "enabled")
}
```

补充配置打印测试：给新配置的密码设置测试字符串，断言 `Config.String()` 不包含该字符串，且原配置密码未被修改。

- [x] **Step 2：运行并确认因新增类型／函数缺失而失败。**

Run: `go test ./internal/config ./internal/mods/dashboard/schema ./internal/mods/dashboard/dal -count=1`

Expected: 新包或新增符号尚不存在；记录实际失败，不把已有环境故障当作 RED。

- [x] **Step 3：实现配置与预聚合查询。**

执行依赖命令时只引入已选定版本，不使用 `@latest`：

```bash
go get github.com/ClickHouse/clickhouse-go/v2@v2.46.0
go mod tidy
```

```go
// package config
type ClickHouseConfig struct {
    Enabled             bool
    Addr                []string
    Database            string
    Username            string
    Password            string
    TLS                 bool
    DialTimeoutSeconds  int `default:"3"`
    QueryTimeoutSeconds int `default:"5"`
}
```

将 `ClickHouse ClickHouseConfig` 添加到 `Storage`。`Config.String()` 序列化配置的值副本，副本中的 `Storage.ClickHouse.Password` 替换为 `"[redacted]"`，不修改配置原值，不扩展到无关配置打印重构。

在本任务实现 `Totals.Tokens` 与 `Totals.Add`：Tokens 返回 Input＋Output；Add 对六个整数计数逐项累加，对 Cost 调用 `decimal.Decimal.Add`。不把缓存计数并入 Input 或 Output，也不经过浮点金额转换。

```sql
SELECT
    api_key_hash, api_key_id, workspace_id, user_id, tenant_id,
    argMax(api_key, time),
    count(),
    countIf(status_code >= 200 AND status_code < 400),
    sum(input_tokens), sum(output_tokens),
    sum(cached_tokens), sum(cache_creation_tokens),
    toString(sum(cost))
FROM access_logs FINAL
WHERE time >= ? AND time < ?
GROUP BY api_key_hash, api_key_id, workspace_id, user_id, tenant_id
```

将上述 SQL 原样定义为 `candidateSQL`。只按候选身份预聚合，不按显示名称、不取 Top N、不连接管理库。一次读取产生所有候选，后续所有分母与排序来自这份结果。

```go
opts := &clickhouse.Options{
    Addr: cfg.Addr,
    Auth: clickhouse.Auth{
        Database: cfg.Database,
        Username: cfg.Username,
        Password: cfg.Password,
    },
    DialTimeout: time.Duration(cfg.DialTimeoutSeconds) * time.Second,
}
if cfg.TLS {
    opts.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
}
```

构造阶段不 Ping。禁用时不调用 `clickhouse.Open`；启用但缺少地址、数据库或发生 Open 错误时，将错误留在 Reader 内，`Read` 返回错误，不能导致整个 Admin 构建依赖失败。`Read` 使用 5 秒上下文、绑定 Window 的 Start／End、逐行 Scan，金额用 `decimal.NewFromString`；Scan、金额解析和 Rows.Err 任一失败均返回错误，不返回部分结果。连接清理函数可安全调用一次。

配置示例：

```toml
[Storage.ClickHouse]
Enabled = false
Addr = ["${ADMIN_CLICKHOUSE_ADDR:127.0.0.1:9000}"]
Database = "${ADMIN_CLICKHOUSE_DATABASE:default}"
Username = "${ADMIN_CLICKHOUSE_USERNAME:}"
Password = "${ADMIN_CLICKHOUSE_PASSWORD:}"
TLS = false
DialTimeoutSeconds = 3
QueryTimeoutSeconds = 5
```

执行 `ParseQuery` 时区分参数缺省与显式空字符串；校验三个白名单。`ResolveWindow` 的今日边界由传入 `end.Location()` 计算，其他窗口用 1h／6h／24h／7×24h 回溯。

- [x] **Step 4：重复 Step 2 并验证全部 PASS。**

额外覆盖 `Read` 连接错误、Scan 错误、金额字符串解析错误、取消和超时；通过最小查询接口注入假 Rows，接口只需 `Query`、`Next`、`Scan`、`Err`、`Close`，不要模拟整个 ClickHouse 客户端。

- [x] **Step 5：提交本任务文件。**

```bash
git add -- internal/config/clickhouse.go internal/config/clickhouse_test.go internal/config/config.go configs/dev/server.toml go.mod go.sum internal/mods/dashboard/schema/api_key_usage.go internal/mods/dashboard/schema/api_key_usage_test.go internal/mods/dashboard/dal/api_key_usage.go internal/mods/dashboard/dal/api_key_usage_test.go
git commit -m "feat: add optional read-only API key usage source"
```

### Task 2：可靠身份解析与安全元数据补充

**Files**

- Create: `internal/mods/dashboard/biz/api_key_identity.go`、`internal/mods/dashboard/biz/api_key_identity_test.go`
- Create: `internal/mods/dashboard/biz/api_key_metadata.go`、`internal/mods/dashboard/biz/api_key_metadata_test.go`

**Interfaces**

- Consumes: `schema.Candidate`／`KeyRef`，Admin `user_api_key`、`tenant`、`user` 表，现有 `opsbiz.PortalUser.ListWorkspaceAPIKeys`。
- Produces: `biz.NewIdentityResolver(*gorm.DB, PortalKeys, string, func() time.Time) *IdentityResolver`，实现固定契约中的 `Resolver`。
- `PortalKeys` 精确签名：`ListWorkspaceAPIKeys(context.Context, string) ([]opsbiz.PortalWorkspaceAPIKey, error)`。
- 纯逻辑函数：`CanonicalKeys([]schema.Candidate, map[schema.KeyRef]string) schema.Resolution`。第二个参数只能包含已被管理库／Portal 验证的无 Hash ID 到规范标识的映射。

- [x] **Step 1：添加 Hash 优先、冲突、缺失及已删除元数据的失败测试。**

```go
func TestCanonicalKeysNeverUsesMaskedDisplay(t *testing.T) {
    a := schema.KeyRef{Hash: "hash-a"}
    b := schema.KeyRef{Hash: "hash-b"}
    unknown := schema.KeyRef{}
    rows := []schema.Candidate{
        {Ref: a, Display: "sk-***same"},
        {Ref: b, Display: "sk-***same"},
        {Ref: unknown, Display: "sk-***same"},
    }
    result := CanonicalKeys(rows, nil)
    require.Equal(t, "h:hash-a", result.Keys[a])
    require.Equal(t, "h:hash-b", result.Keys[b])
    require.Empty(t, result.Keys[unknown])
}
```

元数据测试使用临时 SQLite 文件与 `httptest.Server`，创建一个停用 Key、一条带删除标记的 Key、一条日志有 Hash 但管理表已无记录的 Key；要求三者分别为 `disabled`、`deleted`、`unknown`，且后一项为 `metadata_status=unavailable`。对插入的测试数据用 `t.Cleanup` 按明确 ID 删除，再关闭临时数据库。

- [x] **Step 2：运行新测试并确认 RED。**

Run: `go test ./internal/mods/dashboard/biz -run 'TestCanonical|TestIdentity|TestMetadata' -count=1`

- [x] **Step 3：实现规范标识及有界元数据解析。**

```go
func CanonicalKeys(rows []schema.Candidate, verified map[schema.KeyRef]string) schema.Resolution {
    result := schema.Resolution{Keys: make(map[schema.KeyRef]string)}
    for _, row := range rows {
        if row.Ref.Hash != "" {
            result.Keys[row.Ref] = "h:" + row.Ref.Hash
        } else if key := verified[row.Ref]; key != "" {
            result.Keys[row.Ref] = key
        }
    }
    return result
}
```

`Resolve` 先验证无 Hash 的 ID，再调用纯函数。ID 验证与别名映射规则：

| 候选 | 验证和归并 |
| --- | --- |
| 非空 Hash | 始终独立可识别，不要求当前元数据存在 |
| Admin 用户 ID＋Key ID | Key 记录 ID 与用户归属均匹配；用现有 `gatewaykeys.HashAPIKey` 计算当前凭证 Hash |
| Portal Workspace ID＋Key ID | 必须出现在该工作空间返回的 Key 列表；同一候选集合中只有一个关联 Hash 时归到该 Hash |
| 已验证 Portal ID，无关联 Hash | 使用长度安全编码的来源、工作空间、Key ID 元组作为规范 ID |
| 同一 Portal ID 关联多个 Hash | 保持各 Hash 独立，无 Hash 的该 ID 记录转入未归属，不任选一个 |
| 没有可验证标识 | 不设置 Keys 项，由聚合层计入未归属 |

元组编码使用 JSON 字符串数组，不用容易碰撞的裸字符串拼接。Admin 表读取通过模型 `TableName()` 处理表前缀，不调用过滤了历史记录的列表 API，不给这些读取添加 `deleted='0'`。Hash 对应多条管理记录且归属有冲突时，保留用量但显示部分信息，不任取第一条记录。

`Describe` 只处理 Top N 分组。Portal 按工作空间去重调用，最多并发 4 个，共用 2 秒预算；缓存最多 60 秒，错误与空查找不永久缓存。Admin 名称、租户名称和 Root 名称从本地记录／当前配置解析；Portal 工作空间名称不可得时显示工作空间 ID，不新增 Portal 接口。

当前状态优先级为：有删除证据 → revoked → 已过期 → disabled → enabled；未获得可靠元数据则 unknown。任何“查不到”都不能自动变成 deleted。

不要修改现有 Key 删除方法来实现历史统计；记录已经物理消失时，仍通过 Hash 保留用量，并降级名称和状态。不要在日志中打印候选、完整 Hash 或数据库 Key。

- [x] **Step 4：重复 Step 2 并扩充通过矩阵。**

必须包含 Portal 未配置／超时、不同来源相同 ID、同工作空间重复验证合并、租户换 Key、数据库表前缀、错误后缓存可恢复、并发不超过 4、无 Portal 的企业内部部署。

- [x] **Step 5：提交本任务四个文件。**

```bash
git add -- internal/mods/dashboard/biz/api_key_identity.go internal/mods/dashboard/biz/api_key_identity_test.go internal/mods/dashboard/biz/api_key_metadata.go internal/mods/dashboard/biz/api_key_metadata_test.go
git commit -m "feat: resolve usage key identity and historical metadata"
```

### Task 3：全量归并、排序、分母及隔离缓存

**Files**

- Create: `internal/mods/dashboard/biz/api_key_usage.go`、`internal/mods/dashboard/biz/api_key_usage_test.go`
- Create: `internal/mods/dashboard/biz/api_key_usage_cache.go`、`internal/mods/dashboard/biz/api_key_usage_cache_test.go`

**Interfaces**

- Consumes: Task 1 Reader、Task 2 Resolver、固定契约中的 Query／Window。
- Produces: `Aggregate`、`APIKeyUsageService.Query`；给 Task 4 返回公开 `Response`。

- [x] **Step 1：添加“先合并再 Top N”和“分母包含未知”测试。**

```go
func TestAggregateIncludesUnattributedInDenominator(t *testing.T) {
    known := schema.KeyRef{Hash: "a"}
    rows := []schema.Candidate{
        {Ref: known, Totals: schema.Totals{Requests: 1, Input: 10, Output: 20}},
        {Ref: schema.KeyRef{}, Totals: schema.Totals{Requests: 1, Input: 70}},
    }
    resolution := schema.Resolution{Keys: map[schema.KeyRef]string{known: "h:a"}}
    result := Aggregate(rows, resolution, schema.Query{"today", "tokens", 10})
    require.Len(t, result.Groups, 1)
    require.Equal(t, uint64(100), result.All.Tokens())
    require.Equal(t, uint64(70), result.Unattributed.Tokens())
    require.Equal(t, uint64(1), result.KeyCount)
}

func TestAggregateCountsCachedTokensOnlyAsDetail(t *testing.T) {
    ref := schema.KeyRef{Hash: "a"}
    rows := []schema.Candidate{{
        Ref: ref,
        Totals: schema.Totals{Requests: 1, Input: 100, Output: 20, Cached: 80, CacheCreation: 10},
    }}
    resolution := schema.Resolution{Keys: map[schema.KeyRef]string{ref: "h:a"}}
    result := Aggregate(rows, resolution, schema.Query{"today", "tokens", 10})
    require.Equal(t, uint64(120), result.All.Tokens())
    require.Equal(t, uint64(80), result.Groups[0].Totals.Cached)
}
```

再构造 51 个 Key，其中同一规范 Key 拆成两个候选，各自不足 Top 10、相加后进入 Top 10；验证它实际入榜。缓存测试使用可推进的 `now` 函数，不使用真实 sleep。

- [x] **Step 2：运行并确认 RED。**

Run: `go test ./internal/mods/dashboard/biz -run 'TestAggregate|TestUsageService|TestUsageCache' -count=1`

- [x] **Step 3：实现累加、排序、公开响应转换。**

```go
func (t *Totals) Add(other Totals) {
    t.Requests += other.Requests
    t.Success += other.Success
    t.Input += other.Input
    t.Output += other.Output
    t.Cached += other.Cached
    t.CacheCreation += other.CacheCreation
    t.Cost = t.Cost.Add(other.Cost)
}
func (t Totals) Tokens() uint64 { return t.Input + t.Output }
```

这两个 schema 方法在 Task 1 建立，在本任务通过聚合测试验证完整语义。`Aggregate` 对每条候选先累加 `All`，无规范 ID 时累加 `Unattributed`，否则按规范 ID 累加到一个 Group。合并完成后计算 KeyCount，再按以下比较器排序并截取 Query.Limit：

```go
sort.Slice(groups, func(i, j int) bool {
    left, right := groups[i], groups[j]
    switch query.SortBy {
    case "cost":
        if cmp := left.Totals.Cost.Cmp(right.Totals.Cost); cmp != 0 {
            return cmp > 0
        }
    case "request_count":
        if left.Totals.Requests != right.Totals.Requests {
            return left.Totals.Requests > right.Totals.Requests
        }
    default:
        if left.Totals.Tokens() != right.Totals.Tokens() {
            return left.Totals.Tokens() > right.Totals.Tokens()
        }
    }
    return left.Canonical < right.Canonical
})
```

服务执行顺序固定为：禁用检测 → 窗口 → Read 一次 → Resolve → Aggregate → Describe Top N → 转换 Response。`RowID` 使用独立前缀域的 SHA-256 摘要，如 `sha256("api-key-usage-row-v1:"+canonical)` 的完整十六进制值，不原样暴露规范标识。KeyDisplay 再经服务端掩码函数处理，不能信任历史字段一定已经脱敏。

金额始终 `decimal.Decimal` 累加与排序，返回 `.String()`；成功率和占比最后计算，分母为零时 TokenShare 为 nil。`Unattributed` 使用同一 All 分母。`state=disabled` 的 Window／Summary／GeneratedAt／Unattributed 均为 nil，Items／Warnings 是空数组。

- [x] **Step 4：实现并测试缓存与取消。**

缓存逻辑放入独立文件：

```go
type usageCacheEntry struct {
    Response *schema.Response
    Expires  time.Time
}
```

按 `time_range/sort_by/limit` 缓存，缓存归属于单个数据源绑定的 Service 实例。配置重建实例后不共享旧缓存。TTL 最多 30 秒；`today` 的缓存到本地午夜必须失效。返回深拷贝或不可变响应，缓存命中不能更新 GeneratedAt／Window。

同参数使用已有 `singleflight.Group.DoChan` 合并。共享查询采用最多 9 秒的工作上下文，单个等待者取消只结束其等待，不取消其他等待者；无无限期后台任务。Read 完成后创建一个 2 秒元数据子上下文，Resolve 与 Describe 共用它，不能分别重置预算。数据读取错误不缓存为零，不缓存为 disabled。元数据降级允许缓存，但保留 warnings。

测试：两个并发请求只 Read 一次；不同筛选不共用结果；过期与午夜重查；一个等待者取消另一个仍成功；服务关闭／查询超时无资源泄漏；修改返回对象不污染缓存；错误恢复后重新查询。

- [x] **Step 5：验证并提交。**

Run: `go test -race ./internal/mods/dashboard/schema ./internal/mods/dashboard/biz -count=1`

Expected: 全部 PASS，包含完整聚合和缓存矩阵。

```bash
git add -- internal/mods/dashboard/biz/api_key_usage.go internal/mods/dashboard/biz/api_key_usage_test.go internal/mods/dashboard/biz/api_key_usage_cache.go internal/mods/dashboard/biz/api_key_usage_cache_test.go
git commit -m "feat: aggregate API key usage with complete totals"
```

### Task 4：Root API、明确身份标志与 Wire 接线

**Files**

- Create: `internal/mods/dashboard/api/api_key_usage.api.go`、`internal/mods/dashboard/api/api_key_usage.api_test.go`
- Create: `internal/mods/dashboard/providers.go`、`internal/mods/dashboard/providers_test.go`
- Modify: `internal/mods/dashboard/main.go`、`main_test.go`、`wire.go`、`internal/wirex/wire_gen.go`
- Modify: `internal/mods/rbac/schema/user.go`、`internal/mods/rbac/biz/login.biz.go`
- Create: `internal/mods/rbac/biz/login_root_identity_test.go`

**Interfaces**

- Consumes: `biz.APIKeyUsageService`、`util.FromIsRootUser`。
- Produces: `GET /api/v1/dashboard/api-key-ranking`；当前用户响应 `is_root: boolean`。
- Provider 签名：`ProvideUsageReader() (*dal.ClickHouseReader, func())`、`ProvideUsageResolver(*gorm.DB) *biz.IdentityResolver`、`ProvideUsageService(*dal.ClickHouseReader, *biz.IdentityResolver) *biz.APIKeyUsageService`、`ProvideUsageAPI(*biz.APIKeyUsageService) *api.APIKeyUsage`。

- [x] **Step 1：添加先鉴权后查询的失败测试。**

```go
type usageQueryFunc func(context.Context, schema.Query) (*schema.Response, error)
func (f usageQueryFunc) Query(ctx context.Context, q schema.Query) (*schema.Response, error) {
    return f(ctx, q)
}

func TestUsageAPINonRootNeverQueries(t *testing.T) {
    calls := 0
    handler := &APIKeyUsage{Service: usageQueryFunc(func(context.Context, schema.Query) (*schema.Response, error) {
        calls++
        return nil, nil
    })}
    recorder := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(recorder)
    c.Request = httptest.NewRequest("GET", "/api/v1/dashboard/api-key-ranking", nil)
    handler.Query(c)
    require.Equal(t, http.StatusForbidden, recorder.Code)
    require.Zero(t, calls)
}
```

另测 Root 使用 `util.NewIsRootUser` 的上下文获得响应；非法参数 400；Reader 错误 503；disabled 200；缓存命中仍不能绕过 Root。通过 `json.Marshal` 断言响应中没有测试完整 Key 和 Hash。

- [x] **Step 2：运行并确认 RED。**

Run: `go test ./internal/mods/dashboard/... ./internal/mods/rbac/biz -run 'TestUsageAPI|TestUsageProvider|TestRegisterV1RoutersIncludesAPIKey|TestCurrentUserRoot' -count=1`

- [x] **Step 3：实现 API 和当前用户身份标志。**

```go
func (a *APIKeyUsage) Query(c *gin.Context) {
    ctx := c.Request.Context()
    if !util.FromIsRootUser(ctx) {
        util.ResError(c, errors.Forbidden("api_key_usage_forbidden", "Root access required"))
        return
    }
    query, err := schema.ParseQuery(c.Request.URL.Query())
    if err != nil {
        util.ResError(c, errors.BadRequest("invalid_api_key_usage_query", "Invalid ranking query"))
        return
    }
    result, err := a.Service.Query(ctx, query)
    if err != nil {
        util.ResError(c, errors.New("api_key_usage_unavailable", "API key usage is temporarily unavailable", 503))
        return
    }
    util.ResSuccess(c, result)
}
```

详细错误在安全的服务日志中记录错误类别，不拼接 SQL 参数、连接凭据或候选对象。

在 `schema.User` 增加 `IsRoot bool`，标签为 `json:"is_root" gorm:"-"`。只在 `Login.GetUserInfo` 的 Root 分支设为 true；普通用户明确 false。不能在 UserForm 增加可写入权限标志，不新增数据库字段，也不依赖用户名为 admin 或 ID 字面值为 root。

- [x] **Step 4：接线并重新生成依赖。**

在模块 Dashboard 增加独立 `APIKeyUsageAPI *api.APIKeyUsage` 字段，注册：

```go
g.GET("api-key-ranking", a.APIKeyUsageAPI.Query)
```

给现有路由测试构造器补上空 `APIKeyUsage`，避免注册时空指针。Wire 的新增 Provider 使用 `config.C`、现有 DB 和 `opsbiz.PortalUser`，不在构造器发起 Portal 请求。Reader 的清理函数由 Wire 返回的 cleanup 接管。保留原 `api.Dashboard` 的 DB／RedisClient／RedisSync 接线，不顺便更改 Provider 或其他模块依赖。

```bash
go run github.com/google/wire/cmd/wire@v0.7.0 gen ./internal/wirex
gofmt -w internal/mods/dashboard/providers.go internal/mods/dashboard/api/api_key_usage.api.go internal/mods/dashboard/main.go internal/mods/dashboard/wire.go internal/mods/rbac/schema/user.go internal/mods/rbac/biz/login.biz.go
go test ./internal/mods/dashboard/... ./internal/mods/rbac/biz ./internal/wirex -count=1
```

现有 `internal/wirex/provider_wiring_test.go` 必须通过。测试 disabled 和临时连接故障均不阻止 Injector 构造与关闭。

- [x] **Step 5：提交本任务文件。**

```bash
git add -- internal/mods/dashboard/api/api_key_usage.api.go internal/mods/dashboard/api/api_key_usage.api_test.go internal/mods/dashboard/providers.go internal/mods/dashboard/providers_test.go internal/mods/dashboard/main.go internal/mods/dashboard/main_test.go internal/mods/dashboard/wire.go internal/wirex/wire_gen.go internal/mods/rbac/schema/user.go internal/mods/rbac/biz/login.biz.go internal/mods/rbac/biz/login_root_identity_test.go
git commit -m "feat: expose root-only API key usage ranking"
```

### Task 5：局部错误的前端请求客户端

**Files**

- Create: `frontend/src/utils/api-key-usage-request.js`
- Create: `frontend/src/apis/modules/api-key-usage.js`
- Modify: `frontend/src/apis/modules/dashboard.js`
- Create: `frontend/tests/api-key-usage-request.test.mjs`

**Interfaces**

- Consumes: API 契约、现有 user store 的 token／hasRefreshToken／refreshAccessToken／invalidateLocalSession。
- Produces: `createAPIKeyUsageLoader({ send, getSession, refreshAccessToken, invalidateLocalSession })`，返回 `load(params, { signal } = {})`；API 导出 `getAPIKeyRanking(params, options)`，返回解包后的 Response，不返回 Axios 外壳。

- [x] **Step 1：添加 503 不登出、不弹提示和一次 401 恢复测试。**

```javascript
import test from 'node:test'
import assert from 'node:assert/strict'
import { createAPIKeyUsageLoader } from '../src/utils/api-key-usage-request.js'

test('503 preserves the session and propagates a local error', async () => {
    let invalidations = 0
    let refreshes = 0
    const failure = { response: { status: 503 } }
    const load = createAPIKeyUsageLoader({
        send: async () => { throw failure },
        getSession: () => ({ token: 'fixture-token', hasRefreshToken: true }),
        refreshAccessToken: async () => { refreshes++; return true },
        invalidateLocalSession: () => { invalidations++ },
    })
    await assert.rejects(load({ time_range: 'today', sort_by: 'tokens', limit: 10 }), e => e === failure)
    assert.equal(invalidations, 0)
    assert.equal(refreshes, 0)
})
```

- [x] **Step 2：运行并确认 RED。**

Run（frontend）：`node --test tests/api-key-usage-request.test.mjs`

- [x] **Step 3：实现可注入 Loader，绑定现有会话协调器。**

```javascript
export function createAPIKeyUsageLoader({ send, getSession, refreshAccessToken, invalidateLocalSession }) {
    return async function load(params, { signal } = {}) {
        for (let attempt = 0; attempt < 2; attempt++) {
            const session = getSession()
            try {
                const response = await send({
                    method: 'get',
                    url: '/api/v1/dashboard/api-key-ranking',
                    params,
                    signal,
                    headers: { Authorization: session.token },
                })
                if (response.data?.success !== true) {
                    throw Object.assign(new Error('Invalid usage response'), { response })
                }
                return response.data.data
            } catch (error) {
                if (error.response?.status !== 401 || attempt === 1) throw error
                if (!session.hasRefreshToken) {
                    invalidateLocalSession()
                    throw error
                }
                if (!(await refreshAccessToken())) throw error
                if (signal?.aborted) throw error
            }
        }
    }
}
```

运行时 `send` 使用独立 Axios 实例，baseURL 取 `config('http.apiBasic')`，超时 10 秒；不注册全局 message 提示。`getAPIKeyRanking` 被调用时才获取 user store，不能在模块导入阶段访问尚未初始化的 Pinia。`refreshAccessToken` 绑定现有 user store action，因此并发 401 仍共用现有 refresh coordinator；不在该工具内重建刷新或退出协议。

通过 `dashboard.js` re-export `getAPIKeyRanking`，不改其他 API 的 `request.basic` 行为。测试 refresh 自己失败、没有 refresh token、403、取消、重试后仍 401、重试 Authorization 使用新 token，且发送次数不超过 2。

- [x] **Step 4：格式化并验证。**

```bash
npx prettier --config .prettierrc --write src/utils/api-key-usage-request.js src/apis/modules/api-key-usage.js src/apis/modules/dashboard.js
node --test tests/api-key-usage-request.test.mjs tests/session.test.mjs tests/system-version-data.test.mjs
```

- [x] **Step 5：提交。**

```bash
git add -- frontend/src/utils/api-key-usage-request.js frontend/src/apis/modules/api-key-usage.js frontend/src/apis/modules/dashboard.js frontend/tests/api-key-usage-request.test.mjs
git commit -m "feat: isolate usage requests from global dashboard errors"
```

### Task 6：刷新状态机、可见性与已验证 Root 门控

**Files**

- Create: `frontend/src/utils/api-key-usage-state.js`
- Create: `frontend/src/composables/useAPIKeyUsage.js`
- Modify: `frontend/src/store/modules/user.js`
- Create: `frontend/tests/api-key-usage-state.test.mjs`、`frontend/tests/api-key-usage-data.test.mjs`

**Interfaces**

- Consumes: Task 5 `load(params, { signal })`。
- Produces: `createAPIKeyUsageController({ load, onChange, setTimeoutFn, clearTimeoutFn })`，返回 `getState()`、`setQuery(query)`、`setAuthorized(boolean)`、`setActive(boolean)`、`setVisible(boolean)`、`refresh()`、`dispose()`。
- State 字段固定为 `phase`、`data`、`stale`、`error`、`query`；phase 为 `idle/loading/ready/empty/disabled/error/forbidden`。
- Vue 接口：`useAPIKeyUsage(authorized)`，参数为 boolean ref；返回 `state`、`query`、`setQuery`、`refresh`。

- [x] **Step 1：添加默认不请求和同条件保留数据的失败测试。**

```javascript
test('does not load until authorized, active and visible', async () => {
    let calls = 0
    const controller = createAPIKeyUsageController({
        load: async () => { calls++; return { state: 'disabled' } },
        onChange: () => {},
        setTimeoutFn: () => 1,
        clearTimeoutFn: () => {},
    })
    controller.setActive(true)
    controller.setVisible(true)
    await Promise.resolve()
    assert.equal(calls, 0)
    controller.setAuthorized(true)
    await controller.refresh()
    assert.equal(calls, 1)
    controller.dispose()
})
```

使用可控 Promise 测试：首次成功；同条件 503 后 data 不变且 stale=true；切换条件后 503 不显示旧 data；更早 Promise 后完成不能覆盖新条件。测试文件从新模块导入 Controller，`test` 与 `assert` 分别来自 `node:test` 和 `node:assert/strict`。

- [x] **Step 2：运行并确认 RED。**

Run（frontend）：`node --test tests/api-key-usage-state.test.mjs tests/api-key-usage-data.test.mjs`

- [x] **Step 3：实现状态机和条件缓存。**

```javascript
const defaultQuery = { time_range: 'today', sort_by: 'tokens', limit: 10 }
const queryKey = query => JSON.stringify([query.time_range, query.sort_by, query.limit])
```

控制器初始授权、激活、可见三个开关均为 false。开关全部为 true 时立即加载，然后使用 30,000ms 的单次定时器在上次请求结束后调度下一次，防止叠加和重入。相同条件已有请求时 `refresh()` 返回同一个 Promise；新条件取消旧请求并立即执行新请求。

请求开始记录单调递增序号与 queryKey。只允许最新序号提交结果；取消不标为数据源错误。仅内存缓存最近各条件的成功结果，最多覆盖 5×3×3 个合法组合，不写 localStorage。

| 返回／事件 | 数据与状态 |
| --- | --- |
| ready 且 summary.request_count > 0 | ready，保留榜单及未知汇总 |
| ready 且 summary.request_count = 0 | empty |
| disabled | disabled，清除当前条件旧结果 |
| 503／网络故障 | 相同条件有成功结果则保留并 stale；否则 error |
| 全部记录未归属 | ready，展示未归属汇总，不是 empty |
| 401／403／authorized=false | 清空所有数据及缓存，停止定时器，取消请求 |
| active=false／visible=false | 暂停并取消在途请求，不清除同条件成功结果 |
| 恢复三个开关为 true | 立即刷新，不重复建立定时器 |
| dispose | 永久停止，解绑回调，不再接受异步结果 |

金额不在 Controller 内转换为 Number。用 `data.generated_at` 和 `data.window` 展示成功数据时间，不以尝试刷新时间冒充更新时间。

- [x] **Step 4：桥接 Vue 和用户身份。**

在 user store 新增不持久化的 `userInfoVerified: false`。`getUserInfo` 成功后设为 true，失败设为 false；退出和账号切换清回 false，不能信任 localStorage 中保存的 Root 标志。

```javascript
const canViewAPIKeyUsage = computed(
    () => userStore.isLogin && userStore.userInfoVerified && userStore.userInfo?.is_root === true
)
```

`useAPIKeyUsage` 将 Controller State 写入 Vue reactive 对象，watch 授权 ref；通过 `onMounted/onActivated/onDeactivated/onUnmounted` 管理激活状态，通过 `document.visibilityState` 管理可见性，卸载时移除监听并 dispose。重复 mount／activate 不允许重复定时器。

行为测试仿照仓库已有 `system-version-data.test.mjs` 使用真实 Vue renderer 和替换的网络传输；测试 Node 假时钟推进 29,999ms 无请求、30,000ms 触发下一次，以及 keep-alive 失活／恢复、取消和监听清理。

- [x] **Step 5：格式化、验证与提交。**

```bash
npx prettier --config .prettierrc --write src/utils/api-key-usage-state.js src/composables/useAPIKeyUsage.js src/store/modules/user.js
node --test tests/api-key-usage-state.test.mjs tests/api-key-usage-data.test.mjs tests/session.test.mjs
```

```bash
git add -- frontend/src/utils/api-key-usage-state.js frontend/src/composables/useAPIKeyUsage.js frontend/src/store/modules/user.js frontend/tests/api-key-usage-state.test.mjs frontend/tests/api-key-usage-data.test.mjs
git commit -m "feat: add visibility-aware API key usage refresh"
```

### Task 7：首页只读榜单与双语展示

**Files**

- Create: `frontend/src/views/home/components/APIKeyUsageRanking.vue`
- Modify: `frontend/src/views/home/index.vue`
- Modify: `frontend/src/locales/lang/zh-CN/pages.js`、`frontend/src/locales/lang/en-US/pages.js`
- Create: `frontend/tests/api-key-usage-view.test.mjs`

**Interfaces**

- Consumes: `useAPIKeyUsage(authorized)`、schema Response。
- Produces: `APIKeyUsageRanking` 组件，必需 prop 为 `authorized: boolean`；无点击事件输出。

- [x] **Step 1：添加组件及首页集成失败测试。**

```javascript
test('home places usage ranking after model ranking and gates it by verified Root', async () => {
    const source = await readFile(new URL('../src/views/home/index.vue', import.meta.url), 'utf8')
    assert.ok(source.indexOf('<APIKeyUsageRanking') > source.indexOf('dashboard-ranking-table'))
    assert.match(source, /v-if="canViewAPIKeyUsage"/)
    assert.match(source, /userStore\.userInfoVerified/)
    assert.match(source, /userStore\.userInfo\?\.is_root === true/)
})

test('ranking is not a navigation surface', async () => {
    const source = await readFile(
        new URL('../src/views/home/components/APIKeyUsageRanking.vue', import.meta.url), 'utf8'
    )
    assert.doesNotMatch(source, /router\.push|custom-row|@row-click|cursor:\s*pointer/)
    assert.match(source, /row-key="row_id"/)
    assert.match(source, /:pagination="false"/)
})
```

测试使用 `node:test`、`node:assert/strict`、`node:fs/promises`。同时检查所有渲染文案键在两种语言中存在；保留 Task 6 的真实行为测试，不把源码检查当作生命周期验收。

- [x] **Step 2：运行并确认 RED。**

Run（frontend）：`node --test tests/api-key-usage-view.test.mjs`

- [x] **Step 3：实现独立卡片与七列渲染。**

```vue
<script setup>
import { computed, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAPIKeyUsage } from '@/composables/useAPIKeyUsage'

const props = defineProps({ authorized: { type: Boolean, required: true } })
const { t } = useI18n()
const { state, query, setQuery } = useAPIKeyUsage(toRef(props, 'authorized'))
const rows = computed(() => state.data?.items || [])
const columns = computed(() => [
    { key: 'key', title: t('pages.dashboard.apiKeyRanking.key'), width: 230 },
    { key: 'source', title: t('pages.dashboard.apiKeyRanking.source'), width: 130 },
    { key: 'owner', title: t('pages.dashboard.apiKeyRanking.owner'), width: 180 },
    { key: 'requests', title: t('pages.dashboard.apiKeyRanking.requests'), width: 100 },
    { key: 'tokens', title: t('pages.dashboard.apiKeyRanking.tokens'), width: 210 },
    { key: 'cost', title: t('pages.dashboard.apiKeyRanking.cost'), width: 130 },
    { key: 'success', title: t('pages.dashboard.apiKeyRanking.success'), width: 100 },
])
</script>
```

表格使用 `rows`、`columns`、`:pagination="false"`、`row-key="row_id"` 和窄屏横向滚动。卡片右侧是时间、排序、Top N 三个选择器；所有变化调用 `setQuery`，不使用本地表格 sorter。排序选择器明确固定降序。

Token 单元格的实现核心：

```vue
<a-tooltip :trigger="['hover', 'focus']">
    <template #title>
        <div>{{ t('pages.dashboard.apiKeyRanking.input') }}: {{ record.input_tokens }}</div>
        <div>{{ t('pages.dashboard.apiKeyRanking.output') }}: {{ record.output_tokens }}</div>
        <div>{{ t('pages.dashboard.apiKeyRanking.cached') }}: {{ record.cached_tokens }}</div>
        <div>{{ t('pages.dashboard.apiKeyRanking.cacheCreation') }}: {{ record.cache_creation_tokens }}</div>
    </template>
    <span tabindex="0">{{ record.total_tokens }}</span>
</a-tooltip>
<a-progress :percent="record.token_share ?? 0" :show-info="false" size="small" />
<span>{{ record.token_share == null ? '—' : `${record.token_share.toFixed(2)}%` }}</span>
```

Key 单元格显示名称、脱敏片段和可确认状态；同名且相同脱敏片段时附 RowID 前 8 位。归属优先名称，否则 ID；未知显示国际化文本。金额展示原十进制字符串并使用既有费用单位，不经浮点重新计价。成功率保留两位小数。

未知历史用量在卡片底部单列，显示请求、Token、费用、占比；全量 summary 作为占比说明，不把 Top N 行之和标为全部。错误、disabled、empty、stale 和 metadata warnings 按 Task 6 状态分别展示。时间包含服务端时区偏移，stale 时保留旧时间。

将组件放在模型排行整行之后，首页统一使用 PascalCase `<APIKeyUsageRanking v-if="canViewAPIKeyUsage" :authorized="canViewAPIKeyUsage" />`，组件导入名与测试保持一致。

新增文案固定使用 `pages.dashboard.apiKeyRanking.` 前缀：

| 后缀 | 中文 | English |
| --- | --- | --- |
| title | API Key 用量排行 | API Key Usage |
| key / source / owner | API Key / 来源 / 归属 | API Key / Source / Owner |
| requests / tokens / cost / success | 请求数 / Token 用量 / 费用 / 成功率 | Requests / Tokens / Cost / Success rate |
| input / output / cached / cacheCreation | 输入 / 输出 / 缓存命中 / 缓存创建 | Input / Output / Cache read / Cache creation |
| disabled | 尚未启用 API Key 用量统计 | API key usage reporting is not enabled |
| empty | 该时段暂无调用 | No requests in this period |
| unavailable | 数据暂不可用 | Usage data is temporarily unavailable |
| stale | 更新失败，当前显示上次成功结果 | Update failed; showing the last successful result |
| metadataUnavailable | 信息暂不可用 | Metadata is temporarily unavailable |
| unattributed | 无法归属的历史用量 | Unattributed historical usage |
| denominator | 占比基于该时段全部 Key 用量，包含未入榜及未归属用量 | Share uses all key usage in this period, including keys outside the ranking and unattributed usage |
| updatedAt | 最近成功更新 | Last successful update |

时间选择文本复用现有首页对应范围文案；来源和状态按固定枚举补齐逐项翻译，不直接显示英文枚举。

- [x] **Step 4：格式化、执行视图和行为回归。**

```bash
npx prettier --config .prettierrc --write src/views/home/components/APIKeyUsageRanking.vue src/views/home/index.vue src/locales/lang/zh-CN/pages.js src/locales/lang/en-US/pages.js
node --test tests/api-key-usage-view.test.mjs tests/api-key-usage-data.test.mjs tests/model-performance-trends.test.mjs
npm run build:prod
```

- [x] **Step 5：提交本任务文件。**

```bash
git add -- frontend/src/views/home/components/APIKeyUsageRanking.vue frontend/src/views/home/index.vue frontend/src/locales/lang/zh-CN/pages.js frontend/src/locales/lang/en-US/pages.js frontend/tests/api-key-usage-view.test.mjs
git commit -m "feat: render root-only API key usage ranking on homepage"
```

### Task 8：集成验证、只读接入文档与完整交付检查

**Files**

- Create: `internal/mods/dashboard/dal/api_key_usage_integration_test.go`
- Create: `frontend/tests/api-key-usage.browser.mjs`
- Create: `frontend/tests/fixtures/api-key-usage.html`、`frontend/tests/fixtures/api-key-usage.js`
- Modify: `docs/CONFIGURATION.md`
- 按任务结果更新本计划复选框，不预先勾选未执行步骤。

**Interfaces**

- Consumes: Tasks 1–7 的完整组件、Reader、API、配置与会话能力。
- Produces: 独立 ClickHouse 测试、实际浏览器验证记录、配置说明与最终测试证据；不部署到线上。

- [x] **Step 1：添加隔离 ClickHouse 集成测试。**

测试仅在 `API_KEY_USAGE_CH_TEST_ADDR` 非空时启用；连接凭据从专用测试环境变量读取，不读取生产 DSN。测试创建带 `api_key_usage_test_` 固定前缀及随机后缀的独立数据库，注册 `t.Cleanup` 删除该精确数据库并关闭连接。数据库名必须通过前缀和字符白名单验证，绝不使用配置中的业务数据库执行 DROP。

使用当前 Gateway `scripts/clickhouse_schema.sql` 中 `access_logs` 的字段和排序键创建隔离表，不创建无关汇总视图。插入以下记录：

| 记录 | 预期 |
| --- | --- |
| A 请求重复插入，所有排序键相同 | FINAL 后只计一次 |
| 同 request_id、不同 time 的第二次请求 | 单独计数 |
| Hash 不同、脱敏片段相同 | 两个 Key |
| 同 Hash、不同身份附加字段 | 最终合并为一个 Key |
| 无 Hash、无可靠 ID | 未归属汇总 |
| 缓存 Token 非零 | 不重复加至总量 |
| cost 为 0.1 与 0.2 | 十进制结果准确为 0.3 |
| 查询窗口起点及终点记录 | 起点包含，终点排除 |

固定测试窗口从 UTC 09:00 到 UTC 10:00；插入六行：起点的 A 请求（cost 0.1，input 10，output 20，cached 5，cache_creation 2）、完全相同的 A 补偿副本、09:00:01 复用 request_id 但 Hash 为 B 的请求（cost 0.2，input 40，output 30）、09:00:02 Hash A 但附加 UserID 不同的 499 请求（零 Token 和费用）、09:00:03 无身份请求（零 Token 和费用）、终点 10:00 的请求（窗口外）。时间通过 Go `time.Time` 参数绑定，不使用受服务器时区影响的无偏移字符串插入。

测试核心断言直接针对真实 Reader：

```go
window := schema.Window{
    Start: time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC),
    End: time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC),
    Timezone: "UTC",
}
rows, err := reader.Read(ctx, window)
require.NoError(t, err)
var all schema.Totals
for _, row := range rows {
    all.Add(row.Totals)
}
require.Equal(t, uint64(4), all.Requests)
require.Equal(t, uint64(100), all.Tokens())
require.Equal(t, uint64(5), all.Cached)
require.True(t, decimal.RequireFromString("0.3").Equal(all.Cost))
```

`reader` 由 Task 1 构造器连接本测试创建的独立数据库，`ctx` 使用测试的有界上下文。不能从被测输出反推期望值。没有专用测试环境时明确报告 SKIP，不能声称完成真实 ClickHouse 验证。

- [x] **Step 2：增加真实组件的浏览器 Fixture 与测试。**

Fixture 使用 Vite 解析项目别名，导入真实 `APIKeyUsageRanking`、Ant Design Vue、Pinia 和两种语言。初始化已登录的测试 store，`userInfo={is_root:true}`、`userInfoVerified=true`，将组件作为实际 Vue 组件挂载。`?root=0` 将授权改为 false，`?locale=en-US` 切英文；不要复制组件 HTML 来替代被测实现。

浏览器脚本沿用仓库现有 `PLAYWRIGHT_MODULE`、`TEST_BASE_URL` 和 `TEST_SCREENSHOT_DIR` 参数，mock 网络但不 mock 组件。通过 route.fulfill 返回固定 Response，并记录请求参数。

```javascript
await page.goto(`${baseURL}/tests/fixtures/api-key-usage.html`)
await page.getByText('API Key 用量排行', { exact: true }).waitFor()
const before = page.url()
await page.getByText('Production Agent', { exact: true }).click()
assert.equal(page.url(), before)
assert.equal(await page.locator('.ant-drawer, .ant-modal').count(), 0)
await page.setViewportSize({ width: 390, height: 844 })
await page.screenshot({ path: `${outputDir}/api-key-usage-mobile.png`, fullPage: true })
```

同时验证桌面 1440px、移动 390px、中英文、三种数据异常、未知汇总、状态标签、悬停／键盘聚焦明细、切排序触发服务端请求、非 Root 零请求、401／403 清除旧结果。Fixture 在 false 授权时通过 `v-if` 不挂载组件。

运行入口：

```bash
npm run dev -- --host 127.0.0.1 --port 9211
```

另一个终端从 frontend 执行 `node tests/api-key-usage.browser.mjs`。若缺少 Playwright 运行时，使用执行环境已有的浏览器验证工具完成同一清单并记录缺少的自动化验证，不为此擅自安装全局依赖。

- [x] **Step 3：补充只读配置说明。**

在 `docs/CONFIGURATION.md` 记录 Task 1 的完整配置块、默认关闭、原生端口及 TLS 配对、凭据环境变量、需要已有日志表与 Key 标识列、只读账号的 SELECT 权限、查询超时及三种 UI 状态。

明确说明：

- 此功能不自动启用 Gateway 日志落库、不执行 ClickHouse schema 迁移；
- 无 Key 标识的旧日志只能进入未归属汇总；
- Root 标志来自服务端而非浏览器用户名；
- 历史元数据缺失不等于已删除；
- 占比基于本榜单全量数据，不要求与 Prometheus 卡片逐秒一致；
- 费用为网关记录值，不是本功能重新结算；
- 查询与元数据缓存可能分别延迟最多 30／60 秒；
- 上线启用和真实环境性能验证须获得单独授权。

- [x] **Step 4：执行完整自动化验证与安全检查。**

仓库根目录：

```bash
go test ./internal/config ./internal/mods/dashboard/... ./internal/mods/rbac/biz ./internal/wirex -count=1
go test -race ./internal/mods/dashboard/... -count=1
git diff --check
```

frontend：

```bash
node --test tests/*.test.mjs
npm run build:prod
npx prettier --config .prettierrc --check src/utils/api-key-usage-request.js src/utils/api-key-usage-state.js src/composables/useAPIKeyUsage.js src/apis/modules/api-key-usage.js src/apis/modules/dashboard.js src/store/modules/user.js src/views/home/index.vue src/views/home/components/APIKeyUsageRanking.vue src/locales/lang/zh-CN/pages.js src/locales/lang/en-US/pages.js
```

`tests/*.test.mjs` 会包含工作区原有未跟踪测试，只运行而不修改或提交。若它们失败，区分已有失败与本次回归，并保留证据。

检查最终 diff：只有计划内功能文件；Gateway／Portal 无隐式变更；没有生产写入、密钥打印、全局请求错误行为变更或修改现有 Key 删除逻辑。真实 7 天 Top 50 负载若未测，交付时明确标为未验证。

- [x] **Step 5：提交验证及接入文档，交付证据。**

```bash
git add -- internal/mods/dashboard/dal/api_key_usage_integration_test.go frontend/tests/api-key-usage.browser.mjs frontend/tests/fixtures/api-key-usage.html frontend/tests/fixtures/api-key-usage.js docs/CONFIGURATION.md docs/plans/2026-09-15-api-key-usage-ranking.md
git commit -m "test: verify API key usage ranking and document read-only setup"
```

交付说明列出实际执行的测试、结果、SKIP 项、截图位置及配置尚未启用的事实；不把计划中的预期 PASS 写成实际结果。结束前按验证与代码审查技能检查完成情况，不自动部署。

## 设计覆盖与执行顺序

| 设计要求 | 对应任务 |
| --- | --- |
| 三类 Key、Hash 优先、可靠 ID、历史状态 | 2、3 |
| FINAL 去重、窗口、Token、费用、完整分母、Top N | 1、3、8 |
| 只读配置、启动隔离、查询预算 | 1、2、4、8 |
| Root 后端校验、真实身份标志、不泄露 Key／Hash | 3、4、5、6 |
| 局部错误、共享会话恢复、不反复全局提示 | 5、6 |
| 30 秒刷新、筛选、可见性、缓存与失权清理 | 3、6 |
| 七列表格、无点击、未知用量、双语与窄屏 | 7、8 |
| 数据源失败／未启用／无请求三态 | 1、3、4、6、7 |
| 原有首页及 Wire 不回归、测试数据可清理 | 4、8 |

默认按 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 执行，每项通过自身 RED／GREEN 与审查再继续。前端和后端可以基于固定契约拆分执行，但不得修改另一任务拥有的文件或在未经授权时主动创建子代理。

## 执行记录（2026-09-16）

用户已选择当前会话顺序执行，并明确要求直接修改当前目录；未创建分支或 worktree，未使用子代理，未推送或部署。原有两份未跟踪测试保持原样。

| 任务 | 实际提交／状态 |
| --- | --- |
| 1 只读数据源 | `2735860` |
| 2 身份与历史元数据 | `f4a951d` |
| 3 全量聚合与缓存 | `b4531a0` |
| 4 Root API 与依赖接线 | `71bbbaf` |
| 5 独立前端请求 | `b2d1298` |
| 6 刷新、会话与可见性 | `80d3138` |
| 7 首页组件 | `bc4191d` |
| 8 集成验证与边界修正 | 本次收尾提交；本地验证完成，真实 ClickHouse 测试跳过 |

执行中的验证与调整：

- 基线：相关后端包全部通过；前端 46 项测试通过。
- 各新增模块先执行失败测试，再实现并验证通过。修复了本次元数据查询链复用导致的表模型残留，未修改现有 Key 删除或计费逻辑。
- 前端视图测试改为编译真实 Vue SFC、使用实际 Ant Design Vue 组件进行服务端渲染；刷新测试运行实际 Vue 生命周期，不依赖源码字符串推断行为。
- 新增 `api-key-usage-identity.test.mjs` 验证缓存 Root 不授予权限、迟到的旧账号资料不能覆盖新账号。
- 追加浏览器失败用例后，补齐首页重新加载时的当前用户验证；在原有登录流程之外不改变全局认证协议。
- 追加身份冲突回归：冲突历史只保留可靠 Hash 与用量，不从多条冲突信息中任取归属。
- 浏览器已通过 12 个场景，检查桌面、移动端、暗色英文、无点击、筛选、异常、权限与完整首页挂载。截图为模拟数据，不是线上使用数据。
- 最终回归：相关 Go 包测试通过，Dashboard 全部子包的 `-race` 检查通过；前端 70 项测试通过；生产构建及本次前端文件的 Prettier 检查通过。
- 主程序入口 `go build -o /tmp/tokenlive-api-key-usage.y6jfQD/tokenlive-admin .` 通过。额外的 `go build ./...` 因被 Git 忽略的既有 `scratch/db_check.go` 与 `scratch/update_db.go` 都声明 `main` 而失败；这两个六月创建的本地脚本未作修改。
- 构建仍提示 `useMultiTab` 循环依赖警告，Node 提示项目未声明模块类型；未为本功能改动无关的打包或模块配置。
- 真实 ClickHouse 集成测试已添加，但因未配置 `API_KEY_USAGE_CH_TEST_ADDR` 明确跳过；真实 7 天 Top 50 查询性能也未验证。
- ClickHouse 仍默认关闭。维护人员需按 `docs/CONFIGURATION.md` 配置只读连接后自行启用。
