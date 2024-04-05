package postflight

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func postflight(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	if request.RulesReceived == 0 && request.RulesProcessed == 0 {
		http.Error(w, "Request body is empty or does not contain required fields", http.StatusBadRequest)
		return
	}

	machineID := r.URL.Query().Get("machine_id")

	response := Response{
		ID:             machineID,
		RulesReceived:  request.RulesReceived,
		RulesProcessed: request.RulesProcessed,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Println(string(responseJSON))

	// Set response headers and write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
