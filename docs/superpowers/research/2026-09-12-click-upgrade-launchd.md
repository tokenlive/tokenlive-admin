# 点击升级：macOS 一次性 launchd 执行器研究

日期：2026-09-12。范围：设计第 9 节门槛 2、3，以及门槛 4 中的服务副作用、进程退出和诊断；不评判精确版本安装机制。

## 结论

- **门槛 2：存在有官方文档依据的候选设计，但未通过端到端验证。** 同一非 root UID、独立 label、手动 bootstrap 的一次性 LaunchAgent，与被重启的主服务不是同一个 job。持久工作目录必须在 Cellar/opt 之外；关闭网页不是取消条件。不写入登录自启动目录，不使用 `launchctl submit`。自身 bootout 的退出与清理需要独立环境验证。
- **门槛 3：不能直接采纳 Homebrew 的状态查询作为归属证明；仍有阻塞。** 当前 PID、进程 UID/起始时间/实际可执行文件、可信安装收据、唯一服务 label 和确切域必须串起来。Apple 明确禁止依赖 `launchctl print` 输出结构；SDK 的替代旧 API 也弃用且不稳定。可进一步验证“legacy `list <label>` 的 PID + 域限定查询的退出码 + 安装文件核对”的保守适配，但本次未证明它能在全部目标系统上区分 gui/user。
- **门槛 4：不能把 `brew services restart` 和 worker 超时当成有界、安全结束。** 本机 restart 内部可能无限等 stop；Homebrew 的非 sudo 子命令会进入新的进程组，launchd 杀掉 worker 同进程组并不能证明所有安装后代已退出。未知状态必须 `needs_attention`，不因锁已释放而放行新任务。
- **发布判定：三项尚不能记为“已验证通过”。** 首期候选缩窄至已经验收的 macOS/Homebrew 组合、标准 GUI 用户域、标准直启 TokenLive Formula；其他情况返回不支持。若不接受该适配范围，需继续验证更一般的服务管理适配，不能静默改变服务域或用户。

以上结论对应下文列出的本机一手来源；“建议/候选/推导”不表示运行事实。

## 证据边界

本次只读查看 Apple 随系统提供的 man 源文件、Apple SDK 头文件、Homebrew 源码和项目源码。没有运行 brew、launchctl、服务或进程清单，没有创建 job、安装或重启任何服务，没有读取真实配置或业务数据。

本机 Homebrew Git HEAD 为 `a45b0a0143d4df548cdaa121fcb4282311d3eaff`，`git describe` 为 `6.0.22-310-ga45b0a0143`；已检查的 services、system_command、service、formula 源文件未显示未提交修改。因此下文是这个源码快照的结论，不是所有 Homebrew 版本的兼容承诺。

已调用官方站点 Web 检索并尝试打开 Apple 官方 Daemons and Services Programming Guide 页面；工具没有返回可读内容，**没有以未读网页支撑任何技术结论**。官方在线入口仅供后续核对：

```text
https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html
https://developer.apple.com/library/content/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/Introduction.html
```

后一地址也由本机 Apple [launchd.8:82](/usr/share/man/man8/launchd.8:82) 列为开发者文档。

## 1. 域、用户与独立生命周期

### 文档证实

1. `system` 域修改需要 root。`user/<uid>` 可独立于登录用户存在；`gui/<uid>` 是 GUI 登录域的别名。gui/user 共享部分 Mach 名称空间但有不同的服务集合，不能互换。见 [launchctl.1:32–72](/usr/share/man/man1/launchctl.1:32)。
2. `bootstrap <domain> <plist>` 把指定服务定义载入域；`bootout <domain>/<label>` 移除指定服务。缺少服务参数的 `bootout <domain>` 会移除整个域，必须禁止生成这种参数。见 [launchctl.1:90–104](/usr/share/man/man1/launchctl.1:90)。
3. `UserName`/`GroupName` 仅适用于特权 system 域，不能靠在用户 plist 中设置用户名来切换身份。见 [launchd.plist.5:135–142](/usr/share/man/man5/launchd.plist.5:135)。单独 setuid/setgid 也不能构成完整 macOS 用户上下文，见 [launchd.8:55–65](/usr/share/man/man8/launchd.8:55)。
4. Homebrew 非 root 的服务文件位于当前用户的 `Library/LaunchAgents`；域选择通常为 `gui/<uid>`，SSH 非控制台用户、sudo 环境或 uid/euid 不一致时可能选 `user/<euid>`。查询会遍历两个域并回退到裸 label 的 legacy list。见 [services/system.rb:64–150](/opt/homebrew/Library/Homebrew/services/system.rb:64)。

