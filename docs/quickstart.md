# 版本与更新提醒快速指引

此页说明版本检测，不执行升级。首次启用需要维护人员人工安装包含本功能的 Admin/Gateway 或 standalone 发布；旧程序不能靠新功能自动升级自己。

## 查看与授权

登录后可从侧栏底部版本入口或设置中的「关于」查看当前产品身份。专业版显示 Admin 及观测到的 Gateway 版本分布；all-in-one 只显示一个 standalone 发布单元。入口在侧栏收起和窄屏下仍可访问。

Root 默认可看更新结果并点击检查。为其他维护人员的角色授予 `system.versionUpdates` 菜单能力（读取更新结果和 POST 检查资源）；不需要将用户名改成管理员。无此权限的登录用户仍可看当前版本，但不能看目标版本、发行链接或触发公网请求。更新只以入口标记/对话框展示，不强制弹窗。

打开「关于」只读缓存；手动检查仍受同实例共享的 60 秒冷却约束。关闭联网检查使用 `UPDATE_CHECK_ENABLED=false`，无需关闭内部版本上报。完整默认值与 namespace 说明见 [配置指南](CONFIGURATION.md#产品版本与更新检查)。

## 兼容矩阵

| 组合 | 展示和行为 |
| --- | --- |
| 新 Admin + 新专业版 Gateway | 通过 Redis 或带部署 token 的 HTTP 获取匿名多版本聚合；过期节点自动移除 |
| 新 Admin + 旧 Gateway | 没有有效上报时显示 Gateway `unknown`，不猜测网关版本 |
| 旧 Admin + 新 Gateway | HTTP 接收接口缺失时上报失败并限频告警，不阻断网关业务；Redis 路径只留下可过期记录 |
| 多个 Admin + 内存注册表 | 每实例仅展示本实例收到的上报；负载均衡可能导致分布不同 |
| 多个 Admin + 共享 Redis DB/namespace | 展示该 namespace 中仍有 TTL 的有效节点 |
| Homebrew standalone | 使用安装标记识别渠道；只比较当前 Formula 已就绪的稳定发布 |
| standalone 无有效安装标记（源码、Linux、Docker 等） | 保持 `install_channel=unknown`；展示当前版本，不假定 Homebrew 或专业版升级来源 |
| 开发构建/无法识别的当前版本 | 显示原始版本，但不可与稳定版作升级比较 |

## 人工升级

先阅读当前渠道的发行说明并备份配置和数据；只采用实际存在且已验证的发行版本。控制台不会执行下面的命令。

Homebrew 渠道可复制并由维护人员在目标机器执行：

```bash
brew update
brew upgrade tokenlive
```

如需要重启，由维护人员按既有服务管理流程单独决定，控制台不会发起服务操作。未知 standalone 渠道应按原安装渠道处理，不展示错误的 `brew upgrade` 建议。

专业版独立升级 Admin 和 Gateway。使用 `tokenlive-deploy` 时，在维护窗口手工选择已经发布且验证过的版本：

```bash
# ADMIN_TAG / GATEWAY_TAG 由维护人员设置为已验证的真实镜像标签
: "${ADMIN_TAG:?请先选择已验证的 Admin 镜像标签}"
: "${GATEWAY_TAG:?请先选择已验证的 Gateway 镜像标签}"
bash install.sh --upgrade --admin-version "$ADMIN_TAG" --gateway-version "$GATEWAY_TAG"
```

组件镜像版本优先级为 `ADMIN_VERSION` / `GATEWAY_VERSION` → `VERSION` → `latest`。`--upgrade` 的显式覆盖只对本次操作生效，不改写已保存 `.env`；持久调整请维护 `.env`。镜像标签与本地构建元数据分开：`ADMIN_BUILD_VERSION` / `GATEWAY_BUILD_VERSION` 默认 `dev`，只有受控发布才显式设置 `BUILD_KIND=release`。安装器的详细行为以部署仓库文档为准。

## 源码联合验收与发布前提

跨模块测试位于 standalone 的 `internal/assemble/version_integration_test.go`，通过公开 `adminapp` facade 和 Gateway 真实 HTTP/Redis Sender 连接；Admin 自身的 `internal/versionstatus/integration_test.go` 不依赖 Gateway 模块。测试使用临时 SQLite、内存注册表、独立 miniredis、临时 HTTP 服务和受控 Release transport，不访问真实账号、业务 Redis 或公网。每条测试节点均有删除清理及 TTL；临时文件、数据库和子进程随测试结束清理。

联合源码验证可使用仅位于临时目录的 `go.work` 连接本次 Admin/Gateway/standalone checkout，不提交本机 `replace` 或 `go.work`。测试工作区成功不能代替发布依赖验证：当前 standalone 声明的 Admin `v0.9.7` 不含新增 `pkg/productversion`，干净机器按旧依赖构建不能通过。必须先发布兼容的 Admin/Gateway 模块，再将 standalone 指向实际存在且经过验证的模块 tag，或为打包明确提供匹配的已审核源码。不要为绕过此门槛编造未来版本号。

本次源码验收的临时测试图使用了 genproto 兼容覆盖；独立 `package-release.sh` 临时打包图已经用真实 Go 验证当前源码，无该 genproto 覆盖。它仍不代表旧已发布依赖可用。实际 Docker 镜像编译和容器生命周期需要可用 daemon 后另行验收；没有执行镜像推送或真实部署。

前端 Node 测试已在 Node 22.23.1 验证；既有 Node 18 构建 workflow 不代表同等测试覆盖。已有模块/MockTimers/循环 chunk 警告应保留记录，不因构建成功而忽略。此次验收不覆盖既有 Provider Wire 排除项或无关 API key 算法。
