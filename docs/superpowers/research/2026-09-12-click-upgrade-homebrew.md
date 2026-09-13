# 点击升级：固定 Homebrew 目标的技术验证

日期：2026-09-12。先完成只读研究与可销毁加载器实验，随后经单独许可在临时前缀进行了合成包安装实验。下文先保留原始调查，再记录新增执行证据；这不是实际 TokenLive 自升级或发布就绪证明。

## 结果

在本机 Homebrew 源码 `a45b0a0143d4df548cdaa121fcb4282311d3eaff`
（`6.0.22-310-ga45b0a0143`）上，实际验证了：

- 指向**同一 tap 下、常规 Formula 搜索目录之外的固定文件**的绝对路径，可以经正常 Formula 加载器解析。
- 固定文件为 `1.2.3`，同 tap 常规 Formula 为 `9.9.9`；固定路径仍解析为 `1.2.3`，完整名称仍为 `tokenlive/tokenlive/tokenlive`，不是新建的版本包或其他 tap。
- `Formulary.resolve(snapshot)` 保留 `specified_path`；`latest_formula` 返回原对象；`FormulaInstaller` 构造出的 build argv 仍引用固定快照。
- 固定文件放在 tap 根部 `.tokenlive-upgrade/<task>/tokenlive.rb`，不进入常规 `Formula/**/*.rb` 搜索结果。未覆盖常规 Formula、未移动 opt/Cellar、未伪造安装收据。

因此有一个可以写入实施计划的**候选适配**：用服务端验证过的完整 Formula 字节建立同 tap 固定快照，后续 info/fetch/upgrade 使用该路径。不能用“执行前比较一次版本 + 普通名称 upgrade”代替它。

真实安装、收据和服务兼容性、Homebrew 不同版本、并发终端操作均尚未端到端验证。发布前必须完成相应隔离实验；能力检测不应把未验证组合标为支持。

## 实验隔离与实际输出

实验目录：`/private/tmp/tokenlive-homebrew-probe.eA4Ag4`。

- 复制本机 `bin/brew` 到临时前缀，Library/Homebrew 仅以只读用途引用已有库。
- 伪 tap、Formula、信任记录、缓存、日志、临时目录全部位于实验目录；没有修改真实 Homebrew tap/信任/包。
- 两份 Formula 的 `install` 方法均直接抛出“Probe must never install”，并且实验没有调用 install/fetch/upgrade 或任何服务命令。
- 环境设置 `HOMEBREW_NO_AUTO_UPDATE=1`、`HOMEBREW_NO_ANALYTICS=1`、`HOMEBREW_NO_INSTALL_FROM_API=1`，XDG 配置、Homebrew cache/log/temp 指向实验目录；没有重设用户 HOME。
- 未使用 developer/path/trust 的放宽开关来让固定路径通过检查。

第一次执行真实 `brew info --json=v2 --formula <snapshot>`，exit 0：

```json
{
  "full_name": "tokenlive/tokenlive/tokenlive",
  "tap": "tokenlive/tokenlive",
  "versions": {"stable": "1.2.3", "head": null, "bottle": false},
  "ruby_source_path": ".tokenlive-upgrade/fixed/tokenlive.rb",
  "installed": []
}
```

第二次执行 `brew ruby <实验目录>/check-loader.rb`，只加载公式和构造安装器，不调用安装方法，exit 0：

```json
{
  "full_name": "tokenlive/tokenlive/tokenlive",
  "version": "1.2.3",
  "latest_is_same_object": true,
  "build_formula_path": "/private/tmp/tokenlive-homebrew-probe.eA4Ag4/prefix/Library/Taps/tokenlive/homebrew-tokenlive/.tokenlive-upgrade/fixed/tokenlive.rb",
  "canonical_formula": "/private/tmp/tokenlive-homebrew-probe.eA4Ag4/prefix/Library/Taps/tokenlive/homebrew-tokenlive/Formula/tokenlive.rb"
}
```

实验材料保留在临时目录，不属于产品实现。测试 Formula 和摘要中的版本是合成数据，不是 TokenLive 的真实发行号。

## 一手代码依据

