package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// maxMetricsBodyBytes caps a metrics upload. Santa's exports are a few KB.
const maxMetricsBodyBytes = 1 << 20

// Santa's MetricURL export carries no credentials (it ignores
// SyncExtraHeaders), so this endpoint can't authenticate senders. Instead it
// only accepts bounded bodies in one of Santa's two JSON metric formats, so
// anything else never reaches the logs.
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Only POST and PUT allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMetricsBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			Warnf("Rejected metrics upload larger than %d bytes", maxMetricsBodyBytes)
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		Warnf("Failed to read metrics request body: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	format, metrics := parseMetrics(body)
	if format == "" {
		Warnf("Rejected metrics upload that is not rawjson or monarchjson (%d bytes)", len(body))
		http.Error(w, "Unrecognised metrics format", http.StatusBadRequest)
		return
	}

	Log(SeverityInfo, "Received metrics", map[string]any{
		"format":  format,
		"metrics": metrics,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// parseMetrics identifies which Santa metric format body uses and returns
// the decoded document, or "" if it is neither:
//   - rawjson:     {"metrics": {...}, "root_labels": {...}}
//   - monarchjson: {"metricsCollection": [...]}
func parseMetrics(body []byte) (string, map[string]any) {
	var doc map[string]any
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", nil
	}

	if _, ok := doc["metrics"].(map[string]any); ok {
		return "rawjson", doc
	}
	if _, ok := doc["metricsCollection"].([]any); ok {
		return "monarchjson", doc
	}
	return "", nil
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/metrics", metricsHandler)

	addr := ":8080"
	Infof("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		Fatalf("Server failed: %v", err)
	}
}
