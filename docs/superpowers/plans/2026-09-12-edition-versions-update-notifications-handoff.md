# 单机版/专业版版本与升级提醒：交付记录

日期：2026-09-12

## 结论

本地实施与验收完成：13 项任务通过逐项审查，整项审查发现的 3 项 Important 和 2 项 Minor 均已修复；最终独立复审 F1–F5 全部 ADDRESSED，没有新增问题。代码具备本次功能的技术集成条件，但不等于已经发布或部署。

尚未合回原项目、推送、打 tag、发布、安装或重启服务。四个隔离工作区全部保留；Standalone 按仓库规则保持未暂存、未提交。

## 已交付能力

- 单机版显示 standalone 整包版本；专业版分别显示 Admin 版本和有效 Gateway 版本分组/数量，没有虚构的整套总版本。
- 侧栏有紧凑的 Gateway 混合版本摘要；“关于”展开各组。普通用户可看当前版本，Root 和获委派的更新管理员可看更新与检查入口。
- 后端启动后异步检查，默认每 6 小时刷新；各源共享 5 秒截止时间，实例共享 60 秒手动冷却。
- 只比较可信正式构建与稳定版；失败、过期、未知和未检查状态不会冒充“最新版”或驱动升级操作。
- Homebrew 使用受控 tap 当前 Formula 作为候选，提供复制命令，不执行升级；发布端保证上传资产、公开 Release 后才更新 tap。运行时不额外核验 Release 资产。
- Gateway 复用 Redis 或部署认证 HTTP 通道，启动及每 30 秒上报，3 分钟 TTL；关闭外部检查不影响内部上报。
- 构建版本和发行类型贯穿 Admin/Gateway/Standalone；部署支持独立组件镜像版本并兼容旧 VERSION。
- 不自动弹窗、安装或重启，不新增业务数据库表或完整节点管理。

## 工作区与代码状态

共同分支：`codex/edition-versions-update-notifications`。

| 仓库 | 保留的隔离工作区 | 代码提交/状态 |
| --- | --- | --- |
| Admin | `/private/tmp/tokenlive-edition-work.e9cKUC/admin` | 最终代码提交 `8b03698c81515dd8935ad57ff2c985bf72b2cb07`；交付文档另有提交 |
| Gateway | `/private/tmp/tokenlive-edition-work.e9cKUC/gateway` | `4ccd4b007974033585312dc7d234956b72e2879a` |
| Standalone | `/private/tmp/tokenlive-edition-work.e9cKUC/standalone` | 基线 `bed7691c98d783e0e9417b49cd836611dd9dbbeb`；18 个已审查的修改/新增文件未暂存、未提交 |
| Deploy | `/private/tmp/tokenlive-edition-work.e9cKUC/deploy` | `5e5050b904a8de18df62d1f460d9970877d695db` |

原始 Admin 的 BasicHeader 暂存 blob 为 `035131b0882ef0e2de8beb2f1a1b02a86390d452`，保持不变；原目录的计划/设计文档暂存和 ModelDetail 暂存/工作区改动也未触碰。后续集成必须逐块保留用户内容，不能用隔离目录的整文件覆盖原始 BasicHeader。

## 验证证据

控制器在最终代码提交上运行：

| 范围 | 命令/检查 | 结果 |
| --- | --- | --- |
| Admin | `GOWORK=off go test -race ./pkg/productversion ./pkg/versionregistry ./internal/updatecheck ./internal/versionstatus ./internal/mods/systemversion -count=1` | 全通过 |
| Admin | `GOWORK=off go test ./adminapp ./internal/bootstrap -count=1` | 全通过 |
| Gateway | `GOWORK=off go test -race ./pkg/versionreport ./internal/bootstrap ./cmd/server -count=1` | 全通过 |
| Standalone | 临时联合 go.work 下真实 sender→Admin 集成 `-race -count=1`；main/assemble `-count=1` | 全通过 |
| 前端 | Node 22.23.1，`node --test tests/*.test.mjs` | 39/39 |
| 前端 | `npm run build:prod`，改变文件的 Prettier 检查 | 通过；保留既有 chunk 警告 |
| 浏览器 | 模拟 API，全六种布局、深浅色、折叠/移动端、权限/冷却/状态/复制/链接/单机版/混合摘要/中英文冷状态 | 全通过；临时服务器已停止 |
| Standalone 脚本 | 当前打包脚本的 Python unittest | 22/22；最终修复仅变更 README，脚本未再变化 |
| Deploy | 当前部署脚本的 Python unittest、既有 Shell 回归、Shell 语法 | 27/27 及其余检查通过；最终修复未修改 D |
| 各仓库 | `git diff --check`，Standalone 空索引及 go.mod/go.sum 不变 | 通过 |

测试仅使用临时 SQLite、独立 miniredis、内存存储、受控 HTTP 和模拟浏览器 API，不连接业务服务，不写业务脏数据。初始沙箱本地监听限制经正常提权申请后重跑，不被计为功能失败。未执行可能访问真实服务的全仓 Go 测试。

最后浏览器截图：`/private/tmp/tokenlive-edition-work.e9cKUC/browser-handoff-verified`。

## 发布前仍须完成

1. Standalone 目前声明的已发布 Admin 依赖缺少新增 API。先取得兼容、真实存在的 Admin/Gateway 模块引用，或显式提供匹配的已审核源码，再做干净依赖构建；未编造未来 tag，未提交本地 replace/go.work。
2. 临时联合测试图使用 genproto 覆盖；独立 package-release 打包图已用当前本地源码验证且不需要该覆盖，但这不证明旧已发布依赖可用。
3. Docker daemon 不可用，真实镜像编译及容器生命周期未验证；Compose 只读解析和脚本替身不能替代该验证。
4. Node 18 的 CI 构建/测试未验证；本次执行环境为 Node 22.23.1。发布时应验证或明确统一目标环境。
5. 实际 Homebrew 发布/安装和服务重启未执行；老版本需要先人工升级一次才能获得后续自动提醒。

## 非阻塞后续事项

- 既有模块类型、MockTimers 和 multi-tab 循环 chunk 警告保留，不宣称已修复。
- Deploy 旧卸载脚本可遗留独立固定版本镜像；本次独立版本配置只覆盖安装、构建和升级。
- Provider 的既有关联密钥传播算法未修改，Wire 维持原有 EndpointDAL 排除，避免本功能激活无关缺陷。

## 可追溯记录

本工作区的 `.superpowers/sdd/2026-09-12-edition-versions-update-notifications/` 保留全部 13 项报告、审查包、修复证据及账本：

- `progress.md`：任务和 R1–R22 实施裁定；
- `final-review-report.md`：整项审查、全部 18 项验收覆盖和原始 F1–F5；
- `final-fix-report.md`：真实 RED/GREEN、修复提交、完整验证；
- `final-rereview-report.md`：五项全部关闭、无新增问题。

过程记录与未提交的 Standalone 变更均保留，不删除工作区。后续合并、提交 Standalone 或发布须由用户单独明确选择。
