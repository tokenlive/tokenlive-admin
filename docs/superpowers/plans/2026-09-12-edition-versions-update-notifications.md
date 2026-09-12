# Edition Versions and Update Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 明确展示单机版整包和专业版组件版本，并向获授权的用户提供稳定版更新提醒与升级指引。

**Architecture:** Admin 负责运行时身份、发布源检查、权限和展示；Gateway 通过既有 Redis 或 HTTP 通道上报带时效的进程版本。发布查询缓存与当前节点分布分离，所有确定性升级判断都使用可信运行版本和未过期候选。standalone 显式传递整包身份，Homebrew 的安装标记不从操作系统或版本字符串推断。

**Tech Stack:** 仓库现有 Go 1.27.1、Gin、Wire、Casbin、go-redis/v9、Vue 3、Pinia、Ant Design Vue、Node 内置测试、既有 Playwright 浏览器测试、Python unittest、Homebrew Formula、Docker Compose。

**Spec:** [已确认设计](../specs/2026-09-11-edition-versions-update-notifications-design.md)；执行前同时阅读 [CONTEXT.md](../../../CONTEXT.md)、[ADR-0005](../../adr/0005-edition-version-boundaries.md)、[ADR-0006](../../adr/0006-server-side-update-check-isolation.md)。

**执行状态（2026-09-12）：**13 项任务及最终整项审查发现的 5 项问题均已完成，独立复审无未解决问题，提交后的隔离环境回归通过。以下勾选表示本地实施步骤已有证据，不表示已发布、已合入原项目或真实部署已通过。交付结果见[交付记录](2026-09-12-edition-versions-update-notifications-handoff.md)，完整过程保留在本工作区 `.superpowers/sdd/2026-09-12-edition-versions-update-notifications/progress.md`。

**后续集成（2026-09-12）：**用户进一步明确同意合回各仓库 main 并提交 Standalone。四仓库已完成本地快进合并，Standalone 提交 `cdab01310f880202f96173a2c763c17fe39083eb`；合并后的实际项目目录已完成 Go、脚本、前端及浏览器回归。下文“不提交/不合并”是此前实施阶段的边界记录，现由这次明确授权完成本地集成；仍未推送、发布、安装或重启。Admin 原有 BasicHeader/ModelDetail 改动保留暂存，备份见交付记录。

## Global Constraints

- 自动检查：启动后异步检查一次，此后每 6 小时检查，周期可配置。
- 一轮外部检查超时：5 秒，各来源并行检查并共享该截止时间。
- 手动检查冷却：60 秒；同一 Admin 实例内所有用户共用冷却和进行中的请求。
- Gateway 版本上报间隔：30 秒；有效上报窗口：3 分钟。
- 联网检查关闭后，定时和手动外部检查均停止；本地版本展示及内部版本上报继续运行。
- 仅正式稳定版可作为升级目标；开发构建和无法识别的当前版本不可比较。
- Homebrew 当前 Formula 和各专业版仓库的 latest Release 是候选来源；不回退历史发行，不自行寻找历史最高版本。
- Root 默认拥有版本更新管理权限；其他用户通过角色授权；后端校验，不能只隐藏按钮或判断用户名。
- 普通登录用户可看当前版本与节点聚合，但不能看更新专属信息、节点标识或触发手动检查。
- 不自动下载安装、执行远端命令、重启、创建统一发布清单或扩展完整节点管理。
- 使用现有依赖；Admin 的生产依赖仅将已在 go.sum 中的 `golang.org/x/mod v0.35.0` 提升为直接依赖；节点测试可新增与 Gateway 一致的 `miniredis v2.38.0`。不得顺带升级依赖。
- 新增配置使用默认值与环境变量兼容，不读取、输出或改写用户的实际密钥和 `.env`。
- 修改前端 `.vue`、`.js` 后必须执行项目 Prettier。
- 数据测试使用内存、临时目录或独立测试 Redis；每条记录有删除路径和 TTL。不得清空业务 Redis 或连接生产服务做验收。
- 用户原有 `frontend/src/layouts/components/BasicHeader.vue` 暂存改动不属于本计划；涉及同一文件时只做必要接线并逐块核对。
- **standalone 的 AGENTS.md 禁止未获用户明确要求的 git commit。**该仓库任务只保留经过验证的差异；不要套用其他仓库的提交步骤。
- 不推送、打 tag、发布 Release、安装 Homebrew 包或启动真实部署；这些均不是执行本计划自动获得的权限。

---

## 工作目录、执行边界和顺序

下列别名是文件路径前缀，不是需要写入 shell 的变量。若执行时已创建隔离 worktree，将别名映射到对应 worktree，并记录映射。

| 别名 | 当前绝对路径 |
|---|---|
| A | `/Users/chenzhiguo/Projects/tokenlive-admin` |
| G | `/Users/chenzhiguo/Projects/tokenlive-gateway` |
| S | `/Users/chenzhiguo/Projects/tokenlive-standalone` |
| D | `/Users/chenzhiguo/Projects/tokenlive-deploy` |

本计划是一项有共同接口的端到端功能，不拆成互不衔接的子项目。执行时使用 `using-git-worktrees` 检查隔离条件；规划阶段不创建 worktree、不改业务源码。实现任务保留先红后绿的证据，最终联合验收如已通过则如实记录，不能人为制造失败；本文中的代码是计划，不是已完成实现。

依赖顺序：

```text
1 身份与比较 ─┬─ 2 发布源 ─ 3 检查器 ─┐
              ├─ 4 节点记录 ──────────┼─ 6 服务集成 ─ 7 API/权限 ─ 8 前端状态 ─ 9 页面
              └─ 5 Gateway 上报 ─────┘
1、6、7 ─ 10 standalone 传递与安装标记
1、5、10 ─ 11 构建发布元数据 ─ 12 部署版本兼容
全部 ─ 13 联合验收与文档
```

只有契约固定且文件不重叠的任务才可并行。Task 6/7 会修改共同接线文件，串行执行。每个任务的审阅和提交只包含该任务文件；任何提交均保留其他暂存内容，不能使用 `git add .`。

## 文件责任图

| 文件/目录 | 责任 |
|---|---|
| A/pkg/productversion | 发行身份、正式版本解析、纯比较函数 |
| A/internal/updatecheck | 发布源解析、检查并发/时效/冷却，不依赖 Gin 或数据库 |
| A/pkg/versionregistry | 节点协议、内存/Redis 记录、聚合，不依赖请求指标 |
| A/internal/versionstatus | 将当前身份、有效节点和发布候选组成响应 |
| A/internal/mods/systemversion | 生命周期、HTTP 路由、权限与 Wire provider |
| A/frontend/src/utils/system-version.js | 不依赖 Vue 的显示/权限/标记转换 |
| A/frontend/src/composables/useSystemVersion.js | 拉取缓存结果、刷新、手动检查和组件生命周期 |
| G/pkg/versionreport | 节点协议、Redis/HTTP sender、上报循环 |
| S/internal/assemble、S/cmd/tokenlive | 宿主身份传递；不拥有检查业务或 Gateway 引擎业务 |
| S/packaging/homebrew/tokenlive.rb | 仅在 Homebrew 安装时创建渠道标记 |
| 各仓库构建文件、D 的编排文件 | 真实版本和构建类型注入、独立镜像版本兼容 |

## Task 1: 运行时身份与可比较版本

**目录：** A；无前置任务。

**Files**

- Create: `pkg/productversion/identity.go`, `pkg/productversion/compare.go`, `pkg/productversion/compare_test.go`
- Modify: `go.mod`, `go.sum`, `main.go`, `cmd/start.go`, `adminapp/app.go`, `internal/bootstrap/bootstrap.go`, `internal/config/config.go`
- Test: `adminapp/version_test.go`, `internal/bootstrap/version_test.go`

