package main

import (
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// logger writes access logs to stdout. It is a package-level variable so tests
// can swap the output writer and capture what is logged.
var logger = log.New(os.Stdout, "", log.LstdFlags)

// statusRecorder wraps an http.ResponseWriter and remembers the status code
// written by the handler.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// authMiddleware protects a route by requiring an Authorization header of the
// form "Bearer <AUTH_TOKEN>". The expected token is read lazily from the
// AUTH_TOKEN environment variable on every request, so a token set or changed
// at runtime takes effect without a restart. The comparison uses a
// constant-time routine so the length of a wrong token cannot be probed. A
// missing header, a missing configured token, or a mismatching token all answer
// 401 with a JSON error object; a valid token passes the request through.
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := os.Getenv("AUTH_TOKEN")
		header := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if token == "" || !strings.HasPrefix(header, prefix) ||
			subtle.ConstantTimeCompare([]byte(header[len(prefix):]), []byte(token)) != 1 {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs one line per request with the method, the path WITHOUT
// the query string, the status code and the duration. The query string is
// deliberately dropped so a value such as `user` never appears in the log.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		logger.Printf("method=%s path=%s status=%d duration=%s",
			r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}
