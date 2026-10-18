package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func getAllDevices(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(dbName).Collection("devices")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		Errorf("Failed to retrieve devices: %v", err)
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}
	defer cursor.Close(ctx)

	var devices []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			Warnf("Failed to decode device document: %v", err)
			continue
		}

		if creationTime, ok := doc["CreationTime"].(bson.DateTime); ok {
			doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
		}

		if lastUpdated, ok := doc["LastUpdated"].(bson.DateTime); ok {
			doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		devices = append(devices, doc)
	}

	if err := cursor.Err(); err != nil {
		Errorf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return devices, nil
}

func getDeviceByID(ctx context.Context, machineID string) ([]map[string]interface{}, error) {
	collection := client.Database(dbName).Collection("devices")

	var devices []map[string]interface{}
	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"_id": machineID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			Infof("Device %s not found in MongoDB", machineID)
			return nil, nil
		}
		Errorf("Error fetching device %s: %v", machineID, err)
		return nil, fmt.Errorf("failed to fetch device: %w", err)
	}

	if creationTime, ok := doc["CreationTime"].(bson.DateTime); ok {
		doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
	}

	if lastUpdated, ok := doc["LastUpdated"].(bson.DateTime); ok {
		doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
	}

	devices = append(devices, doc)
	return devices, nil
}

func createDevice(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Device, error) {
	collection := client.Database(dbName).Collection("devices")

	var devices []*Device
	err := json.Unmarshal(jsonData, &devices)
	if err != nil {
		Warnf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	for _, device := range devices {
		if device.Identifier == "" {
			return nil, fmt.Errorf("device is missing identifier")
		}
	}

	for _, device := range devices {
		now := bson.NewDateTimeFromTime(time.Now())

		updateFields, err := deviceUpdateFields(device, now)
		if err != nil {
			Errorf("Failed to build update for device %s: %v", device.Identifier, err)
			return nil, fmt.Errorf("failed to build update for device %s: %w", device.Identifier, err)
		}

		filter := bson.M{"_id": device.Identifier}
		update := bson.M{"$set": updateFields}
		opts := options.UpdateOne().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			Errorf("Failed to update device %s: %v", device.Identifier, err)
			return nil, fmt.Errorf("failed to update device %s: %w", device.Identifier, err)
		}
	}

	return devices, nil
}

func deviceUpdateFields(device *Device, now bson.DateTime) (bson.M, error) {
	device.LastUpdated = now

	updateData, err := bson.Marshal(device)
	if err != nil {
		return nil, fmt.Errorf("failed to convert device to BSON: %w", err)
	}

	var updateMap bson.M
	if err := bson.Unmarshal(updateData, &updateMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal device BSON: %w", err)
	}

	delete(updateMap, "_id")
	return updateMap, nil
}

func saveDevice(ctx context.Context, client *mongo.Client, device *Device, machineID string) error {
	collection := client.Database(dbName).Collection("devices")

	device.ID = machineID

	updateMap, err := deviceUpdateFields(device, bson.NewDateTimeFromTime(time.Now()))
	if err != nil {
		Errorf("Failed to build device update: %v", err)
		return fmt.Errorf("failed to build device update: %w", err)
	}

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": machineID},
		bson.M{"$set": updateMap},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		Errorf("Failed to save device data: %v", err)
		return fmt.Errorf("failed to save device data: %w", err)
	}

	return nil
}
