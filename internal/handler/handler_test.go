package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"url-shortener/internal/model"
)

func TestHealth(t *testing.T) {
	h := &Handler{baseURL: "http://localhost:8080"}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func TestCreateURL_EmptyBody(t *testing.T) {
	h := &Handler{baseURL: "http://localhost:8080"}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()

	h.CreateURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreateURL_InvalidJSON(t *testing.T) {
	h := &Handler{baseURL: "http://localhost:8080"}

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(`not json`))
	w := httptest.NewRecorder()

	h.CreateURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRedirect_NotFound(t *testing.T) {
	h := &Handler{baseURL: "http://localhost:8080"}

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	req.SetPathValue("code", "")
	w := httptest.NewRecorder()

	h.Redirect(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetStats_EmptyCode(t *testing.T) {
	h := &Handler{baseURL: "http://localhost:8080"}

	req := httptest.NewRequest(http.MethodGet, "/api/stats/", nil)
	req.SetPathValue("code", "")
	w := httptest.NewRecorder()

	h.GetStats(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	var resp model.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Error != "code is required" {
		t.Errorf("expected 'code is required', got %s", resp.Error)
	}
}