| 事实 | 本机 Homebrew 一手源码 |
| --- | --- |
| 普通路径默认受限；tap/Cellar 内路径例外，检查实际路径祖先 | [utils/path.rb:240](/opt/homebrew/Library/Homebrew/utils/path.rb:240)、[env_config.rb:1011](/opt/homebrew/Library/Homebrew/env_config.rb:1011) |
| FromPathLoader 识别 tap，文件名决定 Formula 名称 | [formulary.rb:595](/opt/homebrew/Library/Homebrew/formulary.rb:595)、[tap.rb:85](/opt/homebrew/Library/Homebrew/tap.rb:85) |
| 绝对路径 resolve 直接 factory，随后禁止跟随已安装别名 | [formulary.rb:409](/opt/homebrew/Library/Homebrew/formulary.rb:409) |
| latest_formula 只在别名目标变化时替换，通常返回自身 | [formula.rb:2119](/opt/homebrew/Library/Homebrew/formula.rb:2119) |
| upgrade 从已解析 Formula 创建安装器 | [cmd/upgrade.rb:430](/opt/homebrew/Library/Homebrew/cmd/upgrade.rb:430) |
| 构建子进程参数使用 selected specified_path，保存同一 Formula 内容 | [formula_installer.rb:637](/opt/homebrew/Library/Homebrew/formula_installer.rb:637)、[1233](/opt/homebrew/Library/Homebrew/formula_installer.rb:1233) |
| postinstall 使用已选公式或 keg 保存副本 | [formula_installer.rb:1428](/opt/homebrew/Library/Homebrew/formula_installer.rb:1428) |
| 正常安装器写入 tap 和所用 source path；不可由应用自行伪造 | [tab/tab.rb:127](/opt/homebrew/Library/Homebrew/tab/tab.rb:127)、[formula_installer.rb:1706](/opt/homebrew/Library/Homebrew/formula_installer.rb:1706) |
| tap 扫描 Formula 目录，不扫描根部任意隐藏子目录 | [tap.rb:1021](/opt/homebrew/Library/Homebrew/tap.rb:1021) |

收据 `tap_git_head` 来源于当前本地 tap，不保证等于远端固定 Formula 修订。应用应独立保存已确认的 Formula 内容/发布修订指纹；不能把 Homebrew 的这个字段冒充精确目标证据，也不能事后修改收据来“修正”它。

## 副作用与必须保留的限制

1. 完整限定名称 upgrade 在当前版本可能自动记录第三方 Formula 信任；固定路径仍执行现有 trust 检查。见 [cmd/upgrade.rb:204](/opt/homebrew/Library/Homebrew/cmd/upgrade.rb:204)、[trust.rb:224](/opt/homebrew/Library/Homebrew/trust.rb:224)。产品不能自动加信任，预检失败给手动说明。
2. `brew version-install` 会提取到个人 tap 并创建版本名，不能用作“原 TokenLive 安装就地升级”的等价替换。见 [cmd/version-install.rb:14](/opt/homebrew/Library/Homebrew/cmd/version-install.rb:14)。
3. upgrade 默认可能处理 dependents 和 cleanup。候选命令环境设置 `HOMEBREW_NO_AUTO_UPDATE`、`HOMEBREW_NO_INSTALL_CLEANUP`、`HOMEBREW_NO_INSTALLED_DEPENDENTS_CHECK`、`HOMEBREW_NO_ANALYTICS`、`HOMEBREW_NO_ENV_HINTS`；它们不能代替依赖审查，也不能保证缺失的 Ruby/toolchain 永不自举。见 [env_config.rb:535](/opt/homebrew/Library/Homebrew/env_config.rb:535)、[586](/opt/homebrew/Library/Homebrew/env_config.rb:586)、[utils/ruby.sh:153](/opt/homebrew/Library/Homebrew/utils/ruby.sh:153)。
4. 当前 TokenLive Formula 没有包依赖；首期只接受已审核的无依赖 Formula 模板，下载 URL、版本和 SHA 作为受控可变字段；模板增加新行为时先更新适配，不盲目执行。
5. Homebrew 的系统/prefix/user `brew.env` 可以覆盖调用环境。只设置 Go 子进程 Env 白名单并不充分。见 [bin/brew:124](/opt/homebrew/bin/brew:124)。必须对会影响来源、权限、自动更新/清理、代理/包装器和执行路径的有效配置做明确校验，冲突时拒绝；不能替用户改配置或扩大信任。
6. 同 UID 进程可修改其拥有的文件/运行 brew。互斥锁和只读快照不能抵御恶意同 UID，也不承诺阻止手工操作。产品必须检测常规并发变更、验证快照和安装身份，不声称提供操作系统级防篡改。
7. 包管理可能产生不同进程组的后代；超时、worker 崩溃后不能由锁释放推断没有后代执行。服务和退出策略见[launchd 研究](2026-09-12-click-upgrade-launchd.md)。