### 候选约束

- 要求 `uid == euid != 0`；用户名从本机 UID 账户记录解析，不信任环境变量 `USER`。主服务域与任务域都记录为结构化的 `kind + numeric UID`，不是浏览器字符串。
- 初期可只接受 **已证明当前主进程属于 `gui/<uid>/<label>`** 的安装，并把任务也 bootstrap 到同一 gui 域。不尝试建立不存在的 GUI 会话，不 `asuser`/sudo，不以改写 SSH/SUDO 环境伪造域选择。
- `user/<uid>` 不是错误域，但若接受它，服务恢复也必须明确落回原 user 域；本机 `brew services start` 没有显式域参数且会重新计算域，因此不能先在 user 域识别成功、之后无条件调用 start 到 gui 域。
- 独立 job 推导上不受只针对主服务 label 的 bootout 影响；但它仍依赖所在用户域和系统存续。**不承诺退出 macOS 登录、重启或断电后继续。** `user` 域“可存在于未登录状态”不等于永久存活保证。

域选择问题不是只读权限预检就能证实可写。真正 bootstrap/bootout 的权限及行为必须在获授权的隔离测试中确认；正常任务提交时再以操作实际结果处理失败。

## 2. 一次性 job 与文件寿命

### 文档证实

- label 唯一标识 job；`Program` 要求绝对路径，`ProgramArguments` 是 argv 数组。见 [launchd.plist.5:124](/usr/share/man/man5/launchd.plist.5:124)、[181–216](/usr/share/man/man5/launchd.plist.5:181)。
- `KeepAlive` 默认 false；true 或条件字典会导致再次启动。`RunAtLoad` 表示加载时启动一次；`LaunchOnlyOnce` 表示 job 只能运行一次。见 [launchd.plist.5:269–335](/usr/share/man/man5/launchd.plist.5:269)、[629–632](/usr/share/man/man5/launchd.plist.5:629)。
- `launchctl submit` 会在程序失败时保持其运行，不满足失败不重试。见 [launchctl.1:529–539](/usr/share/man/man1/launchctl.1:529)。
- Homebrew 自己有 bootstrap 临时 plist、返回后删除临时文件的路径，但这仅证明其源码如此调用，不能代替新执行器的端到端测试。见 [services/cli.rb:499–505](/opt/homebrew/Library/Homebrew/services/cli.rb:499)。
- TokenLive Formula 把主程序和 admin/web/libexec 安装于版本包内，把 etc/var 放在版本包外；服务直启 `opt_bin/tokenlive`。见 [tokenlive.rb:28–45](/Users/chenzhiguo/Projects/tokenlive-standalone/packaging/homebrew/tokenlive.rb:28)、[69–76](/Users/chenzhiguo/Projects/tokenlive-standalone/packaging/homebrew/tokenlive.rb:69)。

### 候选落地

任务路径示意：`<validated-prefix>/var/tokenlive/upgrades/<installation-id>/<task-id>/`。不是固定使用 `/opt/homebrew`，也不从请求接收路径。父目录及任务目录归执行 UID，目录 0700，元数据/诊断 0600，执行器 0700；拒绝不可信符号链接、非预期所有者和非受控路径。

在调用安装前，复制**当前可信发行物中专用执行器**到任务目录，固定校验值、原子落盘；不要让 job 执行 `opt/bin/tokenlive` 或旧 Cellar 路径。若复用主二进制的隐藏子命令，必须在业务配置加载、DB 初始化、HTTP 监听之前分派，且不依赖被替换的 admin/web 资源。复制文件本身不证明可独立运行：Mach-O 的 `@executable_path`、`@loader_path`、`@rpath` 会影响库解析，见 [dyld.1:205–236](/usr/share/man/man1/dyld.1:205)；发行测试须核查动态依赖并在旧包移走后运行。

候选 plist（是设计字段，不是已安装文件）：

