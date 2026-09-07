package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func rateLimitedHandler() http.Handler {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return rateLimitMiddleware(h)
}

func newRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "192.0.2.1:1234"
	return req
}

func TestRateLimitAllowsUnderLimit(t *testing.T) {
	handler := rateLimitedHandler()
	for i := 0; i < defaultRateCapacity; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newRequest())
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimitExceededReturns429WithJSONError(t *testing.T) {
	handler := rateLimitedHandler()
	for i := 0; i < defaultRateCapacity; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newRequest())
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newRequest())
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body %q: %v", rr.Body.String(), err)
	}
	if body["error"] != "too many requests" {
		t.Fatalf("expected error message, got %q", body["error"])
	}
}

func TestRateLimitDoesNotPassThroughWhenExceeded(t *testing.T) {
	var served int
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimitMiddleware(h)
	for i := 0; i < defaultRateCapacity; i++ {
		handler.ServeHTTP(httptest.NewRecorder(), newRequest())
	}
	handler.ServeHTTP(httptest.NewRecorder(), newRequest())
	if served != defaultRateCapacity {
		t.Fatalf("expected handler to serve exactly %d requests, got %d", defaultRateCapacity, served)
	}
}

func TestRateLimitRecoveryAfterRefill(t *testing.T) {
	capacity := 2
	refill := 10 * time.Millisecond
	limiter := newRateLimiter(refill, capacity)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimitMiddlewareWithLimiter(h, limiter)

	for i := 0; i < capacity; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, newRequest())
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, newRequest())
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 while exhausted, got %d", rr.Code)
	}

	time.Sleep(2 * refill)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, newRequest())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 after refill, got %d", rr.Code)
	}
}

func TestClientIPStripsPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "203.0.113.9:54321"
	if got := clientIP(req); got != "203.0.113.9" {
		t.Fatalf("expected 203.0.113.9, got %q", got)
	}
}

func TestRateLimitBucketsArePerIP(t *testing.T) {
	capacity := 1
	refill := time.Hour
	limiter := newRateLimiter(refill, capacity)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := rateLimitMiddlewareWithLimiter(h, limiter)

	mk := func(ip string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		req.RemoteAddr = ip
		return req
	}

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, mk("198.51.100.1:1"))
	if rr.Code != http.StatusOK {
		t.Fatalf("first IP first request: expected 200, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, mk("198.51.100.1:2"))
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("first IP second request: expected 429, got %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, mk("198.51.100.2:1"))
	if rr.Code != http.StatusOK {
		t.Fatalf("second IP first request: expected 200, got %d", rr.Code)
	}
}
