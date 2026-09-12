# 开发环境配置指南

## 环境变量配置（推荐）

两个项目都支持通过环境变量注入敏感配置，保持配置文件模板化。

### tokenlive-gateway

1. **复制配置模板**：

   ```bash
   cp config/local.yml.example config/local.yml
   ```

2. **创建 .env 文件**：

   ```bash
   cp .env.example .env
   ```

3. **编辑 .env 文件**，填入实际的：
   - Redis 密码
   - OpenAI/Anthropic API Key
   - 其他敏感配置

4. **启动服务**：

   ```bash
   # 环境变量会自动注入到配置文件中
   go run cmd/server/main.go -conf config/local.yml
   ```

### tokenlive-admin

1. **复制配置模板**：

   ```bash
   cp configs/dev/server.toml.example configs/dev/server.toml
   ```

2. **创建 .env 文件**：

   ```bash
   cp .env.example .env
   ```

3. **编辑 .env 文件**，填入实际的：
   - Redis 密码
   - 数据库密码
   - 其他敏感配置

4. **启动服务**：

   ```bash
   # 使用 godotenv 加载 .env 文件，然后启动
   go run main.go server -c configs/dev
   ```

## 环境变量格式

配置文件中使用 `${VAR_NAME}` 或 `${VAR_NAME:default_value}` 格式：

```toml
# 使用环境变量
RedisAddr = "${REDIS_ADDR:localhost:6379}"
RedisPassword = "${REDIS_PASSWORD}"

# 带默认值
DBType = "${DB_TYPE:sqlite3}"
```

## 配置优先级

1. **环境变量**：最高优先级
2. **配置文件**：默认值
3. **代码默认值**：最低优先级

## Docker 部署

```yaml
# docker-compose.yml
services:
  gateway:
    environment:
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=your_password
      - OPENAI_API_KEY=your_key
    volumes:
      - ./config/local.yml:/app/config/local.yml
```

## 不要提交到 Git 的文件

- `.env` - 本地环境变量
- `config/local.yml` - 本地配置（含敏感信息）
- `configs/dev/server.toml` - 开发配置（含敏感信息）
- `configs/prod/server.toml` - 生产配置（含敏感信息）

所有这些文件已经在 `.gitignore` 中，不会被提交。

## 产品版本与更新检查

版本身份来自正在运行的二进制或 all-in-one 宿主，不来自 `General.Version`、镜像标签或前端构建号。开发构建即使使用形似 `v1.2.3` 的名称，也不会冒充正式发布参与升级比较；`latest` 只能用作镜像标签，不能用作程序版本。

Admin 的 TOML 配置可增加：

```toml
[UpdateCheck]
Enabled = true
IntervalSeconds = 21600
TimeoutSeconds = 5
CooldownSeconds = 60

[Gateway]
VersionNamespace = "default"
```

已有 `[Gateway]` 段时直接添加字段，不要重复声明该段。

| 环境变量 | 默认值 | 作用 |
| --- | --- | --- |
| `UPDATE_CHECK_ENABLED` | `true` | 覆盖自动和手动公网检查开关 |
| `UPDATE_CHECK_INTERVAL_SECONDS` | `21600` | 覆盖检查周期，单位秒 |
| `GATEWAY_VERSION_NAMESPACE` | `default` | Admin 与 Gateway 必须一致，区分共享 Redis 中的部署 |
| `GATEWAY_SYNC_TOKEN` | 无 | HTTP 上报必须显式配置，双方值必须完全一致 |

检查启动后异步进行一次，之后每 6 小时检查；一轮各来源并行执行、共享 5 秒截止时间。手动检查具有同一 Admin 实例所有用户共享的 60 秒冷却，并合并进行中的请求。只读版本页面使用缓存，不会因为打开对话框或节点过期而请求公网。关闭 `UPDATE_CHECK_ENABLED` 后，自动/手动公网检查均停止，当前版本、内部上报与节点聚合继续工作；手动接口返回 HTTP 409，冷却中返回 HTTP 429 和 `Retry-After`。

专业版分别查询 Admin 和 Gateway 仓库的 latest Release；只有正式稳定标签可成为候选，不搜索历史最高版本、不回退历史发行。all-in-one 仅在确认 Homebrew 安装渠道时读取官方 tap 当前 Formula，并核实对应 Release 的公开状态、稳定标签和发行资产；不借用专业版来源。失败或过期结果不可作为可执行的升级建议；无网络时仍能显示本地版本。

### Gateway 观测范围与清理

- Gateway 启动成功后异步上报一次，此后每 30 秒上报，进程 UUID 在本次进程内稳定。
- 有配置的 Redis 客户端时优先直接写 Redis；没有 Redis 时才使用 `ADMIN_SERVER_URL` 与 `GATEWAY_SYNC_TOKEN` 的 HTTP 通道。发送失败不切换通道、不阻断业务请求。
- Admin 有 Redis 地址时从同一 DB/namespace 聚合；双方应共享同一 Redis DB。键格式为 `tokenlive:gateway-versions:<namespace>:<UUID>`，每次覆盖写并设置 3 分钟 TTL，不添加永久索引。
- namespace 区分大小写，只允许 1–64 位字母、数字、`_`、`-`。共用 Redis DB 的独立部署应使用不同 namespace。
- Admin 没有 Redis 地址时使用内存注册表，只能看到上报到本实例的节点（`scope=this_admin`）；负载均衡后的多个 Admin 不保证看到相同分布。共享 Redis 时为 `scope=shared`。
- 节点超过 3 分钟未上报自动消失，旧版本组的升级提示随之消失，不触发额外公网检查。停止节点无需人工删除记录；注册表也提供按 UUID 删除路径。不要清空业务 Redis 做验证。
- all-in-one 展示单一 standalone 版本，内嵌 Gateway 不作为专业版节点上报。

### 权限与首次启用

Root 默认具有版本更新管理能力；其他用户通过角色菜单中的 `system.versionUpdates` 授权，资源为 `GET /api/v1/system/updates` 和 `POST /api/v1/system/updates/check`。后端以 POST 能力作统一授权判断，不依据用户名或只隐藏按钮。普通登录用户可以读取 `GET /api/v1/current/version` 的当前版本和匿名聚合，不能查看更新专属信息或主动检查。所有版本界面都不显示节点 UUID。

旧版本没有这套检测和提醒能力，首次仍需维护人员人工升级到包含此功能的受控发布。界面仅提供发行说明和复制升级命令，不下载、不安装、不重启。兼容矩阵与人工操作见 [快速指引](quickstart.md)。
