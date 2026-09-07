package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prev := logger.Writer()
	logger.SetOutput(buf)
	t.Cleanup(func() { logger.SetOutput(prev) })
	return buf
}

func TestLoggingMiddlewareLogsMethodPathStatusDuration(t *testing.T) {
	buf := captureLog(t)

	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/flags", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rr.Code)
	}

	out := buf.String()
	if !strings.Contains(out, "method=POST") {
		t.Fatalf("log line missing method: %q", out)
	}
	if !strings.Contains(out, "path=/flags") {
		t.Fatalf("log line missing path: %q", out)
	}
	if !strings.Contains(out, "status=201") {
		t.Fatalf("log line missing status: %q", out)
	}
	durationRe := regexp.MustCompile(`duration=\d+(\.\d+)?(ns|µs|ms|s|m|h)`)
	if !durationRe.MatchString(out) {
		t.Fatalf("log line missing duration: %q", out)
	}
}

func TestLoggingMiddlewareLogsDefaultStatus200(t *testing.T) {
	buf := captureLog(t)

	h := loggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if !strings.Contains(buf.String(), "status=200") {
		t.Fatalf("log line missing default status 200: %q", buf.String())
	}
}

func TestLoggingMiddlewareOmitsQueryStringOnEvaluate(t *testing.T) {
	buf := captureLog(t)

	h := loggingMiddleware(newMux())
	req := httptest.NewRequest(http.MethodGet, "/flags/feature-key/evaluate?user=secret-user", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	out := buf.String()
	if !strings.Contains(out, "path=/flags/feature-key/evaluate") {
		t.Fatalf("log line missing path: %q", out)
	}
	if strings.Contains(out, "secret-user") {
		t.Fatalf("log line leaks the user query value: %q", out)
	}
	if strings.Contains(out, "?") {
		t.Fatalf("log line contains a query string: %q", out)
	}
	if !strings.Contains(out, "method=GET") {
		t.Fatalf("log line missing method: %q", out)
	}
}