```text
Label                 = "com.tokenlive.upgrade.<opaque-install-id>.<opaque-task-id>"
Program               = "<task-dir>/executor"
ProgramArguments      = ["<task-dir>/executor", "upgrade-worker", "--task-id", "<server-created-id>"]
WorkingDirectory      = "<task-dir>"
RunAtLoad             = true
KeepAlive             = false
LaunchOnlyOnce        = true
AbandonProcessGroup   = false
Umask                 = 63  # decimal; means octal 077
ExitTimeOut           = 15  # candidate stop grace, NOT task deadline
```

不设置启动定时器、WatchPaths、QueueDirectories、MachServices、PressuredExit、UserName 或 GroupName。不写入 `~/Library/LaunchAgents` 或系统启动目录，只在受控工作目录保存 plist 后显式 bootstrap；不使用会跨启动持久改变 enable/disable 状态的命令。`enable/disable` 持久性见 [launchctl.1:105–110](/usr/share/man/man1/launchctl.1:105)；Umask/ExitTimeOut 语义见 [launchd.plist.5:380–408](/usr/share/man/man5/launchd.plist.5:380)。

建议 launchd 标准输出仅写无敏感信息的启动诊断；业务命令输出由 worker 自己有界读取和脱敏。plist 的 `EnvironmentVariables` 只是添加变量，不代表清空继承环境，见 [launchd.plist.5:373–379](/usr/share/man/man5/launchd.plist.5:373)。worker 必须为每条外部命令重新构建白名单环境；stdin 关闭，不允许等待交互。

### 启动握手和清理

1. 主服务先在安装级锁内写入不可重放的 queued 记录，再 bootstrap；worker 取得安装级进程锁后写入自身 PID、起始时间及 ready 状态。需要测试父锁释放/子锁取得的交接：queued 状态应阻止第二次提交，但状态文件不能替代 worker 运行期的真实锁。
2. Bootstrap 调用超时/连接断开不能解释为“未创建”。先按同一个任务 ID 对账；不生成新任务重试，不 kickstart 已可能启动的 job。
3. worker 结束前必须已确认所管理子进程结束、状态及有限日志落盘，再写 `cleanup_pending`；之后可候选 `exec` 成 `/bin/launchctl bootout <exact-task-target>` 进行自身注销。
4. **自身 bootout 会终止自身 job，不能假定之后还能写“清理成功”。** 此路径的精确退出行为尚未运行验证。下次主服务启动或显式本地 CLI 对账时验证该 task target 已不存在，再删除该任务的临时执行文件/plist；终态记录按保留策略保留。没有可验证结果时保留 `cleanup_pending` 及资源，不能顺手清空目录。
5. `LaunchOnlyOnce` 不等于自动注销；崩溃遗留的注册和文件仍需对账。若设计要求“离线失败时也立刻、可确认地注销并删除全部文件”，单一自清理 worker 目前没有证据满足，须另行验证并确认方案调整。

两阶段清理是候选折中：没有常驻升级服务、没有自动重跑安装；允许故障遗留有界且可诊断的记录。它不能被描述为已证明的即时完整自清理。

## 3. 当前进程 → 服务 → Formula 身份

### 不能使用的捷径

- `installedChannel()` 仅解析实际可执行文件旁的 `homebrew` 标记，没有检查服务归属，见 [main.go:158–173](/Users/chenzhiguo/Projects/tokenlive-standalone/cmd/tokenlive/main.go:158)。
- Homebrew 的 `owner` 部分由 plist 或路径推导，PID 来自状态文本正则，JSON 只是把这些结果包装起来。见 [formula_wrapper.rb:279–349](/opt/homebrew/Library/Homebrew/services/formula_wrapper.rb:279)、[498–524](/opt/homebrew/Library/Homebrew/services/formula_wrapper.rb:498)。因此 `brew services info --json` 不是独立可信的强归属 API。
- `launchctl print` 文档明确不允许依赖输出结构或内容；`procinfo` 要求 root 且只供诊断；`list -x` 不再支持。见 [launchctl.1:261–300](/usr/share/man/man1/launchctl.1:261)、[560–578](/usr/share/man/man1/launchctl.1:560)。
- `SMJobCopyDictionary` 文档说明其字典内容不稳定、API 已弃用且无替代；`launch_msg` 同样弃用。见 [ServiceManagement.h:59–84](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/System/Library/Frameworks/ServiceManagement.framework/Headers/ServiceManagement.h:59)、[launch.h:4–14](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/launch.h:4)、[400–403](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/launch.h:400)。

