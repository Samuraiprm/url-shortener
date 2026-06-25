package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"url-shortener/internal/model"
)

type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(addr string) (*RedisCache, error) {
	opts, err := redis.ParseURL("redis://" + addr)
	if err != nil {
		opts = &redis.Options{Addr: addr}
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisCache{client: client, ttl: 10 * time.Minute}, nil
}

func NewRedisCacheFromURL(redisURL string) (*RedisCache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &RedisCache{client: client, ttl: 10 * time.Minute}, nil
}

func (c *RedisCache) Get(ctx context.Context, code string) (*model.URL, error) {
	data, err := c.client.Get(ctx, "url:"+code).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var url model.URL
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}
	return &url, nil
}

func (c *RedisCache) Set(ctx context.Context, code string, url *model.URL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "url:"+code, data, c.ttl).Err()
}

func (c *RedisCache) Invalidate(ctx context.Context, code string) error {
	return c.client.Del(ctx, "url:"+code).Err()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}
