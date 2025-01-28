package xsrf

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var validAPIKey = os.Getenv("API_KEY")

func init() {
	functions.HTTP("xsrf", xsrf)
}

type Response struct {
	ID string `json:"id"`
}

func xsrf(w http.ResponseWriter, r *http.Request) {

	// Extract the API key from the header
	apiKey := r.Header.Get("X-API-Key")

	// Validate the API key
	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	machineID := r.URL.Query().Get("machine_id")

	response := Response{
		ID: machineID,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Println(string(responseJSON))

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
	}); err != nil {
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}
}
