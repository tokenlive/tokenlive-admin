# Homebrew 单机版点击升级实施计划

依据：[设计](../specs/2026-09-12-homebrew-click-upgrade-design.md)（已获用户确认）、
[Homebrew 固定目标研究](../research/2026-09-12-click-upgrade-homebrew.md)、
[launchd 一次性任务研究](../research/2026-09-12-click-upgrade-launchd.md)。

目标：获授权用户在“关于”中对 Homebrew 单机版点击升级，二次确认后由独立一次性任务
完成固定目标安装、服务重启与本机核验；不再要求手工复制 brew 命令。

## 全局约束（来自设计，逐条可验收）

- 仅支持：macOS、Homebrew 安装、`brew services` 托管、`uid==euid!=0`、gui 域、
  受控 tap `tokenlive/homebrew-tokenlive`、无依赖 Formula 模板。其余一律“不支持”并给原因。
- 无本机开关；Root 默认拥有新权限“单机版升级执行”，其他角色单独授权。
- 发现新版只提示；升级必须由用户发起并二次确认；确认凭据 5 分钟有效、一次性、绑定用户与目标。
- 同一安装同时最多一个任务；状态原子落盘；主服务重启后可查询。
- 固定目标：prepare 时抓取远端 Formula 字节并校验模板与版本，submit 后 worker 把
  **同一字节**快照写入 tap 内 `.tokenlive-upgrade/<task>/tokenlive.rb`，
  以 `brew upgrade --formula <快照路径>` 安装；禁止裸 `brew upgrade`、cleanup、
  dependents、auto-update（环境白名单 + 显式变量）。
- 一次性任务为同用户 launchd gui 域 job（RunAtLoad/KeepAlive=false/LaunchOnlyOnce），
  执行器复制自当前发行物，独立 label，不写登录自启动目录。
- 预检/下载失败不停服务；安装/重启/核验超时或结果未知一律 `needs_attention`；
  不自动降级、不循环重试、不删 Homebrew 锁。
- 只有核验（新 PID + `/health` 版本与可执行路径匹配目标）通过才记 `succeeded`。
- 检查关闭（update_check.enabled=false）时不允许新建升级任务；已进入安装的任务不中断。

## 架构与数据流

```
浏览器(关于) ──JWT──▶ Admin API(systemversion) ──进程内──▶ upgradehost.Host(standalone 实现)
                                                            │ prepare: 抓 Formula+校验+建任务(awaiting_confirmation)
                                                            │ submit:  验凭据→queued→复制执行器→写 plist→launchctl bootstrap
                                                            └ task:    读状态文件(+对账)
launchd gui job: <task>/executor upgrade-worker --task-id <id>
  worker: 快照入 tap → brew upgrade --formula <快照> → brew services stop/start → 轮询核验 → 终态 → 自 bootout
CLI: tokenlive upgrade-status（只读+安全对账，不加载业务配置/DB）
```

- Admin 不 import standalone；接口与 DTO 定义于 admin 公共包 `pkg/upgradehost`，
  standalone 实现之。运行时经 `adminapp.Options.HostUpgrade` 注入，
  `adminapp.New` 内 `upgradehost.Register(...)`（沿 `util.OnConfigChanged` 先例），
  独立 Admin 未注入时一律返回 `supported=false, reasons=["host_unsupported"]`。

## 文件地图

tokenlive-admin：
- 新建 `pkg/upgradehost/upgradehost.go`：Host 接口、Capability/Preparation/TaskView DTO、Register/Current。
- 改 `adminapp/app.go`：Options.HostUpgrade → bootstrap.RunConfig。
- 改 `internal/bootstrap/bootstrap.go`：RunConfig.HostUpgrade 透传并 Register。
- 改 `internal/mods/systemversion/api.go`：+4 路由处理（capability/prepare/submit/task）、Summary 增加 can_manage_upgrades。
- 改 `internal/mods/systemversion/main.go`：注册路由。
- 改 `internal/versionstatus/service.go,view.go`：Summary.CanManageUpgrades。
- 改 `configs/menu_cn.json,menu.json`：system 下新增 button `versionUpgrades`（prepare/submit/task 资源）。
- 前端 `src/apis/modules/system.js`、`src/composables/useUpgrade.js`（含可测 controller）、
  `src/layouts/BasicLayout.vue`、`src/layouts/components/SystemAboutDialog.vue`、
  `src/locales/lang/{zh-CN,en-US}/settingDrawer.js`。
- 测试：`pkg/upgradehost/*_test.go`、`internal/mods/systemversion/api_test.go` 扩展、`frontend/tests/upgrade*.test.mjs`。

tokenlive-standalone：
- 新建 `internal/upgrade/`：`identity.go`（安装/服务身份探针）、`state.go`（任务结构+原子落盘）、
  `store.go`（flock+任务目录）、`manager.go`（Capability/Prepare/Submit/Task/Reconcile）、
  `formula.go`（模板校验+远端抓取+解析）、`launchd.go`（plist 生成+launchctl 有界调用）、
  `worker.go`（阶段状态机）、`verify.go`（健康握手核验）。
