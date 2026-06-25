//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

)
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
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
		t.Fatalf("failed to connect to pool: %v", err)
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

	cleanup := func() {
		pool.Close()
		pgContainer.Terminate(ctx)
	}

	return pool, cleanup
}

func TestRepository_CreateAndGetByCode(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewURLRepository(pool)
	ctx := context.Background()

	url, err := repo.Create(ctx, "abc123", "https://example.com")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if url.Code != "abc123" {
		t.Errorf("expected code abc123, got %s", url.Code)
	}
	if url.Original != "https://example.com" {
		t.Errorf("expected original https://example.com, got %s", url.Original)
	}

	got, err := repo.GetByCode(ctx, "abc123")
	if err != nil {
		t.Fatalf("GetByCode failed: %v", err)
	}
	if got.ID != url.ID {
		t.Errorf("expected ID %d, got %d", url.ID, got.ID)
	}
}

func TestRepository_CodeExists(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewURLRepository(pool)
	ctx := context.Background()

	exists, err := repo.CodeExists(ctx, "nope")
	if err != nil {
		t.Fatalf("CodeExists failed: %v", err)
	}
	if exists {
		t.Error("expected false for non-existent code")
	}

	_, err = repo.Create(ctx, "exists", "https://example.com")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	exists, err = repo.CodeExists(ctx, "exists")
	if err != nil {
		t.Fatalf("CodeExists failed: %v", err)
	}
	if !exists {
		t.Error("expected true for existing code")
	}
}

func TestRepository_RecordClickAndStats(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewURLRepository(pool)
	ctx := context.Background()

	url, err := repo.Create(ctx, "stats1", "https://example.com/stats")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	for i := range 5 {
		err = repo.RecordClick(ctx, url.ID, fmt.Sprintf("192.168.1.%d", i), "test-agent", "https://google.com")
		if err != nil {
			t.Fatalf("RecordClick failed: %v", err)
		}
	}

	stats, err := repo.GetStats(ctx, "stats1")
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalClicks != 5 {
		t.Errorf("expected 5 clicks, got %d", stats.TotalClicks)
	}

	if len(stats.RecentClicks) != 5 {
		t.Errorf("expected 5 recent clicks, got %d", len(stats.RecentClicks))
	}

	if stats.URL.Code != "stats1" {
		t.Errorf("expected code stats1, got %s", stats.URL.Code)
	}
}

func TestRepository_DuplicateCode(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewURLRepository(pool)
	ctx := context.Background()

	_, err := repo.Create(ctx, "dup1", "https://example.com/1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	_, err = repo.Create(ctx, "dup1", "https://example.com/2")
	if err == nil {
		t.Error("expected error for duplicate code")
	}
}

func TestRepository_DeleteCascade(t *testing.T) {
	pool, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewURLRepository(pool)
	ctx := context.Background()

	url, err := repo.Create(ctx, "del1", "https://example.com/del")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = repo.RecordClick(ctx, url.ID, "1.2.3.4", "agent", "ref")
	if err != nil {
		t.Fatalf("RecordClick failed: %v", err)
	}

	_, err = pool.Exec(ctx, "DELETE FROM urls WHERE code = $1", "del1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM clicks WHERE url_id = $1", url.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count clicks failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 clicks after cascade delete, got %d", count)
	}
}
