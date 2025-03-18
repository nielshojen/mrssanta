package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	apiKey := r.Header.Get("X-API-Key")

	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	Endpoint := r.URL.Query().Get("endpoint")
	fmt.Println("Request for:", Endpoint)
	if Endpoint != "" {
		fmt.Println("Request for:", Endpoint)
	}

	ID := r.URL.Query().Get("id")
	if ID != "" {
		fmt.Println("With ID for:", ID)
	}

	switch {
	case r.Method == http.MethodGet && Endpoint == "rules" && ID == "":
		rules, err := getAllRules(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && Endpoint == "rules" && ID != "":
		rules, err := getRuleByID(ctx, ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && Endpoint == "rules":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		rules, err := createRules(ctx, w, reqBody)
		if err != nil {
			log.Printf("Failed to create rules: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create rules: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rules saved successfully", "count": %d}`, len(rules))

	case r.Method == http.MethodGet && Endpoint == "devices" && ID == "":
		devices, err := getAllDevices(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching devices: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(devices); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && Endpoint == "devices" && ID != "":
		device, err := getDeviceByID(ctx, ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching device: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(device); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && Endpoint == "devices":
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		devices, err := createDevice(ctx, w, reqBody)
		if err != nil {
			log.Printf("Failed to create devices: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create devices: %v", err), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Devices saved successfully", "count": %d}`, len(devices))

	case r.Method == http.MethodGet && Endpoint == "events" && ID == "":
		events, err := getAllEvents(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(events); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && Endpoint == "events":
		events, err := getEventByID(ctx, ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(events); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