**Interfaces**

```go
// package productversion
type Build struct {
    Version string `json:"version"`
    Kind    string `json:"kind"` // release | dev
}
type Identity struct {
    Edition        string `json:"edition"` // standalone | professional
    InstallChannel string `json:"install_channel"` // homebrew | release | unknown
    Build          Build  `json:"build"`
}
func StableVersion(raw string) (string, bool)
func Compare(current Build, target string) string
// Results: available | current | ahead | uncomparable | no_candidate
func ResolveIdentity(explicit *Identity, legacyVersion string) Identity
```

- [x] **1. 写纯函数失败测试。**文件使用 `package productversion`，导入 `testing`。

```go
func TestCompare(t *testing.T) {
    cases := []struct{ current, kind, target, want string }{
        {"v1.2.3", "release", "1.2.4", "available"},
        {"v1.2.3", "release", "1.2.3", "current"},
        {"v2.0.0", "release", "1.9.0", "ahead"},
        {"v1.2.3", "dev", "1.2.4", "uncomparable"},
        {"git-abc", "dev", "1.2.4", "uncomparable"},
        {"1.2.3", "release", "1.3.0-rc.1", "no_candidate"},
        {"v01.2.3", "release", "1.2.4", "uncomparable"},
    }
    for _, tc := range cases {
        if got := Compare(Build{tc.current, tc.kind}, tc.target); got != tc.want {
            t.Errorf("%+v: got %s", tc, got)
        }
    }
}
```

- [x] **2. 验证红灯。**`go test ./pkg/productversion -run TestCompare -count=1`；预期因待实现函数缺失失败。
- [x] **3. 实现身份和比较，再接入启动链。**`StableVersion` 使用严格三段版本形式，再调用 `semver.IsValid`、`semver.Prerelease`、`semver.Canonical`；允许前导 `v` 和合法 build metadata，不接受缩写 `v1`、预发布或前导零。

```go
func Compare(current Build, target string) string {
    cv, ok := StableVersion(current.Version)
    if current.Kind != "release" || !ok { return "uncomparable" }
    tv, ok := StableVersion(target)
    if !ok { return "no_candidate" }
    switch semver.Compare(cv, tv) {
    case -1: return "available"
    case 0: return "current"
    default: return "ahead"
    }
}
```

  `main.VERSION` 默认改为 `dev`，增加 `main.BUILD_KIND="dev"`。现有 `StartCmd(version ...string)` 保持兼容：第一个参数是版本，第二个可选参数是构建类型；旧调用未传类型时按 dev。`adminapp.Options` 与 `bootstrap.RunConfig` 各增加 `Identity *productversion.Identity`。

  `ResolveIdentity` 优先使用显式身份；缺省为 professional、release 渠道、legacyVersion 或 dev、构建类型 dev。非法显式 edition 转为 unknown、渠道 unknown、构建类型 dev，保留原始版本供诊断；正常启动入口只生成 standalone/professional。非法渠道转为 unknown，不升级为猜测的正式身份。Config 新字段固定为 `RuntimeIdentity productversion.Identity`，使用 `json:"-" toml:"-"` 排除配置反序列化；启动时填入可信身份，并将主版本同步到 `General.Version`，保留既有公开接口。

- [x] **4. 绿灯及兼容测试。**`go test ./pkg/productversion ./adminapp ./internal/bootstrap -run 'Test.*(Version|Identity|Compare)' -count=1`。必须覆盖 standalone 显式身份优先、legacy 调用不能被当作正式发行；对启动测试禁用外部检查，不运行真实配置。
- [x] **5. 审阅任务差异。**`git diff --check`，核对新依赖仅为已有版本的 x/mod。记录红/绿灯后按本仓库授权仅提交本任务文件。

## Task 2: 两类受控发布源

**目录：** A；依赖 Task 1。

**Files**

- Create: `internal/updatecheck/source.go`, `github.go`, `homebrew.go`, `source_test.go`（均位于 `internal/updatecheck/`）

**Interfaces**

```go
type Candidate struct {
    Version    string `json:"version"`
    ReleaseURL string `json:"release_url"`
}
type Source interface {
    Latest(context.Context) (Candidate, error)
}
var ErrNoCandidate = errors.New("no stable candidate")
func NewGitHubSource(client *http.Client, component string) Source
func NewHomebrewSource(client *http.Client) Source
func ParseGitHubRelease(body []byte, component string) (Candidate, error)
func ParseFormula(body []byte) (Candidate, error)
```

- [x] **1. 写解析失败测试。**文件 `package updatecheck`，导入 `errors`、`testing`；使用如下真实输入，不访问公网。

```go
func TestCandidateFilters(t *testing.T) {
    c, err := ParseGitHubRelease([]byte(`{"tag_name":"v1.2.4","draft":false,"prerelease":false}`), "admin")
    if err != nil || c.Version != "v1.2.4" { t.Fatalf("%+v %v", c, err) }
    _, err = ParseGitHubRelease([]byte(`{"tag_name":"v1.3.0-rc.1","draft":false,"prerelease":false}`), "gateway")
    if !errors.Is(err, ErrNoCandidate) { t.Fatalf("got %v", err) }
    if _, err = ParseFormula([]byte("version \"1.2.3\"\nversion \"1.2.4\"\n")); err == nil {
        t.Fatal("duplicate version must fail")
    }
}
```

- [x] **2. 验证红灯。**`go test ./internal/updatecheck -run TestCandidateFilters -count=1`。
- [x] **3. 实现适配器。**固定来源如下，不能由浏览器传入 URL：

```text
admin:   GET https://api.github.com/repos/tokenlive/tokenlive-admin/releases/latest
gateway: GET https://api.github.com/repos/tokenlive/tokenlive-gateway/releases/latest
brew:    GET https://api.github.com/repos/tokenlive/homebrew-tokenlive/contents/Formula/tokenlive.rb
```

  所有请求使用 `http.NewRequestWithContext`、静态 User-Agent、`Accept: application/vnd.github+json`。Contents 响应必须是 `type=file, encoding=base64`，解码其 `content` 后仅解析锚定行首的唯一 `version "..."` 声明，不执行 Ruby。响应上限 1 MiB，Formula 解码上限 64 KiB；多声明、格式损坏、非 2xx 返回检查错误，合法响应但非稳定候选返回 `ErrNoCandidate`。

```go
// Shared bounded read; call after checking HTTP status.
body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
if err != nil { return Candidate{}, err }
if len(body) > 1<<20 { return Candidate{}, errors.New("response too large") }
```

  Release 排除 draft/prerelease，tag 再经 `StableVersion` 校验；不要调用历史列表排序。ReleaseURL 根据受控 component 对应仓库及已验证 tag 构造，不照收任意外链。Homebrew 链接指向 standalone 相应 tag，不能反向用 standalone Release 覆盖 Formula 候选。重定向仅允许预期 HTTPS 官方主机，不泄露认证信息。

- [x] **4. 绿灯与 HTTP 契约测试。**`go test ./internal/updatecheck -run 'Test(Candidate|Source|Formula)' -count=1`。为 http.Client 注入测试 RoundTripper，断言请求路径、无业务 header/body、超大响应、取消、403/429/404、base64 异常、无稳定候选、Formula 旧于 Release 时仍采用 Formula。
- [x] **5. 审阅后只提交本任务文件。**此时测试通过只证明模拟契约，不能声称公网源已验证。

## Task 3: 检查器、冷却及过期状态

**目录：** A；依赖 Task 2。

**Files**

- Create: `internal/updatecheck/checker.go`, `state.go`, `checker_test.go`

**Interfaces**