### 尚待隔离验证的保守证据链

1. 当前程序自取 PID、真实/有效 UID、进程起始时间和实际可执行文件路径。SDK 存在 `proc_pidinfo`、`proc_pidpath`；`proc_bsdinfo` 包含 UID/PPID/PGID/起始时间。见 [libproc.h:92–103](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/libproc.h:92)、[proc_info.h:59–81](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/include/sys/proc_info.h:59)。这些字段帮助识别进程，但不自行证明 launchd job 归属。
2. 由受控 Formula/收据推导确切前缀、tap、keg、opt 链接、程序文件与预期 plist。检验实际二进制属于该 keg，注册 plist 的程序参数是该安装标准直启命令，路径/所有者/内容符合预期；自定义 `--file`、包装脚本、多实例先拒绝。Formula/安装可信性的更完整证明属于门槛 1。
3. 标签来自经核验的服务元数据，不硬编码旧标签。本机 Homebrew 同时认识 canonical `sh.brew.<formula>` 与 legacy `homebrew.mxcl.<formula>`，见 [service.rb:85–105](/opt/homebrew/Library/Homebrew/service.rb:85)。任何另一标签/另一域冲突应拒绝，不能取“第一个成功”。
4. 候选使用 legacy `launchctl list <label>` 的严格有界解析确认 **PID 等于当前服务自身 PID**，不是“有 PID”。Apple 对 legacy list 输出仅给出应与旧版匹配的说明，见 [launchctl.1:675–684](/usr/share/man/man1/launchctl.1:675)，因此仍需限定支持系统、拒绝未知结构、并测试 caller context 的域语义。
5. 逐个对 `gui/<uid>/<allowed-label>` 与 `user/<uid>/<allowed-label>` 做限定查询，只以 exit status 作存在性辅助（0 表示成功，见 [launchctl.1:715–720](/usr/share/man/man1/launchctl.1:715)），不解析 print 内容。必须验证唯一正确域；“查询失败”也可能是权限/域故障，不能全部当作不存在。若无法把 list 的 PID 与唯一域关联，不开放能力。
6. 接受确认、worker 开始安装前、准备重启前分别重核身份；记录 PID 起始时间防 PID 复用。主进程重启后使用新 PID/起始时间，重新核对目标安装、目标运行版本和本机专用健康握手。不能让一次旧 PID 证据永久授权后续服务操作。

**该证据链是可测试候选，不是已经取得的稳定跨版本 API。** 不建议以“Homebrew 自己也解析 print”为由突破 Apple 的明确限制。无法完成这条链时错误类别应为 `service_identity_unverifiable`，保留手动升级。

## 4. 服务操作副作用、退出和 needs_attention

### 一手源码发现

1. `brew services restart` 会 stop 后 start/run，更新服务文件；内部 stop 未传 `max_wait`，默认是 0，而循环把 0 作为无限等待。restart 命令本身没有 `--max-wait`。见 [restart.rb:11–46](/opt/homebrew/Library/Homebrew/services/subcommand/restart.rb:11)、[cli.rb:229](/opt/homebrew/Library/Homebrew/services/cli.rb:229)、[303–326](/opt/homebrew/Library/Homebrew/services/cli.rb:303)。
2. 独立 `brew services stop --max-wait=<seconds>` 支持有限等待，但内部 `launchctl` 每次调用本身仍需外层监督；不能把它当作整条调用的硬 deadline。见 [stop.rb:17–38](/opt/homebrew/Library/Homebrew/services/subcommand/stop.rb:17)、[cli.rb:303–326](/opt/homebrew/Library/Homebrew/services/cli.rb:303)。
3. start 会生成/安装目标 plist，再 enable/bootstrap 到重新计算的域，见 [cli.rb:180–215](/opt/homebrew/Library/Homebrew/services/cli.rb:180)、[462–469](/opt/homebrew/Library/Homebrew/services/cli.rb:462)、[522–560](/opt/homebrew/Library/Homebrew/services/cli.rb:522)。stop 遍历服务标签及候选域，并可能回退裸标签 `launchctl stop`，见 [cli.rb:294–327](/opt/homebrew/Library/Homebrew/services/cli.rb:294)。存在其他同名实例或标签迁移时，不能保证只影响当前已识别服务，须拒绝或改用另行验证的精确域适配。
4. `TimeOut` 不再实现；`ExitTimeOut` 仅是停止 job 时 TERM→KILL 的等待，**不是最大运行时间**。见 [launchd.plist.5:397–408](/usr/share/man/man5/launchd.plist.5:397)。
5. launchd 默认只杀死与退出 job **同进程组**的残余进程，见 [launchd.plist.5:609–613](/usr/share/man/man5/launchd.plist.5:609)。本机 Homebrew `SystemCommand` 为非 sudo 子命令设置 `pgroup: true`，其 timeout 也是可选参数；见 [system_command.rb:430–478](/opt/homebrew/Library/Homebrew/system_command.rb:430)。据此推导：杀掉 worker/顶层 brew 或其原进程组并不足以证明包管理所有后代结束。
6. TokenLive 当前 `/health` 仅返回固定 status/mode，不提供 PID、运行构建指纹或目标身份，见 [assemble.go:98–102](/Users/chenzhiguo/Projects/tokenlive-standalone/internal/assemble/assemble.go:98)。该接口本身不能用于最终升级成功判定。

