# Domain Context Glossary

## User API Key (用户 API 密钥)
面向 toC 场景的个人级别访问凭证，绑定到具体的用户（User）。用于网关对下游客户端的鉴权与计量，计费统计维度为个人用户。
_Avoid_: 租户密钥、组织密钥

## Tenant API Key (租户 API 密钥)
面向 toB 场景的组织级别访问凭证，直接绑定到租户（Tenant），不关联具体用户。代表整个企业或组织的共享凭证，适合服务间调用。计费统计维度为租户组织。与 User API Key 互斥——同一个 apikey 只能是租户级别或用户级别，不会同时关联两者。
_Avoid_: 用户密钥、个人密钥

## Provider (供应商)
AI 模型的上游服务商或自定义接入端点，例如 OpenAI、Azure OpenAI、Anthropic 等。每个 Provider 下可挂载多个 Model。

## Model (模型)
供应商提供的具体 AI 模型（如 gpt-4、claude-3）。每个 Model 支持多别名（ModelAlias）和多接入点（Endpoint），通过加权路由实现流量分配。

## Endpoint (接入点)
模型的实际服务地址，带有权重配置，用于实现负载均衡和灰度发布。

## Policy (治理策略)
网关层面的流量治理规则集合，包括路由策略（policy_route）、限流策略（policy_limit）、熔断隔离策略（policy_circuit_break）、调用管理与重试策略（policy_invoke）、负载均衡策略（policy_load_balance）、染色打标策略（policy_tagging）等。这些策略作为独立无状态的“模板资产”存储在各自的表结构中，本身不包含具体的应用实体对象。

## Policy Binding (策略绑定)
通过多对多关系绑定表（`policy_binding`），将具体的治理策略（`policy_id`）应用到指定的业务维度上。绑定维度由 `tenant_code`、`user_id`、`model_code` 三个核心物理字段组成，不限则为空字符串。
- **可叠加策略（Cumulative Policy）**：如限流、染色打标、路由、熔断。支持同一维度重复绑定不同策略实例，运行时网关将它们融合并累加生效（如在 Policy struct 中以 Slice 数组形式合并）。
- **单选覆盖策略（Exclusive Policy）**：如负载均衡、调用重试。同一维度同种类型只能生效唯一实例。如果在多级维度发生冲突，运行时会根据优先级链条进行高优先级指针非 nil 覆盖合并（在 Policy struct 中以单指针形式合并）。

## Tagging Action (染色打标动作/注入动作)
染色打标策略（policy_tagging）触发时执行的操作。动作可以有不同的操作类型（Type）：
- **TAG**：默认动作，将键值对注入到网关请求上下文标签（GatewayContext.Tags）中，用于后续路由等模块消费。
- **REQ_HEADER**：注入或修改发送至上游模型的 HTTP 请求头部（Request Header）。
- **RSP_HEADER**：注入或修改返回给客户端的 HTTP 响应头部（Response Header）。
- **REQ_COOKIE**：注入或修改发送至上游模型的 HTTP Cookie。
- **RSP_COOKIE**：注入或修改返回给客户端的 HTTP Cookie (Set-Cookie)。
- **REQ_BODY**：修改或改写发送至上游模型的 HTTP 请求体 JSON 字段。

## Priority Chain (可配置优先级链条)
网关在运行时通过策略匹配器（`PolicyMatcher`）合并策略时的优先级顺位。该链条完全可配置（如 `global -> tenant -> user -> model -> tenant_model -> user_model`），自底向上遍历进行合并，后者的 non-nil 字段将覆盖前者的零值与非空值，最终生成当次请求的动态策略参数。


## Space (空间)
资源隔离的逻辑边界，用于在多租户或团队场景下隔离供应商、模型和策略的归属范围。

## Tenant (租户)
系统管理和资源隔离的顶层账户实体，代表一个独立的企业、组织或团队。
- **用户归属**：单租户关系（1:N），即一个用户必须且仅能属于一个租户。
- **逻辑关联**：通过租户唯一英文标识（Tenant Code，如 `default`、`company-a`）对系统中的用户（User）、角色（Role）、空间（Space）和数据权限（Data Permission）进行软关联，从而实现无物理强外键侵入的业务分区与数据隔离。

## Tenant-Model Association (租户模型关联)
将大模型（Model）授权给指定租户（Tenant）的物理配置关联。通过 `tenant_model` 表多对多关联模型 ID 与租户编码（Tenant Code），决定了该租户下的所有用户及客户端 API Key 能够访问的合法模型范围。管理员可在租户详情页使用穿梭框对其进行一次性批量配置。

## RBAC (角色权限控制)
基于 Casbin 的角色访问控制系统。通过 Role → MenuGroup 的绑定关系，控制用户可访问的菜单和 API 资源。

