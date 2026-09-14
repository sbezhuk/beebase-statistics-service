// Package http wires the HTTP transport: routing, middleware, and the
// handlers that don't yet belong to a specific domain (health, readiness).
package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	httpmw "github.com/sbezhuk/beebase-common/authmw"
	statisticshttp "github.com/sbezhuk/beebase-statistics-service/internal/transport/http/statistics"
)

// NewRouter builds the root HTTP handler for the service.
func NewRouter(
	log *slog.Logger,
	statisticsHandler *statisticshttp.Handler,
	tokenParser httpmw.AccessTokenParser,
) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// This service holds no data of its own - every request it serves
	// depends on apiary/hive/inspection-service being reachable, not on
	// anything local it could probe - so, like the gateway, liveness and
	// readiness are the same trivial check.
	r.Get("/health", HealthHandler)
	r.Get("/ready", HealthHandler)

	r.Group(func(r chi.Router) {
		r.Use(httpmw.RequireAuth(tokenParser))

		r.Route("/api/v1/statistics", func(r chi.Router) {
			r.Get("/overview", statisticsHandler.Overview)
			r.Get("/apiaries", statisticsHandler.Apiaries)
			r.Get("/inspections", statisticsHandler.Inspections)
			r.Get("/activity", statisticsHandler.Activity)
			r.Get("/harvest", statisticsHandler.Harvest)
		})
	})

	return r
}

// requestLogger logs each request's method, path, status, and duration
// through slog instead of chi's default stdlib logger.
func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