### 候选固定 argv 与接口

以下全部是未来实现接口候选，本次没有执行：

```text
ProbeService(label):
  ["/bin/launchctl", "list", label]
  ["/bin/launchctl", "print", "gui/<uid>/<label>"]   # status-only candidate
  ["/bin/launchctl", "print", "user/<uid>/<label>"]  # status-only candidate

BootstrapTask(domain, exactPlist):
  ["/bin/launchctl", "bootstrap", domain, exactPlist]

RemoveTask(exactTaskTarget):
  ["/bin/launchctl", "bootout", exactTaskTarget]

RestartCandidate(verifiedFormula):  # only after isolated acceptance of its label/domain scope
  [verifiedBrew, "services", "stop", "--max-wait=60", verifiedFormula]
  [verifiedBrew, "services", "start", verifiedFormula]
```

禁止 `--all`、域级 bootout、裸 stop label、`kickstart -k`、`submit`、用户提供的路径/URL/Formula 参数、全局 cleanup、sudo。直接用 launchctl 重载主服务的替代路线可以精确指定域，但需另行证明新版 plist 生成、注册文件保留、标签迁移与 Homebrew 管理语义；不能把它当成已经验证的 brew services 等价替换。

建议将系统依赖封装成以下接口，便于纯替身测试：

```text
InstallationProbe -> immutable InstallationIdentity
ServiceProbe      -> VerifiedServiceIdentity | Unsupported(reason)
TaskLauncher      -> Accepted | NotAccepted | OutcomeUnknown
ProcessRunner     -> ExitEvidence{pid, start, exit, descendantsKnownGone, timedOut}
StateStore        -> atomic state + persistent unresolved latch
TaskReconciler    -> cleanup/recovery evidence; never automatically rerun installation
```

### 超时和恢复规则建议

- 使用单调时钟监督各阶段和总期限；到达 deadline 是“预算耗尽”，不自动产生已退出证据。不得依赖 plist `TimeOut`，也不得用强杀来制造“失败已恢复”。
- 下载等未变更阶段的取消只有在确实等待命令退出、无存活后代证据时才能记录 `failed`；安装/链接阶段超时、worker 异常退出、进程树失联、bootout/start 结果未知一律 `needs_attention`。
- 在每次外部状态变更前原子写入“已进入该步骤，结果待定”。即使 worker 被 KILL 来不及写最终状态，下次只读对账也能将非终态且已失去执行证据的任务视为未决，而不是当成无任务。
- 安装级 advisory lock 只协调遵守锁协议的执行器，不能阻止同 UID 在终端操作 brew；见 [flock.2:50–74](/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk/usr/share/man/man2/flock.2:50)。同时保留持久 unresolved latch；**锁取得成功不代表上次安装已安全结束**。不删除 Homebrew 锁、不按 PID 名称泛化杀进程。
- 若 worker 仍在监督未知的安装进程，保留活动锁并停止推进后续步骤；如果为了有界 worker 寿命需要退出，先可靠写入 unresolved 状态，并把“可能仍有外部执行”明确交给人工恢复。该策略不等于保证外部命令在 30 分钟内终止。
- 离线 CLI 只读状态/安全诊断的实现不需要登录 token，不应初始化业务 DB。只有明确的人工恢复流程和新的可验证证据才能解除 unresolved，不因下一次 Web 查询自动清空。

