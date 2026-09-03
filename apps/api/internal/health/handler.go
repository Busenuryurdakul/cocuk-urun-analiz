package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type response struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

var readinessCheck atomic.Value // func(context.Context) error

func SetReadinessCheck(fn func(context.Context) error) {
	readinessCheck.Store(fn)
}

// Liveness returns 200 when the API process is running.
func Liveness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{Status: "ok", Service: "miyuna-api"})
}

// Readiness returns 200 when dependencies are reachable.
func Readiness(w http.ResponseWriter, r *http.Request) {
	fn, ok := readinessCheck.Load().(func(context.Context) error)
	if ok && fn != nil {
		if err := fn(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, response{Status: "not_ready", Service: "miyuna-api"})
			return
		}
	}
	writeJSON(w, http.StatusOK, response{Status: "ready", Service: "miyuna-api"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
