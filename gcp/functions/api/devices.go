package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
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

func getDeviceByID(ctx context.Context, machineID string) ([]*Device, error) {

	query := client.Collection(os.Getenv("DB_PREFIX")+"_devices").Where("name", "==", machineID)
	iter := query.Documents(ctx)

	var devices []*Device
	for {
		var d Device
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		err = doc.DataTo(&d)
		if err != nil {
			return nil, err
		}
		devices = append(devices, &d)
	}
	return devices, nil
}

func updateDevice(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Rule, error) {
	// Define a slice to hold the rules
	var rules []*Rule

	// Unmarshal the JSON data into the slice
	err := json.Unmarshal(jsonData, &rules)
	if err != nil {
		// Log and return an error if JSON parsing fails
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate each rule
	for _, rule := range rules {
		if rule.Identifier == "" {
			return nil, fmt.Errorf("rule is missing identifier")
		}
		if rule.Policy == "" {
			return nil, fmt.Errorf("rule with identifier %s is missing policy", rule.Identifier)
		}
		if rule.RuleType == "" {
			return nil, fmt.Errorf("rule with identifier %s is missing rule type", rule.Identifier)
		}
	}

	// Save each rule to Firestore
	for _, rule := range rules {
		saveRule(ctx, client, w, rule)
	}

	// Return the successfully saved rules
	return rules, nil
}
