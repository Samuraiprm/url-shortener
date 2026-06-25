# URL Shortener

High-performance URL shortening service with analytics and caching, built with Go.

## Features

- **URL Shortening** — create short codes for long URLs
- **302 Redirect** — fast redirect to original URL
- **Analytics** — track clicks with IP, User-Agent, Referrer, timestamp
- **Redis Caching** — hot URLs served from cache (TTL 10min)
- **Rate Limiting** — token bucket per IP (10 req/s, burst 20)
- **Security Headers** — HSTS, CSP, X-Frame-Options, X-XSS-Protection
- **Request Validation** — blocks javascript/data/vbscript URIs, max 2KB
- **Prometheus Metrics** — `/metrics` endpoint for monitoring
- **Swagger/OpenAPI** — interactive API docs at `/swagger/`

## Tech Stack

| Component | Technology |
|---|---|
| Language | Go 1.23+ |
| Router | chi |
| Database | PostgreSQL 16 |
| Cache | Redis 7 |
| DB Driver | pgx/v5 |
| Cache Client | go-redis/v9 |
| API Docs | swaggo |
| Tests | stdlib + testcontainers |

## Project Structure

```
url-shortener/
├── cmd/server/
│   └── main.go                    # Entry point, router setup, graceful shutdown
├── internal/
│   ├── cache/
│   │   ├── redis.go               # Redis cache: Get/Set/Invalidate
│   │   └── redis_integration_test.go
│   ├── config/
│   │   └── config.go              # Env-based configuration
│   ├── handler/
│   │   ├── handler.go             # HTTP handlers with Swagger annotations
│   │   ├── handler_test.go        # Unit tests
│   │   └── e2e_test.go            # End-to-end integration tests
│   ├── middleware/
│   │   ├── metrics.go             # Prometheus-style metrics
│   │   ├── ratelimit.go           # Token bucket rate limiter
│   │   ├── security.go            # Security headers, body limit, request ID
│   │   ├── security_test.go       # Security middleware tests
│   │   └── errors.go              # Middleware error types
│   ├── model/
│   │   └── model.go               # Domain models and DTOs
│   ├── repository/
│   │   ├── url_repo.go            # PostgreSQL data access
│   │   └── url_repo_integration_test.go
│   └── service/
│       ├── url_service.go         # Business logic + URL validation
│       └── url_service_test.go    # Validation unit tests
├── migrations/
│   └── 001_init.sql               # Database schema
├── docs/
│   ├── docs.go                    # Generated Swagger Go code
│   ├── swagger.json
│   └── swagger.yaml
├── docker-compose.yml             # PostgreSQL + Redis
├── Dockerfile                     # Multi-stage build
├── Makefile                       # Build/run/test commands
└── go.mod
```

## Quick Start

### Prerequisites

- Go 1.23+
- Docker & Docker Compose

### Run locally

```bash
# Start PostgreSQL and Redis
docker compose up -d

# Run the server
go run ./cmd/server
```

Server starts on `http://localhost:8080`.

### Run with Docker

```bash
docker compose up -d --build
```

## API

### Create short URL

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

Response:

```json
{
  "code": "a1b2c3d4",
  "short_url": "http://localhost:8080/a1b2c3d4",
  "original_url": "https://go.dev/doc/"
}
```

### Redirect

```bash
curl -v http://localhost:8080/a1b2c3d4
# 302 Found → Location: https://go.dev/doc/
```

### Get analytics

```bash
curl http://localhost:8080/api/stats/a1b2c3d4
```

Response:

```json
{
  "total_clicks": 42,
  "url": {
    "id": 1,
    "code": "a1b2c3d4",
    "original_url": "https://go.dev/doc/",
    "created_at": "2026-06-25T19:07:44Z"
  },
  "recent_clicks": [
    {
      "id": 42,
      "url_id": 1,
      "ip": "192.168.1.1",
      "user_agent": "Mozilla/5.0...",
      "referrer": "https://google.com",
      "created_at": "2026-06-25T20:15:00Z"
    }
  ]
}
```

### Health check

```bash
curl http://localhost:8080/health
# {"status": "ok"}
```

### Metrics

```bash
curl http://localhost:8080/metrics
# Prometheus-format metrics
```

### Swagger UI

Open `http://localhost:8080/swagger/` in browser.

## Testing

```bash
# Unit tests
go test ./... -v

# Integration tests (requires Docker)
go test -tags=integration ./... -v -timeout 120s
```

### Test coverage

| Layer | Type | Tests | Framework |
|---|---|---|---|
| Handler | Unit | 5 | net/http/httptest |
| Middleware | Unit | 2 | net/http/httptest |
| Service | Unit | 1 | stdlib testing |
| Repository | Integration | 5 | testcontainers |
| Cache | Integration | 3 | testcontainers |
| E2E | Integration | 1 | testcontainers |
| **Total** | | **17** | |

## Security

Implemented following OWASP Top 10 guidelines:

- **XSS Prevention** — CSP header, input sanitization, blocks `javascript:` / `data:` / `vbscript:` URIs
- **Clickjacking** — `X-Frame-Options: DENY`
- **MIME Sniffing** — `X-Content-Type-Options: nosniff`
- **HSTS** — `Strict-Transport-Security` with 1-year max-age
- **DoS Protection** — rate limiting (token bucket), request body size limit (1MB), server timeouts
- **SSRF Prevention** — only `http` and `https` schemes allowed
- **Audit Trail** — request ID header, click tracking with IP/User-Agent/Referrer

## Architecture Decisions

- **302 Redirect** over 301 — allows analytics tracking on every click
- **Token bucket** rate limiter — smooths bursts, auto-refills, per-IP isolation
- **Async click recording** — `go RecordClick()` doesn't block redirect response
- **Redis cache with 10min TTL** — balances freshness vs performance
- **pgx pool** — connection pooling built-in, no ORM overhead
- **chi router** — lightweight, stdlib-compatible, middleware-friendly

## Configuration

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | `postgres://shortener:shortener@localhost:5433/shortener?sslmode=disable` | PostgreSQL connection |
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `BASE_URL` | `http://localhost:8080` | Base URL for short links |

## License

MIT
