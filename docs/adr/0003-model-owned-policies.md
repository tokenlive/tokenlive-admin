# Policy templates and model-owned policy instances

Status: accepted

Governance policies are split into two shapes:

- Policy Template: `model_id` is empty. It is reusable configuration maintained by platform operators. It does not bind to runtime dimensions, does not sync to Redis, and does not take effect directly.
- Model Policy: `model_id` is set. It belongs to exactly one Model, its data permissions follow that Model, and it is the only policy shape that can be bound and synced to runtime configuration.

Policy Binding remains an internal runtime relationship for tenant/user/model matching. It is not a separately managed user-facing asset and does not have its own enabled state; policy enablement lives on the policy itself.

We chose this over independently permissioned reusable policies because reusable effective policies made authorization ambiguous: users could have permission to manage a Model but not the Policy bound to it, forcing redaction and dual authorization rules. Copying templates into model-owned instances keeps reuse convenient while anchoring effective policy authorization to existing Model data permissions.
