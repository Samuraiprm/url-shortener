// @title           URL Shortener API
// @version         1.0
// @description     URL Shortener service with analytics and caching
// @host            localhost:8080
// @BasePath        /
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "url-shortener/docs"
	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/handler"
	myMiddleware "url-shortener/internal/middleware"
	"url-shortener/internal/repository"
	"url-shortener/internal/service"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("unable to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("unable to ping database: %v", err)
	}

	var redisCache *cache.RedisCache

	if cfg.RedisURL != "" {
		redisCache, err = cache.NewRedisCacheFromURL(cfg.RedisURL)
	} else {
		redisCache, err = cache.NewRedisCache(cfg.RedisAddr)
	}
	if err != nil {
		log.Fatalf("unable to connect to redis: %v", err)
	}
	defer redisCache.Close()

	repo := repository.NewURLRepository(pool)
	svc := service.NewURLService(repo, redisCache)
	h := handler.NewHandler(svc, cfg.BaseURL)

	rl := myMiddleware.NewRateLimiter(10, 20)
	metrics := myMiddleware.NewMetrics()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(5))
	r.Use(metrics.Middleware)
	r.Use(myMiddleware.SecurityHeaders)
	r.Use(myMiddleware.RequestID)
	r.Use(myMiddleware.MaxBodySize(1 << 20))

	r.Get("/health", h.Health)
	r.Handle("/metrics", metrics.Handler())
	r.Get("/swagger/*", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	r.Group(func(r chi.Router) {
		r.Use(rl.Middleware)
		r.Post("/api/shorten", h.CreateURL)
		r.Get("/api/stats/{code}", h.GetStats)
		r.Get("/{code}", h.Redirect)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("server starting on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
