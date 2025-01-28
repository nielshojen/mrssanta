package preflight

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var device Device

	// Extract the API key from the header
	apiKey := r.Header.Get("X-API-Key")

	// Validate the API key
	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	// Validate the Content-Type header
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	// Extract machine ID from the query parameters
	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		log.Println("Machine ID is missing in the request URL")
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	// Read the request body
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot parse request body: %v\n", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decompress the data (if needed)
	decompressedData, err := decompressZlib(reqBody)
	if err != nil {
		log.Printf("Failed to decompress body: %v", err)
		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		return
	}

	// Unmarshal JSON into the Device struct
	if err := json.Unmarshal(decompressedData, &device); err != nil {
		log.Printf("Failed to decode JSON from decompressed data: %v", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	// Save the device data (assumes saveDevice is implemented)
	saveDevice(ctx, client, w, &device, machineID)

	// Prepare the response
	response := Response{
		BatchSize:             100,
		FullSyncInterval:      60,
		ClientMode:            &device.ClientMode,
		EnableBundles:         true,
		EnableTransitiveRules: false,
	}

	// Set the sync type based on the request
	if device.RequestCleanSync {
		response.SyncType = "clean_all"
	} else {
		response.SyncType = "normal"
	}

	// Write the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response to JSON: %v", err)
		http.Error(w, "Failed to encode response to JSON", http.StatusInternalServerError)
		return
	}
}

// decompressZlib decompresses zlib-compressed data
func decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer reader.Close()

	var decompressedData bytes.Buffer
	if _, err := io.Copy(&decompressedData, reader); err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}
	return decompressedData.Bytes(), nil
}
