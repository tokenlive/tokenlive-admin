# TokenLive Admin Deployment Guide

> **Language / 语言:** English | [简体中文](DEPLOY-zh.md)

This document provides detailed deployment guidelines for TokenLive Admin (the management console), covering local development execution, standalone container Docker deployment, and production-grade cluster orchestration using **`tokenlive-deploy`**.

---

## Table of Contents

- [1. Local Development Build and Run](#1-local-development-build-and-run)
- [2. Docker Single-Image Deployment](#2-docker-single-image-deployment)
- [3. Orchestrated Deployment with tokenlive-deploy (Recommended)](#3-orchestrated-deployment-with-tokenlive-deploy-recommended)
  - [3.1 Quick Start Methods](#31-quick-start-methods)
  - [3.2 Co-Building with Local Source Code](#32-co-building-with-local-source-code)
  - [3.3 Sync and Distributed Running Modes](#33-sync-and-distributed-running-modes)
  - [3.4 Optional Add-ons (Monitoring, Logging)](#34-optional-add-ons-monitoring-logging)
- [4. Image Structure and Configuration Details](#4-image-structure-and-configuration-details)
- [5. Health Check and Security Best Practices](#5-health-check-and-security-best-practices)

---

## 1. Local Development Build and Run

In a local development environment, you can use commands defined in the `Makefile` to compile both the frontend and backend, and run the service.

### 1.1 Prerequisites

- **Go**: 1.19+
- **Node.js**: 18+
- **Database**: MySQL 5.7+ / PostgreSQL / SQLite 3
- **Cache**: Redis 6.0+ (Optional, in-memory cache is used by default)

### 1.2 Initialize Database

Create a database and import the initial schema:

```sql
CREATE DATABASE tokenlive CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
```

```bash
mysql -u root tokenlive < scripts/init.sql
```

### 1.3 Local Build Commands

```bash
# 1. Compile frontend static assets and build backend binary
make build-all

# 2. Run the service (foreground execution: starts both frontend server and backend API)
make serve

# 3. Start the service in the background
make serve-d
```

> [!NOTE]
> For local development, the system loads `configs/dev/server.toml` as the default config file, and attempts to load placeholders from `configs/.env`.

---

## 2. Docker Single-Image Deployment

TokenLive Admin is built on an integrated frontend-backend architecture. The Docker image uses a **multi-stage build** to compile the Vue 3 frontend static assets and Golang backend binary, packaging them into a single lightweight Alpine Linux image.

### 2.1 Automated Build and Run

You can directly use the Docker commands packaged in the Makefile:

```bash
# Build the local Docker image
make docker-build

# Build and push the image to a remote registry
make docker-push
```

### 2.2 Manual Build and Run

If you do not want to use the `Makefile`, execute the following Docker commands manually:

```bash
# 1. Build the image at the project root directory
docker build -t tokenlive-admin:latest .

# 2. Run the container (SQLite is used by default and saved inside /app/data in the container)
docker run -d -p 8040:8040 --name tokenlive-admin tokenlive-admin:latest
```

---

## 3. Orchestrated Deployment with tokenlive-deploy (Recommended)

For production environments or when the full gateway capability is required, it is highly recommended to use **[tokenlive-deploy](https://github.com/tokenlive/tokenlive-deploy)**. It provides a Docker Compose orchestration stack containing **Admin Console, Gateway, Caddy reverse proxy, Redis, and Prometheus**.

### 3.1 Quick Start Methods

#### Method 1: Online One-Click Install (Recommended)

Execute the command below on your target server to start the interactive configuration wizard:

```bash
curl -fsSL https://raw.githubusercontent.com/tokenlive/tokenlive-deploy/main/install.sh | bash
```

- **Non-Interactive Deployment** (for CI/CD or automated scenarios):
  ```bash
  curl -fsSL https://raw.githubusercontent.com/tokenlive/tokenlive-deploy/main/install.sh | bash -s -- --domain api.example.com --password your-password --yes
  ```

#### Method 2: Git Clone Deployment

```bash
git clone https://github.com/tokenlive/tokenlive-deploy.git
cd tokenlive-deploy

# Run the interactive installer
chmod +x install.sh
./install.sh
```

Once deployment succeeds, you can access the platform via Caddy:
- **Admin Console**: `http://<your-domain-or-ip>` (Default Credentials: `admin` / `admin`)
- **Gateway API**: `http://<your-domain-or-ip>/v1`

---

### 3.2 Co-Building with Local Source Code

If you modified the code in `tokenlive-admin` or `tokenlive-gateway` and want to test these changes locally together with the stack:

1. **Verify Project Structure**: Place all three projects in the same parent directory:
   ```
   ├── tokenlive-admin/    (Current admin console repository)
   ├── tokenlive-gateway/  (Gateway API repository)
   └── tokenlive-deploy/   (Orchestration deployment repository)
   ```

2. **Build Local Images**:
   ```bash
   cd ../tokenlive-deploy
   chmod +x build-images.sh
   
   # Build local images for both Admin and Gateway
   ./build-images.sh
   
   # Or build local Admin image only
   ./build-images.sh --admin
   ```

3. **Start Compose with Build Overrides**:
   ```bash
   docker compose -f docker-compose.yml -f docker-compose.build.yml up -d --build
   ```

---

### 3.3 Sync and Distributed Running Modes

In `tokenlive-deploy`, the Admin Console and Gateway support two synchronization and running modes:

#### Mode A: Standalone Mode (Default)
- **Features**: Zero external dependencies, one-click startup.
- **Config Sync**: The Gateway periodically pulls the latest models, API keys, and routing policies from the Admin Console via **HTTP Polling** and applies them hot.
- **State Store**: Rate limiting, circuit breakers, and token quota statistics are stored in the Gateway's **in-memory storage** locally.
- **Usage**: Do not launch Redis in `.env` or Compose. Simply run the stack directly.

#### Mode B: Redis Distributed Cluster Mode (Recommended)
- **Features**: Instant real-time config sync, state sharing across multiple Gateway instances. Suitable for scaling out and high availability.
- **Sync & Share**: Changes in Admin are published and synced instantly via Redis. Multiple Gateway instances share rate limits, circuit breaker, and quota states.
- **Startup and Configuration**:
  1. Start the stack with the Redis profile enabled:
     ```bash
     docker compose --profile with-redis up -d
     ```
  2. Edit `.env` to enable Redis sync:
     ```env
     GATEWAY_CONFIG_SOURCE=redis
     GATEWAY_STATE_STORE=redis
     REDIS_ADDR=redis:6379
     REDIS_PASSWORD=
     REDIS_DB=0
     ```

---

### 3.4 Optional Add-ons (Monitoring, Logging)

#### A. Enable Prometheus Monitoring
1. Run Compose with the monitoring profile:
   ```bash
   docker compose --profile with-monitoring up -d
   ```
2. Set the Prometheus server URL in `.env` to show metrics charts on the Admin Console Dashboard:
   ```env
   PROMETHEUS_SERVER_URL=http://prometheus:9090
   ```

#### B. Enable ClickHouse Audit Logs
ClickHouse is used for high-performance gateway access logging. It connects to an external ClickHouse service. Add the connection parameters in `.env`:
```env
CLICKHOUSE_ENABLED=true
CLICKHOUSE_ADDR=clickhouse.example.com:9000
CLICKHOUSE_DATABASE=tokenlive_gateway
CLICKHOUSE_USERNAME=default
CLICKHOUSE_PASSWORD=your_password
```

---

## 4. Image Structure and Configuration Details

### 4.1 Internal Image Directory Structure

- **/usr/bin/tokenlive-admin**: Backend executable binary
- **/app/dist**: Frontend static UI assets compiled by Vue
- **/app/configs**: Directory hosting the configuration file (`server.toml`)
- **/app/data**: Default directory for the local SQLite database

### 4.2 Essential Environment Variables

When running the `tokenlive-admin` container, you can inject environment variables to override the default settings in `configs/prod/server.toml`:

| Env Name | Default Value | Description |
|---|---|---|
| `GIN_MODE` | `release` | Gin framework environment mode (`debug`/`release`) |
| `ROOT_PASSWORD` | `admin` | Initial admin password for the root user |
| `DB_TYPE` | `sqlite3` | Database engine (`mysql`/`postgres`/`sqlite3`) |
| `DB_DSN` | `/app/data/tokenlive.db` | Database connection DSN |
| `STORAGE_CACHE_TYPE` | `memory` | Cache type (`memory`/`redis`/`badger`) |
| `REDIS_ADDR` | `""` | Redis address (e.g., `redis:6379`) |
| `REDIS_PASSWORD` | `""` | Redis password |
| `PROMETHEUS_SERVER_URL` | `http://prometheus:9090` | Prometheus server URL (for Dashboard display) |
| `GATEWAY_SYNC_TOKEN` | `default_tokenlive_sync_secret_token_2026` | Security token for synchronizing configs with the Gateway |

---

## 5. Health Check and Security Best Practices

### 5.1 Health Check

Verify whether the Admin API is running by hitting the health endpoint:

```bash
curl http://localhost:8040/health
```

An HTTP 200 response returning `{"status":"OK"}` confirms that the server is healthy.

### 5.2 Security Best Practices

1. **Change Default Admin Password**: Always change the default admin password by setting `ROOT_PASSWORD` in `.env` before running in production.
2. **Restrict Internal Ports**: In a production setup, expose only the Caddy HTTP/HTTPS ports (`80`/`443`) to the public internet. Keep `8040` (Admin Console) and `8000` (Gateway API) private.
3. **Enable HTTPS**: Configure your actual domain name in `DOMAIN` under `.env` so Caddy automatically retrieves and renews SSL certificates.
4. **Data Backup**: Periodically backup mounted volumes or the SQLite database file:
   ```bash
   # SQLite backup example
   docker cp tokenlive-admin:/data/tokenlive.db ./backup/
   ```
