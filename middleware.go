package main

import (
	"log"
	"net/http"
	"os"
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