- 改 `cmd/tokenlive/main.go`：`upgrade-worker`、`upgrade-status` 子命令在业务配置前分派。
- 改 `internal/assemble/assemble.go`：构建 Manager 注入 adminapp；/health 增加 version/executable 供核验握手。
- 测试：`internal/upgrade/*_test.go`（全部用注入的 Runner/Storer 替身，不触真实 brew/launchctl）。

## 任务分解

### Task 1 admin/pkg/upgradehost
接口与 DTO；不依赖 internal。单测覆盖 Register/Current 与默认值。

### Task 2 systemversion API + 权限
- `canUpgrade`：Root 直通；否则 Casbin enforce(role, POST /api/v1/system/upgrade/prepare)。
- capability 读取允许 canManageUpdates || canUpgrade。
- prepare/submit/task 仅 canUpgrade；错误区分：无权限(403)、不支持(capability 内)、
  冲突(409 upgrade_task_conflict)、目标变化(409 upgrade_target_changed)、
  凭据过期(409 upgrade_confirmation_expired)、检查关闭(409 update_check_disabled)。
- Summary.CanManageUpgrades；menu seed 增 button。
- 单测：权限矩阵 + fake Host 全路径。

### Task 3 standalone 安装识别 identity.go
可执行文件→Cellar keg→前缀；INSTALL_RECEIPT tap 校验；opt 链接归一；
LaunchAgents 内 ProgramArguments[0] 归一到本安装 opt_bin 的唯一 plist → label；
`launchctl list <label>` PID == 自身 PID；`launchctl print gui/<uid>/<label>` 仅 exit code。
输出 Installation 或有序 reasons（不满足即不支持）。

### Task 4 状态与存储 state.go/store.go
状态机与终态集合；任务 JSON 原子写（0700/0600、tmp+rename+fsync）；
安装级 flock；活动任务判定（非终态或 needs_attention 未决 latch）。

### Task 5 manager.go
- Prepare(targetVersion, user)：活动任务冲突→409；抓远端 Formula（≤64KB）→模板校验→
  版本==target→建 awaiting_confirmation 任务（快照字节、SHA256、TTL、凭据哈希）。
- Submit(taskID, credential, user)：验凭据/目标/状态→queued→复制执行器→写 plist→
  bootstrap（exit 0 继续；结果未知→needs_attention，不重试不新建）。
- Task/LatestTask：读盘+Reconcile；Reconcile 规则：
  awaiting 超 TTL→confirmation_expired；非终态且无 worker 心跳且执行器进程已消失→needs_attention；
  终态且 cleanup_pending→验证 job 不存在后清理任务目录（保留终态记录于 index）。
- Index 文件记录最近 20 条终态摘要。

### Task 6 worker.go + launchd.go + formula.go + verify.go
- worker：取锁→验 queued→写 ready(pid,start)→逐阶段“先落盘待定态再执行”：
  tap 快照写入+哈希校验→`brew upgrade --formula <snap>`（10min 预算，退出码非零→failed/安装阶段；
  超时→needs_attention）→`brew services stop --max-wait=60 tokenlive`（超时→needs_attention，
  不得 start）→`brew services start tokenlive`→核验（3min：launchctl list 新 PID + 
  GET 127.0.0.1:<port>/health 版本==target 且 executable 在本安装 Cellar 内）→succeeded+cleanup_pending→
  exec 自 bootout。brew.env 已知位置存在覆盖→预检拒绝。
- plist 字段按研究候选；launchctl 调用全部有界超时+受限解析。

### Task 7 CLI 与装配
- main.go：`upgrade-worker --task-id`、`upgrade-status` 在配置加载前分派；
  upgrade-status 打印任务/原因，不初始化 DB/HTTP。
- assemble：homebrew+darwin 时构建 Manager 传入 adminapp；/health 增加 version/executable。

### Task 8 前端
- system.js +4 API；useUpgrade.js：controller（可注入 fetch/timer）管理 capability、
  prepare→确认→submit、2s 可见轮询、终态停止、成功后触发 summary 刷新。
- SystemAboutDialog：standalone+homebrew+can_manage_upgrades 时展示“立即更新”；
  不支持/阻止原因展示；确认对话框含目标版本、重启影响、备份提醒；
  任务进行中展示阶段；failed/needs_attention 展示失败阶段+手动命令回退。
- i18n zh/en；node:test 单测 controller 状态机；Prettier。

### Task 9 验证与收尾
- go test ./...（三仓）；前端 node --test；Prettier 检查。
- 行为对齐设计验收 1–12 中本期可测项；发布门槛（真实 brew 端到端矩阵）单列不冒充已完成。
- 交付记录 handoff 文档。

## 刻意不做（本期）

- 非 Homebrew 渠道、Linux brew、root/UserName 服务、`user/<uid>` 域、多实例/自定义 plist。
- 在线修改配置页、全局发布清单、自动备份/回滚、安装中取消。
- go.mod 版本对齐与发布流程（发布时另行 bump）。
