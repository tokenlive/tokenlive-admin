<div align="center">
  <a href="https://github.com/tokenlive/tokenlive-admin">
    <img alt="TokenLive-Admin Logo" width="188" src="./frontend/public/images/logo.png">
  </a>
  <br>
  <br>

[![license](https://img.shields.io/github/license/tokenlive/tokenlive-admin.svg)](LICENSE)

  <h1>TokenLive Admin</h1>
</div>

> [中文文档](README-zh.md)
>
> 📖 **"In the syntax of code, let governance endure and life remain forever green."** — Learn more about the tribute and story behind our name: [The Origin of TokenLive](./docs/origin_of_tokenlive_en.md).

## Introduction

TokenLive Admin is the admin console for the [TokenLive](https://github.com/tokenlive/tokenlive-gateway) ecosystem. This project is a high-performance, enterprise-grade large model (LLM) gateway designed specifically for the LLM computing power ecosystem. The gateway is designed based on a mature microservice governance model, with built-in rich intelligent routing and traffic governance strategies, and naturally supports massive concurrent traffic and elastic horizontal scaling. By deeply optimizing the request chain, the gateway can greatly reduce the failure rate of LLM calls, providing a solid stability guarantee for high-concurrency, high-availability AI application scenarios.

![Dashboard Screenshot](./docs/images/dashboard.png)

![Ops Screenshot](./docs/images/ops.png)

## Features

### Resource Management

Manage AI model **providers** (e.g. OpenAI, Azure, custom endpoints) and their **models**. Each model can have multiple aliases and endpoints with weighted routing.

### Governance Policies

A rich set of traffic governance policies applied at the gateway level:

- **Route Policies** — Tag-based and detail-level request routing
- **Rate Limiting** — Server-side and client-side flow control
- **Circuit Breaking** — Automatic fault isolation with configurable thresholds (failure rate, slow-call rate, TTFT-based)
- **Fault Injection** — Simulate delays and errors for resilience testing
- **Load Balancing** — Pluggable load-balancing strategies across endpoints
- **Service Authentication** — Mutual authentication between services
- **Invocation Management** — Cross-service call chain configuration
- **Access Permission** — Fine-grained API-level permission control

### RBAC & System Management

- Role-based access control powered by Casbin
- Menu and permission management with resource-group support
- User management with department grouping
- API Key management for downstream client authentication
- Operation log audit

### Space Management

Multi-space (tenant-level) resource isolation for organizing providers, models, and policies.

## Project Structure

This project uses an integrated frontend-backend architecture:

- **Frontend**: Vue 3 + Vite + Ant Design Vue
- **Backend**: Go + Gin + GORM
- **Deployment**: Multi-stage Docker build, single image containing both frontend and backend

```
tokenlive-admin/
├── frontend/              # Frontend Vue 3 SPA
│   └── src/
│       ├── apis/modules/  # API service modules (mirrors backend modules)
│       ├── router/routes/ # Route definitions (menu-driven, dynamic)
│       ├── views/         # Page components by domain
│       └── store/         # Pinia state management
├── internal/              # Backend Go code
│   ├── mods/              # Domain modules (rbac, resource, space, policy)
│   │   ├── api/           # HTTP handlers
│   │   ├── biz/           # Business logic
│   │   ├── dal/           # Data access layer (GORM)
│   │   └── schema/        # Data models & DTOs
│   └── wirex/             # Google Wire dependency injection
├── pkg/                   # Shared packages (cachex, gormx, jwtx, middleware, etc.)
├── configs/               # TOML configuration files
├── cmd/                   # CLI commands (start/stop/version)
├── scripts/               # Database init scripts & utilities
├── docs/                  # Swagger docs, ADRs, and specs
├── main.go                # Application entry point
├── Makefile               # Build scripts
├── Dockerfile             # Docker multi-stage build
└── docker-compose.yml     # Docker Compose configuration
```

## Tech Stack

| Layer    | Technology                            |
|----------|---------------------------------------|
| Frontend | Vue 3, Vite, Ant Design Vue, Pinia    |
| Backend  | Go, Gin, GORM, Google Wire            |
| Auth     | JWT, Casbin RBAC                      |
| Database | MySQL / PostgreSQL / SQLite            |
| Cache    | Redis / Badger / In-memory             |
| Deploy   | Docker multi-stage, docker-compose     |

## Quick Start

### Local Development

#### 1. Prerequisites

- Go 1.19+
- Node.js 18+
- MySQL 5.7+ (or PostgreSQL / SQLite)
- Redis 6.0+ (optional, can use in-memory cache)

#### 2. Initialize Database

Create the database and import the schema:

```sql
CREATE DATABASE tokenlive CHARACTER SET utf8mb4 COLLATE utf8mb4_bin;
```

```bash
mysql -u root tokenlive < scripts/init.sql
```

#### 3. Configure

Edit the backend config in `configs/dev/server.toml` to match your database and cache settings.

#### 4. Build and Run

```bash
# Build frontend and backend, then start
make serve

# Or run backend only (with hot-reload via air)
make start
```

#### 5. Access

Open your browser and navigate to `http://localhost:8040`. Default admin credentials:

- Username: `admin`
- Password: `admin`

### Docker Deployment

```bash
# Build image
make docker-build

# Run with Docker Compose (recommended)
docker-compose up -d
```

See [DEPLOY.md](DEPLOY.md) for detailed deployment instructions.

## Build Commands

```bash
make start             # Run backend with hot-reload (air)
make build             # Build backend binary to bin/tokenlive-admin
make build-frontend    # Build frontend to frontend/dist
make build-all         # Build frontend + backend
make serve             # Build all then start server on :8040
make wire              # Regenerate Wire dependency injection
make swagger           # Regenerate Swagger docs
make docker-build      # Build Docker image
make docker-push       # Build and push image
make clean             # Clean build artifacts
make build-cross-all   # Cross-compile for linux/darwin/windows
```

## Configuration

### Frontend

- `frontend/.env.dev` — Development environment
- `frontend/.env.prod` — Production environment

### Backend (TOML)

Configuration files are located in `configs/`:

- `configs/dev/` — Development (MySQL + Redis)
- `configs/prod/` — Production

Key config sections: `[General]`, `[Storage]`, `[Storage.DB]`, `[Storage.Cache]`, `[Middleware]`.

#### Configuration with Environment Variables

Sensitive configurations (passwords, tokens, connection strings) use environment variable placeholders with defaults. Create a `.env` file in the project root for local development:

```bash
# Copy the example file
cp .env.example .env

# Edit .env with your actual values
vi .env
```

**Environment variable format**: `${VAR_NAME:default_value}`

Examples:
- `${ROOT_PASSWORD:admin}` — Uses `admin` if `ROOT_PASSWORD` not set
- `${REDIS_ADDR:localhost:6379}` — Uses `localhost:6379` if `REDIS_ADDR` not set
- `${DB_DSN:data/tokenlive-admin.db}` — Uses SQLite if `DB_DSN` not set

The `.env` file is automatically loaded at startup (not committed to git). Environment variables override file values when both exist.

## API Structure

All APIs are prefixed with `/api/v1/`. Standard CRUD pattern:

| Method   | Path               | Description  |
|----------|--------------------|--------------|
| `GET`    | `/<resource>`      | Query list   |
| `GET`    | `/<resource>/:id`  | Get by ID    |
| `POST`   | `/<resource>`      | Create       |
| `PUT`    | `/<resource>/:id`  | Update       |
| `DELETE` | `/<resource>/:id`  | Delete       |

Swagger docs are auto-generated from annotations — run `make swagger` to regenerate.

## FAQ

### Q: How do I change the frontend API endpoint?

Modify `VITE_API_HTTP` in `frontend/.env.prod`, then rebuild.

### Q: How do I persist data?

When using Docker Compose, data is automatically mounted to the `./data` directory.

### Q: How do I view logs?

```bash
docker-compose logs -f tokenlive-admin
```

## License

This project is licensed under the Apache License. See the [LICENSE](LICENSE) file for details.
