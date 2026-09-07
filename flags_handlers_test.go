package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// resetStore reassigns the package-level store so each test starts empty.
func resetStore() {
	store = NewStore()
}

// do performs an HTTP request against newMux and returns the recorder.
func do(t *testing.T, method, target, contentType string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rr := httptest.NewRecorder()
	newMux().ServeHTTP(rr, req)
	return rr
}

func jsonBody(t *testing.T, rr *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON response %q: %v", rr.Body.String(), err)
	}
	return m
}

func TestCreateFlagThenGet(t *testing.T) {
	resetStore()
	rr := do(t, http.MethodPost, "/flags", "application/json",
		`{"key":"feature-a","enabled":true,"description":"desc","rollout_percent":50}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var flag Flag
	if err := json.Unmarshal(rr.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid flag JSON: %v", err)
	}
	if flag.Key != "feature-a" || !flag.Enabled || flag.Description != "desc" || flag.RolloutPercent != 50 {
		t.Fatalf("unexpected flag: %+v", flag)
	}

	rr = do(t, http.MethodGet, "/flags/feature-a", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	got := jsonBody(t, rr)
	if got["key"] != "feature-a" || got["enabled"] != true || got["rollout_percent"] != float64(50) {
		t.Fatalf("unexpected GET body: %v", got)
	}
}

func TestCreateFlagDefaults(t *testing.T) {
	resetStore()
	rr := do(t, http.MethodPost, "/flags", "application/json", `{"key":"min","enabled":false}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var flag Flag
	if err := json.Unmarshal(rr.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid flag JSON: %v", err)
	}
	if flag.RolloutPercent != 0 {
		t.Fatalf("expected default rollout_percent 0, got %d", flag.RolloutPercent)
	}
}

func TestCreateFlagValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty key", `{"key":"","enabled":true}`},
		{"missing enabled", `{"key":"k"}`},
		{"rollout above 100", `{"key":"k","enabled":true,"rollout_percent":101}`},
		{"rollout below 0", `{"key":"k","enabled":true,"rollout_percent":-1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetStore()
			rr := do(t, http.MethodPost, "/flags", "application/json", c.body)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
			}
			if got := jsonBody(t, rr)["error"]; got == nil || got == "" {
				t.Fatalf("expected error object, got %v", got)
			}
		})
	}
}

func TestCreateFlagDuplicate(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"dup","enabled":true}`)
	rr := do(t, http.MethodPost, "/flags", "application/json", `{"key":"dup","enabled":true}`)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := jsonBody(t, rr)["error"]; got == nil || got == "" {
		t.Fatalf("expected error object, got %v", got)
	}
}

func TestListFlags(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"zebra","enabled":true}`)
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"apple","enabled":false}`)
	rr := do(t, http.MethodGet, "/flags", "", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var flags []Flag
	if err := json.Unmarshal(rr.Body.Bytes(), &flags); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
	if flags[0].Key != "apple" || flags[1].Key != "zebra" {
		t.Fatalf("expected sorted keys [apple zebra], got %v", flags)
	}
}

func TestGetFlagNotFound(t *testing.T) {
	resetStore()
	rr := do(t, http.MethodGet, "/flags/missing", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if got := jsonBody(t, rr)["error"]; got == nil || got == "" {
		t.Fatalf("expected error object, got %v", got)
	}
}

func TestUpdateFlag(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"upd","enabled":true,"description":"old","rollout_percent":10}`)

	rr := do(t, http.MethodPut, "/flags/upd", "application/json",
		`{"enabled":false,"description":"new","rollout_percent":80}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var flag Flag
	if err := json.Unmarshal(rr.Body.Bytes(), &flag); err != nil {
		t.Fatalf("invalid flag JSON: %v", err)
	}
	if flag.Enabled || flag.Description != "new" || flag.RolloutPercent != 80 {
		t.Fatalf("unexpected updated flag: %+v", flag)
	}

	rr = do(t, http.MethodPut, "/flags/missing", "application/json", `{"enabled":false}`)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown key, got %d", rr.Code)
	}

	rr = do(t, http.MethodPut, "/flags/upd", "application/json", `{"enabled":true,"rollout_percent":101}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid rollout, got %d", rr.Code)
	}

	rr = do(t, http.MethodPut, "/flags/upd", "application/json", `{"description":"no enabled"}`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing enabled, got %d", rr.Code)
	}
}

func TestDeleteFlag(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"del","enabled":true}`)

	rr := do(t, http.MethodDelete, "/flags/del", "", "")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}

	rr = do(t, http.MethodDelete, "/flags/del", "", "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", rr.Code)
	}
}

func TestCreateFlagBodyTooLarge(t *testing.T) {
	resetStore()
	big := strings.Repeat("x", maxBodyBytes+1)
	rr := do(t, http.MethodPost, "/flags", "application/json", `{"key":"big","enabled":true,"description":"`+big+`"}`)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
	if got := jsonBody(t, rr)["error"]; got == nil || got == "" {
		t.Fatalf("expected error object, got %v", got)
	}
}

func TestUpdateFlagBodyTooLarge(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"big2","enabled":true}`)
	big := strings.Repeat("x", maxBodyBytes+1)
	rr := do(t, http.MethodPut, "/flags/big2", "application/json", `{"enabled":true,"description":"`+big+`"}`)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rr.Code)
	}
}

func TestCreateFlagWrongContentType(t *testing.T) {
	resetStore()
	rr := do(t, http.MethodPost, "/flags", "text/plain", `{"key":"ct","enabled":true}`)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
	if got := jsonBody(t, rr)["error"]; got == nil || got == "" {
		t.Fatalf("expected error object, got %v", got)
	}

	rr = do(t, http.MethodPost, "/flags", "", `{"key":"ct","enabled":true}`)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 for missing content type, got %d", rr.Code)
	}
}

func TestUpdateFlagWrongContentType(t *testing.T) {
	resetStore()
	do(t, http.MethodPost, "/flags", "application/json", `{"key":"ctu","enabled":true}`)
	rr := do(t, http.MethodPut, "/flags/ctu", "text/plain", `{"enabled":false}`)
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", rr.Code)
	}
}

func TestErrorDoesNotLeakInternal(t *testing.T) {
	resetStore()
	rr := do(t, http.MethodPost, "/flags", "application/json", `{"key":"bad",`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", rr.Code)
	}
	body := jsonBody(t, rr)
	msg, _ := body["error"].(string)
	if strings.Contains(msg, "json:") || strings.Contains(msg, "invalid character") {
		t.Fatalf("error message leaks internal details: %q", msg)
	}
	if msg == "" {
		t.Fatalf("expected a non-empty error message")
	}
}