```go
type Options struct {
    Enabled bool
    Interval, Timeout, Cooldown time.Duration
    Now func() time.Time
    After func(time.Duration) <-chan time.Time
}
type SourceState struct {
    Status string `json:"status"` // unchecked | checking | ready | no_candidate | unavailable | disabled
    Candidate *Candidate `json:"candidate,omitempty"`
    LastAttempt time.Time `json:"last_attempt"`
    LastSuccess time.Time `json:"last_success"`
    Stale bool `json:"stale"`
    ErrorCode string `json:"error_code,omitempty"`
}
type CheckResult struct {
    Enabled bool `json:"enabled"`
    Sources map[string]SourceState `json:"sources"`
    RetryAfterSeconds int `json:"retry_after_seconds"`
}
type flight struct {
    done chan struct{}
    result CheckResult
    err error
}
type Checker struct {
    mu sync.Mutex
    wg sync.WaitGroup
    startOnce sync.Once
    closeOnce sync.Once
    ctx context.Context
    cancel context.CancelFunc
    opts Options
    sources map[string]Source
    states map[string]SourceState
    running *flight
    lastStart time.Time
}
func NewChecker(ctx context.Context, opts Options, sources map[string]Source) (*Checker, error)
func (c *Checker) Start()
func (c *Checker) Check(ctx context.Context, manual bool) (CheckResult, error)
func (c *Checker) Snapshot() CheckResult
func (c *Checker) Close()
var ErrDisabled = errors.New("update checks disabled")
var ErrCooldown = errors.New("check cooldown")
```

  checker 不关闭借用的 HTTP client；所有结果 map 和 Candidate 指针在返回调用方前复制，不能泄漏内部可变状态。

- [x] **1. 写失败测试。**`sourceFunc` 在本测试文件定义，`Options.Now/After` 允许推进时间而不睡眠六小时。

```go
type sourceFunc func(context.Context) (Candidate, error)
func (f sourceFunc) Latest(ctx context.Context) (Candidate, error) { return f(ctx) }

func TestDisabledNeverCallsSource(t *testing.T) {
    var calls atomic.Int32
    c, err := NewChecker(context.Background(), Options{Enabled:false}, map[string]Source{
        "admin": sourceFunc(func(context.Context) (Candidate,error) {
            calls.Add(1); return Candidate{Version:"v1.2.4"},nil
        }),
    })
    if err != nil { t.Fatal(err) }
    defer c.Close()
    c.Start()
    _, err = c.Check(context.Background(), true)
    if !errors.Is(err, ErrDisabled) || calls.Load() != 0 { t.Fatal(err, calls.Load()) }
}
```

- [x] **2. 验证红灯。**`go test ./internal/updatecheck -run TestDisabledNeverCallsSource -count=1`。
- [x] **3. 实现状态机。**Options 的零值时长补为 6h/5s/60s，Now/After 为空时使用 time.Now/time.After；负时长返回 error。服务端显式配置的周期必须在 provider 层验证为正，不能把错误配置静默当作缺省。`Start` 幂等，启动一次异步检查，随后用 `After(Interval)` 调度；`Close` 幂等并取消、等待所有工作，关闭后 Check 不再创建工作。

```text
Check:
  disabled -> ErrDisabled
  flight exists -> join done channel, caller cancellation only stops its own wait
  manual && now-lastStart < cooldown -> cached state + retry seconds + ErrCooldown
  else -> record lastStart; launch one flight with background-context 5s deadline
flight:
  fetch sources concurrently using the SAME deadline
  success candidate -> ready, replace candidate, LastSuccess=now, Stale=false
  ErrNoCandidate -> no_candidate, mark prior candidate stale, keep prior LastSuccess
  other error -> unavailable, mark prior candidate stale, keep prior LastSuccess
  publish each source result independently; close done channel
Snapshot:
  deep-copy state; if now-LastSuccess >= Interval mark old candidate stale
```

  请求等待取消不能取消其他用户共享的后台工作。Snapshot 不发网络请求。候选历史仅供展示，不保留旧的确定性“available”比较状态。

- [x] **4. 验证绿灯。**`go test -race ./internal/updatecheck -count=1`。覆盖启动异步、并发合并、冷却第 59/60 秒、两个来源共享截止时间、一个源失败、候选撤回、快照深拷贝、停止后无工作。测试时用注入时钟与受控 channel，不以大段 sleep 验证。
- [x] **5. 审阅后只提交本任务文件。**

## Task 4: Admin 节点协议和带 TTL 的记录

**目录：** A；依赖 Task 1。

**Files**

- Create: `pkg/versionregistry/protocol.go`, `memory.go`, `redis.go`, `groups.go`, `registry_test.go`
- Create: `pkg/versionregistry/testdata/node-v1.json`
- Modify: `go.mod`, `go.sum`（仅新增测试依赖 miniredis v2.38.0，与 Gateway 一致）

**Interfaces**

```go
type Node struct {
    SchemaVersion int    `json:"schema_version"`
    Namespace     string `json:"namespace"`
    InstanceID    string `json:"instance_id"`
    Version       string `json:"version"`
    BuildKind     string `json:"build_kind"`
}
type Group struct {
    Version string `json:"version"`
    BuildKind string `json:"build_kind"`
    Count int `json:"count"`
}
type Store interface {
    Upsert(context.Context, Node) error
    List(context.Context) ([]Node, error)
    Delete(context.Context, string) error
}
func NewMemoryStore(namespace string, now func() time.Time) Store
func NewRedisStore(client *redis.Client, namespace string) Store
func Validate(node Node, expectedNamespace string) error
func GroupNodes(nodes []Node) []Group
const TTL = 3 * time.Minute
```

  固定 key 为 `tokenlive:gateway-versions:<namespace>:<instance_id>`；namespace 默认 `default`，只接受 `[A-Za-z0-9_-]{1,64}`，instance ID 为 UUID。相同 Redis DB 中的不同部署必须显式使用不同 namespace。不要拿已有指标前缀或租户字段推断部署身份。

- [x] **1. 写失败测试与唯一协议 fixture。**

```json
{"schema_version":1,"namespace":"default","instance_id":"00000000-0000-4000-8000-000000000001","version":"v1.2.3","build_kind":"release"}
```

```go
func TestMemoryExpiryAndDedup(t *testing.T) {
    now := time.Unix(100,0)
    s := NewMemoryStore("default", func() time.Time { return now })
    n := Node{1,"default","00000000-0000-4000-8000-000000000001","v1.2.3","release"}
    ctx := context.Background()
    if err := s.Upsert(ctx,n); err != nil { t.Fatal(err) }
    if err := s.Upsert(ctx,n); err != nil { t.Fatal(err) }
    nodes, err := s.List(ctx)
    if err != nil || len(nodes)!=1 { t.Fatal(nodes,err) }
    now = now.Add(TTL)
    nodes, err = s.List(ctx)
    if err != nil || len(nodes)!=0 { t.Fatal(nodes,err) }
}
```

- [x] **2. 验证红灯。**`go test ./pkg/versionregistry -run TestMemoryExpiryAndDedup -count=1`。
- [x] **3. 实现记录。**Memory 使用 mutex + `{Node, expiresAt}` map；Upsert 覆盖，List 删除到期记录。Redis 使用 SET JSON EX/TTL、SCAN 固定前缀、GET、PTTL；跳过已到期 key，无 TTL 或损坏记录返回可诊断错误而不是伪装成零节点。Delete 仅删除本 namespace 的精确实例键；Store 不关闭借用的 Redis client。

```go
payload, err := json.Marshal(node)
if err != nil { return err }
return client.Set(ctx, key, payload, TTL).Err()
```

  SchemaVersion 必须为 1，namespace 必须匹配，版本文本最大 128 字节，build_kind 仅 release/dev。版本为空可记录为未知。GroupNodes 按正规化稳定版本与 build_kind 分组，非法版本保留原始文本，空值显示 unknown；排序稳定，不包含实例 ID。

