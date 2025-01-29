package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getAllDevices(ctx context.Context) ([]map[string]interface{}, error) {
	// Query Firestore collection
	query := client.Collection(os.Getenv("DB_PREFIX") + "_devices")
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Failed to retrieve devices: %v", err)
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}

	// Convert Firestore documents to a generic JSON-friendly format
	var devices []map[string]interface{}
	for _, doc := range docs {
		data := doc.Data() // Retrieve document as a map

		// Convert Firestore timestamps to RFC3339 formatted strings
		if creationTime, ok := data["CreationTime"].(time.Time); ok {
			data["CreationTime"] = creationTime.Format(time.RFC3339)
		}
		if lastUpdated, ok := data["LastUpdated"].(time.Time); ok {
			data["LastUpdated"] = lastUpdated.Format(time.RFC3339)
		}

		devices = append(devices, data)
	}

	return devices, nil
}

func getDeviceByID(ctx context.Context, machineID string) (*Device, error) {
	// Reference Firestore document directly by ID
	docRef := client.Collection(os.Getenv("DB_PREFIX") + "_devices").Doc(machineID)

	// Get document snapshot
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		// Handle case where document does not exist
		if status.Code(err) == codes.NotFound {
			log.Printf("Device %s not found in Firestore", machineID)
			return nil, nil // Return nil instead of an empty slice
		}
		log.Printf("Error fetching device %s: %v", machineID, err)
		return nil, fmt.Errorf("failed to fetch device: %w", err)
	}

	// Unmarshal document into Rule struct
	var device Device
	if err := docSnap.DataTo(&device); err != nil {
		log.Printf("Failed to decode rule document: %v", err)
		return nil, fmt.Errorf("failed to decode device: %w", err)
	}

	return &device, nil
}

func updateDevice(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Device, error) {
	// Define a slice to hold the rules
	var devices []*Device

	// Unmarshal the JSON data into the slice
	err := json.Unmarshal(jsonData, &devices)
	if err != nil {
		// Log and return an error if JSON parsing fails
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate each rule
	for _, device := range devices {
		if device.Identifier == "" {
			return nil, fmt.Errorf("rule is missing identifier")
		}
	}

	// Save each rule to Firestore
	for _, device := range devices {
		saveDevice(ctx, client, w, device)
	}

	// Return the successfully saved rules
	return devices, nil
}
