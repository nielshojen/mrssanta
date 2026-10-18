package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/nielshojen/mrssanta/xsrf"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := xsrf.Init(ctx); err != nil {
		xsrf.Fatalf("xsrf.Init: %v", err)
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
	mux.HandleFunc("/santa/xsrf/", xsrf.Handler)

	server := &http.Server{
		Addr:              host + ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	xsrf.Infof("Listening on %s:%s", hostOrDefault(host), port)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		xsrf.Fatalf("ListenAndServe: %v", err)
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