- [x] **4. 绿灯。**`go test -race ./pkg/versionregistry -count=1`。miniredis 使用 `FastForward(TTL)`；覆盖不同 namespace、重复上报、混合版本、未知版本、Redis 失败和精确删除，不连接用户 Redis。
- [x] **5. 审阅后只提交本任务文件。**

## Task 5: Gateway 身份与双通道上报

**目录：** G；依赖 Task 4 的协议，不引入对 Admin 模块的 Go 依赖。

**Files**

- Create: `pkg/versionreport/protocol.go`, `sender.go`, `reporter.go`, `reporter_test.go`, `testdata/node-v1.json`
- Create: `cmd/server/version_test.go`
- Modify: `cmd/server/main.go`, `internal/bootstrap/engine.go`

**Interfaces**

  在 `versionreport` 中复制 Task 4 的 Node wire 字段，fixture 必须逐字节一致。

```go
type Sender interface { Send(context.Context, Node) error }
func NewRedisSender(client *redis.Client) Sender
func NewHTTPSender(client *http.Client, adminURL, token string) Sender
func SelectSender(rdb *redis.Client, client *http.Client, adminURL, token string) Sender
func Run(ctx context.Context, sender Sender, node Node, after func(time.Duration)<-chan time.Time)
```

- [x] **1. 写失败测试。**测试文件定义 recordingSender，并通过 cancel 结束循环。

```go
type recordingSender struct{ calls chan Node }
func (s recordingSender) Send(_ context.Context, n Node) error { s.calls <- n; return nil }
func TestRunReportsImmediately(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    ch := make(chan Node,1)
    done := make(chan struct{})
    n := Node{1,"default","00000000-0000-4000-8000-000000000001","v1.2.3","release"}
    go func(){ Run(ctx,recordingSender{ch},n,func(time.Duration)<-chan time.Time{return make(chan time.Time)});close(done) }()
    select { case got:=<-ch: if got.Version!=n.Version {t.Fatal(got)}
    case <-time.After(time.Second): t.Fatal("no initial report") }
    cancel()
    select {case <-done: case <-time.After(time.Second):t.Fatal("did not stop")}
}
```

- [x] **2. 验证红灯。**`go test ./pkg/versionreport -run TestRunReportsImmediately -count=1`。
- [x] **3. 实现 sender 和接线。**Redis sender 按 Task 4 的精确 key 和 TTL 写 JSON；HTTP sender POST `/api/v1/gateway/version`，只携带 `X-Sync-Token`，请求 5 秒超时；未配置 token 时不发送匿名报告。Redis client 非 nil 时优先 Redis，不能失败后擅自切 HTTP。

  无可用通道时 SelectSender 返回 nil，Run 直接返回，不创建空转循环。Run 有 sender 时立即发送一次，此后 `after(30*time.Second)`；一次失败只记录限频错误，继续下次上报；取消停止，不阻断业务。每个 Gateway 进程只生成一次 `uuid.NewString()`。

  main 增加默认 dev 的 `VERSION` 和 `BUILD_KIND`，`-version` 在加载配置前退出；启动前写入 viper 的 `runtime.version`、`runtime.build_kind`。在 `internal/bootstrap/engine.go` 成功构建末端启动 reporter，并加入既有 cleanup；`config_source=embedded` 时完全禁用 reporter。namespace 读取 `GATEWAY_VERSION_NAMESPACE`，空值 default；不改请求指标 filter。

- [x] **4. 绿灯。**`go test -race ./pkg/versionreport ./internal/bootstrap -run 'Test.*(Version|Report|Sender)' -count=1`，另运行 `go test ./cmd/server -run TestVersion -count=1`。覆盖 Redis-only 不调用 HTTP、HTTP token、401/404/超时不阻断、停止、同实例重复发送、embedded 零上报。
- [x] **5. 审阅后仅提交本任务文件。**不改 `pkg/filters/outbound/status_collector.go` 的聚合语义。

## Task 6: Admin 汇总服务与生命周期

**目录：** A；依赖 Tasks 1、3、4。

**Files**

- Create: `internal/versionstatus/service.go`, `view.go`, `service_test.go`
- Create: `internal/mods/systemversion/main.go`, `wire.go`, `provider.go`
- Modify: `internal/config/config.go`, `internal/mods/mods.go`, `internal/wirex/wire_gen.go`
- Test: `internal/mods/systemversion/provider_test.go`

**Interfaces**

```go
// package versionstatus
type GatewayView struct {
    Status string `json:"status"` // observed | unknown | unavailable | not_applicable
    Scope string `json:"scope"`   // shared | this_admin
    Groups []versionregistry.Group `json:"groups"`
}
type Summary struct {
    Identity productversion.Identity `json:"identity"`
    Gateway GatewayView `json:"gateway"`
    CanManageUpdates bool `json:"can_manage_updates"`
}
type ComponentView struct {
    Component string `json:"component"`
    Current string `json:"current"`
    Latest string `json:"latest,omitempty"`
    State string `json:"state"`
    Count int `json:"count,omitempty"`
    Source updatecheck.SourceState `json:"source"`
}
type Updates struct {
    Enabled bool `json:"enabled"`
    Components []ComponentView `json:"components"`
    RetryAfterSeconds int `json:"retry_after_seconds"`
}
type Service struct {
    identity productversion.Identity
    store versionregistry.Store
    checker *updatecheck.Checker
    scope string
}
func New(identity productversion.Identity, store versionregistry.Store, checker *updatecheck.Checker, scope string) *Service
func (s *Service) Summary(ctx context.Context, canManage bool) (Summary,error)
func (s *Service) Updates(ctx context.Context) (Updates,error)
func (s *Service) Check(ctx context.Context) (Updates,error)
func (s *Service) Report(ctx context.Context, node versionregistry.Node) error
func (s *Service) Start()
func (s *Service) Close()
func Compose(identity productversion.Identity, groups []versionregistry.Group, state updatecheck.CheckResult) Updates
```

- [x] **1. 写失败测试。**无需启动应用和数据库。

```go
func TestComposeDoesNotTrustStaleCandidate(t *testing.T) {
    id:=productversion.Identity{"professional","release",productversion.Build{"v1.0.0","release"}}
    states:=updatecheck.CheckResult{Enabled:true,Sources:map[string]updatecheck.SourceState{
        "admin":{Status:"ready",Candidate:&updatecheck.Candidate{Version:"v2.0.0"},Stale:true},
    }}
    got:=Compose(id,nil,states)
    for _, c:=range got.Components { if c.State=="available" {t.Fatal("stale update",c)} }
}
```

- [x] **2. 验证红灯。**`go test ./internal/versionstatus -run TestComposeDoesNotTrustStaleCandidate -count=1`。
- [x] **3. 实现 Compose、service 和 provider。**Compose 是纯函数，Updates.Enabled 复制自 CheckResult.Enabled；优先保留 disabled/unavailable/no_candidate/stale，再在未过期 ready 状态调用 Compare。每次读取 Updates 使用当前 Store.List，不缓存 Gateway 的最终比较结果；standalone 仅产生一个 standalone 单元。

  Config 增加 `UpdateCheck`（Enabled 默认 true，IntervalSeconds=21600，TimeoutSeconds=5，CooldownSeconds=60）和 `Gateway.VersionNamespace`。provider 显式解析 `UPDATE_CHECK_ENABLED`、`UPDATE_CHECK_INTERVAL_SECONDS`、`GATEWAY_VERSION_NAMESPACE`，不假设现有 TOML loader 自动处理环境变量。

  provider 借用现有 `*redis.Client`：非 nil 创建 RedisStore，否则 MemoryStore；standalone 可使用空 MemoryStore，但永不展示节点分布。关闭检查仍创建只读服务，但不注册外部任务。来源 map 的 key 固定为 `admin`、`gateway`、`standalone`：professional 注册前两个，已确认 homebrew 的 standalone 仅用 `standalone` key 注册 NewHomebrewSource；未知身份/渠道注册空来源 map。provider 检查 NewChecker 返回的 error，配置错误不启动检查器。

  新 `systemversion.SystemVersion` 持有 Service 和 `*rbac.RBAC`，提供与其他 mod 相同的 Init/RegisterV1Routers/Release 方法。把模块加入 `internal/mods/mods.go` 的 Set、结构体和调用链；Release 在释放依赖前停止 checker。provider 放在本模块，避免 rbac/api 反向 import rbac 的循环依赖。

