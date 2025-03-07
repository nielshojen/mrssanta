package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getAllDevices(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("devices")

	// Find all documents
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to retrieve devices: %v", err)
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB documents to a JSON-friendly format
	var devices []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode device document: %v", err)
			continue
		}

		// Convert MongoDB timestamps (primitive.DateTime) to RFC3339 formatted strings
		if creationTime, ok := doc["CreationTime"].(primitive.DateTime); ok {
			doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
		}
		if lastUpdated, ok := doc["LastUpdated"].(primitive.DateTime); ok {
			doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		devices = append(devices, doc)
	}

	// Check for cursor errors
	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return devices, nil
}

func getDeviceByID(ctx context.Context, machineID string) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("devices")

	// Fetch document by ID into a flexible `bson.M`
	var devices []map[string]interface{}
	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"_id": machineID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("Device %s not found in MongoDB", machineID)
			return nil, nil // Return nil to indicate no result
		}
		log.Printf("Error fetching device %s: %v", machineID, err)
		return nil, fmt.Errorf("failed to fetch device: %w", err)
	}

	// Convert MongoDB timestamps (primitive.DateTime) to RFC3339 formatted strings
	if creationTime, ok := doc["CreationTime"].(primitive.DateTime); ok {
		doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
	}
	if lastUpdated, ok := doc["LastUpdated"].(primitive.DateTime); ok {
		doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
	}

	devices = append(devices, doc)

	return devices, nil
}

func createDevice(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Device, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("devices")

	// Decode JSON into slice of Device structs
	var devices []*Device
	err := json.Unmarshal(jsonData, &devices)
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate devices
	for _, device := range devices {
		if device.Identifier == "" {
			return nil, fmt.Errorf("device is missing identifier")
		}
	}

	// Update each device in MongoDB
	for _, device := range devices {
		// Set LastUpdated timestamp
		now := primitive.NewDateTimeFromTime(time.Now())

		// Convert device to bson.M to dynamically update only provided fields
		updateFields := bson.M{}
		jsonData, _ := json.Marshal(device)     // Convert struct to JSON
		json.Unmarshal(jsonData, &updateFields) // Convert JSON to bson.M

		updateFields["last_updated"] = now // Always update LastUpdated

		// Update MongoDB document without removing missing fields
		filter := bson.M{"_id": device.Identifier}
		update := bson.M{"$set": updateFields}
		opts := options.Update().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Failed to update device %s: %v", device.Identifier, err)
			return nil, fmt.Errorf("failed to update device %s: %w", device.Identifier, err)
		}
	}

	// Return updated devices
	return devices, nil
}

func saveDevice(ctx context.Context, w http.ResponseWriter, device *Device) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("devices")

	// Set LastUpdated timestamp
	now := primitive.NewDateTimeFromTime(time.Now())

	// Convert the device struct to a bson.M map
	updateFields := bson.M{}
	jsonData, _ := json.Marshal(device)     // Convert struct to JSON
	json.Unmarshal(jsonData, &updateFields) // Convert JSON to bson.M

	updateFields["last_updated"] = now // Always update LastUpdated

	// Update MongoDB document without removing missing fields
	filter := bson.M{"_id": device.Identifier}
	update := bson.M{"$set": updateFields}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Failed to update device %s: %v", device.Identifier, err)
		http.Error(w, "Failed to update device", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Device saved successfully"})
}
