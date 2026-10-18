package eventupload

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
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

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	machineID, _ := machineIDFromRequest(r, "eventupload", "")
	if machineID != "" {
		Infof("Request from: %s", machineID)
	}

	defer r.Body.Close()

	var request syncv1.EventUploadRequest
	format, err := readSyncRequest(r, &request)
	if err != nil {
		Log(SeverityError, "Failed to decode eventupload request", map[string]any{
			"error":      err.Error(),
			"machine_id": machineID,
		})
		status := http.StatusBadRequest
		if errors.Is(err, errUnsupportedContentType) {
			status = http.StatusUnsupportedMediaType
		}
		http.Error(w, "Failed to decode request", status)
		return
	}

	if machineID == "" {
		var ok bool
		machineID, ok = machineIDFromRequest(r, "eventupload", request.GetMachineId())
		if !ok {
			http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
			return
		}
		Infof("Request from: %s", machineID)
	}

	if !checkXSRF(w, r, machineID) {
		return
	}

	skipDecisions := map[string]bool{}
	var savedBinaries []string

	for _, protoEvent := range request.GetEvents() {
		event := eventFromProto(protoEvent)

		if err := logEvent(event); err != nil {
			Errorf("Error logging event: %v", err)
			http.Error(w, "Failed to log event", http.StatusInternalServerError)
			return
		}

		sanitizedEvent := sanitizeEvent(event)

		if !skipDecisions[sanitizedEvent.Decision] {
			if _, err := saveEvent(ctx, client, sanitizedEvent); err != nil {
				Errorf("Error saving event: %v", err)
				http.Error(w, "Failed to save event", http.StatusInternalServerError)
				return
			}
		}

		savedBinaries = append(savedBinaries, sanitizedEvent.FileSha256)
	}

	now := time.Now().UTC()

	if err := saveFileAccessEvents(ctx, client, request.GetFileAccessEvents(), machineID, now); err != nil {
		Errorf("Error saving file access events for %s: %v", machineID, err)
		http.Error(w, "Failed to save file access events", http.StatusInternalServerError)
		return
	}

	if err := saveStandaloneApprovals(ctx, client, request.GetAuditEvents(), machineID, now); err != nil {
		Errorf("Error saving audit events for %s: %v", machineID, err)
		http.Error(w, "Failed to save audit events", http.StatusInternalServerError)
		return
	}

	response := syncv1.EventUploadResponse_builder{
		EventUploadBundleBinaries: savedBinaries,
	}.Build()

	if err := writeSyncResponse(w, format, response); err != nil {
		Errorf("Failed to write eventupload response for %s: %v", machineID, err)
	}
}

// saveFileAccessEvents stores each File Access Authorization event as its
// own document, since every occurrence matters for auditing.
func saveFileAccessEvents(ctx context.Context, client *mongo.Client, events []*syncv1.FileAccessEvent, machineID string, now time.Time) error {
	if len(events) == 0 {
		return nil
	}

	docs := make([]any, 0, len(events))
	for _, event := range events {
		docs = append(docs, fileAccessEventFromProto(event, machineID, now))
	}

	if _, err := client.Database(dbName).Collection("file_access_events").InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert file access events: %w", err)
	}

	Infof("Saved %d file access event(s) from %s", len(docs), machineID)
	return nil
}

// saveStandaloneApprovals stores the rules users created themselves in
// STANDALONE mode so admins can review them.
func saveStandaloneApprovals(ctx context.Context, client *mongo.Client, events []*syncv1.AuditEvent, machineID string, now time.Time) error {
	approvals := standaloneApprovalsFromProto(events, machineID, now)
	if len(approvals) == 0 {
		return nil
	}

	docs := make([]any, 0, len(approvals))
	for _, approval := range approvals {
		docs = append(docs, approval)
	}

	if _, err := client.Database(dbName).Collection("standalone_approvals").InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert standalone approvals: %w", err)
	}

	Infof("Saved %d standalone approval(s) from %s", len(docs), machineID)
	return nil
}

func saveEvent(ctx context.Context, client *mongo.Client, event Event) (Event, error) {
	if event.FileSha256 == "" {
		return Event{}, errors.New("no file_sha256 field in the event")
	}

	collection := client.Database(dbName).Collection("events")

	event.LastUpdated = primitive.NewDateTimeFromTime(time.Now().UTC())
	event.ID = event.FileSha256

	updateData, err := bson.Marshal(event)
	if err != nil {
		return Event{}, fmt.Errorf("failed to convert event to BSON: %v", err)
	}

	var updateMap bson.M
	err = bson.Unmarshal(updateData, &updateMap)
	if err != nil {
		return Event{}, fmt.Errorf("failed to unmarshal BSON: %v", err)
	}

	delete(updateMap, "_id")

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": event.ID},
		bson.M{"$set": updateMap},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return Event{}, fmt.Errorf("failed to store data in MongoDB: %v", err)
	}

	return event, nil
}

func machineIDFromPath(path string, operation string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 {
		return "", false
	}

	if parts[0] != "santa" || parts[1] != operation || parts[2] == "" {
		return "", false
	}

	return parts[2], true
}

func machineIDFromRequest(r *http.Request, operation string, bodyMachineID string) (string, bool) {
	if machineID := strings.TrimSpace(r.URL.Query().Get("machine_id")); machineID != "" {
		return machineID, true
	}

	if machineID, ok := machineIDFromPath(r.URL.Path, operation); ok {
		return machineID, true
	}

	if machineID := strings.TrimSpace(bodyMachineID); machineID != "" {
		return machineID, true
	}

	return "", false
}