```go
// package systemversion
type SystemVersion struct {
    Service *versionstatus.Service
    RBAC *rbac.RBAC
}
func ProvideService(ctx context.Context, rdb *redis.Client) (*versionstatus.Service, func(), error)
```

  provider 的清理函数调用幂等的 Service.Close，并由 Wire 纳入清理链，保证后续模块初始化失败时也不会遗留检查 goroutine；它不能关闭借用的 Redis client。

- [x] **4. 绿灯与生成代码。**

```sh
go run github.com/google/wire/cmd/wire ./internal/wirex
go test ./internal/versionstatus ./internal/mods/systemversion -count=1
go test ./internal/bootstrap ./adminapp -run 'Test.*(Version|Identity)' -count=1
```

  覆盖节点过期后旧提醒消失且源调用次数不增加、一个组件失败不掩盖另一个、读取节点失败不冒充零节点、关闭开关与环境变量覆盖。不要手改 Wire 生成代码。

- [x] **5. 审阅生成差异及关闭路径，再仅提交本任务文件。**

## Task 7: API、可委派权限和文档

**目录：** A；依赖 Task 6。

**Files**

- Create: `internal/mods/systemversion/api.go`, `auth.go`, `api_test.go`, `auth_test.go`
- Modify: `internal/mods/systemversion/main.go`, `internal/bootstrap/http.go`, `configs/menu_cn.json`, `configs/menu.json`
- Modify generated: `internal/swagger/docs.go`, `internal/swagger/swagger.json`, `internal/swagger/swagger.yaml`

**Interfaces**

```text
GET  /api/v1/current/version       -> Summary; 必须登录，不要求更新权限
GET  /api/v1/system/updates        -> Updates; 必须有更新权限
POST /api/v1/system/updates/check  -> Updates; 必须有更新权限；disabled/cooldown 不联网
POST /api/v1/gateway/version       -> 部署同步 token 校验后写 Node，不走用户登录
```

```go
func CanManage(isRoot bool, roles []string, enforce func(...interface{})(bool,error)) bool
```

- [x] **1. 写权限失败测试。**

```go
func TestCanManageFailsClosed(t *testing.T) {
    if !CanManage(true,nil,nil) {t.Fatal("root must be allowed")}
    if CanManage(false,[]string{"admin"},nil) {t.Fatal("role name is not authority")}
    enforce:=func(args ...interface{})(bool,error){
        return args[0]=="operators" && args[1]=="/api/v1/system/updates/check" && args[2]=="POST",nil
    }
    if !CanManage(false,[]string{"operators"},enforce) {t.Fatal("delegated user denied")}
    if CanManage(false,[]string{"viewer"},enforce) {t.Fatal("viewer allowed")}
}
```

- [x] **2. 验证红灯。**`go test ./internal/mods/systemversion -run TestCanManageFailsClosed -count=1`。
- [x] **3. 实现权限和路由。**Root 从 `util.FromIsRootUser` 取得；角色取 `util.FromUserCache(ctx).RoleIDs`。以 `POST /api/v1/system/updates/check` 为统一能力判定资源；enforce 为空/报错必须 deny。全局 Casbin 被关闭时仍不能自动授予非 Root 更新权限。

```go
func CanManage(root bool, roles []string, enforce func(...interface{})(bool,error)) bool {
    if root { return true }
    if enforce == nil { return false }
    for _, role := range roles {
        ok,err:=enforce(role,"/api/v1/system/updates/check","POST")
        if err==nil && ok { return true }
    }
    return false
}
```

  新增“版本更新管理”授权节点同时关联 GET results 和 POST check 两个 API，不自动给已有普通角色授权。新 current/version 只跳过 Casbin，不跳过登录；新 gateway/version 精确加入用户 Auth/Casbin 跳过清单，handler 必须校验非空 `GATEWAY_SYNC_TOKEN` 和 `X-Sync-Token`，使用常量时间比较。限制 Node 请求体 4 KiB，校验 namespace/schema/UUID 后写入。

  API 使用现有 `util.ResSuccess/ResError`；无权限 `errors.Forbidden`，关闭检查返回明确 disabled、冷却返回 Retry-After 和剩余秒数（不得被映射成成功检查）。全局“关于”的 capability 由后端返回；普通 Summary 不包含目标版本、发布 URL、节点 ID 或 token。

- [x] **4. 绿灯。**`go test ./internal/mods/systemversion -count=1`。用 Gin httptest 覆盖匿名、Root、授权/未授权角色、伪造用户名、Casbin disabled、非法同步 token、超大 Node、cooldown/disabled 源调用为零。用临时 DB 验证新菜单授权数据，测试结束销毁临时库；生成 Swagger 后核对只有新增接口相关变化。
- [x] **5. 审阅后仅提交本任务文件。**

## Task 8: 前端状态转换和数据获取

**目录：** A/frontend；依赖 Task 7。

**Files**

- Create: `src/utils/system-version.js`, `src/composables/useSystemVersion.js`, `tests/system-version.test.mjs`
- Modify: `src/apis/modules/system.js`

**Interfaces**

```text
system-version.js:
  hasAvailableUpdate(summary, updates) -> boolean
  formatIdentity(identity) -> string（纯显示；unknown 不猜测）
  copyVersionText(summary) -> string（发行形态、当前组件版本和组数量）
useSystemVersion.js:
  useSystemVersion() -> { summary, updates, fallbackVersion, load, check,
                         checking, error, retryAfterSeconds }
  load() -> Promise<void>；check() -> Promise<void>
  其他返回值为 Vue ref/computed，组件直接绑定
```

- [x] **1. 写 Node 失败测试，不引入新的前端单测框架。**

```js
import test from 'node:test'
import assert from 'node:assert/strict'
import { hasAvailableUpdate } from '../src/utils/system-version.js'
test('only authorized and fresh available results create a badge', () => {
    const summary = { can_manage_updates: true }
    const updates = { enabled: true, components: [{ state: 'available', source: { stale: false } }] }
    assert.equal(hasAvailableUpdate(summary, updates), true)
    assert.equal(hasAvailableUpdate({ can_manage_updates: false }, updates), false)
    updates.components[0].source.stale = true
    assert.equal(hasAvailableUpdate(summary, updates), false)
})
```

- [x] **2. 验证红灯。**`node --test tests/system-version.test.mjs`。
- [x] **3. 实现纯转换和 API。**

```js
export function hasAvailableUpdate(summary, updates) {
    return summary?.can_manage_updates === true && updates?.enabled === true &&
        updates.components?.some((item) => item.state === 'available' && !item.source?.stale) === true
}
```

  `system.js` 增加 `getVersionSummary()`、`getUpdates()`、`checkUpdates()`，对应 Task 7 路由。composable 加载 Summary 后，仅在 capability=true 时取 Updates；使用既有 request 封装处理响应。首次进入、窗口重新获得焦点和“关于”打开时读缓存结果；页面可见时每 30 秒刷新缓存，隐藏时暂停，卸载清理 timer/listener。这些 GET 不触发公网。

  后端不可用时回退 `__APP_INFO__.version` 并标记构建信息，不构造可信 identity。服务端取消权限后立即清除旧更新专属数据。手动检查禁用和倒计时由服务端结果控制，客户端按钮禁用不能替代后端限制。

