package main

import (
	"log"
	"net/http"
	"time"
)

// newMux registers all seven routes onto a fresh mux and returns it.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("POST /flags", authMiddleware(http.HandlerFunc(handleCreateFlag)))
	mux.Handle("GET /flags", authMiddleware(http.HandlerFunc(handleListFlags)))
	mux.Handle("GET /flags/{key}", authMiddleware(http.HandlerFunc(handleGetFlag)))
	mux.Handle("PUT /flags/{key}", authMiddleware(http.HandlerFunc(handleUpdateFlag)))
	mux.Handle("DELETE /flags/{key}", authMiddleware(http.HandlerFunc(handleDeleteFlag)))
	mux.Handle("GET /flags/{key}/evaluate", authMiddleware(http.HandlerFunc(handleEvaluate)))
	mux.HandleFunc("GET /healthz", handleHealthz)
	return mux
}

func main() {
	mux := newMux()
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("feature-flag service listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
