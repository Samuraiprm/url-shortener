//go:build integration

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"url-shortener/internal/cache"
	"url-shortener/internal/model"
	"url-shortener/internal/repository"
	"url-shortener/internal/service"
)

func setupTestEnv(t *testing.T) (*Handler, func()) {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			id BIGSERIAL PRIMARY KEY,
			code VARCHAR(8) NOT NULL UNIQUE,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		CREATE TABLE IF NOT EXISTS clicks (
			id BIGSERIAL PRIMARY KEY,
			url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
			ip VARCHAR(45),
			user_agent TEXT,
			referrer TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start redis: %v", err)
	}

	redisHost, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get redis host: %v", err)
	}
	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get redis port: %v", err)
	}
	redisAddr := redisHost + ":" + redisPort.Port()

	redisCache, err := cache.NewRedisCache(redisAddr)
	if err != nil {
		t.Fatalf("failed to create redis cache: %v", err)
	}

	repo := repository.NewURLRepository(pool)
	svc := service.NewURLService(repo, redisCache)
	h := NewHandler(svc, "http://localhost:8080")

	cleanup := func() {
		redisCache.Close()
		pool.Close()
		pgContainer.Terminate(ctx)
		redisContainer.Terminate(ctx)
	}

	return h, cleanup
}

func TestE2E_ShortenRedirectStats(t *testing.T) {
	h, cleanup := setupTestEnv(t)
	defer cleanup()

	// 1. Create short URL
	body, _ := json.Marshal(model.CreateURLRequest{URL: "https://go.dev/doc/"})
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.CreateURL(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createResp model.CreateURLResponse
	json.NewDecoder(w.Body).Decode(&createResp)
	if createResp.Code == "" {
		t.Fatal("expected code in response")
	}
	if createResp.ShortURL != "http://localhost:8080/"+createResp.Code {
		t.Errorf("unexpected short_url: %s", createResp.ShortURL)
	}

	// 2. Redirect
	req = httptest.NewRequest(http.MethodGet, "/"+createResp.Code, nil)
	req.SetPathValue("code", createResp.Code)
	w = httptest.NewRecorder()

	h.Redirect(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", w.Code)
	}
	if w.Header().Get("Location") != "https://go.dev/doc/" {
		t.Errorf("expected redirect to https://go.dev/doc/, got %s", w.Header().Get("Location"))
	}

	// 3. Wait for goroutine click recording and check stats
	time.Sleep(100 * time.Millisecond)

	// 4. Check stats
	req = httptest.NewRequest(http.MethodGet, "/api/stats/"+createResp.Code, nil)
	req.SetPathValue("code", createResp.Code)
	w = httptest.NewRecorder()

	h.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stats model.Stats
	json.NewDecoder(w.Body).Decode(&stats)
	if stats.TotalClicks != 1 {
		t.Errorf("expected 1 click, got %d", stats.TotalClicks)
	}
	if len(stats.RecentClicks) != 1 {
		t.Errorf("expected 1 recent click, got %d", len(stats.RecentClicks))
	}

	// 5. Duplicate code check
	body2, _ := json.Marshal(model.CreateURLRequest{URL: "https://example.com"})
	req = httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body2))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.CreateURL(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for second URL, got %d", w.Code)
	}

	// 6. Stats for nonexistent code
	req = httptest.NewRequest(http.MethodGet, "/api/stats/doesnotexist", nil)
	req.SetPathValue("code", "doesnotexist")
	w = httptest.NewRecorder()
	h.GetStats(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	_ = fmt.Sprintf("All E2E tests passed for code: %s", createResp.Code)
}