- [x] **4. 绿灯。**`node --test tests/system-version.test.mjs`。补充 current/ahead/uncomparable/no_candidate/disabled/stale、权限撤销和混合组复制文本的断言。

```sh
npx prettier --config .prettierrc --write src/utils/system-version.js src/composables/useSystemVersion.js src/apis/modules/system.js
```

- [x] **5. 审阅后仅提交本任务文件。**

## Task 9: 侧栏、关于弹窗和浏览器验收

**目录：** A/frontend；依赖 Task 8。

**Files**

- Modify: `src/layouts/BasicLayout.vue`, `src/layouts/components/SidebarVersion.vue`, `src/layouts/components/BasicSide.vue`, `src/layouts/components/SystemAboutDialog.vue`
- Modify only necessary wiring: `src/layouts/components/BasicHeader.vue`（保留用户现有改动）
- Modify: `src/locales/lang/zh-CN/settingDrawer.js`, `src/locales/lang/en-US/settingDrawer.js`
- Modify: `tests/system-about.browser.mjs`, `tests/fixtures/system-about.js`

**Interfaces:** BasicLayout 只持有一个 `useSystemVersion()`；各布局入口共享同一 summary 和 badge，不各自创建检查器。SystemAboutDialog 使用 summary/updates/capability，发出 `check` 事件，不自行请求公网。

- [x] **1. 扩展既有浏览器失败测试。**沿用现有 `.route('**/api/**')` 模拟后端及 fixture，不启动真实 API。

```js
assert.equal(await page.getByRole('dialog').count(), 0) // 不主动弹窗
await versionButton.click()
const dialog = page.getByRole('dialog', { name: '关于 TokenLive' })
assert.match(await dialog.innerText(), /专业版/)
assert.match(await dialog.innerText(), /v1\.2\.3.*2|2.*v1\.2\.3/s)
assert.match(await dialog.innerText(), /最近 3 分钟/)
assert.equal(await dialog.getByRole('button', { name: '检查更新' }).count(), 1)
```

  把测试 fixture 的 API 数据改为 Task 7/8 结构；测试另设普通用户、stale、disabled、单机版、unknown 渠道场景并记录 POST 次数，连续点击不得突破服务端模拟冷却。

- [x] **2. 验证红灯。**启动仅本地 Vite：`npm run dev -- --host 127.0.0.1 --port 9211 --strictPort`；独立终端运行 `node tests/system-about.browser.mjs`。沿用该脚本的 `PLAYWRIGHT_MODULE` 配置；依赖不可用必须先解决测试环境，不能把未运行记为通过。
- [x] **3. 实现组件和中英文文案。**增加徽标、逐单元状态、检查时间、已过期历史说明、Gateway 统计范围、复制指引；普通用户不渲染更新专属内容。保留键盘打开/关闭、焦点回归、折叠侧栏、所有菜单布局和复制降级逻辑。

```vue
<a-tag v-if="hasUpdate" color="blue">{{ $t('app.about.updateAvailable') }}</a-tag>
<a-button v-if="summary?.can_manage_updates"
    :disabled="!updates?.enabled || checking || retryAfterSeconds > 0"
    @click="$emit('check')">
    {{ $t('app.about.checkUpdates') }}
</a-button>
```

  字符串全部由 `$t` 处理。新窗口链接校验来源并设置 `rel="noopener noreferrer"`；不用 v-html 渲染 release 文本。Homebrew 指令由自身模板生成，未知渠道不提供 brew 专属动作；services restart 明确为条件性手动步骤。

- [x] **4. 绿灯与格式化。**

```sh
npx prettier --config .prettierrc --write src/layouts/BasicLayout.vue src/layouts/components/SidebarVersion.vue src/layouts/components/BasicSide.vue src/layouts/components/SystemAboutDialog.vue src/layouts/components/BasicHeader.vue src/locales/lang/zh-CN/settingDrawer.js src/locales/lang/en-US/settingDrawer.js
node --test tests/system-version.test.mjs
node tests/system-about.browser.mjs
npm run build:prod
```

  截图检查浅/深主题、折叠、长版本和混合组。若 Prettier 修改了用户现有 BasicHeader 改动，只保留格式化后的同等内容，不覆盖回旧版。
- [x] **5. 分块审阅该文件；未获用户授权不把其原有改动一起提交。**其他本任务文件可以独立提交，保留明确说明。

## Task 10: standalone 身份与 Homebrew 安装标记

**目录：** S；依赖 Tasks 1、6、7。此仓库不自动 commit。

**Files**

- Modify: `cmd/tokenlive/main.go`, `internal/assemble/assemble.go`, `packaging/homebrew/tokenlive.rb`, `scripts/brew-install-local.sh`
- Modify: `cmd/tokenlive/main_test.go`, `internal/assemble/assemble_test.go`, `scripts/update_homebrew_formula_test.py`
- Create: `internal/assemble/version_test.go`, `cmd/tokenlive/install_channel_test.go`
- Modify: `go.mod`, `go.sum`（仅在真实依赖版本可用时；本地验证用临时 go.work）
- Modify: `configs/admin/menu_cn.json`, `configs/admin/menu.json`（同步新增授权节点）

**Interfaces**

```go
// assemble.Options adds:
// Version string; BuildKind string; InstallChannel string
func adminIdentity(version, kind, channel string) productversion.Identity
// package main
func installedChannel(executable string) string
```

- [x] **1. 写失败测试。**

```go
func TestInstalledChannelRequiresMarker(t *testing.T) {
    root:=t.TempDir()
    exe:=filepath.Join(root,"bin","tokenlive")
    if err:=os.MkdirAll(filepath.Dir(exe),0755); err!=nil {t.Fatal(err)}
    if err:=os.WriteFile(exe,[]byte("test"),0755); err!=nil {t.Fatal(err)}
    if got:=installedChannel(exe);got!="unknown"{t.Fatal(got)}
    marker:=filepath.Join(root,"libexec","tokenlive-install-channel")
    if err:=os.MkdirAll(filepath.Dir(marker),0755);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(marker,[]byte("homebrew\n"),0644);err!=nil{t.Fatal(err)}
    if got:=installedChannel(exe);got!="homebrew"{t.Fatal(got)}
}
```

- [x] **2. 验证红灯。**先用受控临时 go.work 指向本次 A/G/S 工作目录，再 `go test ./cmd/tokenlive -run TestInstalledChannelRequiresMarker -count=1`；不要给已跟踪 go.mod 加本机绝对路径 replace。
- [x] **3. 实现安装来源和身份传递。**installedChannel 用 `filepath.EvalSymlinks` 定位实际可执行文件，再读取其上级安装目录下 `libexec/tokenlive-install-channel`；只有文件内容恰为 homebrew 才确认，读取失败/其他内容都 unknown，不启动 brew 子进程。

```ruby
# Formula install 阶段，只有真实 Homebrew 安装创建，不放入通用 tarball：
(libexec/"tokenlive-install-channel").write("homebrew\n")
```

  自带 `brew-install-local.sh` 在自己的 KEG 中创建同名标记；仅增加安装步骤，不运行该脚本验收。main 将自身 version/buildKind/installedChannel 传给 assemble.Options；`adminIdentity` 返回 standalone 身份，并通过 `adminapp.Options.Identity` 传入。standalone 的 Gateway 保持 embedded，零专业版节点上报。

  通用 tarball 不带安装标记，直接运行仍为单机版但渠道 unknown。Homebrew 软链接、前台运行与 service 指向同一标记；无需通过操作系统猜测。

