package eventupload

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func sanitizeEvent(event Event) Event {
	sanitized := event
	eventType := reflect.TypeOf(event)
	eventValue := reflect.ValueOf(&sanitized).Elem()

	for _, fieldName := range []string{
		"ExecutingUser",
		"ExecutionTime",
		"LoggedinUsers",
		"CurrentSessions",
		"PID",
		"PPID",
		"Labels",
		"Severity",
	} {
		field, found := eventType.FieldByName(fieldName)
		if found {
			sanitizedField := eventValue.FieldByName(fieldName)
			zeroValue := reflect.Zero(field.Type)
			sanitizedField.Set(zeroValue)
		}
	}

	return sanitized
}

func eventuploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var request Request
	var response Response

	// Extract the API key from the header
	apiKey := r.Header.Get("X-API-Key")

	// Validate the API key
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

	// Get machine_id from query parameters
	machineID := r.URL.Query().Get("machine_id")
	if machineID != "" {
		fmt.Println("Request from:", machineID)
	}

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot parse request body: %v\n", err)
	}
	defer r.Body.Close()

	// Decompress the data
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

	// Define a set of Decision values to skip
	skipDecisions := map[string]bool{}

	// Process events
	var savedBinaries []string
	for _, event := range request.Events {

		err := logEvent(event)
		if err != nil {
			fmt.Println("Error logging event:", err)
			return
		}

		sanitizedEvent := sanitizeEvent(event)

		// Check if the Decision is in the skip list
		if !skipDecisions[sanitizedEvent.Decision] {
			_, err = saveEvent(ctx, client, sanitizedEvent)
			if err != nil {
				fmt.Println("Error saving event:", err)
				return
			}
		}

		savedBinaries = append(savedBinaries, sanitizedEvent.FileSha256)
	}

	// Update response
	response.EventUploadBundleBinaries = savedBinaries

	// Encode response to JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
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

func saveEvent(ctx context.Context, client *mongo.Client, event Event) (Event, error) {
	// Check if FileSha256 field exists
	if event.FileSha256 == "" {
		return Event{}, errors.New("no file_sha256 field in the event")
	}

	// Get MongoDB collection
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("events")

	// Convert time.Time to primitive.DateTime
	event.LastUpdated = primitive.NewDateTimeFromTime(time.Now().UTC())

	// Set `_id` to `FileSha256`
	event.ID = event.FileSha256

	// Convert event struct to BSON (excluding `_id` to prevent modification errors)
	updateData, err := bson.Marshal(event)
	if err != nil {
		return Event{}, fmt.Errorf("failed to convert event to BSON: %v", err)
	}

	// Convert BSON to map to avoid modifying `_id`
	var updateMap bson.M
	err = bson.Unmarshal(updateData, &updateMap)
	if err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal BSON: %v", err)
	}

	delete(updateMap, "_id") // Ensure `_id` is not updated

	// Perform upsert (insert if not exists, update if exists)
	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": event.ID},   // Match by `_id`
		bson.M{"$set": updateMap}, // Dynamically update all fields
		options.Update().SetUpsert(true),
	)

	if err != nil {
		return Event{}, fmt.Errorf("failed to store data in MongoDB: %v", err)
	}

	return event, nil
}
