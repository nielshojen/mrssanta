package preflight

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
	"google.golang.org/protobuf/encoding/protojson"
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

	machineID, ok := machineIDFromRequest(r, "preflight")
	if !ok {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	if !checkXSRF(w, r, machineID) {
		return
	}

	defer r.Body.Close()

	var request syncv1.PreflightRequest
	format, err := readSyncRequest(r, &request)
	if err != nil {
		Warnf("Failed to decode preflight request for %s: %v", machineID, err)
		status := http.StatusBadRequest
		if errors.Is(err, errUnsupportedContentType) {
			status = http.StatusUnsupportedMediaType
		}
		http.Error(w, "Failed to decode request", status)
		return
	}

	device := deviceFromPreflight(&request, machineID)

	if device.BinaryRuleCount == 0 &&
		device.CertificateRuleCount == 0 &&
		device.CompilerRuleCount == 0 &&
		device.TransitiveRuleCount == 0 &&
		device.TeamIDRuleCount == 0 &&
		device.SigningIDRuleCount == 0 &&
		device.CDHashRuleCount == 0 {
		Infof("Device %s has 0 rules in all categories - flagging for clean sync", machineID)
		device.NeedsCleanSync = true
	}

	existingDevice, err := getDevice(ctx, client, machineID)
	if err != nil {
		http.Error(w, "Error retrieving device", http.StatusInternalServerError)
		return
	}

	globalSettings, err := loadGlobalSettings(ctx, client)
	if err != nil {
		Errorf("Failed to load global settings for %s: %v", machineID, err)
		http.Error(w, "Error retrieving settings", http.StatusInternalServerError)
		return
	}

	incomingTotalRules := device.BinaryRuleCount +
		device.CertificateRuleCount +
		device.CompilerRuleCount +
		device.TransitiveRuleCount +
		device.TeamIDRuleCount +
		device.SigningIDRuleCount +
		device.CDHashRuleCount

	if existingDevice != nil {
		var diff uint32
		if incomingTotalRules > existingDevice.RuleCount {
			diff = incomingTotalRules - existingDevice.RuleCount
		} else {
			diff = existingDevice.RuleCount - incomingTotalRules
		}

		var allowedDiff uint32 = device.CompilerRuleCount
		if diff > allowedDiff {
			Infof("Device %s rule count differs by %d (allowed=%d) - flagging for clean sync",
				machineID, diff, allowedDiff)
			device.NeedsCleanSync = true
		}
	}

	if existingDevice != nil && existingDevice.RulesHash != "" && device.RulesHash != "" &&
		existingDevice.RulesHash != device.RulesHash {
		Infof("Device %s rules hash mismatch: stored=%s, incoming=%s - forcing clean sync",
			machineID, existingDevice.RulesHash, device.RulesHash)
		device.NeedsCleanSync = true
	}

	now := time.Now().UTC()
	syncType, syncReason := determineSyncType(existingDevice, &device, now, cleanSyncMaxAge)
	Infof("preflight: setting %s sync for device %s (%s)", syncType, machineID, syncReason)

	// Ruledownload reads PendingSyncType to decide whether to send the full rule
	// set, and postflight uses SyncStartedAt as the new last_sync once the sync
	// completes. last_clean_sync is only recorded by postflight, so a clean sync
	// that fails part way is issued again on the next preflight.
	device.PendingSyncType = syncType
	device.SyncStartedAt = primitive.NewDateTimeFromTime(now)

	var deviceOverrides *Settings
	if existingDevice == nil {
		device.ClientMode = globalSettings.defaultClientMode()
		device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
		err = saveDevice(ctx, client, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to create new device", http.StatusInternalServerError)
			return
		}
	} else {
		deviceOverrides = existingDevice.Settings

		if !validClientMode(existingDevice.ClientMode) {
			Warnf("Invalid ClientMode=%d for device %s in DB, defaulting to LOCKDOWN",
				existingDevice.ClientMode, machineID)
			device.ClientMode = clientModeLockdown
		} else {
			device.ClientMode = existingDevice.ClientMode
		}

		if device.SyncInterval == 0 {
			device.SyncInterval = existingDevice.SyncInterval
		}

		device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())
		device.LastCleanSync = existingDevice.LastCleanSync

		err = saveDevice(ctx, client, &device, machineID)
		if err != nil {
			http.Error(w, "Failed to update device data", http.StatusInternalServerError)
			return
		}
	}

	syncInterval := syncIntervalLockdown
	if device.ClientMode == clientModeMonitor {
		syncInterval = syncIntervalMonitor
	}

	if device.SyncInterval > 0 {
		syncInterval = device.SyncInterval
	}

	Infof(
		"Device %s sync interval: %d seconds (ClientMode=%d, override=%t)",
		machineID,
		syncInterval,
		device.ClientMode,
		device.SyncInterval > 0,
	)

	response := buildPreflightResponse(
		mergeSettings(globalSettings, deviceOverrides),
		device.ClientMode,
		syncType,
		syncInterval,
		machineID,
	)

	Log(SeverityInfo, "Preflight response", map[string]any{
		"machine_id": machineID,
		"response":   protojson.Format(response),
	})

	if err := writeSyncResponse(w, format, response); err != nil {
		Errorf("Failed to write preflight response for %s: %v", machineID, err)
	}
}

