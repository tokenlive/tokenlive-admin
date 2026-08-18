# TokenLive Admin — 领域上下文术语表

Admin 是 TokenLive 平台的管理后台。其用户角色随部署场景变化：作为企业内部工具时，Admin 用户既包含管理员也包含终端消费者；作为公共 API 平台（搭配 Portal）时，Admin 用户仅为平台运营团队。

## Language

### 用户与权限

**Admin User (管理用户)**:
Admin 系统中的注册用户。在企业内部部署场景下，涵盖管理员、运维和普通开发者（即终端消费者）；在公共平台部署场景下（搭配 Portal），仅为平台运营团队。用户角色通过 RBAC 控制。
_Avoid_: 将 Admin User 固定定义为"仅管理员" — 其含义取决于部署场景。

**External Identity (第三方身份)**:
由外部身份提供方证明的登录身份，用于绑定到一个 Admin User；同一已验证邮箱对应同一个 Admin User。
_Avoid_: 将 Google 身份和 GitHub 身份直接视为两个 Admin User。

**RBAC (角色权限控制)**:
基于 Casbin 的角色访问控制系统。通过 Role → MenuGroup 的绑定关系，控制用户可访问的菜单和 API 资源。

### 业务资源

**Tenant (租户)**:
网关层面的资源配额与策略隔离单元。其含义随部署场景变化：
- **企业内部场景**：可代表部门或团队（可选使用）。
- **公共平台场景**：代表签约的外部 B 端企业客户。
由运维人员在 Admin 后台创建和管理。
_Avoid_: 混淆 Tenant 与 Admin User 的归属关系 — Tenant 是"被管理的业务对象"。

**Self-Service Tenant (自助租户)**:
企业内部部署场景下，Admin User 首次通过第三方身份登录时自动产生的个人资源隔离单元。
_Avoid_: 将自助租户用于公共 API 平台的外部客户注册 — 外部客户自助注册应归属 Portal。

**Tenant API Key (租户 API 密钥)**:
面向 toB 场景的组织级别访问凭证，直接绑定到 Tenant，不关联具体用户。适用于仅部署 Admin + Gateway 的轻量模式，或服务间调用场景。计费统计维度为租户组织。
_Avoid_: 用户密钥、个人密钥

**User API Key (用户 API 密钥)**:
面向终端消费者的个人级别访问凭证，绑定到 Admin 的具体用户。在企业内部场景下，这是开发者获取 API 访问权限的主要方式。
_Avoid_: 管理员密钥

**Provider (供应商)**:
AI 模型的上游服务商或自定义接入端点，例如 OpenAI、Azure OpenAI、Anthropic 等。每个 Provider 下可挂载多个 Model。

**Model (模型)**:
供应商提供的具体 AI 模型（如 gpt-4、claude-3）。每个 Model 支持多别名（ModelAlias）和多接入点（Endpoint），通过加权路由实现流量分配。

**Endpoint (接入点)**:
模型的实际服务地址，带有权重配置，用于实现负载均衡和灰度发布。

**Space (空间)**:
资源隔离的逻辑边界，用于在团队场景下隔离供应商、模型和策略的归属范围。

### 治理策略

**Policy (治理策略)**:
网关层面的流量治理规则集合，包括路由策略（policy_route）、限流策略（policy_limit）、熔断隔离策略（policy_circuit_break）、调用管理与重试策略（policy_invoke）、负载均衡策略（policy_load_balance）、染色打标策略（policy_tagging）等。Policy 可以是策略模板，也可以是模型策略实例；只有模型策略实例参与运行时生效。

**Policy Template (策略模板)**:
`model_id` 为空的治理策略配置，由平台运营方维护，用于跨模型复用。策略模板不直接生效、不创建策略绑定、不同步到 Redis；用户在模板详情或模型详情中选择目标 Model 后，系统复制出一份新的模型策略实例。
_Avoid_: 将策略模板直接绑定到租户、用户或模型运行时维度。

