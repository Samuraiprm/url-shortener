//go:build integration

package cache

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"url-shortener/internal/model"
)

func setupTestRedis(t *testing.T) (*RedisCache, func()) {
	t.Helper()

	ctx := context.Background()

	redisContainer, err := redis.Run(ctx,
		"redis:7-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start redis: %v", err)
	}

	host, err := redisContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get redis host: %v", err)
	}

	port, err := redisContainer.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get redis port: %v", err)
	}

	addr := host + ":" + port.Port()

	cache, err := NewRedisCache(addr)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	cleanup := func() {
		cache.Close()
		redisContainer.Terminate(ctx)
	}

	return cache, cleanup
}

func TestCache_SetAndGet(t *testing.T) {
	cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	url := &model.URL{ID: 1, Code: "test1", Original: "https://example.com"}

	err := cache.Set(ctx, "test1", url)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, err := cache.Get(ctx, "test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected url, got nil")
	}
	if got.Code != "test1" {
		t.Errorf("expected code test1, got %s", got.Code)
	}
	if got.Original != "https://example.com" {
		t.Errorf("expected original https://example.com, got %s", got.Original)
	}
}

func TestCache_Miss(t *testing.T) {
	cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	got, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestCache_Invalidate(t *testing.T) {
	cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()

	url := &model.URL{ID: 2, Code: "inv1", Original: "https://example.com/inv"}
	_ = cache.Set(ctx, "inv1", url)

	got, err := cache.Get(ctx, "inv1")
	if err != nil || got == nil {
		t.Fatal("expected url to be cached")
	}

	err = cache.Invalidate(ctx, "inv1")
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	got, err = cache.Get(ctx, "inv1")
	if err != nil {
		t.Fatalf("Get after invalidate failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil after invalidate")
	}
}
