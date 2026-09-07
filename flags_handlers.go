package main

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
)

// maxBodyBytes is the request body size limit for POST /flags and PUT /flags/{key}.
const maxBodyBytes = 1 << 20 // 1 MiB

// flagRequest is the request body for POST /flags. Enabled and RolloutPercent
// are pointers so a missing field is distinguishable from an explicit zero value.
type flagRequest struct {
	Key            string `json:"key"`
	Enabled        *bool  `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent *int   `json:"rollout_percent"`
}

// hasJSONContentType reports whether r carries a Content-Type of
// application/json (ignoring any parameters such as charset).
func hasJSONContentType(r *http.Request) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}
	return mt == "application/json"
}

// readLimitedBody reads the full request body, capping it at maxBodyBytes. On
// success it returns the raw bytes and true; otherwise it writes the
// appropriate error response and returns false.
func readLimitedBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	data, err := io.ReadAll(r.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return nil, false
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return nil, false
	}
	return data, true
}

// handleCreateFlag handles POST /flags.
func handleCreateFlag(w http.ResponseWriter, r *http.Request) {
	if !hasJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	data, ok := readLimitedBody(w, r)
	if !ok {
		return
	}
	var req flagRequest
	if err := json.Unmarshal(data, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}
	if req.Enabled == nil {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	rollout := 0
	if req.RolloutPercent != nil {
		rollout = *req.RolloutPercent
		if rollout < 0 || rollout > 100 {
			writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
			return
		}
	}
	f := Flag{
		Key:            req.Key,
		Enabled:        *req.Enabled,
		Description:    req.Description,
		RolloutPercent: rollout,
	}
	if err := store.Create(f); err != nil {
		if errors.Is(err, ErrKeyExists) {
			writeError(w, http.StatusConflict, "flag already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// handleListFlags handles GET /flags.
func handleListFlags(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, store.List())
}

// handleGetFlag handles GET /flags/{key}.
func handleGetFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	f, ok := store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// handleUpdateFlag handles PUT /flags/{key}.
func handleUpdateFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !hasJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "content type must be application/json")
		return
	}
	data, ok := readLimitedBody(w, r)
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if _, present := raw["enabled"]; !present {
		writeError(w, http.StatusBadRequest, "enabled is required")
		return
	}
	var upd flagUpdate
	if err := json.Unmarshal(data, &upd); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if upd.RolloutPercent != nil && (*upd.RolloutPercent < 0 || *upd.RolloutPercent > 100) {
		writeError(w, http.StatusBadRequest, "rollout_percent must be between 0 and 100")
		return
	}
	f, err := store.Update(key, upd)
	if err != nil {
		if errors.Is(err, ErrKeyNotFound) {
			writeError(w, http.StatusNotFound, "flag not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, f)
}

// handleDeleteFlag handles DELETE /flags/{key}.
func handleDeleteFlag(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if !store.Delete(key) {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
