package preflight

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var device Device

	apiKey := r.Header.Get("X-API-Key")

	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
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

	if err := json.Unmarshal(decompressedData, &device); err != nil {
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	existingDevice, err := getDevice(ctx, client, machineID)
	if err != nil {
		http.Error(w, "Error retrieving device", http.StatusInternalServerError)
		return
	}

	device.Identifier = machineID

	needsCleanSync := existingDevice == nil ||
		(existingDevice.LastCleanSync.Time().IsZero()) ||
		time.Since(existingDevice.LastCleanSync.Time()) > 24*time.Hour

	if existingDevice == nil {
		device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
		device.LastCleanSync = primitive.NewDateTimeFromTime(time.Now())
		err = saveDevice(ctx, client, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to create new device", http.StatusInternalServerError)
			return
		}
	} else {
		device.ClientMode = existingDevice.ClientMode
		device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
		if needsCleanSync {
			device.LastCleanSync = primitive.NewDateTimeFromTime(time.Now())
		} else {
			device.LastCleanSync = existingDevice.LastCleanSync
		}

		err = saveDevice(ctx, client, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to update device data", http.StatusInternalServerError)
			return
		}
	}

	syncType := "NORMAL"
	if needsCleanSync {
		syncType = "CLEAN"
		log.Printf("Setting CLEAN sync for device %s", machineID)
	}

	response := Response{
		BatchSize:             100,
		FullSyncInterval:      60,
		ClientMode:            &device.ClientMode,
		EnableBundles:         true,
		EnableTransitiveRules: true,
		SyncType:              syncType,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

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

func saveDevice(ctx context.Context, client *mongo.Client, device *Device, machineID string) error {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("devices")

	device.ID = machineID

	device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())

	updateData, err := bson.Marshal(device)
	if err != nil {
		log.Printf("Failed to convert device to BSON: %v", err)
		return fmt.Errorf("failed to convert device to BSON: %w", err)
	}

	var updateMap bson.M
	err = bson.Unmarshal(updateData, &updateMap)
	if err != nil {
		log.Printf("Failed to unmarshal BSON: %v", err)
		return fmt.Errorf("failed to unmarshal BSON: %w", err)
	}

	delete(updateMap, "_id")

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": machineID},
		bson.M{"$set": updateMap},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		log.Printf("Failed to save device data: %v", err)
		return fmt.Errorf("failed to save device data: %w", err)
	}

	return nil
}

func getDevice(ctx context.Context, client *mongo.Client, machineID string) (*Device, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("devices")

	var existingDevice Device
	err := collection.FindOne(ctx, bson.M{"_id": machineID}).Decode(&existingDevice)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		log.Printf("Failed to retrieve existing device: %v", err)
		return nil, fmt.Errorf("failed to retrieve existing device: %w", err)
	}

	return &existingDevice, nil
}
