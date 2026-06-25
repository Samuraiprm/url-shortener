package model

import "time"

type URL struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Original  string    `json:"original_url"`
	CreatedAt time.Time `json:"created_at"`
}

type Click struct {
	ID        int64     `json:"id"`
	URLID     int64     `json:"url_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
	CreatedAt time.Time `json:"created_at"`
}

type Stats struct {
	TotalClicks int64   `json:"total_clicks"`
	URL         *URL    `json:"url"`
	RecentClicks []Click `json:"recent_clicks,omitempty"`
}

type CreateURLRequest struct {
	URL string `json:"url"`
}

type CreateURLResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
	Original string `json:"original_url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
