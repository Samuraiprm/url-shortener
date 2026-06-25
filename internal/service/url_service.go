package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"url-shortener/internal/cache"
	"url-shortener/internal/model"
	"url-shortener/internal/repository"
)

type URLService struct {
	repo  *repository.URLRepository
	cache *cache.RedisCache
}

func NewURLService(repo *repository.URLRepository, cache *cache.RedisCache) *URLService {
	return &URLService{repo: repo, cache: cache}
}

func (s *URLService) Shorten(ctx context.Context, originalURL string) (*model.CreateURLResponse, error) {
	if len(originalURL) > 2048 {
		return nil, fmt.Errorf("url too long (max 2048 characters)")
	}

	parsed, err := url.ParseRequestURI(originalURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("only http and https schemes are allowed")
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("url must have a host")
	}

	lower := strings.ToLower(originalURL)
	for _, block := range []string{"javascript:", "data:", "vbscript:"} {
		if strings.Contains(lower, block) {
			return nil, fmt.Errorf("url contains blocked scheme")
		}
	}

	code, err := s.generateUniqueCode(ctx)
	if err != nil {
		return nil, err
	}

	u, err := s.repo.Create(ctx, code, originalURL)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, code, u)

	return &model.CreateURLResponse{
		Code:     code,
		Original: originalURL,
	}, nil
}

func (s *URLService) Resolve(ctx context.Context, code string) (*model.URL, error) {
	if u, err := s.cache.Get(ctx, code); err == nil && u != nil {
		return u, nil
	}

	u, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, code, u)
	return u, nil
}

func (s *URLService) GetStats(ctx context.Context, code string) (*model.Stats, error) {
	return s.repo.GetStats(ctx, code)
}

func (s *URLService) RecordClick(ctx context.Context, urlID int64, ip, userAgent, referrer string) error {
	return s.repo.RecordClick(ctx, urlID, ip, userAgent, referrer)
}

func (s *URLService) generateUniqueCode(ctx context.Context) (string, error) {
	for range 10 {
		b := make([]byte, 4)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		code := hex.EncodeToString(b)
		exists, err := s.repo.CodeExists(ctx, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code")
}
