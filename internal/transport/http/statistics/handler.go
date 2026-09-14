// Package statistics holds the HTTP handlers for the Dashboard endpoints.
// Handlers stay thin: they pull the caller's raw access token (verified
// by httpmw.RequireAuth) from the request, call into the application
// service, and map the result to a response. No business logic or
// upstream calls happen here.
package statistics

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	httpmw "github.com/sbezhuk/beebase-common/authmw"
	"github.com/sbezhuk/beebase-common/httpx"
	appstatistics "github.com/sbezhuk/beebase-statistics-service/internal/application/statistics"
)

// Default and bounds for the ?limit= query param on Activity, distinct
// from beebase-common/pagination's since Recent Activity isn't a paged
// collection - just "the N most recent items".
const (
	defaultActivityLimit = 10
	maxActivityLimit     = 50
)

// CodeInvalidLimit is returned when ?limit= on Activity isn't a positive
// integer within bounds.
const CodeInvalidLimit = "invalid_limit"

// Handler exposes the Dashboard HTTP endpoints. Every method requires the
// request to have already passed through httpmw.RequireAuth.
type Handler struct {
	service *appstatistics.Service
	log     *slog.Logger
}

// NewHandler returns a Handler backed by service.
func NewHandler(service *appstatistics.Service, log *slog.Logger) *Handler {
	return &Handler{service: service, log: log}
}

// Overview handles GET /statistics/overview.
func (h *Handler) Overview(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	overview, err := h.service.Overview(r.Context(), token)
	if err != nil {
		httpx.WriteInternalError(w, h.log, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newOverviewResponse(overview))
}

// Apiaries handles GET /statistics/apiaries.
func (h *Handler) Apiaries(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	stats, err := h.service.ApiaryStats(r.Context(), token)
	if err != nil {
		httpx.WriteInternalError(w, h.log, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newApiaryStatsResponse(stats))
}

// Inspections handles GET /statistics/inspections.
func (h *Handler) Inspections(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	stats, err := h.service.InspectionStats(r.Context(), token)
	if err != nil {
		httpx.WriteInternalError(w, h.log, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newInspectionStatsResponse(stats))
}

// Activity handles GET /statistics/activity.
func (h *Handler) Activity(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	limit, ok := h.parseLimit(w, r)
	if !ok {
		return
	}

	items, err := h.service.RecentActivity(r.Context(), token, limit)
	if err != nil {
		httpx.WriteInternalError(w, h.log, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newActivityResponse(items))
}

// Harvest handles GET /statistics/harvest.
func (h *Handler) Harvest(w http.ResponseWriter, r *http.Request) {
	token, ok := h.requireToken(w, r)
	if !ok {
		return
	}

	stats, err := h.service.HarvestStats(r.Context(), token)
	if err != nil {
		httpx.WriteInternalError(w, h.log, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, newHarvestStatsResponse(stats))
}

// requireToken returns the caller's raw access token, read back off the
// request's own Authorization header, which httpmw.RequireAuth already
// validated as a well-formed bearer token before this handler ran.
func (h *Handler) requireToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	if _, ok := httpmw.UserIDFromContext(r.Context()); !ok {
		httpx.WriteError(w, http.StatusUnauthorized, httpmw.CodeMissingAuthorization, "missing authentication")
		return "", false
	}

	const prefix = "Bearer "
	token := strings.TrimPrefix(r.Header.Get("Authorization"), prefix)
	return token, true
}

func (h *Handler) parseLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultActivityLimit, true
	}

	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 || v > maxActivityLimit {
		httpx.WriteValidationError(w, map[string]string{"limit": CodeInvalidLimit})
		return 0, false
	}

	return v, true
}