## 发布前最小补充实验

- 在独立、可销毁的 macOS 测试用户/机器中实际安装旧测试包，再用上述快照路径升级。
- 校验原完整 tap/Formula 身份、安装收据、opt 链接及未来正常 `brew upgrade`/services 兼容性，不人工重写收据。
- 在确认后把常规 Formula 改为不同版本/同版本不同内容，证明只安装固定字节所指的产物；不只事后报错。
- 验证信任不足、pin、Homebrew 配置覆盖、依赖/工具缺失、已有 brew 锁、进程异常/超时都安全拒绝或进入 needs_attention。
- 记录实际通过的 macOS/Homebrew/架构组合；不宣称本机加载器实验自动覆盖所有正式版本。

Web 官方检索没有返回可读内容；本报告使用明确版本的本机一手源码和实际无安装实验，不以未读网络页面作证。

## 后续获授权的真实包管理实验

范围仍为 `/private/tmp/tokenlive-homebrew-probe.eA4Ag4/prefix`，合成包名
`tokenlive-upgrade-probe`，只打印版本，无服务。归档为本地 `file://`，没有下载
发行产物。实际 Homebrew 前缀、用户 TokenLive 和原有 tap/信任配置未修改。

1. 初次旧包安装因归档只有 `bin/` 一个顶层目录，被 Homebrew staging 自动进入该目录，
   导致 `bin.install "bin/..."` 找不到文件。依据
   [abstract_download_strategy.rb:108](/opt/homebrew/Library/Homebrew/download_strategy/abstract_download_strategy.rb:108)
   修正合成包布局（增加 README），保留第一次失败事实，不算产品缺陷。
2. 旧版 `1.2.2` 实际安装成功，执行测试程序输出 `1.2.2`。
3. 把常规 Formula 改成 `9.9.9`，固定快照保持 `1.2.3`。升级调用在 120 秒超时；
   此后只读检查发现 `1.2.3`、opt 链接和正确收据已经落盘，匹配实验目录/包名的
   进程未发现残留。不能据此把这次调用判为成功。超时的确切收尾原因尚未证明。
4. 对已装好的同一目标做一次诊断，返回 `already installed`，exit 0，没有重新安装。
5. 换用另一个明确的固定合成目标 `1.2.4`，记录实时日志；常规 Formula 仍为 `9.9.9`。
   真实 `brew upgrade --formula --verbose <固定快照>` 完整 exit 0，打印
   `1.2.3 -> 1.2.4`，测试程序输出 `1.2.4`，收据保留：

```json
{
  "tap": "tokenlive/tokenlive",
  "path": "/private/tmp/tokenlive-homebrew-probe.eA4Ag4/prefix/Library/Taps/tokenlive/homebrew-tokenlive/.tokenlive-upgrade/install-proof-second/tokenlive-upgrade-probe.rb",
  "versions": {"stable": "1.2.4", "head": null, "version_scheme": 0}
}
```

临时 Cellar 中只有 `tokenlive-upgrade-probe`；没有其他包被安装或升级。
Homebrew 输出了非标准前缀 Tier 3 警告，因此不将该证据当作标准发行环境覆盖。
完整脚本及日志保留于实验目录。没有为让实验通过修改 Homebrew 库或放宽信任/路径检查。

**门槛判定：**固定字节路径→加载器→真实安装→收据/opt 的可行性已有正向执行证据。
一次调用超时仍要求产品持久化未决状态，不能凭收据已写就返回成功。正式 macOS/
Homebrew/架构矩阵、TokenLive 全包和真实服务联合验收仍是发布门槛。

另外发现当前 macOS 的升级收尾可能独立检查并重装 pkgconf：
[cmd/upgrade.rb:361](/opt/homebrew/Library/Homebrew/cmd/upgrade.rb:361)、
[extend/os/mac/reinstall.rb:16](/opt/homebrew/Library/Homebrew/extend/os/mac/reinstall.rb:16)。
不能把关闭 dependents/cleanup 当成禁止一切其他包操作。实施预检必须识别会触发
此类迁移的环境并拒绝，或以另行审核的原生安装入口替代；不 monkey-patch Homebrew。