- [x] **4. 绿灯。**`go test ./cmd/tokenlive ./internal/assemble -run 'Test.*(Version|Identity|InstalledChannel)' -count=1`；`python3 -m unittest discover -s scripts -p '*_test.py'`。增加无标记、错误标记、软链接、整包版本不是 Admin 版本、菜单资源同步测试。用临时 SQLite 配置验证公开版本和 current/version，无真实数据写入。
- [x] **5. 保留已验证差异和测试证据，不提交。**若需要更新依赖，先用实际已发布的 A/G tag；发布权限未获得时不发 tag，临时 go.work 只作联合测试。

## Task 11: 二进制、镜像和发布脚本的版本一致性

**目录：** A、G、S；依赖 Tasks 1、5、10。

**Files**

- A Modify: `Makefile`, `deploy/build/Dockerfile`, `deploy/build/CN.Dockerfile`, `.github/workflows/release.yml`, `.github/workflows/deploy.yml`
- A Create: `scripts/version-build_test.sh`
- G Modify: `Makefile`, `deploy/build/Dockerfile`, `.github/workflows/release.yml`
- G Create: `scripts/version-build_test.sh`
- S Modify: `scripts/package-release.sh`, `scripts/publish-brew-release.sh`, `scripts/publish-linux-release.sh`, `.github/workflows/release-brew.yml`
- S Create: `scripts/package-version_test.py`

**Interfaces:** 所有正式产物注入 version 与 build kind 两个字段；开发构建默认 dev。A/G 使用 `main.VERSION`、`main.BUILD_KIND`，S 使用 `main.version`、`main.buildKind`。

- [x] **1. 写失败的 Make dry-run 和包装脚本测试。**以下断言加入 A 的 shell 测试；只运行 `make -n`，不运行 npm ci。

```sh
#!/usr/bin/env bash
set -euo pipefail
output="$(RELEASE_TAG=v9.8.7 BUILD_KIND=release make -n build-frontend build)"
[[ "$output" == *"VITE_APP_VERSION=v9.8.7"* ]]
[[ "$output" == *"main.VERSION=v9.8.7"* ]]
[[ "$output" == *"main.BUILD_KIND=release"* ]]
```

  G 的测试同样断言 `VERSION` 和 `BUILD_KIND` 在 make build 命令中。S 的 Python 测试用临时目录与假 go/npm/rsync 工具记录参数，不触碰真实安装；断言正式包强制重建匹配版本的前端、带 buildKind、通用包不含 Homebrew 安装标记。

- [x] **2. 验证红灯。**分别运行 A/G `bash scripts/version-build_test.sh`、S `python3 -m unittest discover -s scripts -p '*_test.py'`。
- [x] **3. 实现元数据注入。**

```make
# Admin: preserve external RELEASE_TAG; local default remains a dev build.
RELEASE_TAG ?= $(RELEASE_VERSION).$(GIT_COUNT).$(GIT_HASH)
BUILD_KIND ?= dev
# All build commands append:
# -X main.VERSION=$(RELEASE_TAG) -X main.BUILD_KIND=$(BUILD_KIND)
```

  正式 release job 显式传 BUILD_KIND=release，预发布标签仍由稳定版过滤拒绝；普通部署/本地构建不因为标签看起来像 semver 就自动声明 release。Docker 各 stage 的 VERSION/RELEASE_TAG 和前端 VITE_APP_VERSION 必须统一；`latest` 标签不能成为二进制版本。

  S package-release 对正式构建不可复用不明版本的旧 frontend/dist；显式设置 VERSION、buildKind 和 VITE_APP_VERSION。只在 tap 推送完成后认为 Homebrew 发布完成；SKIP_TAP 不产生“已就绪”声明。不引入发布 JSON 索引。

- [x] **4. 绿灯和真实本地构建烟雾验证。**先重跑脚本测试，再在临时目录构建：

```sh
build_tmp="$(mktemp -d)"
go build -ldflags "-X main.VERSION=v9.8.7 -X main.BUILD_KIND=release" -o "$build_tmp/app" .
"$build_tmp/app" version
```

  上述为 A；G 用 `./cmd/server` 和 `-version`，S 用 `./cmd/tokenlive` 与自身符号名及 `-version`。仅运行不加载配置的版本命令。镜像可用时仅本地 build/load 并覆盖 entrypoint 读取版本，不 push、不启动服务；缺失 Docker 环境要报告未验证。

- [x] **5. A/G 分仓审阅和授权提交；S 只保留差异。**不得由本任务实际触发发布工作流。

## Task 12: 专业版独立镜像版本与旧配置兼容

**目录：** D；依赖 Task 11。

**Files**

- Modify: `docker-compose.yml`, `docker-compose.build.yml`, `build-images.sh`, `install.sh`, `.env.example`, `README.md`, `README.zh.md`
- Create: `tests/component_versions_test.py`

**Interfaces**

```text
ADMIN_VERSION   > VERSION > latest
GATEWAY_VERSION > VERSION > latest
UPDATE_CHECK_ENABLED=true
UPDATE_CHECK_INTERVAL_SECONDS=21600
GATEWAY_VERSION_NAMESPACE=default
```

- [x] **1. 写不启动容器的失败测试。**Python unittest 创建临时 env 文件，调用 `docker compose --env-file <临时文件> -f docker-compose.yml config --images`，断言独立版本与 fallback。

```python
def test_component_versions(self):
    with tempfile.TemporaryDirectory() as d:
        env_file = pathlib.Path(d) / "test.env"
        env_file.write_text("VERSION=v1.0.0\nADMIN_VERSION=v1.1.0\nGATEWAY_VERSION=v1.2.0\n")
        result = subprocess.check_output(
            ["docker", "compose", "--env-file", str(env_file),
             "-f", str(ROOT / "docker-compose.yml"), "config", "--images"],
            cwd=ROOT, text=True)
        self.assertIn("ghcr.io/tokenlive/tokenlive-admin:v1.1.0", result)
        self.assertIn("ghcr.io/tokenlive/tokenlive-gateway:v1.2.0", result)
```

  测试模块显式导入 pathlib/subprocess/tempfile/unittest，ROOT 为该测试文件父目录的父目录。测试进程清理 ADMIN_VERSION/GATEWAY_VERSION/VERSION 等继承环境，避免用户环境覆盖 fixture。

- [x] **2. 验证红灯。**`python3 -m unittest discover -s tests -p 'component_versions_test.py'`；该命令只要求 Compose CLI，不需要 Docker daemon。
- [x] **3. 实现兼容。**

```yaml
services:
  admin:
    image: "${REGISTRY:-ghcr.io/tokenlive}/tokenlive-admin:${ADMIN_VERSION:-${VERSION:-latest}}"
  gateway:
    image: "${REGISTRY:-ghcr.io/tokenlive}/tokenlive-gateway:${GATEWAY_VERSION:-${VERSION:-latest}}"
```

  build override 使用相同规则，传递真实组件 VERSION/BUILD_KIND。`build-images.sh` 增加 `--admin-version`、`--gateway-version`，保留 `--version`；每种组件的 build/tag/push 使用同一解析结果。`install.sh` 接受对应参数/环境变量，保留已有值，升级路径的镜像标识也按组件解析，不残留统一 version_val 假设。

  Compose 向 Admin 传递检查开关/周期和 namespace，向 Gateway 传递相同 namespace。不更改真实 `.env`；说明纯内存多 Admin 只显示本实例观测、共用 Redis 的不同部署须区分 namespace。

- [x] **4. 绿灯。**重跑 Python 测试，补充仅 VERSION、无 VERSION、仅一个组件覆盖和 build override。对 build/install 脚本使用假 docker 记录 argv 做分支测试；不得运行真实 `install.sh --upgrade`、docker down/rmi/pull/up。
- [x] **5. 审阅后仅提交本任务文件。**

