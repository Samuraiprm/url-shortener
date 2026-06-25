# URL Shortener

[Русский](README.md) | [English](README.en.md) | **Español** | [Deutsch](README.de.md)

Servicio de acortamiento de URL con analítica y caché, construido en Go.

## Características

- **Acortamiento de URL** — generación de códigos cortos para enlaces largos
- **Redirección 302** — redirección rápida a la URL original
- **Analítica** — seguimiento de clics con IP, User-Agent, Referrer, marca de tiempo
- **Caché Redis** — URLs calientes servidas desde caché (TTL 10 min)
- **Rate Limiting** — cubo de tokens por IP (10 req/s, ráfaga 20)
- **Headers de seguridad** — HSTS, CSP, X-Frame-Options, X-XSS-Protection
- **Validación de solicitudes** — bloquea URIs javascript/data/vbscript, máx. 2KB
- **Métricas Prometheus** — endpoint `/metrics` para monitoreo
- **Swagger/OpenAPI** — documentación interactiva de API en `/swagger/`

## Stack tecnológico

| Componente | Tecnología |
|---|---|
| Lenguaje | Go 1.23+ |
| Router | chi |
| Base de datos | PostgreSQL 16 |
| Caché | Redis 7 |
| DB Driver | pgx/v5 |
| Cache Client | go-redis/v9 |
| Documentación API | swaggo |
| Tests | stdlib + testcontainers |

## Estructura del proyecto

```
url-shortener/
├── cmd/server/
│   └── main.go                    # Punto de entrada, configuración del router, apagado graceful
├── internal/
│   ├── cache/
│   │   ├── redis.go               # Caché Redis: Get/Set/Invalidate
│   │   └── redis_integration_test.go
│   ├── config/
│   │   └── config.go              # Configuración basada en variables de entorno
│   ├── handler/
│   │   ├── handler.go             # Handlers HTTP con anotaciones Swagger
│   │   ├── handler_test.go        # Tests unitarios
│   │   └── e2e_test.go            # Tests de integración end-to-end
│   ├── middleware/
│   │   ├── metrics.go             # Métricas en formato Prometheus
│   │   ├── ratelimit.go           # Rate limiter con cubo de tokens
│   │   ├── security.go            # Headers de seguridad, límite de cuerpo, request ID
│   │   ├── security_test.go       # Tests del middleware de seguridad
│   │   └── errors.go              # Tipos de error del middleware
│   ├── model/
│   │   └── model.go               # Modelos de dominio y DTOs
│   ├── repository/
│   │   ├── url_repo.go            # Acceso a datos PostgreSQL
│   │   └── url_repo_integration_test.go
│   └── service/
│       ├── url_service.go         # Lógica de negocio + validación de URL
│       └── url_service_test.go    # Tests unitarios de validación
├── migrations/
│   └── 001_init.sql               # Esquema de base de datos
├── docs/
│   ├── docs.go                    # Código Go Swagger generado
│   ├── swagger.json
│   └── swagger.yaml
├── docker-compose.yml             # PostgreSQL + Redis
├── Dockerfile                     # Construcción multi-etapa
├── Makefile                       # Comandos de build/ejecución/tests
└── go.mod
```

## Inicio rápido

### Requisitos

- Go 1.23+
- Docker y Docker Compose

### Ejecutar localmente

```bash
# Iniciar PostgreSQL y Redis
docker compose up -d

# Ejecutar el servidor
go run ./cmd/server
```

El servidor inicia en `http://localhost:8080`.

### Ejecutar con Docker

```bash
docker compose up -d --build
```

## API

### Crear URL corta

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

Respuesta:

```json
{
  "code": "a1b2c3d4",
  "short_url": "http://localhost:8080/a1b2c3d4",
  "original_url": "https://go.dev/doc/"
}
```

### Redirección

```bash
curl -v http://localhost:8080/a1b2c3d4
# 302 Found → Location: https://go.dev/doc/
```

### Obtener analítica

```bash
curl http://localhost:8080/api/stats/a1b2c3d4
```

Respuesta:

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

### Métricas

```bash
curl http://localhost:8080/metrics
# Métricas en formato Prometheus
```

### Swagger UI

Abra `http://localhost:8080/swagger/` en el navegador.

## Testing

```bash
# Tests unitarios
go test ./... -v

# Tests de integración (requiere Docker)
go test -tags=integration ./... -v -timeout 120s
```

### Cobertura de tests

| Capa | Tipo | Tests | Framework |
|---|---|---|---|
| Handler | Unitario | 5 | net/http/httptest |
| Middleware | Unitario | 2 | net/http/httptest |
| Service | Unitario | 1 | stdlib testing |
| Repository | Integración | 5 | testcontainers |
| Cache | Integración | 3 | testcontainers |
| E2E | Integración | 1 | testcontainers |
| **Total** | | **17** | |

## Seguridad

Implementado siguiendo las directrices OWASP Top 10:

- **Prevención de XSS** — header CSP, sanitización de entrada, bloqueo de URIs `javascript:` / `data:` / `vbscript:`
- **Clickjacking** — `X-Frame-Options: DENY`
- **MIME Sniffing** — `X-Content-Type-Options: nosniff`
- **HSTS** — `Strict-Transport-Security` con max-age de 1 año
- **Protección contra DoS** — rate limiting (cubo de tokens), límite de tamaño del cuerpo (1MB), timeouts del servidor
- **Prevención de SSRF** — solo se permiten esquemas `http` y `https`
- **Auditoría** — header request ID, seguimiento de clics con IP/User-Agent/Referrer

## Decisiones arquitectónicas

- **Redirección 302** en lugar de 301 — permite rastrear analítica en cada clic
- **Cubo de tokens** rate limiter — suaviza ráfagas, recarga automática, aislamiento por IP
- **Registro asíncrono de clics** — `go RecordClick()` no bloquea la respuesta de redirección
- **Caché Redis con TTL 10 min** — equilibrio entre frescura y rendimiento
- **pgx pool** — pool de conexiones integrado, sin sobrecoste de ORM
- **Router chi** — ligero, compatible con stdlib, amigable con middleware

## Configuración

Variables de entorno:

| Variable | Por defecto | Descripción |
|---|---|---|
| `PORT` | `8080` | Puerto del servidor |
| `DATABASE_URL` | `postgres://shortener:shortener@localhost:5433/shortener?sslmode=disable` | Conexión a PostgreSQL |
| `REDIS_ADDR` | `localhost:6379` | Dirección de Redis |
| `BASE_URL` | `http://localhost:8080` | URL base para enlaces cortos |

## Licencia

MIT
