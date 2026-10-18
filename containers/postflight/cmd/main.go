package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/nielshojen/mrssanta/postflight"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := postflight.Init(ctx); err != nil {
		postflight.Fatalf("postflight.Init: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	host := ""
	if os.Getenv("LOCAL_ONLY") == "true" {
		host = "127.0.0.1"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/santa/postflight/", postflight.Handler)

	server := &http.Server{
		Addr:              host + ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	postflight.Infof("Listening on %s:%s", hostOrDefault(host), port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		postflight.Fatalf("ListenAndServe: %v", err)
	}
}

func hostOrDefault(host string) string {
	if host == "" {
		return "0.0.0.0"
	}

	return host
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
