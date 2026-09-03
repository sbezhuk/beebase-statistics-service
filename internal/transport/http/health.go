package http

import (
	"net/http"

	"github.com/sbezhuk/beebase-common/httpx"
)

type statusResponse struct {
	Status string `json:"status"`
}

// HealthHandler reports liveness: the process is up and serving requests.
// This service has no database or other local dependency of its own to
// probe, so it doubles as the readiness check too - see NewRouter.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}
