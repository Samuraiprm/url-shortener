package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"url-shortener/internal/model"
	"url-shortener/internal/service"
)

type Handler struct {
	svc     *service.URLService
	baseURL string
}

func NewHandler(svc *service.URLService, baseURL string) *Handler {
	return &Handler{svc: svc, baseURL: baseURL}
}

// CreateURL
// @Summary      Shorten a URL
// @Description  Create a short URL from a long one
// @Tags         urls
// @Accept       json
// @Produce      json
// @Param        request body model.CreateURLRequest true "URL to shorten"
// @Success      201  {object}  model.CreateURLResponse
// @Failure      400  {object}  model.ErrorResponse
// @Failure      422  {object}  model.ErrorResponse
// @Router       /api/shorten [post]
func (h *Handler) CreateURL(w http.ResponseWriter, r *http.Request) {
	var req model.CreateURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.URL == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "url is required"})
		return
	}

	resp, err := h.svc.Shorten(r.Context(), req.URL)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, model.ErrorResponse{Error: err.Error()})
		return
	}
	resp.ShortURL = h.baseURL + "/" + resp.Code

	writeJSON(w, http.StatusCreated, resp)
}

// Redirect
// @Summary      Redirect to original URL
// @Description  302 redirect to the original URL by short code
// @Tags         urls
// @Param        code path string true "Short code"
// @Success      302
// @Failure      404
// @Router       /{code} [get]
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.NotFound(w, r)
		return
	}

	u, err := h.svc.Resolve(r.Context(), code)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	go h.svc.RecordClick(context.Background(), u.ID, r.RemoteAddr, r.UserAgent(), r.Referer())

	http.Redirect(w, r, u.Original, http.StatusFound)
}

// GetStats
// @Summary      Get URL analytics
// @Description  Get click count and recent clicks for a short URL
// @Tags         analytics
// @Param        code path string true "Short code"
// @Success      200  {object}  model.Stats
// @Failure      404  {object}  model.ErrorResponse
// @Router       /api/stats/{code} [get]
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "code is required"})
		return
	}

	stats, err := h.svc.GetStats(r.Context(), code)
	if err != nil {
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{Error: "url not found"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// Health
// @Summary      Health check
// @Description  Returns server health status
// @Tags         system
// @Success      200  {object}  map[string]string
// @Router       /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
