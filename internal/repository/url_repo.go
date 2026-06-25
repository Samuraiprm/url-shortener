package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"url-shortener/internal/model"
)

type URLRepository struct {
	pool *pgxpool.Pool
}

func NewURLRepository(pool *pgxpool.Pool) *URLRepository {
	return &URLRepository{pool: pool}
}

func (r *URLRepository) Create(ctx context.Context, code, originalURL string) (*model.URL, error) {
	var url model.URL
	err := r.pool.QueryRow(ctx,
		`INSERT INTO urls (code, original_url) VALUES ($1, $2) RETURNING id, code, original_url, created_at`,
		code, originalURL,
	).Scan(&url.ID, &url.Code, &url.Original, &url.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert url: %w", err)
	}
	return &url, nil
}

func (r *URLRepository) GetByCode(ctx context.Context, code string) (*model.URL, error) {
	var url model.URL
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, original_url, created_at FROM urls WHERE code = $1`,
		code,
	).Scan(&url.ID, &url.Code, &url.Original, &url.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get url by code: %w", err)
	}
	return &url, nil
}

func (r *URLRepository) CodeExists(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM urls WHERE code = $1)`, code).Scan(&exists)
	return exists, err
}

func (r *URLRepository) RecordClick(ctx context.Context, urlID int64, ip, userAgent, referrer string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO clicks (url_id, ip, user_agent, referrer) VALUES ($1, $2, $3, $4)`,
		urlID, ip, userAgent, referrer,
	)
	return err
}

func (r *URLRepository) GetStats(ctx context.Context, code string) (*model.Stats, error) {
	var url model.URL
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, original_url, created_at FROM urls WHERE code = $1`, code,
	).Scan(&url.ID, &url.Code, &url.Original, &url.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get url for stats: %w", err)
	}

	var totalClicks int64
	err = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM clicks WHERE url_id = $1`, url.ID,
	).Scan(&totalClicks)
	if err != nil {
		return nil, fmt.Errorf("count clicks: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, url_id, ip, user_agent, referrer, created_at
		 FROM clicks WHERE url_id = $1 ORDER BY created_at DESC LIMIT 10`, url.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent clicks: %w", err)
	}
	defer rows.Close()

	var clicks []model.Click
	for rows.Next() {
		var c model.Click
		if err := rows.Scan(&c.ID, &c.URLID, &c.IP, &c.UserAgent, &c.Referrer, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan click: %w", err)
		}
		clicks = append(clicks, c)
	}

	return &model.Stats{
		TotalClicks:  totalClicks,
		URL:          &url,
		RecentClicks: clicks,
	}, nil
}
