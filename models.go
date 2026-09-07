package main

import (
	"encoding/json"
	"net/http"
)

// Flag is the single shared representation of a feature flag.
type Flag struct {
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description,omitempty"`
	RolloutPercent int    `json:"rollout_percent"`
}

// flagUpdate carries the fields that may be changed via PUT /flags/{key}.
// Description and RolloutPercent are pointers so a missing field in the JSON
// body is distinguishable from an explicit zero value.
type flagUpdate struct {
	Enabled        bool    `json:"enabled"`
	Description    *string `json:"description"`
	RolloutPercent *int    `json:"rollout_percent"`
}

// writeJSON serializes v as JSON and writes it with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error object of the shape {"error":"..."}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
