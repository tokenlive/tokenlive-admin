# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

JoyLive-Dashboard is the admin console for the JoyLive AI Gateway ecosystem. It's a full-stack application with a Go backend (Gin + GORM) and Vue 3 frontend (Vite + Ant Design Vue), deployed as a single Docker container serving both. It manages AI model providers/models, governance policies, RBAC, and multi-space resource isolation.

## Common Commands

### Development

```bash
# Backend (requires MySQL/Redis running, config in configs/dev/server.toml)
make start                    # Run backend with hot-reload via air
make build                    # Build backend binary to bin/tokenlive-admin

# Frontend
cd frontend
npm run dev                   # Vite dev server (proxies /api to backend)
npm run build:prod            # Production build to frontend/dist

# Full build (frontend + backend)
make build-all
make serve                    # Build all then start server on :8040
```

### Code Generation

```bash
make wire                     # Regenerate Wire dependency injection (internal/wirex/wire_gen.go)
make swagger                  # Regenerate Swagger docs from annotations
```

### Testing

```bash
# Integration tests (uses in-memory DB, no external dependencies needed)
go test ./test/...

# Run a single test
go test ./test/ -run TestXxx -v
```

### Docker

```bash
make docker-build             # Build image
docker-compose up -d          # Start with compose
```

## Architecture

### Backend: Layered Module Pattern

Each domain module follows a strict layered architecture:

```
internal/mods/<module>/
├── api/       # HTTP handlers (request parsing, response formatting)
├── biz/       # Business logic (validation, orchestration)
├── dal/       # Data access layer (GORM queries)
├── schema/    # Data models and request/response DTOs
├── main.go    # Module struct, Init/RegisterRouters/Release lifecycle
└── wire.go    # Wire provider set for DI
```

Current modules:

- **`rbac`** — User, Role, Menu, MenuResource, MenuResourceGroup, UserRole, RoleMenu, UserApiKey, Login, Logger
- **`resource`** — Provider, Model, ModelAlias, Endpoint, DataPermission
- **`space`** — Space management (tenant-level resource isolation)
- **`policy`** — Auth, CircuitBreak, Fault, Invocation, Limit, Loadbalance, Permission, Route, RouteDetail

### Dependency Injection (Google Wire)

All modules are wired together in `internal/wirex/`:

- `wire.go` — Wire injector definition (build tag: `wireinject`)
- `wire_gen.go` — Auto-generated, do not edit manually
- `injector.go` — Top-level `Injector` struct holding DB, Cache, Auth, Mods

After modifying Wire providers, run `make wire`.

### Configuration

TOML-based config in `configs/`:

- `configs/dev/` — Development (MySQL + Redis by default)
- `configs/prod/` — Production

Config struct is in `internal/config/config.go`. Key sections: `[General]`, `[Storage]`, `[Storage.DB]`, `[Storage.Cache]`, `[Util]`, `[Middleware]`.

DB supports: `sqlite3`, `mysql`, `postgres`. Cache supports: `memory`, `badger`, `redis`.

### RBAC & Auth

- Casbin for role-based access control (`internal/mods/rbac/casbin.go`)
- JWT-based authentication (`pkg/jwtx/`)
- Login at `POST /api/v1/login`, token refresh at `POST /api/v1/current/refresh-token`
- Default admin: `admin`/`admin`

### Frontend

Vue 3 + Vite SPA in `frontend/`:

- `src/apis/modules/` — API service modules (mirrors backend modules)
- `src/router/routes/` — Route definitions (menu-driven, dynamic routes)
- `src/views/` — Page components by domain (policy, resource, space, system)
- `src/components/` — Reusable UI components
- `src/config/` — App configuration
- `src/store/` — Pinia state management

Lint: `npm run lint`. Formatter: `npm run prettier`.

### API Structure

All APIs are prefixed with `/api/v1/`. Routes registered in each module's `RegisterV1Routers`. Standard CRUD pattern:

- `GET /<resource>` — Query list
- `GET /<resource>/:id` — Get by ID
- `POST /<resource>` — Create
- `PUT /<resource>/:id` — Update
- `DELETE /<resource>/:id` — Delete

### Key Packages

- `pkg/cachex/` — Cache abstraction (memory/badger/redis)
- `pkg/gormx/` — GORM wrapper with multi-DB support
- `pkg/jwtx/` — JWT auth with token store
- `pkg/middleware/` — Gin middleware (auth, CORS, rate limit, etc.)
- `pkg/logging/` — Structured logging with zap
- `pkg/util/` — Shared utilities

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`chenzhiguo/tokenlive-admin`). See `docs/agents/issue-tracker.md`.

### Triage labels

Using default label vocabulary (needs-triage, needs-info, ready-for-agent, ready-for-human, wontfix). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — one `CONTEXT.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
