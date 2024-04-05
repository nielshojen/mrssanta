package preflight

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var device Device

	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		log.Println("Machine ID is missing in the request URL")
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	saveDevice(ctx, client, w, &device, machineID)

	response := Response{
		BatchSize:             100,
		FullSyncInterval:      600,
		ClientMode:            "MONITOR",
		EnableBundles:         true,
		EnableTransitiveRules: false,
	}

	if device.RequestCleanSync {
		response.SyncType = "clean_all"
	} else {
		response.SyncType = "normal"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response to JSON: %v", err)
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}
}
