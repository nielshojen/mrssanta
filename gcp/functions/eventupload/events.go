package eventupload

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Get machine_id from query parameters
	machineID := r.URL.Query().Get("machine_id")
	if machineID != "" {
		fmt.Println("Request from:", machineID)
	}

	// Parse request body
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
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
