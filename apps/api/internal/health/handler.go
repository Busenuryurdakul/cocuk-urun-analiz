package health

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

// Liveness returns 200 when the API process is running.
func Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok", Service: "miyuna-api"})
}

// Readiness returns 200 when dependencies are reachable (Phase 1: process-only).
func Readiness(w http.ResponseWriter, _ *http.Request) {
	// Phase 1: readiness mirrors liveness; Mongo/Redis checks in Phase 2+
	writeJSON(w, http.StatusOK, response{Status: "ready", Service: "miyuna-api"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
