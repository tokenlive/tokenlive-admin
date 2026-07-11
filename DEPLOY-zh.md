# TokenLive Admin 部署指南

> **语言 / Language:** [English](DEPLOY.md) | 简体中文

本文档提供了 TokenLive Admin (管理控制台) 的详细部署指引，涵盖本地开发部署、单容器 Docker 部署，以及在生产环境中通过 **`tokenlive-deploy`** 编排方案进行一键式完整集群部署。

---

## 目录

- [一、本地开发构建与运行](#一本地开发构建与运行)
- [二、Docker 单镜像部署](#二docker-单镜像部署)
- [三、使用 tokenlive-deploy 统一编排部署（推荐）](#三使用-tokenlive-deploy-统一编排部署推荐)
  - [1. 快速启动方式](#1-快速启动方式)
  - [2. 本地镜像协同构建部署](#2-本地镜像协同构建部署)
  - [3. 配置与分布式运行模式](#3-配置与分布式运行模式)
  - [4. 可选组件（监控、日志）](#4-可选组件监控日志)
- [四、镜像结构与配置说明](#四镜像结构与配置说明)
- [五、健康检查与安全建议](#五健康检查与安全建议)

---

## 一、本地开发构建与运行

在本地开发环境中，你可以使用 `Makefile` 中定义的命令来完成前后端的编译和启动。

### 1. 准备工作

- **Go**: 1.19+
- **Node.js**: 18+
- **数据库**: MySQL 5.7+ / PostgreSQL / SQLite 3
- **缓存**: Redis 6.0+ (可选，默认可使用内存缓存)

### 2. 数据库初始化

创建数据库并导入初始化表结构：

```sql
CREATE DATABASE tokenlive CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
```

```bash
mysql -u root tokenlive < scripts/init.sql
```

### 3. 本地构建命令

```bash
# 1. 编译前端静态资源，编译后端二进制文件
make build-all

# 2. 启动服务（前台运行，包含前端静态服务和后端 API）
make serve

# 3. 后台启动服务
make serve-d
```

> [!NOTE]
> 本地开发时，系统默认加载 `configs/dev/server.toml` 配置文件，并尝试读取 `configs/.env` 加载环境变量占位符。

---

## 二、Docker 单镜像部署

TokenLive Admin 采用前后端一体化架构。在 Docker 镜像构建中，使用**多阶段构建**（Multi-stage Build）把 Vue 3 前端静态产物和 Golang 后端二进制文件打包进同一个 Alpine 镜像中。

### 1. 自动构建与运行

你可以直接使用 Makefile 封装的 Docker 命令：

```bash
# 构建本地 Docker 镜像
make docker-build

# 构建并推送镜像到远程仓库
make docker-push
```

### 2. 手动构建与运行

如果不使用 `Makefile`，也可以手动执行以下 Docker 命令：

```bash
# 1. 在项目根目录下构建镜像
docker build -t tokenlive-admin:latest .

# 2. 运行容器（默认使用 SQLite 数据库存放在容器内的 /app/data 目录中）
docker run -d -p 8040:8040 --name tokenlive-admin tokenlive-admin:latest
```

---

## 三、使用 tokenlive-deploy 统一编排部署（推荐）

对于生产环境或需要完整网关能力的场景，强烈建议使用 **[tokenlive-deploy](https://github.com/tokenlive/tokenlive-deploy)** 一键部署项目。它提供了集成了 **Admin 控制台、Gateway 网关、Caddy 反向代理、Redis、Prometheus** 等核心组件的 Docker Compose 编排方案。

### 1. 快速启动方式

#### 方式一：在线一键安装（推荐）

无需手动克隆仓库，直接在目标服务器上执行以下命令即可启动交互式向导：

```bash
curl -fsSL https://raw.githubusercontent.com/tokenlive/tokenlive-deploy/main/install.sh | bash
```

- **非交互式部署**（CI/CD 或自动化部署场景）：
  ```bash
  curl -fsSL https://raw.githubusercontent.com/tokenlive/tokenlive-deploy/main/install.sh | bash -s -- --domain api.example.com --password your-password --yes
  ```

#### 方式二：克隆仓库部署

```bash
git clone https://github.com/tokenlive/tokenlive-deploy.git
cd tokenlive-deploy

# 运行交互式安装脚本
chmod +x install.sh
./install.sh
```

一键部署完成后，可以通过以下统一入口进行访问：
- **Admin 后台**: `http://<your-domain-or-ip>` (默认账号/密码: `admin` / `admin`)
- **网关 API**: `http://<your-domain-or-ip>/v1`

---

### 2. 本地镜像协同构建部署

当你在本地对 `tokenlive-admin` 或 `tokenlive-gateway` 代码进行了修改，想在本地联合测试最新容器时，可以采用协同构建：

1. **准备目录结构**：确保这三个项目的目录在同级目录下：
   ```
   ├── tokenlive-admin/    (当前控制台项目)
   ├── tokenlive-gateway/  (网关项目)
   └── tokenlive-deploy/   (编排部署项目)
   ```

2. **本地构建镜像**：
   ```bash
   cd ../tokenlive-deploy
   chmod +x build-images.sh
   
   # 构建 Admin 和 Gateway 的本地镜像
   ./build-images.sh
   
   # 或者只构建本地 Admin 镜像
   ./build-images.sh --admin
   ```

3. **使用构建配置启动 Compose**：
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
   ```

---

### 3. 配置与分布式运行模式

在 `tokenlive-deploy` 中，控制台与网关存在两种配置与运行状态同步模式：

#### 模式一：极简单机模式（默认）
- **特点**：零外部依赖，一键启动。
- **配置同步**：网关默认通过 **HTTP 轮询 (HTTP Polling)** 定时从控制台（Admin）拉取最新的模型、密钥及路由策略并热更新。
- **状态存储**：限流、熔断及配额扣减状态保存在网关的**单机本地内存**中。
- **配置方式**：在 `tokenlive-deploy` 目录的 `.env` 中无需配置 Redis，直接启动即可。

#### 模式二：Redis 分布式集群模式（推荐）
- **特点**：配置实时同步，状态多实例共享，适合横向扩容和高可用部署。
- **同步与共享**：控制台的变更将通过 Redis 实时同步至网关，且多台网关实例共享限流、熔断和配额状态。
- **启动与配置**：
  1. 使用带 Redis 的 Profile 启动：
     ```bash
     docker compose --profile with-redis up -d
     ```
  2. 编辑 `.env` 文件，启用并配置 Redis 模式：
     ```env
     GATEWAY_CONFIG_SOURCE=redis
     GATEWAY_STATE_STORE=redis
     REDIS_ADDR=redis:6379
     REDIS_PASSWORD=
     REDIS_DB=0
     ```

---

### 4. 可选组件（监控、日志）

#### A. 启用 Prometheus 监控
1. 使用带监控的 Profile 启动：
   ```bash
   docker compose --profile with-monitoring up -d
   ```
2. 在 `.env` 中指定 Prometheus 服务器地址，此时 Admin 后台可以直接在前端展示调用监控大屏：
   ```env
   PROMETHEUS_SERVER_URL=http://prometheus:9090
   ```

#### B. 启用 ClickHouse 审计与日志记录
ClickHouse 为网关日志的高性能存储组件，需连接至已有的 ClickHouse 服务。在 `.env` 中启用并注入连接配置：
```env
CLICKHOUSE_ENABLED=true
CLICKHOUSE_ADDR=clickhouse.example.com:9000
CLICKHOUSE_DATABASE=tokenlive_gateway
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=your_password
```

---

## 四、镜像结构与配置说明

### 1. 容器内镜像结构

- **/usr/bin/tokenlive-admin**: 后端二进制可执行文件
- **/app/dist**: 前端打包静态资源
- **/app/configs**: 配置文件目录 (加载 `server.toml`)
- **/app/data**: 默认的 SQLite 数据库存储目录

### 2. 关键环境变量

运行 `tokenlive-admin` 容器时，支持以下环境变量注入以覆盖 `configs/prod/server.toml` 默认配置：

| 环境变量名 | 默认值 | 说明 |
|---|---|---|
| `GIN_MODE` | `release` | Gin 框架运行模式 (`debug`/`release`) |
| `ROOT_PASSWORD` | `admin` | 超级管理员的初始默认密码 |
| `DB_TYPE` | `sqlite3` | 数据库类型 (`mysql`/`postgres`/`sqlite3`) |
| `DB_DSN` | `/app/data/tokenlive.db` | 数据库连接串 (DSN) |
| `STORAGE_CACHE_TYPE` | `memory` | 缓存类型 (`memory`/`redis`/`badger`) |
| `REDIS_ADDR` | `""` | Redis 地址 (例如 `redis:6379`) |
| `REDIS_PASSWORD` | `""` | Redis 连接密码 |
| `PROMETHEUS_SERVER_URL` | `http://prometheus:9090` | Prometheus 服务端地址 (用于展示监控大屏) |
| `GATEWAY_SYNC_TOKEN` | `default_tokenlive_sync_secret_token_2026` | 与网关进行配置同步的安全密钥令牌 |

---

## 五、健康检查与安全建议

### 1. 健康检查

可以通过访问以下接口判断 Admin 服务的运行状态：

```bash
curl http://localhost:8040/health
```

返回包含 `{"status":"OK"}` 的 HTTP 200 响应表示服务运行正常。

### 2. 安全建议

1. **修改默认管理员密码**：在容器首次运行前或在 `.env` 中通过 `ROOT_PASSWORD` 注入复杂的初始密码。
2. **隐藏内部端口**：生产环境中只对外暴露 Caddy 的 `80` 和 `443` 端口，严禁向外网直接暴露 `8040` (Admin API) 和 `8000` (Gateway API)。
3. **开启 HTTPS**：利用 Caddy 自动申请免费 SSL 证书，在 `.env` 中配置正确的 `DOMAIN` 域名。
4. **数据备份**：定期备份挂载的外部数据卷或容器内部数据库：
   ```bash
   # SQLite 备份示例
   docker cp tokenlive-admin:/data/tokenlive.db ./backup/
   ```
