# URL Shortener

[English](README.en.md) | [Español](README.es.md) | [Deutsch](README.de.md)

Сервис сокращения URL с аналитикой и кэшированием, написанный на Go.

## Возможности

- **Сокращение URL** — генерация коротких кодов для длинных ссылок
- **302 редирект** — быстрый редирект на оригинальный URL
- **Аналитика** — учёт кликов с IP, User-Agent, Referrer, временной меткой
- **Redis кэш** — горячие URL раздаются из кэша (TTL 10 мин)
- **Rate Limiting** — токен-бакет по IP (10 req/s, burst 20)
- **Security-заголовки** — HSTS, CSP, X-Frame-Options, X-XSS-Protection
- **Валидация запросов** — блокировка javascript/data/vbscript URI, макс. 2KB
- **Prometheus метрики** — эндпоинт `/metrics` для мониторинга
- **Swagger/OpenAPI** — интерактивная документация API на `/swagger/`

## Стек технологий

| Компонент | Технология |
|---|---|
| Язык | Go 1.23+ |
| Роутер | chi |
| База данных | PostgreSQL 16 |
| Кэш | Redis 7 |
| DB Driver | pgx/v5 |
| Cache Client | go-redis/v9 |
| Документация API | swaggo |
| Тесты | stdlib + testcontainers |

## Структура проекта

```
url-shortener/
├── cmd/server/
│   └── main.go                    # Точка входа, настройка роутера, graceful shutdown
├── internal/
│   ├── cache/
│   │   ├── redis.go               # Redis кэш: Get/Set/Invalidate
│   │   └── redis_integration_test.go
│   ├── config/
│   │   └── config.go              # Конфигурация из переменных окружения
│   ├── handler/
│   │   ├── handler.go             # HTTP-хендлеры с Swagger-аннотациями
│   │   ├── handler_test.go        # Unit-тесты
│   │   └── e2e_test.go            # End-to-end интеграционные тесты
│   ├── middleware/
│   │   ├── metrics.go             # Метрики в формате Prometheus
│   │   ├── ratelimit.go           # Rate limiter на токен-бакете
│   │   ├── security.go            # Security-заголовки, лимит тела, request ID
│   │   ├── security_test.go       # Тесты security-мидлвари
│   │   └── errors.go              # Типы ошибок мидлвари
│   ├── model/
│   │   └── model.go               # Доменные модели и DTO
│   ├── repository/
│   │   ├── url_repo.go            # Работа с PostgreSQL
│   │   └── url_repo_integration_test.go
│   └── service/
│       ├── url_service.go         # Бизнес-логика + валидация URL
│       └── url_service_test.go    # Unit-тесты валидации
├── migrations/
│   └── 001_init.sql               # Схема базы данных
├── docs/
│   ├── docs.go                    # Сгенерированный Swagger Go-код
│   ├── swagger.json
│   └── swagger.yaml
├── docker-compose.yml             # PostgreSQL + Redis
├── Dockerfile                     # Multi-stage сборка
├── Makefile                       # Команды сборки/запуска/тестов
└── go.mod
```

## Быстрый старт

### Требования

- Go 1.23+
- Docker и Docker Compose

### Запуск локально

```bash
# Запуск PostgreSQL и Redis
docker compose up -d

# Запуск сервера
go run ./cmd/server
```

Сервер стартует на `http://localhost:8080`.

### Запуск через Docker

```bash
docker compose up -d --build
```

## API

### Создать короткую ссылку

```bash
curl -X POST http://localhost:8080/api/shorten \
  -H "Content-Type: application/json" \
  -d '{"url": "https://go.dev/doc/"}'
```

Ответ:

```json
{
  "code": "a1b2c3d4",
  "short_url": "http://localhost:8080/a1b2c3d4",
  "original_url": "https://go.dev/doc/"
}
```

### Редирект

```bash
curl -v http://localhost:8080/a1b2c3d4
# 302 Found → Location: https://go.dev/doc/
```

### Получить аналитику

```bash
curl http://localhost:8080/api/stats/a1b2c3d4
```

Ответ:

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

### Метрики

```bash
curl http://localhost:8080/metrics
# Метрики в формате Prometheus
```

### Swagger UI

Откройте `http://localhost:8080/swagger/` в браузере.

## Тестирование

```bash
# Unit-тесты
go test ./... -v

# Интеграционные тесты (требуется Docker)
go test -tags=integration ./... -v -timeout 120s
```

### Покрытие тестами

| Слой | Тип | Тестов | Фреймворк |
|---|---|---|---|
| Handler | Unit | 5 | net/http/httptest |
| Middleware | Unit | 2 | net/http/httptest |
| Service | Unit | 1 | stdlib testing |
| Repository | Интеграционный | 5 | testcontainers |
| Cache | Интеграционный | 3 | testcontainers |
| E2E | Интеграционный | 1 | testcontainers |
| **Итого** | | **17** | |

## Безопасность

Реализовано по руководствам OWASP Top 10:

- **Защита от XSS** — CSP-заголовок, санитайзинг ввода, блокировка `javascript:` / `data:` / `vbscript:` URI
- **Защита от Clickjacking** — `X-Frame-Options: DENY`
- **Защита от MIME Sniffing** — `X-Content-Type-Options: nosniff`
- **HSTS** — `Strict-Transport-Security` с max-age 1 год
- **Защита от DoS** — rate limiting (токен-бакет), лимит размера тела запроса (1MB), таймауты сервера
- **Защита от SSRF** — разрешены только схемы `http` и `https`
- **Аудит-трейл** — заголовок request ID, отслеживание кликов с IP/User-Agent/Referrer

## Архитектурные решения

- **302 редирект** вместо 301 — позволяет отслеживать аналитику каждого клика
- **Токен-бакет** rate limiter — сглаживает всплески, автоматическое пополнение, изоляция по IP
- **Асинхронная запись кликов** — `go RecordClick()` не блокирует ответ редиректа
- **Redis кэш с TTL 10 мин** — баланс между свежестью и производительностью
- **pgx pool** — встроенный пул соединений, без накладных расходов ORM
- **chi роутер** — лёгкий, совместимый с stdlib, дружелюбный к мидлвари

## Конфигурация

Переменные окружения:

| Переменная | По умолчанию | Описание |
|---|---|---|
| `PORT` | `8080` | Порт сервера |
| `DATABASE_URL` | `postgres://shortener:shortener@localhost:5433/shortener?sslmode=disable` | Подключение к PostgreSQL |
| `REDIS_ADDR` | `localhost:6379` | Адрес Redis |
| `BASE_URL` | `http://localhost:8080` | Базовый URL для коротких ссылок |

## Лицензия

MIT
