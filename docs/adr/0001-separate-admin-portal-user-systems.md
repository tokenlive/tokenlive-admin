# Admin 与 Portal 双模式用户体系

TokenLive 支持两种部署模式，Admin 的用户角色随部署场景自然变化，两种模式共享同一套代码，无需分叉。

**企业内部工具模式**（Admin + Gateway）：Admin 用户涵盖管理员和终端消费者（内部开发者），通过 RBAC 角色区分权限。Tenant 可选用于部门/团队隔离。终端消费者通过 Admin 的 User API Key 调用 Gateway。`policy_binding.user_id` 直接使用 Admin 的 `user.id`。

**公共 API 平台模式**（Admin + Gateway + Portal）：Admin 用户退化为仅平台运营团队。终端消费者在 Portal 注册，通过 Workspace 管理 API Key 和余额。Tenant 代表签约的外部 B 端企业，Workspace 关联到 Tenant（1:N）。Gateway 通过统一的 Redis 缓存解析 API Key 身份，不区分来源系统。

## Considered Options

1. **Admin 强制多租户 + 客户开 Admin 账号**：被否决，公共平台场景下安全隔离复杂度过高。
2. **Portal 完全替代 Admin 用户体系**：被否决，企业内部场景不需要 Portal，Admin 自身已能承载终端消费者。
3. **（采用）双模式共存**：Admin 代码不变，用户角色含义随部署场景自然适配。Portal 是叠加层，不是替代层。

## Consequences

- Admin 的 `user` 表和 RBAC 体系保持不变，同时服务两种部署模式。
- `user.tenant` 字段在企业内部模式下有实际隔离意义（部门/团队），在公共平台模式下对 admin 用户不再重要。
- `policy_binding.user_id` 存储 API Key 所关联的用户 ID — 企业内部模式来自 admin `user.id`，公共平台模式来自 Portal `users.id`。
- Portal `workspaces` 表需增加 `tenant_code` 字段以关联 Admin 的 Tenant。
- Gateway 通过 Redis 缓存统一解析 API Key，不区分来源系统。
