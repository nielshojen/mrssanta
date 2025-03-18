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
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("devices")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to retrieve devices: %v", err)
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}
	defer cursor.Close(ctx)

	var devices []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode device document: %v", err)
			continue
		}

		if creationTime, ok := doc["CreationTime"].(primitive.DateTime); ok {
			doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
		}

		if lastUpdated, ok := doc["LastUpdated"].(primitive.DateTime); ok {
			doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		devices = append(devices, doc)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return devices, nil
}

func getDeviceByID(ctx context.Context, machineID string) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("devices")

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
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("devices")

	var devices []*Device
	err := json.Unmarshal(jsonData, &devices)
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	for _, device := range devices {
		if device.Identifier == "" {
			return nil, fmt.Errorf("device is missing identifier")
		}
	}

	for _, device := range devices {
		now := primitive.NewDateTimeFromTime(time.Now())

		updateFields := bson.M{}
		jsonData, _ := json.Marshal(device)
		json.Unmarshal(jsonData, &updateFields)

		updateFields["last_updated"] = now

		filter := bson.M{"_id": device.Identifier}
		update := bson.M{"$set": updateFields}
		opts := options.Update().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Failed to update device %s: %v", device.Identifier, err)
			return nil, fmt.Errorf("failed to update device %s: %w", device.Identifier, err)
		}
	}

	return devices, nil
}
