package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	// Extract the API key from the header
	apiKey := r.Header.Get("X-API-Key")

	// Validate the API key
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
		// Handle GET /rules
		rules, err := getAllRules(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Serialize and write the rules as JSON
		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && Endpoint == "rules" && ID != "":
		// Handle GET /rules/{id}
		rules, err := getRuleByID(ctx, ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching rules: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Serialize and write the rules as JSON
		if err := json.NewEncoder(w).Encode(rules); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && Endpoint == "rules":
		// Handle POST /rules

		// Read the request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Call createRules to parse and save the rules
		rules, err := createRules(ctx, w, reqBody)
		if err != nil {
			log.Printf("Failed to create rules: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create rules: %v", err), http.StatusBadRequest)
			return
		}

		// Respond with success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Rules saved successfully", "count": %d}`, len(rules))

	case r.Method == http.MethodGet && Endpoint == "devices" && ID == "":
		// Handle GET /rules
		devices, err := getAllDevices(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching devices: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Serialize and write the devices as JSON
		if err := json.NewEncoder(w).Encode(devices); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodGet && Endpoint == "devices" && ID != "":
		// Handle GET /rules/{id}
		device, err := getDeviceByID(ctx, ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching device: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Serialize and write the rules as JSON
		if err := json.NewEncoder(w).Encode(device); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	case r.Method == http.MethodPost && Endpoint == "devices":
		// Handle POST /rules

		// Read the request body
		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Call createRules to parse and save the rules
		devices, err := createDevice(ctx, w, reqBody)
		if err != nil {
			log.Printf("Failed to create devices: %v", err)
			http.Error(w, fmt.Sprintf("Failed to create devices: %v", err), http.StatusBadRequest)
			return
		}

		// Respond with success
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"message": "Devices saved successfully", "count": %d}`, len(devices))

	case r.Method == http.MethodGet && Endpoint == "events" && ID == "":
		// Handle GET /rules
		events, err := getAllEvents(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error fetching events: %v", err), http.StatusInternalServerError)
			return
		}

		// Set response headers
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Serialize and write the rules as JSON
		if err := json.NewEncoder(w).Encode(events); err != nil {
			http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		}

	default:
		// Handle unsupported paths or methods
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