## Task 13: 联合验收、兼容矩阵和交付

**目录：** A/G/S/D；依赖全部任务。

**Files**

- A Create: `internal/versionstatus/integration_test.go`
- A Modify: `docs/CONFIGURATION.md`, `docs/quickstart.md`
- G Create: `docs/version-reporting.md`
- S Modify: `README.md`
- D 文档沿用 Task 12

**Interfaces:** 不新增产品接口；使用前述 Node fixture、API 和 source adapter，验证跨仓库契约。

- [x] **1. 增加联合回归测试。**将 Gateway sender 发往测试 Gin handler，再从 Admin Summary/Updates 读取分布。受控 source 返回 v1.2.4；报告两个 v1.2.3 节点与一个 v1.2.4 节点，断言只有旧组 available。推进 registry 时钟至 3 分钟，断言旧组消失但 source 请求数不增加。

```text
断言序列：
source.calls=1 -> report(old1), report(old2), report(new1)
Summary.groups=[v1.2.3×2,v1.2.4×1]
Updates.old.state=available; Updates.new.state=current
expire records -> Summary.gateway.status=unknown
Updates 不保留旧组 available；source.calls 仍为 1
```

  测试 Go 文件使用真实 NewMemoryStore/NewChecker/versionstatus.New 和 HTTP httptest，不创建真实用户或发送真实网关业务。错误 token 不增加节点数，关闭公网检查不停止内部报告。

- [x] **2. 先运行联合回归测试并记录实际结果。**发现接线缺陷时，以该失败为依据修复后重跑；如果前面任务已使测试通过，则如实记录首次即通过，不为了制造红灯破坏正确实现。故障注入用于断言既定失败处理，而不是伪造开发过程。
- [x] **3. 完成兼容矩阵和配置文档。**记录新版 Admin+旧 Gateway 为未知、旧 Admin+新版 Gateway 上报失败不阻断、standalone unknown 安装渠道、内存多 Admin 观测范围、首次人工升级、手动命令及可委派权限。说明所有测试数据如何自动清理。

  联合源码验证通过临时 go.work 使用本次 A/G/S 目录，不提交本机 replace/go.work。真正发布时再选取已存在且经过验证的模块 tag，不编造未来版本。

- [x] **4. 执行最终验证命令并记录结果。**

```text
A: go test -race ./pkg/productversion ./pkg/versionregistry ./internal/updatecheck ./internal/versionstatus ./internal/mods/systemversion
A: go test ./adminapp ./internal/bootstrap
A/frontend: node --test tests/*.test.mjs
A/frontend: node tests/system-about.browser.mjs
A/frontend: npm run build:prod
G: go test -race ./pkg/versionreport
G: go test ./internal/bootstrap ./cmd/server
S: go test ./cmd/tokenlive ./internal/assemble
S: python3 -m unittest discover -s scripts -p '*_test.py'
D: python3 -m unittest discover -s tests -p '*_test.py'
所有仓库: git diff --check
```

  先审查测试初始化，确认不读取真实外部服务配置；不得贸然执行含真实数据库测试的全仓 `go test ./...`。缺失工具或环境的检查逐项标记未运行，不声称全部通过。公网只读烟雾检查是独立可选项，必须设 5 秒超时，不把它变成单测依赖；网络不可用保持设计中的不可检查状态。

- [x] **5. 完成逐任务审阅和交付。**核对所有 spec 验收点与以下覆盖表，说明已实施/未验证项目及保留的用户改动。S 不自动提交；其他仓库只按授权提交本任务文件。不推送、不发布、不重启用户服务。

## 规格覆盖与计划自审

| Spec 验收项 | 实施任务 |
|---|---|
| 1–4：身份、版本来源、稳定比较、回退 | 1、2、6、8、10、11 |
| 5：Homebrew 就绪与未知渠道 | 2、10、11 |
| 6–9：关闭、周期、失败、过期、组件隔离 | 3、6、8、13 |
| 10–11：权限与非打扰交互 | 7、8、9 |
| 12–15：双通道、多节点、TTL、数据清理 | 4、5、6、7、13 |
| 16：二进制/镜像/内嵌版本一致 | 1、5、10、11、12 |
| 17：仅指引、无执行升级 | 2、9、10、13 |
| 18：Prettier 与实际测试 | 8、9、13 |

执行前自审已检查：类型/方法、wire/API/JSON 契约、默认配置、测试数据隔离、Homebrew 与 professional 来源边界。批准时的示例保留为计划记录，实际实现中的有依据调整见工作记录 R1–R22；例如显式 freshness 检查、联合测试位置和发布就绪顺序，均以设计要求为准。

## 执行结果与交付边界

隔离工作区：`/private/tmp/tokenlive-edition-work.e9cKUC/{admin,gateway,standalone,deploy}`；分支均为 `codex/edition-versions-update-notifications`。

2026-09-12 的最终修复前验证：

- Admin：版本身份、节点注册表、检查器、汇总和 API 的竞态测试，以及 adminapp/bootstrap 测试通过。
- Gateway：reporter 竞态测试及 bootstrap/server 测试通过。
- Standalone：真实 HTTP/Redis sender → Admin 集成竞态测试、main/assemble 测试及 22 项打包脚本测试通过。
- Deploy：27 项隔离脚本测试、原有 Shell 回归、语法检查通过；真实 Compose 仅用于只读配置解析。
- 前端：Node 22.23.1 下 39 项测试、生产构建、完整浏览器矩阵及 Prettier 检查通过。浏览器使用模拟 API，临时服务器已停止。
- 原始 Admin 的 BasicHeader 暂存 blob 保持 `035131b0882ef0e2de8beb2f1a1b02a86390d452`；原目录中另有用户的 ModelDetail 改动，均未触碰。

### 整体审查后的最后修复

- [x] HTTP 版本上报沿用已有显式 TLS 配置，默认仍校验证书。
- [x] 侧栏显示紧凑的 Gateway 混合版本摘要。
- [x] Provider 拒绝显式零检查周期，保留底层 API 的默认值语义。
- [x] “尚未检查”状态使用独立文案，不降格为未知。
- [x] 文档准确区分 Formula 运行时读取和发布端资产/公开 Release 就绪保障。
- [x] 修复回归、独立复审及最终交付记录完成。

修复提交：Admin `8b03698c81515dd8935ad57ff2c985bf72b2cb07`、Gateway `4ccd4b007974033585312dc7d234956b72e2879a`；Standalone 文档修正未提交。最终复审 F1–F5 全部 ADDRESSED，没有新问题。控制器在这些提交上重新执行 Admin/Gateway/Standalone Go 测试、39 项前端测试、生产构建、完整浏览器矩阵和格式检查，均通过。

### 未执行与保留门槛

- 尚未合并、推送、打 tag、发布、安装 Homebrew 包或重启服务；standalone 按仓库规则保持未暂存、未提交。
- Standalone 现有已发布 Admin 依赖不含新增 API。真正发布前须先取得兼容的已发布 Admin/Gateway 引用，或显式提供匹配的已审核源码，再做干净依赖构建；没有编造 tag 或提交本地 replace/go.work。
- 临时联合测试图的 genproto 覆盖仅用于测试。独立打包图已验证当前本地源码，不代表旧已发布依赖可用。
- 缺少 Docker daemon，真实镜像编译和容器生命周期未验证；真实 Homebrew 发布/安装也未执行。
- Node 18 的 CI 构建/测试未验证；现有模块类型、MockTimers 和 multi-tab 循环 chunk 警告保留为非本功能阻塞的后续事项。
- Deploy 原有卸载脚本仍用共享 VERSION 清理镜像，可能遗留独立固定版本镜像；未扩展修改。
- Provider 原有 EndpointDAL 排除和关联密钥算法保持基线行为，本功能不宣称修复该无关算法。
