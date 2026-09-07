package main

import (
	"hash/fnv"
	"net/http"
)

// evaluateResult is the JSON body answered by GET /flags/{key}/evaluate.
type evaluateResult struct {
	Key     string `json:"key"`
	User    string `json:"user"`
	Enabled bool   `json:"enabled"`
	Result  bool   `json:"result"`
}

// handleEvaluate handles GET /flags/{key}/evaluate?user={id}. The user
// identifier is read from the query, hashed together with the key and never
// stored. A missing user answers 400 and an unknown key 404.
func handleEvaluate(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")

	user := r.URL.Query().Get("user")
	if user == "" {
		writeError(w, http.StatusBadRequest, "missing user")
		return
	}

	flag, ok := store.Get(key)
	if !ok {
		writeError(w, http.StatusNotFound, "flag not found")
		return
	}

	result := false
	if flag.Enabled {
		h := fnv.New64a()
		_, _ = h.Write([]byte(key + ":" + user))
		result = h.Sum64()%100 < uint64(flag.RolloutPercent)
	}

	writeJSON(w, http.StatusOK, evaluateResult{
		Key:     key,
		User:    user,
		Enabled: flag.Enabled,
		Result:  result,
	})
}
