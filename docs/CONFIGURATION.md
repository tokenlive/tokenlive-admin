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
