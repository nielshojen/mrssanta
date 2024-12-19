package eventupload

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
	"reflect"
)

func sanitizeEvent(event Event) Event {
	sanitized := event // Initialize sanitized struct with all fields from the original event
	eventType := reflect.TypeOf(event)
	eventValue := reflect.ValueOf(&sanitized).Elem()

	for _, fieldName := range []string{
		"ExecutingUser", "ExecutionTime", "LoggedinUsers", "CurrentSessions", "Decision",
		"FileBundlePath", "FileBundleHashBillis", "PID", "PPID", "ParentName",
		"QuarantineTimestamp", "QuarantineAgentBundleID", "Labels",
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

	reqBody, err := ioutil.ReadAll(r.Body)
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

	// Process events
	var savedBinaries []string
	for _, event := range request.Events {

		err := logEvent(event)
		if err != nil {
			fmt.Println("Error logging event:", err)
			return
		}

		sanitizedEvent := sanitizeEvent(event)

		_, err = saveEvent(ctx, client, sanitizedEvent)
		if err != nil {
			fmt.Println("Error saving event:", err)
			return
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