**Model Policy (模型策略实例)**:
`model_id` 非空的治理策略配置，归属于一个具体 Model，权限跟随该 Model 的数据权限。模型策略实例可以从策略模板复制生成，也可以在模型详情页直接创建；同一 Model 下策略名称唯一。
_Avoid_: 将模型策略实例修改回写到原始策略模板。

**Policy Binding (策略绑定)**:
将模型策略实例应用到指定业务维度上的关系。策略绑定是系统内部运行时关系，不作为独立用户资产做数据权限管理；租户与用户等作用维度由绑定关系表达，启停状态由 Policy 本身决定。
- **可叠加策略（Cumulative Policy）**：如限流、染色打标、路由、熔断。支持同一维度重复绑定不同策略实例，运行时网关将它们融合并累加生效。
- **单选覆盖策略（Exclusive Policy）**：如负载均衡、调用重试。同一维度同种类型只能生效唯一实例，多级维度冲突时根据优先级链条覆盖合并。

**Policy Seed (策略种子)**:
新装环境首次启动时自动注入的初始 Policy Template 集合，用于提供开箱即用的推荐配置。种子仅在目标记录从未存在过时创建一次；注入后即成为运营数据，运营方对其的修改与删除不被后续重启覆盖或复活。
_Avoid_: 将策略种子视为每次启动强制同步的系统配置；与菜单初始化（结构化系统数据，启动时覆盖同步）混为一谈。

**Tagging Action (染色打标动作/注入动作)**:
染色打标策略（policy_tagging）触发时执行的操作。动作类型包括：TAG（注入上下文标签）、REQ_HEADER、RSP_HEADER、REQ_COOKIE、RSP_COOKIE、REQ_BODY。

**Priority Chain (可配置优先级链条)**:
网关在运行时通过策略匹配器（`PolicyMatcher`）合并策略时的优先级顺位。该链条完全可配置（如 `global -> tenant -> user -> model -> tenant_model -> user_model`），自底向上遍历进行合并，后者的 non-nil 字段将覆盖前者的零值与非空值。

**Tenant-Model Association (租户模型关联)**:
将大模型（Model）授权给指定租户（Tenant）的物理配置关联。通过 `tenant_model` 表多对多关联模型 ID 与租户编码（Tenant Code），决定了该租户下所有用户、Workspace 及 API Key 能够访问的合法模型范围。

## Relationships

- 一个 **Admin User** 通过 **RBAC** 被授予一个或多个角色，控制其可操作的菜单和 API
- 一个 **Admin User** 可绑定多个 **External Identity**
- 一个 **Admin User** 可通过 `tenant` 字段归属于一个 **Tenant**（企业内部场景下代表部门/团队）
- 一个首次第三方登录的 **Admin User** 可自动拥有一个 **Self-Service Tenant**
- 一个 **Tenant** 可绑定多个 **Model**（通过 Tenant-Model Association）
- 一个 **Tenant** 可对应零个或多个 Portal 的 **Workspace**（1:N，仅公共平台场景）
- **Policy Template** 可复制生成多个 **Model Policy**
- **Model Policy** 属于一个 **Model**，权限跟随该 **Model**
- **Policy Binding** 将 **Model Policy** 应用到 (tenant_code, user_id, model_code) 等维度组合上
- 一个 **Model** 挂载到一个 **Provider**，并有多个 **Endpoint**

## Example dialogue

> **运维:** "我们在公司内部部署了 Admin + Gateway，开发团队的人要怎么拿到 API Key？"
> **架构师:** "给每个开发者创建一个 **Admin User** 账号，分配普通用户角色，然后为他们创建 **User API Key**。可以通过 **Tenant** 按部门隔离资源和策略。"

> **运维:** "现在我们要对外开放 API 服务了，需要加 Portal 吗？"
> **架构师:** "是的。外部客户通过 **Portal** 自助注册 **Workspace**，管理自己的 API Key 和余额。Admin 只留给我们运营团队使用。签约的 B 端客户对应一个 **Tenant**，他们的 Workspace 关联到这个 Tenant。"

## Flagged ambiguities

- `policy_binding.user_id` 在公共平台场景下，需要能查询到 Portal 用户 — Admin 策略绑定页面如何获取 Portal 用户列表待定。
