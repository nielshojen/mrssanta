package postflight

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

	defer r.Body.Close()

	var protoRequest syncv1.PostflightRequest
	format, err := readSyncRequest(r, &protoRequest)
	if err != nil {
		Warnf("Failed to decode postflight request: %v", err)
		status := http.StatusBadRequest
		if errors.Is(err, errUnsupportedContentType) {
			status = http.StatusUnsupportedMediaType
		}
		http.Error(w, "Failed to decode request", status)
		return
	}
	request := requestFromProto(&protoRequest)

	machineID, ok := machineIDFromRequest(r, "postflight", request.MachineID)
	if !ok {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	if !checkXSRF(w, r, machineID) {
		return
	}
	request.MachineID = machineID

	existingDevice, err := getDevice(ctx, client, request.MachineID)
	if err != nil {
		http.Error(w, "Error retrieving device", http.StatusInternalServerError)
		return
	}

	if existingDevice == nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	if request.RulesHash != "" {
		existingDevice.RulesHash = request.RulesHash
		Infof("postflight: updated RulesHash for %s to %s", request.MachineID, request.RulesHash)
	}

	pendingSyncType := existingDevice.PendingSyncType
	hadRequestedSyncType := existingDevice.RequestedSyncType != ""
	if completeSync(existingDevice, request, time.Now().UTC()) {
		Infof("postflight: %s sync completed for %s - received=%d, processed=%d, last_sync=%v",
			pendingSyncType, request.MachineID, request.RulesReceived, request.RulesProcessed,
			existingDevice.LastSync.Time())
	} else {
		Infof("postflight: rule mismatch for %s - received=%d, processed=%d (allowed diff=%d) - marking NeedsCleanSync = true",
			request.MachineID, request.RulesReceived, request.RulesProcessed, existingDevice.CompilerRuleCount)
	}

	clearRequested := hadRequestedSyncType && existingDevice.RequestedSyncType == ""
	if err := saveDevice(ctx, client, existingDevice, request.MachineID, clearRequested); err != nil {
		http.Error(w, "Failed to update device record", http.StatusInternalServerError)
		return
	}

	if err := writeSyncResponse(w, format, &syncv1.PostflightResponse{}); err != nil {
		Errorf("Failed to write postflight response for %s: %v", request.MachineID, err)
	}
}

// completeSync updates device to reflect the end of the sync started by the
// last preflight and reports whether the client applied the rules it was sent.
//
// On success last_sync is advanced to the time the sync started rather than to
// now, so rules changed while the sync was running are sent again next time
// instead of being skipped. A completed CLEAN_ALL also records last_clean_sync,
// and any completed clean sync satisfies a clean sync requested by an admin.
// On failure the device is flagged for a clean sync and last_sync is left
// alone, so the next sync resends everything.
func completeSync(device *Device, request Request, now time.Time) bool {
	pendingSyncType := device.PendingSyncType
	device.PendingSyncType = ""

	allowedDiff := int(device.CompilerRuleCount)
	diff := int(request.RulesReceived) - int(request.RulesProcessed)
	if diff < 0 {
		diff = -diff
	}

	if diff > allowedDiff {
		device.NeedsCleanSync = true
		return false
	}

	completedAt := device.SyncStartedAt
	if completedAt == 0 {
		completedAt = primitive.NewDateTimeFromTime(now)
	}

	device.LastSync = completedAt
	if pendingSyncType == "CLEAN_ALL" {
		device.LastCleanSync = completedAt
	}
	if pendingSyncType != "" && pendingSyncType != "NORMAL" {
		device.RequestedSyncType = ""
	}
	device.NeedsCleanSync = false
	device.RequestCleanSync = false

	return true
}

// saveDevice writes the postflight outcome. requested_sync_type is only
// cleared when clearRequestedSyncType is set, so a clean sync an admin
// requests while this sync was running is not lost.
func saveDevice(ctx context.Context, client *mongo.Client, device *Device, machineID string, clearRequestedSyncType bool) error {
	collection := client.Database(dbName).Collection("devices")

	device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())

	update := bson.M{
		"needs_clean_sync":   device.NeedsCleanSync,
		"last_updated":       device.LastUpdated,
		"request_clean_sync": device.RequestCleanSync,
		"pending_sync_type":  device.PendingSyncType,
	}

	if clearRequestedSyncType {
		update["requested_sync_type"] = ""
	}

	if device.LastSync != 0 {
		update["last_sync"] = device.LastSync
	}

	if device.LastCleanSync != 0 {
		update["last_clean_sync"] = device.LastCleanSync
	}

	if device.RulesHash != "" {
		update["rules_hash"] = device.RulesHash
	}

	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": machineID},
		bson.M{"$set": update},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		Errorf("Failed to update device: %v", err)
		return fmt.Errorf("failed to update device: %w", err)
	}

	return nil
}

func getDevice(ctx context.Context, client *mongo.Client, machineID string) (*Device, error) {
	collection := client.Database(dbName).Collection("devices")

	var existingDevice Device
	err := collection.FindOne(ctx, bson.M{"_id": machineID}).Decode(&existingDevice)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}

		Errorf("Failed to retrieve existing device: %v", err)
		return nil, fmt.Errorf("failed to retrieve existing device: %w", err)
	}

	return &existingDevice, nil
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
