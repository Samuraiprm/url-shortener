# URL Shortener

[Русский](README.md) | [English](README.en.md) | [Español](README.es.md) | **Deutsch**

URL-Verkürzungsservice mit Analytik und Caching, gebaut in Go.

## Funktionen

- **URL-Verkürzung** — Generierung kurzer Codes für lange Links
- **302-Weiterleitung** — schnelle Weiterleitung zur Original-URL
- **Analytik** — Klick-Tracking mit IP, User-Agent, Referrer, Zeitstempel
- **Redis-Caching** — heiße URLs werden aus dem Cache ausgeliefert (TTL 10 Min.)
- **Rate Limiting** — Token Bucket pro IP (10 Req/s, Burst 20)
- **Sicherheits-Header** — HSTS, CSP, X-Frame-Options, X-XSS-Protection
- **Request-Validierung** — blockiert javascript/data/vbscript URIs, max. 2KB
- **Prometheus-Metriken** — `/metrics`-Endpoint für Monitoring
- **Swagger/OpenAPI** — interaktive API-Dokumentation unter `/swagger/`

## Tech-Stack

| Komponente | Technologie |
|---|---|
| Sprache | Go 1.23+ |
| Router | chi |
| Datenbank | PostgreSQL 16 |
| Cache | Redis 7 |
| DB Driver | pgx/v5 |
| Cache Client | go-redis/v9 |
| API-Doku | swaggo |
| Tests | stdlib + testcontainers |

## Projektstruktur

```
url-shortener/
├── cmd/server/
│   └── main.go                    # Einstiegspunkt, Router-Setup, Graceful Shutdown
├── internal/
│   ├── cache/
│   │   ├── redis.go               # Redis-Cache: Get/Set/Invalidate
│   │   └── redis_integration_test.go
│   ├── config/
│   │   └── config.go              # Umgebungsvariablen-basierte Konfiguration
│   ├── handler/
│   │   ├── handler.go             # HTTP-Handler mit Swagger-Annotationen
│   │   ├── handler_test.go        # Unit-Tests
│   │   └── e2e_test.go            # End-to-End-Integrationstests
│   ├── middleware/
│   │   ├── metrics.go             # Prometheus-Metriken
│   │   ├── ratelimit.go           # Token-Bucket-Rate-Limiter
│   │   ├── security.go            # Sicherheits-Header, Body-Limit, Request-ID
│   │   ├── security_test.go       # Sicherheits-Middleware-Tests
│   │   └── errors.go              # Middleware-Fehlertypen
│   ├── model/
│   │   └── model.go               # Domänenmodelle und DTOs
│   ├── repository/
│   │   ├── url_repo.go            # PostgreSQL-Datenzugriff
│   │   └── url_repo_integration_test.go
│   └── service/
│       ├── url_service.go         // Geschäftslogik + URL-Validierung
│       └── url_service_test.go    // Unit-Tests der Validierung
├── migrations/
│   └── 001_init.sql               # Datenbankschema
├── docs/
│   ├── docs.go                    # Generierter Swagger-Go-Code
│   ├── swagger.json
│   └── swagger.yaml
├── docker-compose.yml             # PostgreSQL + Redis
├── Dockerfile                     # Multi-Stage-Build
├── Makefile                       # Build/Run/Test-Befehle
└── go.mod
```

## Schnellstart

### Voraussetzungen

- Go 1.23+
- Docker und Docker Compose

### Lokal ausführen

```bash
# PostgreSQL und Redis starten
docker compose up -d

# Server starten
go run ./cmd/server
```

Der Server startet auf `http://localhost:8080`.

### Mit Docker ausführen

```bash
docker compose up -d --build
```

## API

### Kurze URL erstellen

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

Antwort:

```json
{
  "code": "a1b2c3d4",
  "short_url": "http://localhost:8080/a1b2c3d4",
  "original_url": "https://go.dev/doc/"
}
```

### Weiterleitung

```bash
curl -v http://localhost:8080/a1b2c3d4
# 302 Found → Location: https://go.dev/doc/
```

### Analytik abrufen

```bash
curl http://localhost:8080/api/stats/a1b2c3d4
```

Antwort:

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

### Health Check

```bash
curl http://localhost:8080/health
# {"status": "ok"}
```

### Metriken

```bash
curl http://localhost:8080/metrics
# Metriken im Prometheus-Format
```

### Swagger UI

Öffnen Sie `http://localhost:8080/swagger/` im Browser.

## Testing

```bash
# Unit-Tests
go test ./... -v

# Integrationstests (erfordert Docker)
go test -tags=integration ./... -v -timeout 120s
```

### Testabdeckung

| Schicht | Typ | Tests | Framework |
|---|---|---|---|
| Handler | Unit | 5 | net/http/httptest |
| Middleware | Unit | 2 | net/http/httptest |
| Service | Unit | 1 | stdlib testing |
| Repository | Integration | 5 | testcontainers |
| Cache | Integration | 3 | testcontainers |
| E2E | Integration | 1 | testcontainers |
| **Gesamt** | | **17** | |

## Sicherheit

Implementiert nach OWASP Top 10-Richtlinien:

- **XSS-Schutz** — CSP-Header, Input-Sanitisierung, Blockierung von `javascript:` / `data:` / `vbscript:` URIs
- **Clickjacking** — `X-Frame-Options: DENY`
- **MIME Sniffing** — `X-Content-Type-Options: nosniff`
- **HSTS** — `Strict-Transport-Security` mit 1 Jahr max-age
- **DoS-Schutz** — Rate Limiting (Token Bucket), Request-Body-Größenlimit (1MB), Server-Timeouts
- **SSRF-Schutz** — nur `http`- und `https`-Schemata erlaubt
- **Audit Trail** — Request-ID-Header, Klick-Tracking mit IP/User-Agent/Referrer

## Architekturentscheidungen

- **302-Weiterleitung** statt 301 — ermöglicht Analytik-Tracking bei jedem Klick
- **Token Bucket** Rate Limiter — glättet Burst, automatische Auffüllung, IP-Isolierung
- **Async Klick-Aufzeichnung** — `go RecordClick()` blockiert nicht die Weiterleitungsantwort
- **Redis-Cache mit 10 Min. TTL** — Gleichgewicht zwischen Aktualität und Leistung
- **pgx Pool** — integrierte Verbindungspoolung, kein ORM-Overhead
- **chi Router** — leichtgewichtig, stdlib-kompatibel, middleware-freundlich

## Konfiguration

Umgebungsvariablen:

| Variable | Standard | Beschreibung |
|---|---|---|
| `PORT` | `8080` | Server-Port |
| `DATABASE_URL` | `postgres://shortener:shortener@localhost:5433/shortener?sslmode=disable` | PostgreSQL-Verbindung |
| `REDIS_ADDR` | `localhost:6379` | Redis-Adresse |
| `BASE_URL` | `http://localhost:8080` | Basis-URL für Kurzlinks |

## Lizenz

MIT
