package main

import "net/http"

// handleHealthz answers GET /healthz with 200 {"status":"ok"}.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
