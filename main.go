package main

import (
	"log"
	"net/http"
	"time"
)

// newMux registers all seven routes onto a fresh mux and returns it.
func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /flags", handleCreateFlag)
	mux.HandleFunc("GET /flags", handleListFlags)
	mux.HandleFunc("GET /flags/{key}", handleGetFlag)
	mux.HandleFunc("PUT /flags/{key}", handleUpdateFlag)
	mux.HandleFunc("DELETE /flags/{key}", handleDeleteFlag)
	mux.HandleFunc("GET /flags/{key}/evaluate", handleEvaluate)
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