// buildPreflightResponse assembles the response from the effective settings
// and the per-sync decisions.
func buildPreflightResponse(settings Settings, clientMode uint32, syncType string, syncInterval uint32, machineID string) *syncv1.PreflightResponse {
	response := syncv1.PreflightResponse_builder{
		ClientMode:              syncv1.ClientMode(clientMode),
		SyncType:                syncv1.SyncType(syncv1.SyncType_value[syncType]).Enum(),
		FullSyncIntervalSeconds: syncInterval,
	}.Build()

	settings.applyTo(response, machineID)
	return response
}

// adminSyncTypes are the clean syncs an admin can request for a device.
var adminSyncTypes = map[string]bool{
	"CLEAN":            true,
	"CLEAN_ALL":        true,
	"CLEAN_RULES":      true,
	"CLEAN_STANDALONE": true,
}

// determineSyncType decides which sync type preflight hands to the client.
// existing is the stored device (nil for a new device) and incoming is the
// device as reported in this preflight. A device whose last completed clean
// sync is older than maxAge gets a CLEAN_ALL. A clean sync requested by an
// admin wins over a client requested one, but not over a server-forced
// CLEAN_ALL. It returns the sync type and a short reason for logging.
func determineSyncType(existing *Device, incoming *Device, now time.Time, maxAge time.Duration) (string, string) {
	switch {
	case existing == nil:
		return "CLEAN_ALL", "device not found"
	case existing.LastCleanSync == 0:
		return "CLEAN_ALL", "no recorded clean sync"
	case now.Sub(existing.LastCleanSync.Time()) > maxAge:
		hours := now.Sub(existing.LastCleanSync.Time()).Hours()
		return "CLEAN_ALL", fmt.Sprintf("last clean sync was %.1f hours ago", hours)
	case existing.NeedsCleanSync:
		return "CLEAN_ALL", "device marked needs_clean_sync"
	case incoming.NeedsCleanSync:
		return "CLEAN_ALL", "incoming device flagged needs_clean_sync"
	case adminSyncTypes[strings.ToUpper(existing.RequestedSyncType)]:
		return strings.ToUpper(existing.RequestedSyncType), "requested by admin"
	case incoming.RequestCleanSync:
		return "CLEAN", "client requested clean sync"
	default:
		return "NORMAL", "incremental"
	}
}

func saveDevice(ctx context.Context, client *mongo.Client, device *Device, machineID string) error {
	collection := client.Database(dbName).Collection("devices")

	device.ID = machineID
	device.LastUpdated = primitive.NewDateTimeFromTime(time.Now())

	updateData, err := bson.Marshal(device)
	if err != nil {
		Errorf("Failed to convert device to BSON: %v", err)
		return fmt.Errorf("failed to convert device to BSON: %w", err)
	}

	var updateMap bson.M
	err = bson.Unmarshal(updateData, &updateMap)
	if err != nil {
		Errorf("Failed to unmarshal BSON: %v", err)
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
		Errorf("Failed to save device data: %v", err)
		return fmt.Errorf("failed to save device data: %w", err)
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

func machineIDFromRequest(r *http.Request, operation string) (string, bool) {
	if machineID := strings.TrimSpace(r.URL.Query().Get("machine_id")); machineID != "" {
		return machineID, true
	}

	return machineIDFromPath(r.URL.Path, operation)
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
