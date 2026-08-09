# Admin 会话恢复与登录状态稳定性设计

## 背景

Admin 已支持同一账户在多台设备分别登录。每次登录会得到独立的 refresh token，单设备退出只撤销该设备的 token。

当前前端仍可能把菜单初始化失败、网络中断、服务端 5xx 等临时故障当作认证失效，调用完整 logout 并删除本地 refresh token。只剩 refresh token、没有 access token 时，路由守卫也不会尝试恢复；进入登录页还会无条件清除现有 token。这些行为会让用户频繁重新登录，并造成“多设备登录互相影响”的错觉。

## 目标

- 临时网络错误、超时、5xx 和业务初始化错误不得撤销或删除登录凭据。
- access token 过期时通过 refresh token 自动恢复，并重放原请求。
- 应用启动时即使 access token 缺失，只要 refresh token 有效也能恢复登录。
- 只有 refresh 接口明确返回 401，或用户主动退出时，才清除持久化凭据。
- refresh 失败后的本地失效流程不得再次调用受保护的 logout 接口。
- 保持多设备会话互不影响：设备 A 主动退出不影响设备 B。

## 非目标

- 不实现无需再次认证的跨设备会话同步或扫码授权。
- 不新增全设备退出、会话列表或设备管理页面。
- 不迁移 token 到 HttpOnly Cookie；本次延续现有 localStorage 方案。
- 不扩大到 RBAC、菜单或 OAuth 业务逻辑重构。

## 方案

### 会话状态与操作

前端区分三类操作：

1. **主动退出**：用户点击退出。尽力调用后端 logout 撤销本设备 token，随后始终清除本地会话和应用状态。
2. **认证失效**：refresh 接口明确返回 401。只执行本地会话失效，不再调用后端 logout，避免过期 access token 引发递归 401。
3. **临时故障**：网络错误、超时、5xx、菜单加载失败或其他初始化异常。保留所有 token，向上抛出错误或显示重试状态，不退出登录。

用户 store 提供独立的本地失效动作，集中清除 token、用户信息和依赖登录态的 store。主动 logout 复用该清理动作，但认证失效路径不会发起网络请求。

### 启动恢复

路由守卫不再只以 access token 是否存在作为唯一依据：

- 有 access token：按现有流程初始化应用；若接口返回 401，由请求拦截器刷新并重放。
- 无 access token、有 refresh token：先执行一次 refresh，再初始化应用。
- 两者都没有：进入登录页。
- refresh 明确失效：本地清理并进入登录页。
- refresh 遇到临时错误：保留 refresh token，导航到登录页或展示失败状态，但不得撤销会话；用户后续刷新页面仍可重试恢复。

恢复过程需要共享同一个 Promise，避免并发导航或并发 401 发起多个 refresh 请求。

### 登录页行为

登录页挂载时不再无条件清除 token。到达登录页可能是用户主动访问、临时初始化失败或认证失效；凭据清理由明确的状态转换负责，页面展示本身不产生破坏性副作用。

### HTTP 401 处理

- 普通受保护请求返回 401 且存在 refresh token：加入单飞刷新流程。
- refresh 成功：更新 token，并重放全部等待请求。
- refresh 接口返回 401：执行一次本地失效，拒绝等待请求。
- refresh 网络错误或 5xx：保留 refresh token，拒绝等待请求，不执行 logout。
- 没有 refresh token：执行本地失效，不调用后端 logout。

等待队列必须同时支持成功与失败结算，防止刷新失败后已有请求永久 pending。

## 数据契约

现有 `LoginToken.expires_at` 表示 access token 到期时间。前端当前将其保存为 `refreshExpiresAt`，语义错误且没有任何消费者。本次删除该错误状态与存储键，不以 access token 到期时间推断 refresh token 生命周期。

后端 refresh token 仍按 `Middleware.Auth.RefreshExpired` 控制，默认 30 天。若未来前端需要展示 refresh 到期时间，应由 API 新增独立的 `refresh_expires_at` 字段，不能复用 `expires_at`。

## 错误处理准则

| 场景 | 是否清 token | 是否调用后端 logout | 行为 |
| --- | --- | --- | --- |
| 用户主动退出 | 是 | 是，尽力而为 | 进入登录页 |
| access token 401、refresh 成功 | 否 | 否 | 更新 token 并重放请求 |
| refresh token 401 | 是 | 否 | 会话失效，进入登录页 |
| refresh 网络错误/超时/5xx | 否 | 否 | 保留凭据，允许重试 |
| 菜单初始化网络错误/5xx | 否 | 否 | 保留凭据，停止本次导航 |
| 进入登录页 | 否 | 否 | 只展示页面 |

## 测试设计

前端目前没有 Vitest/Jest。新增 Node 原生测试可直接测试抽出的纯会话决策模块，避免为本次修复引入大型测试框架。回归测试覆盖：

- 临时 refresh 故障不会要求清除凭据。
- refresh 401 会要求本地失效。
- 初始化失败不会调用主动 logout。
- 仅持有 refresh token 时选择启动恢复。
- 登录页加载不会清除凭据。
- 刷新等待队列在成功时重放，在失败时全部 reject。

同时运行 Prettier、ESLint、生产构建和现有 Go 认证测试。手工验证使用两个独立浏览器上下文：A、B 分别登录；A 退出后 B 继续可用；模拟 access token 过期后两者均可独立 refresh；模拟菜单 500 后刷新页面仍可恢复。

## 部署注意事项

上线需构建并部署包含本次提交的 Admin 镜像，不能只依赖未锁定内容的 `latest` 缓存。若部署多个 Admin 实例，所有实例必须共享 `JWT_SIGNING_KEY`，认证撤销 store 应从本地 Badger 改为 Redis；该部署调整不属于本次单实例代码修复。
