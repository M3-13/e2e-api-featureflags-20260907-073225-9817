package main

import "net/http"

// handleEvaluate handles GET /flags/{key}/evaluate. Implemented by
// "Evaluierungs-Endpunkt mit FNV-1a-Hash implementieren".
func handleEvaluate(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
