# Production-Ready Go URL Shortener API

A high-performance, production-ready URL Shortener REST API built in Go, backed by PostgreSQL, containerized with Docker, and automated with GitHub Actions CI/CD.

---

## Table of Contents

- [Features](#features)
- [Architecture & Design](#architecture--design)
- [Prerequisites](#prerequisites)
- [Local Development Setup](#local-development-setup)
- [Running with Docker Compose](#running-with-docker-compose)
- [API Reference & Examples](#api-reference--examples)
- [Testing](#testing)
- [Security & Reliability](#security--reliability)
- [CI/CD Workflow](#cicd-workflow)
- [Cloud Deployment Guide](#cloud-deployment-guide)

---

## Features

- **Fast & Scalable Shortening:** Generates collision-resistant Base62 short codes (e.g. `aB82xQ`) with bounded retry and custom alias support.
- **Temporary Redirects (HTTP 307):** Uses HTTP 307 Temporary Redirect with `Cache-Control: no-cache, no-store, must-revalidate` to prevent CDN/browser cache pollution while ensuring real-time click tracking and preserving the request method.
- **Full CRUD Management:** Create, list with pagination, retrieve details, update target URLs, and soft-delete codes.
- **Click Analytics:** Atomic, race-condition-free click counter increments (`UPDATE ... SET clicks = clicks + 1`) and last-accessed timestamps.
- **Resilient Database Layer:** Connection pooling via `pgx/v5/pgxpool` with automated embedded migrations.
- **Production Observability:** Structured JSON logging (`log/slog`), correlation IDs (`X-Request-ID`), panic recovery, and `/healthz` & `/readyz` probes.
- **In-Flight Abuse Prevention:** Token-bucket rate limiter per client IP with automatic idle-client eviction.
- **Multi-Stage Container:** Ultra-lightweight non-root Alpine container image (< 20MB).

---

## Architecture & Design

The project strictly follows Clean Architecture principles, ensuring clear boundaries between transport, domain logic, and persistence:

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
[ Service Layer ] (Business logic, URL validation, Base62 code generator & collision retry)
     │
     ▼
[ Repository Layer ] (pgxpool PostgreSQL transactions, atomic click increments)
     │
     ▼
[ PostgreSQL Database ]
```

### Directory Structure

```text
├── cmd/
│   └── api/
│       └── main.go         # Application entrypoint & graceful shutdown
├── internal/
│   ├── config/             # Environment variable parsing & validation
│   ├── domain/             # Core entities, DTOs, repository interfaces, errors
│   ├── handler/            # HTTP handlers, JSON validation, router setup
│   ├── middleware/         # Logging, request ID, rate limiting, panic recovery, CORS
│   ├── repository/         # PostgreSQL persistence (pgxpool) & in-memory test store
│   └── service/            # Base62 generator, URL validator, business logic
├── migrations/             # SQL migrations (up/down) & Go embed wrapper
├── docs/                   # OpenAPI 3.1 specification & API contract documentation
├── scripts/                # Developer helper scripts (dev.ps1)
├── .github/
│   └── workflows/          # GitHub Actions CI pipeline (lint, vet, test, build)
├── Dockerfile              # Multi-stage production container build
├── compose.yaml            # Docker Compose orchestration (API + PostgreSQL)
├── .env.example            # Sample configuration values
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

---

## Prerequisites

- **Go:** 1.22+ (tested with Go 1.27)
- **Git**
- **Docker & Docker Compose** (optional for local container execution)
- **PostgreSQL 15+** (or use local in-memory fallback mode)

---

## Local Development Setup

### 1. Copy Environment Configuration
```bash
cp .env.example .env
```

### 2. Run the Application
```bash
go run ./cmd/api
```
*Note:* If `DATABASE_URL` is omitted, the service automatically runs in an in-memory mode for rapid local prototyping and zero-dependency testing.

---

## Running with Docker Compose

Start the application and PostgreSQL together with persistent volume storage and automated healthchecks:

```bash
docker compose up --build
```

To stop:
```bash
docker compose down
```

---

## API Reference & Examples

### 1. Health and Readiness
```bash
# Liveness probe
curl -i http://localhost:8080/healthz

# Readiness probe (verifies database connectivity)
curl -i http://localhost:8080/readyz
```

### 2. Create Short URL
```bash
curl -i -X POST http://localhost:8080/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://go.dev/doc/tutorial",
    "custom_code": "my-tutorial"
  }'
```
**Response (201 Created):**
```json
{
  "id": "e10a498d-1d09-4a9a-a439-0271f2e4e91e",
  "code": "my-tutorial",
  "original_url": "https://go.dev/doc/tutorial",
  "short_url": "http://localhost:8080/my-tutorial",
  "clicks": 0,
  "created_at": "2026-10-21T07:31:08Z",
  "updated_at": "2026-10-21T07:31:08Z"
}
```

### 3. Redirect to Original URL
```bash
curl -i http://localhost:8080/my-tutorial
```
**Response (307 Temporary Redirect):**
```http
HTTP/1.1 307 Temporary Redirect
Location: https://go.dev/doc/tutorial
Cache-Control: no-cache, no-store, must-revalidate
```

### 4. Get Click Analytics
```bash
curl -i http://localhost:8080/api/v1/urls/my-tutorial/stats
```
**Response (200 OK):**
```json
{
  "code": "my-tutorial",
  "original_url": "https://go.dev/doc/tutorial",
  "clicks": 1,
  "last_accessed_at": "2026-10-21T07:31:08Z",
  "created_at": "2026-10-21T07:31:08Z"
}
```

### 5. List URLs (Paginated)
```bash
curl -i "http://localhost:8080/api/v1/urls?limit=10&offset=0"
```

### 6. Update Target URL
```bash
curl -i -X PATCH http://localhost:8080/api/v1/urls/my-tutorial \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/learn"}'
```

### 7. Delete Short URL
```bash
curl -i -X DELETE http://localhost:8080/api/v1/urls/my-tutorial
```
**Response (204 No Content)**

---

## Challenge 5: In-Memory URL Shortener (Duplicate Code Handling)

In addition to the database-backed persistent engine, the service provides high-performance, strictly in-memory endpoints implementing Go standard library URL shortening with atomic duplicate-code rejection.

### Architecture & Concurrency Safety
- **Storage:** Safe in-memory hash map (`map[string]string`) storing `short_code -> destination_url`.
- **Atomic Locking:** Guarded with `sync.RWMutex`. The existence check and insertion occur atomically inside a single critical section under `Lock()`. Separate unlocked reads followed by writes are strictly avoided to eliminate race conditions.
- **Duplicate Prevention:** If a code already exists, creating it again immediately returns `409 Conflict` and **never overwrites** the original destination.
- **In-Memory Lifetime Limitation:** Mappings exist strictly within process memory and are lost when the service restarts. In multi-instance / distributed deployments, mappings are isolated to individual instances.

### Endpoints & Status Codes

| Endpoint | Method | Success Code | Conflict Code | Not Found Code | Description |
|---|---|---|---|---|---|
| `/shorten` | `POST` | `201 Created` | `409 Conflict` | — | Store code and destination URL atomically |
| `/resolve/{code}` | `GET` | `200 OK` | — | `404 Not Found` | Retrieve destination URL |
| Any other method | Various | — | — | `405 Method Not Allowed` | Unsupported HTTP methods |

### Usage Examples

#### 1. Create a New Short Code (201 Created)
```bash
curl -i -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"code":"abc123","url":"https://example.com"}'
```
**Response (201 Created):**
```json
{
  "code": "abc123",
  "url": "https://example.com"
}
```

#### 2. Attempt Duplicate Short Code (409 Conflict - Never Overwrites)
```bash
curl -i -X POST http://localhost:8080/shorten \
  -H "Content-Type: application/json" \
  -d '{"code":"abc123","url":"https://another-example.com"}'
```
**Response (409 Conflict):**
```json
{
  "error": "short code already exists"
}
```

#### 3. Resolve an Existing Short Code (200 OK)
```bash
curl -i http://localhost:8080/resolve/abc123
```
**Response (200 OK):**
```json
{
  "code": "abc123",
  "url": "https://example.com"
}
```

#### 4. Resolve a Missing Short Code (404 Not Found)
```bash
curl -i http://localhost:8080/resolve/missing123
```
**Response (404 Not Found):**
```json
{
  "error": "short code not found"
}
```

#### 5. Method Not Allowed (405 Method Not Allowed)
```bash
curl -i http://localhost:8080/shorten
```
**Response (405 Method Not Allowed):**
```json
{
  "error": "method not allowed"
}
```

---

## Testing

Run the full automated test suite:

```bash
# Run all tests
go test -v ./...

# Run static analysis
go vet ./...

# Verify code formatting
gofmt -l .
```

---

## Security & Reliability

| Security Safeguard | Implementation |
|---|---|
| **SQL Injection Prevention** | 100% parameterized queries via `pgx/v5` binary protocol |
| **Open Redirect / XSS Prevention** | Strict scheme whitelisting (`http://` & `https://` only; rejects `javascript:`, `file:`, `data:`) |
| **Abuse & DoS Prevention** | Per-IP token bucket rate limiter with automatic stale client memory eviction |
| **Payload Bombs** | Request body hard limit capped at 1 MB via `http.MaxBytesReader` |
| **Slowloris Prevention** | Strict server timeouts (`ReadTimeout: 5s`, `WriteTimeout: 10s`, `IdleTimeout: 120s`) |
| **Concurrency Safety** | Atomic SQL click increments (`UPDATE urls SET clicks = clicks + 1`) |
| **Secret Masking** | Database connection string sanitization in log outputs |
| **Container Security** | Multi-stage Dockerfile running as non-root user (UID 10001) on Alpine |

---

## Cloud Deployment Guide

The service is cloud-agnostic and runs on any container or PaaS provider (Render, Railway, Fly.io, AWS ECS, Google Cloud Run).

### Recommended Free/Low-Cost Option: Render / Railway + Neon PostgreSQL
1. **Database:** Provision a serverless PostgreSQL database on [Neon.tech](https://neon.tech) or [Supabase](https://supabase.com).
2. **Compute:** Deploy the container on [Render](https://render.com) or [Railway](https://railway.app).
3. **Environment Variables:** Set `DATABASE_URL`, `ENVIRONMENT=production`, and `BASE_URL`.
4. The service will automatically run embedded database migrations on startup and begin serving traffic.
