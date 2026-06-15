# 快速开始

## 本地开发配置

### tokenlive-gateway

```bash
cd tokenlive-gateway

# 1. 复制配置模板
cp config/local.yml.example config/local.yml

# 2. 创建环境变量文件
cp .env.example .env

# 3. 编辑 .env 文件，填入：
#    - REDIS_PASSWORD (Redis 密码)
#    - OPENAI_API_KEY (OpenAI API Key)
#    - ANTHROPIC_API_KEY (Anthropic API Key, 如需要)

# 4. 启动服务
go run cmd/server/main.go -conf config/local.yml
```

### tokenlive-admin

```bash
cd tokenlive-admin

# 1. 复制配置模板
cp configs/dev/server.toml.example configs/dev/server.toml

# 2. 创建环境变量文件
cp .env.example .env

# 3. 编辑 .env 文件，填入：
#    - REDIS_PASSWORD (Redis 密码)
#    - DB_DSN (数据库连接字符串)
#    - 其他敏感配置

# 4. 启动服务
go run main.go server -c configs/dev
```

## 环境变量优先级

配置支持三级优先级（从高到低）：

1. **系统环境变量** - 直接设置 `export REDIS_PASSWORD=xxx`
2. **.env 文件** - 在项目根目录的 `.env` 文件中设置
3. **配置文件默认值** - 在 `xxx.example` 文件中定义的默认值

示例：
```toml
# 配置文件中
RedisAddr = "${REDIS_ADDR:localhost:6379}"
```

- 如果设置了 `REDIS_ADDR` 环境变量，使用该值
- 否则使用默认值 `localhost:6379`

## Docker 部署

```bash
# 使用环境变量直接注入
docker run -e REDIS_PASSWORD=xxx \
           -e OPENAI_API_KEY=xxx \
           -v ./config/local.yml:/app/config/local.yml \
           your-image

# 或使用 .env 文件
docker run --env-file .env \
           -v ./config/local.yml:/app/config/local.yml \
           your-image
```

## 常见场景

### 场景 1: 本地开发，使用远程 Redis

`.env` 文件：
```bash
REDIS_ADDR=your-redis-server:6379
REDIS_PASSWORD=your-redis-password
```

### 场景 2: 测试环境，使用 MySQL

`.env` 文件：
```bash
DB_DSN=your-mysql-dsn
```

### 场景 3: 生产部署，使用环境变量

直接设置系统环境变量或 Docker 环境变量，不使用 `.env` 文件。

## 安全提示

- ✅ `.env` 文件已在 `.gitignore` 中，不会提交到 Git
- ✅ 配置模板 `*.example` 可以安全提交，不包含敏感信息
- ✅ 实际配置文件 `local.yml`、`server.toml` 已被忽略
- ❌ 不要将包含密码的文件提交到版本控制
