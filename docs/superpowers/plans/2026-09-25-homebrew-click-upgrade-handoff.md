# Homebrew 单机版点击升级交付记录

日期：2026-09-25。实施依据：[设计](../specs/2026-09-12-homebrew-click-upgrade-design.md)、
[实施计划](2026-09-25-homebrew-click-upgrade.md)。

## 已交付

### tokenlive-admin

- `pkg/upgradehost`：Host 接口、Capability/Preparation/TaskView DTO、全量任务状态常量、
  哨兵错误（conflict/target_changed/confirmation_expired/invalid_credential/check_disabled/
  unsupported/not_allowed）、进程级 Register/Current（沿 `util.OnConfigChanged` 先例）。
- `adminapp.Options.HostUpgrade`：嵌入宿主注入点；独立 Admin 不注册时所有路径返回
  `host_unsupported`，不虚报能力。
- `internal/mods/systemversion`：
  - 新路由 `GET /system/upgrade/capability`、`POST /system/upgrade/prepare`、
    `POST /system/upgrade/submit`、`GET /system/upgrade/task`；
  - 权限：Root 直通；独立 Casbin 资源 `POST /api/v1/system/upgrade/prepare` 判定
    `canUpgrade`；capability 读取接受两种权限之一；Summary 新增 `can_manage_upgrades`；
  - 错误映射为稳定错误 ID（409/403），宿主未知错误保持 500。
- `configs/menu_cn.json`/`menu.json`：system 下新增 button `versionUpgrades`
  （单机版升级执行），含 4 个 API 资源；**存量安装需在角色管理里对新按钮授权，
  Root 不受影响**。
- 前端：
  - `utils/upgrade-state.js`：纯 controller（能力读取、prepare→确认→submit、2s 可见轮询、
    终态停轮、成功后刷新回调、授权丢失清态），node:test 全覆盖；
  - `composables/useUpgrade.js` + `BasicLayout.vue`：随 About 打开激活、随 summary 授权；
  - `SystemAboutDialog.vue`：独立“一键升级”区块——不支持时展示原因，运行中仅展示阶段，
    终态展示结果与有界诊断，确认面板含版本、发行说明、重启与备份警告；失败回退展示
    手动 brew 命令；i18n 中英齐全。

### tokenlive-standalone

- `internal/upgrade`：
  - `identity.go`：安装/服务身份证据链（安装标记→Cellar keg→INSTALL 前缀→受控 tap→
    opt 链接归一→唯一 LaunchAgents plist→`launchctl list` PID==自身 PID→gui 域 exit-code
    探测）；brew.env 覆盖、root、非 darwin、非标准布局一律拒绝并给出稳定原因；
  - `state.go`/`tasks.go`/`lock.go`：任务原子落盘（0700/0600、tmp+rename+fsync）、
    安装级 flock（TryLock，忙即冲突）、活动任务与 needs_attention 未决判定；
  - `formula.go`：受控 GitHub 源抓取（≤64KB）+ 已审核模板校验（无依赖、受控 URL、
    版本/URL/SHA 计数一致、稳定版解析），字节级目标固定；
  - `launchd.go`：固定 argv launchctl 调用、legacy list 严格解析、bootstrap
    already-bootstrapped 容错、bootout not-found 容错；
  - `manager.go`：Prepare（候选校验+凭据 5min/一次性/绑定用户与目标）、Submit（凭据校验→
    queued→复制执行器→写 plist→bootstrap；未知结果→needs_attention，绝不重试）、
    Reconcile（过期确认、launch 未决、验证后清理）；
  - `worker.go`/`verify.go`：一次性任务阶段机（先落盘待定态再执行）——tap 内快照写入并
    哈希校验→`brew upgrade --formula <快照>`→`services stop --max-wait=60`→`start`→
    轮询核验（`/health` 版本+可执行路径须匹配确认目标 keg）→succeeded+cleanup_pending→
    exec 自 bootout；预算超时/未知一律 needs_attention，停服超时绝不 start；
  - 环境白名单（`brewEnv`）：NO_AUTO_UPDATE/NO_INSTALL_CLEANUP/NO_INSTALLED_DEPENDENTS_
    CHECK/NO_ANALYTICS/NO_ENV_HINTS/NO_INSTALL_FROM_API。
- `cmd/tokenlive`：`upgrade-worker --task-id`（业务配置加载前分派）、
  `upgrade-status`（只读+安全对账，不初始化业务配置/DB/HTTP）。
- `internal/assemble`：/health 增加 `version`/`install_channel`/`executable`
  （核验握手）；homebrew+darwin 构建宿主 Manager 注入 adminapp；启动对账。

## 测试

- standalone `internal/upgrade`：身份链正/反路径、plist 生成与严格解析、Formula 校验
  （真实双架构模板通过；depends_on/head/外部 URL/多版本/预发布/缺 SHA 拒绝）、
  Manager 全流程（冲突/目标变化/凭据/用户绑定/幂等/未知 bootstrap）、Reconcile 三路径、
  Worker 成功路径与失败矩阵（快照篡改/安装超时/安装失败/停服超时/错误版本核验）、
  launchctl 解析、存储完整性。
- admin：upgradehost 单测；systemversion 权限矩阵与哨兵错误映射、Summary 新字段。
- 前端：`tests/upgrade-state.test.mjs` 9 项；全套 107 项通过；`npm run build:prod` 成功；
  改动文件已过 Prettier。
- `go test ./...`：两仓通过；admin `test/TestUser` 失败为存量问题（stash 后基线复现，
  与本功能无关）。
- CLI 冒烟：非 brew 环境执行 `upgrade-status` 诚实输出 `install_unsupported`。

## 未决 / 发布门槛（不得冒充已完成）

1. 真实 Homebrew 固定目标升级 + brew services 重启的端到端验证（设计 §5、研究 §5 矩阵）。
2. 自 bootout 清理、断电/登录退出、双架构矩阵的隔离环境验收。
3. 存量安装的菜单种子迁移：现库已有菜单数据的实例需手动或迁移脚本补 `versionUpgrades`。
4. 发布时 standalone `go.mod` 需 bump tokenlive-admin 版本（本地经 go.work 联动）。
5. admin 既有 `TestUser` 失败需另行修复。

## 本地开发注意

- `/Users/chenzhiguo/Projects/go.work`（gitignored）已建：use 三仓 + replace 旧
  `google.golang.org/genproto` 至 2026 版，消除新旧 genproto 歧义导入。
- 遵循仓库规则：未做 git commit。
