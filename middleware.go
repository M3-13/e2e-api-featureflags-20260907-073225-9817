package main

import "net/http"

// loggingMiddleware passes next through unchanged. The logging behaviour is
// implemented by "Logging-Middleware implementieren".
func loggingMiddleware(next http.Handler) http.Handler {
	return next
}
