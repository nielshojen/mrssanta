package xsrf

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("xsrf", xsrf)
}

type Response struct {
	ID string `json:"id"`
}

func xsrf(w http.ResponseWriter, r *http.Request) {

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
