package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Metrics struct {
	requestCount   map[string]int64
	responseCodes  map[int]int64
	totalDuration  time.Duration
	durationCount  int64
	mu             sync.RWMutex
}

func NewMetrics() *Metrics {
	return &Metrics{
		requestCount:  make(map[string]int64),
		responseCodes: make(map[int]int64),
	}
}

func (m *Metrics) recordRequest(method, path string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := method + " " + path
	m.requestCount[key]++
	m.responseCodes[status]++
	m.totalDuration += duration
	m.durationCount++
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		m.recordRequest(r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

		for key, count := range m.requestCount {
			w.Write([]byte("# HELP http_requests_total Total HTTP requests\n"))
			w.Write([]byte("# TYPE http_requests_total counter\n"))
			w.Write([]byte("http_requests_total{path=\"" + key + "\"} " + strconv.FormatInt(count, 10) + "\n"))
		}

		for code, count := range m.responseCodes {
			w.Write([]byte("# HELP http_responses_total Total responses by status code\n"))
			w.Write([]byte("# TYPE http_responses_total counter\n"))
			w.Write([]byte("http_responses_total{code=\"" + strconv.Itoa(code) + "\"} " + strconv.FormatInt(count, 10) + "\n"))
		}

		if m.durationCount > 0 {
			avg := float64(m.totalDuration.Milliseconds()) / float64(m.durationCount)
			w.Write([]byte("# HELP http_request_duration_ms Average request duration\n"))
			w.Write([]byte("# TYPE http_request_duration_ms gauge\n"))
			w.Write([]byte("http_request_duration_ms " + strconv.FormatFloat(avg, 'f', 2, 64) + "\n"))
		}
	})
}
