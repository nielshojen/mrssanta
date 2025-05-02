package postflight

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

func postflight(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var request Request

	apiKey := r.Header.Get("X-API-Key")

	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot parse request body: %v\n", err)
	}
	defer r.Body.Close()

	decompressedData, err := decompressZlib(reqBody)
	if err != nil {
		log.Printf("Failed to decompress body: %v", err)
		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(decompressedData, &request); err != nil {
		log.Printf("Failed to decode JSON from decompressed data: %v", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return

	}

	existingDevice, err := getDevice(ctx, client, request.MachineID)
	if err != nil {
		http.Error(w, "Error retrieving device", http.StatusInternalServerError)
		return
	}

	if existingDevice.NeedsCleanSync {
		existingDevice.NeedsCleanSync = false
		if err := saveDevice(ctx, client, existingDevice, request.MachineID); err != nil {
			http.Error(w, "Failed to update device", http.StatusInternalServerError)
			return
		}
	}

	log.Printf("Rules Received: %d, Rules Processed: %d", request.RulesReceived, request.RulesProcessed)

	w.WriteHeader(http.StatusOK)
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
