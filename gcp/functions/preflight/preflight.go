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
	"time"
)

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var device Device

	// Validate API Key
	if r.Header.Get("X-API-Key") != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	// Validate Content-Type
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	// Extract machine ID from the query parameters
	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	// Read and decompress request body
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	decompressedData, err := decompressZlib(reqBody)
	if err != nil {
		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		return
	}

	// Unmarshal JSON into Device struct
	if err := json.Unmarshal(decompressedData, &device); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	// Get existing device data from Firestore
	existingDevice, err := getDevice(ctx, machineID)
	if err != nil {
		http.Error(w, "Error retrieving device", http.StatusInternalServerError)
		return
	}

	// Set Identifier to machineID before saving
	device.Identifier = machineID

	// Determine if a clean sync is needed
	needsCleanSync := existingDevice == nil || existingDevice.LastCleanSync.IsZero() || time.Since(existingDevice.LastCleanSync) > 24*time.Hour

	// If the device does not exist, create a new one
	if existingDevice == nil {
		device.LastUpdated = time.Now()
		device.LastCleanSync = time.Now() // Initial clean sync timestamp
		err = saveDevice(ctx, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to create new device", http.StatusInternalServerError)
			return
		}
	} else {
		// Keep existing Firestore values where necessary
		device.ClientMode = existingDevice.ClientMode
		device.LastUpdated = time.Now()
		if needsCleanSync {
			device.LastCleanSync = time.Now()
		} else {
			device.LastCleanSync = existingDevice.LastCleanSync
		}

		// Save the updated device
		err = saveDevice(ctx, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to update device data", http.StatusInternalServerError)
			return
		}
	}

	// Determine SyncType
	syncType := "NORMAL"
	if needsCleanSync {
		syncType = "CLEAN"
		log.Printf("Setting CLEAN sync for device %s", machineID)
	}

	// Prepare response
	response := Response{
		BatchSize:             100,
		FullSyncInterval:      60,
		ClientMode:            &device.ClientMode,
		EnableBundles:         true,
		EnableTransitiveRules: false,
		SyncType:              syncType,
	}

	// Send JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
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