## 5. 发布前最小隔离验证

下列需要可销毁 macOS 测试环境及单独授权；本次没有运行。纯替身单测先覆盖状态机、解析器、参数和时限，但不能代替系统验收。

1. **域/身份矩阵**：同 UID gui 与 user；服务直接启动与 shell 包装；普通前台进程；root 启动但 UserName 为普通用户；同标签不同域；canonical/legacy 双标签；第二安装前缀；PID 复用；未知/截断的 list 输出。错误身份均必须拒绝，且不得调用任何变更命令。
2. **生命周期**：先用两个无业务测试 job 验证主 job bootout/rebootstrap 后 worker 持续运行、PID/起始时间不变；关闭客户端不影响任务；退出登录/重启后不自动重跑。后两项只承诺保留未决诊断，不承诺继续。
3. **资源**：启动 worker 后移走模拟旧 keg/opt、替换主程序及资源，确认 worker 不再访问旧路径；测试 dylib/相对资源缺失时在停止主服务前拒绝。
4. **一次性与清理**：RunAtLoad + KeepAlive=false + LaunchOnlyOnce 的成功、非零退出、崩溃都只运行一次；bootstrap 返回未知；self-exec bootout；bootout 失败；cleanup_pending 对账；不得删除非本任务资源；不残留登录自启动项。
5. **进程监督**：替身命令生成同组子进程、另组孙进程、忽略 TERM、持有输出管道、写盘后阻塞。每种超时/worker 崩溃都不得产生虚假 `failed/succeeded`，持久 unresolved 必须阻止下一次升级。需要在真正 Homebrew 安装路径上复核后代行为。
6. **服务操作**：限定 Formula 的 stop/start 只操作预期标签与域、stop 超时不继续 start、注册文件更新符合目标 Formula、无额外服务迁移或 enable 修改。然后才能进行真实测试 Formula 的一次安装/重启。
7. **核验**：错误版本、同端口无关 HTTP、健康接口提前响应、正确版本但不同安装路径都不能成功；实际目标进程和必要初始化完成后才能持久化 succeeded。

最低阻塞清单：尚无验收过的 gui/user 归属适配；尚无自 bootout/文件回收结果；尚无完整 Homebrew 后代终止证据；尚无目标身份健康握手。实现计划应把这些列为发布门槛，不把本次只读研究改写为“已经实现自升级”。

## 后续获授权的合成 job 实验

控制器在 2026-09-12 单独取得许可后运行
`/private/tmp/tokenlive-launchd-probe.tZivJU/probe.py`。
创建三个随机 label、无业务的一次性 job；只针对本次明确列出的标签
bootstrap/bootout，不操作 TokenLive，不写登录自启动目录。

实际结果（exit 0）：

```json
{
  "independent_worker_survived_restart": true,
  "same_uid": 501,
  "legacy_list_matches_running_pids": true,
  "main_pid_before": 18826,
  "main_pid_after": 18841,
  "worker_pid": 18832,
  "user_lookup_exit": 113,
  "self_bootout_after_terminal": true,
  "failure_launch_count": 1,
  "scope": "synthetic gui-domain jobs only; not Homebrew E2E"
}
```

- 模拟主 job 注销、重新 bootstrap 后，worker PID/实例标识保持不变，心跳继续增加。
- worker 内部针对自己和主 job 的 legacy list PID 与双方自报 PID 一致；同标签
  user 域查询返回非零，gui 域存在。此单一组合不证明所有错误码均表示不存在。
- worker 原子写入 `succeeded + cleanup_pending` 后 exec 为精确 bootout；
  控制器观测 job 已不存在，终态文件仍保留。
- 故障 job 以 7 退出，在超过常规节流窗口的观测期内没有二次启动。
- 最后所有三个精确 job 查询均确认不再存在，`cleanup.json` 为
  `{"remaining":[]}`。文件、日志和合成代码保留在临时目录，没有常驻测试服务。

这补充了门槛 2 的最小生命周期/清理实证和门槛 3 的单一 GUI 域证据链。
不覆盖真实 brew 服务、canonical/legacy 冲突、其他用户域、两种架构、断电、
动态库资源搬迁或包管理后代。产品必须保留未知即拒绝和持久 needs_attention，
并完成第 5 节矩阵后才把对应环境判为支持。
