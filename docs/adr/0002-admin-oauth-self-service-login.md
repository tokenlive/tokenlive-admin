# Admin OAuth 自助登录边界

Admin 支持 Google 与 GitHub 登录时，仅将自助注册能力用于企业内部工具模式。首次通过允许的第三方身份登录时，系统可创建一个 Admin User 与对应的 Self-Service Tenant；公共 API 平台的外部客户自助注册仍归属 Portal，不通过 Admin 完成。

## Considered Options

1. **所有 Google/GitHub 用户都可直接注册 Admin**：被否决，容易制造垃圾用户与租户，也会模糊 Admin 与 Portal 的边界。
2. **第三方登录只允许绑定已有 Admin User**：被否决，无法满足企业内部模式下开发者自助进入 Admin 的需求。
3. **（采用）受控的 Admin OAuth 自助登录**：通过开关、已验证邮箱与 allowlist 限制首次注册；External Identity 绑定到 Admin User，同一已验证邮箱对应同一个 Admin User。

## Consequences

- OAuth 自动注册用户默认不设置可用密码，只能通过已绑定的 External Identity 登录。
- Self-Service Tenant 默认不自动获得全部模型权限，最多获得显式配置的默认模型集合。
- 首次注册不自动创建 User API Key，用户或管理员后续按现有流程创建。
- OAuth 回调不把 Admin access token 放入 URL，而是通过短生命周期一次性票据换取现有 LoginToken。
- 不保存 Google/GitHub access token 或 refresh token，仅保存非敏感身份快照。
- 删除 OAuth 自动注册用户时不自动级联删除 Self-Service Tenant，避免误删后续被授权或使用过的业务资源。
