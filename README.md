# Go URL Shortener API

A production-ready, high-performance URL shortener REST API built in Go, backed by PostgreSQL, containerized with Docker, and automated with GitHub Actions CI/CD.

## Features

- **Fast & Scalable Shortening:** Generate collision-resistant Base62 short codes with bounded retry.
- **Temporary Redirects:** HTTP 307 temporary redirects preserving request semantics while avoiding browser permanent cache pitfalls.
- **Full CRUD Management:** Create, read, update target URL, and soft/hard delete short URLs.
- **Analytics:** Real-time click counting with concurrent-safe updates (`UPDATE ... SET clicks = clicks + 1`) and last-accessed timestamps.
- **Paginated Listings:** Query URLs with deterministic keyset or limit/offset pagination.
- **Resilient Database Layer:** Connection pooling via `pgx/v5` with timeout management and automated SQL migrations.
- **Production Observability:** Structured JSON logging, per-request correlation IDs (`X-Request-ID`), recovery middleware, in-flight rate limiting, and `/healthz` & `/readyz` probes.
- **Containerized:** Multi-stage scratch/distroless Docker builds and Docker Compose orchestration.

## Architecture

```text
HTTP Request
     │
     ▼
[ Middleware Pipeline ] (RequestID -> Structured Logger -> Recovery -> RateLimiter -> CORS)
     │
     ▼
[ HTTP Handler ] (JSON validation, status code mapping, HTTP 307 redirects)
     │
     ▼
[ Service Layer ] (Business logic, URL validation, Base62 code generation & collision retry)
     │
     ▼
[ Repository Layer ] (pgxpool PostgreSQL transactions, atomic click increments)
     │
     ▼
[ PostgreSQL Database ]
```

## Prerequisites

- **Go:** 1.22+ (tested on Go 1.27)
- **PostgreSQL:** 15+ (or Docker Compose)
- **Git**

## Quick Start (Local Development)

### 1. Clone & Configure
```bash
cp .env.example .env
```

### 2. Run the Application
```bash
go run ./cmd/api
```

### 3. Verify Health Probe
```bash
curl http://localhost:8080/healthz
```

## Directory Structure

```text
├── cmd/
│   └── api/
│       └── main.go         # Application entrypoint & graceful shutdown
├── internal/
│   ├── config/             # Environment variable parsing & validation
│   ├── domain/             # Core entities, DTOs, and error types
│   ├── handler/            # HTTP handlers and router registration
│   ├── middleware/         # Logging, request ID, rate limiting, recovery
│   ├── repository/         # PostgreSQL persistence and SQL queries
│   └── service/            # Business logic and code generation
├── migrations/             # Versioned SQL migrations (up/down)
├── tests/
│   └── integration/        # Database-backed integration tests
├── docs/                   # OpenAPI specification & architecture notes
├── scripts/                # Database migration and utility scripts
├── .github/
│   └── workflows/          # CI/CD pipeline (lint, test, build)
├── Dockerfile              # Multi-stage production container build
├── compose.yaml            # Docker Compose setup for API and PostgreSQL
├── .env.example            # Sample configuration values
├── .gitignore
├── go.mod
└── README.md
```

## License

MIT
