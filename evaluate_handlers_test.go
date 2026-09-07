package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func evaluateRequest(t *testing.T, key, user string) (int, evaluateResult) {
	t.Helper()
	setAuthEnv(t)
	mux := newMux()
	path := "/flags/" + key + "/evaluate"
	if user != "" {
		path += "?user=" + user
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+testAuthToken)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	var res evaluateResult
	if rr.Code == http.StatusOK {
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("invalid JSON body %q: %v", rr.Body.String(), err)
		}
	}
	return rr.Code, res
}

func seedFlag(t *testing.T, f Flag) {
	t.Helper()
	if err := store.Create(f); err != nil {
		t.Fatalf("seed Create(%q): %v", f.Key, err)
	}
	t.Cleanup(func() { _ = store.Delete(f.Key) })
}

func TestEvaluateDeterministic(t *testing.T) {
	seedFlag(t, Flag{Key: "deterministic", Enabled: true, RolloutPercent: 50})

	code1, res1 := evaluateRequest(t, "deterministic", "user-1")
	code2, res2 := evaluateRequest(t, "deterministic", "user-1")

	if code1 != http.StatusOK || code2 != http.StatusOK {
		t.Fatalf("expected 200, got %d and %d", code1, code2)
	}
	if res1.Result != res2.Result {
		t.Fatalf("expected deterministic result, got %v then %v", res1.Result, res2.Result)
	}
	if res1.Key != "deterministic" || res1.User != "user-1" {
		t.Fatalf("unexpected response %+v", res1)
	}
	if !res1.Enabled {
		t.Fatal("expected enabled=true in response")
	}
}

func TestEvaluateDisabledAlwaysFalse(t *testing.T) {
	seedFlag(t, Flag{Key: "disabled", Enabled: false, RolloutPercent: 100})

	for _, user := range []string{"a", "b", "c"} {
		code, res := evaluateRequest(t, "disabled", user)
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if res.Result {
			t.Fatalf("expected result=false for disabled flag, user %q", user)
		}
	}
}

func TestEvaluateRolloutPercentZero(t *testing.T) {
	seedFlag(t, Flag{Key: "zero", Enabled: true, RolloutPercent: 0})

	code, res := evaluateRequest(t, "zero", "any-user")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if res.Result {
		t.Fatal("expected result=false when rollout_percent is 0")
	}
}

func TestEvaluateRolloutPercentHundred(t *testing.T) {
	seedFlag(t, Flag{Key: "hundred", Enabled: true, RolloutPercent: 100})

	code, res := evaluateRequest(t, "hundred", "any-user")
	if code != http.StatusOK {
		t.Fatalf("expected 200, got %d", code)
	}
	if !res.Result {
		t.Fatal("expected result=true when rollout_percent is 100")
	}
}

func TestEvaluateMissingUser(t *testing.T) {
	seedFlag(t, Flag{Key: "missing-user", Enabled: true, RolloutPercent: 50})

	code, _ := evaluateRequest(t, "missing-user", "")
	if code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing user, got %d", code)
	}
}

func TestEvaluateUnknownKey(t *testing.T) {
	code, _ := evaluateRequest(t, "does-not-exist", "user-1")
	if code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown key, got %d", code)
	}
}
