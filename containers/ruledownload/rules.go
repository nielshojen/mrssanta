package ruledownload

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getBatchSize() int64 {
	v := strings.TrimSpace(os.Getenv("BATCH_SIZE"))
	if v == "" {
		return 500
	}

	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		Warnf("Invalid BATCH_SIZE=%q, defaulting to 500", v)
		return 500
	}

	return int64(n)
}

func encodeCursor(cursor CursorMetadata) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(data)
}

func decodeCursor(cursorStr string) (CursorMetadata, error) {
	data, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return CursorMetadata{}, err
	}

	var cursor CursorMetadata
	if err := json.Unmarshal(data, &cursor); err != nil {
		return CursorMetadata{}, err
	}

	return cursor, nil
}

func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if validAPIKey == "" {
		Errorf("API_KEY is not configured")
		writeJSONError(w, http.StatusInternalServerError, "Server configuration error")
		return
	}

	if r.Header.Get("X-API-Key") != validAPIKey {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	machineID, _ := machineIDFromRequest(r, "ruledownload", "")

	defer r.Body.Close()

	var request syncv1.RuleDownloadRequest
	format, err := readSyncRequest(r, &request)
	if err != nil {
		Warnf("Failed to decode ruledownload request for machine_id=%s: %v", machineID, err)
		status := http.StatusBadRequest
		if errors.Is(err, errUnsupportedContentType) {
			status = http.StatusUnsupportedMediaType
		}
		writeJSONError(w, status, "Failed to decode request")
		return
	}

	if machineID == "" {
		var ok bool
		machineID, ok = machineIDFromRequest(r, "ruledownload", request.GetMachineId())
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "Machine ID is missing in the request URL")
			return
		}
	}

	if !checkXSRF(w, r, machineID) {
		return
	}

	dbName := strings.TrimSpace(os.Getenv("MONGO_DB"))
	if dbName == "" {
		Errorf("MONGO_DB is not configured")
		writeJSONError(w, http.StatusInternalServerError, "Server configuration error")
		return
	}

	batchSize := getBatchSize()

	devicesColl := client.Database(dbName).Collection("devices")
	rulesColl := client.Database(dbName).Collection("rules")

	var device Device
	err = devicesColl.FindOne(ctx, bson.M{"machine_id": machineID}).Decode(&device)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			Warnf("Device not found for machine_id=%s", machineID)
			writeJSONError(w, http.StatusNotFound, "Device not found")
			return
		}

		Errorf("Failed to load device for machine_id=%s: %v", machineID, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to load device")
		return
	}

	var lastSyncTime time.Time
	if device.LastSync != 0 {
		lastSyncTime = device.LastSync.Time().UTC()
	}

	needsFullSync, syncReason := fullSyncReason(&device)

	totalRuleCount, err := rulesColl.CountDocuments(ctx, activeRulesFilter(machineID))
	if err != nil {
		Errorf("Failed to count rules for machine_id=%s: %v", machineID, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to count rules")
		return
	}

	var cursorMeta *CursorMetadata
	if request.GetCursor() != "" {
		decoded, err := decodeCursor(request.GetCursor())
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid cursor")
			return
		}
		cursorMeta = &decoded
	}

	pageFilter := ruleDownloadFilter(machineID, needsFullSync, lastSyncTime, cursorMeta)

	findOpts := options.Find().
		SetSort(bson.D{
			{Key: "last_updated", Value: 1},
			{Key: "identifier", Value: 1},
		}).
		SetLimit(batchSize + 1)

	cursor, err := rulesColl.Find(ctx, pageFilter, findOpts)
	if err != nil {
		Errorf("Failed to fetch rules for machine_id=%s: %v", machineID, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch rules")
		return
	}
	defer cursor.Close(ctx)

	rules := make([]*Rule, 0, batchSize+1)
	for cursor.Next(ctx) {
		var rule Rule
		if err := cursor.Decode(&rule); err != nil {
			Errorf("Failed to decode rule for machine_id=%s: %v", machineID, err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to decode rule")
			return
		}
		rules = append(rules, &rule)
	}

	if err := cursor.Err(); err != nil {
		Errorf("Cursor error for machine_id=%s: %v", machineID, err)
		writeJSONError(w, http.StatusInternalServerError, "Failed while reading rules")
		return
	}

	nextCursor := ""
	if int64(len(rules)) > batchSize {
		last := rules[batchSize-1]
		nextCursor = encodeCursor(CursorMetadata{
			Identifier:  last.Identifier,
			LastUpdated: last.LastUpdated.Time().UTC().UnixMilli(),
		})
		rules = rules[:batchSize]
	}

	protoRules := make([]*syncv1.Rule, 0, len(rules))
	removals := 0
	for _, rule := range rules {
		protoRule, err := rule.toProto(machineID)
		if err != nil {
			Warnf("Skipping rule %s for machine_id=%s: %v", rule.Identifier, machineID, err)
			continue
		}
		if protoRule.GetPolicy() == syncv1.Policy_REMOVE {
			removals++
		}
		protoRules = append(protoRules, protoRule)
	}

	// last_sync is deliberately not touched here: it must stay constant across
	// every page of this sync so the incremental filter and the cursor agree.
	// Postflight advances it once the client has applied the rules.
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"rule_count":   uint32(totalRuleCount),
			"last_updated": now,
		},
	}

	if _, err := devicesColl.UpdateOne(ctx, bson.M{"machine_id": machineID}, update); err != nil {
		Errorf("Failed to update device metadata for machine_id=%s: %v", machineID, err)
	}

	if len(protoRules) > 0 {
		lastSync := ""
		if !lastSyncTime.IsZero() {
			lastSync = lastSyncTime.Format(time.RFC3339Nano)
		}

		Log(SeverityInfo, "Device downloading rules", map[string]any{
			"machine_id":             machineID,
			"hostname":               device.Hostname,
			"sync_reason":            syncReason,
			"needs_full_sync":        needsFullSync,
			"needs_clean_sync":       device.NeedsCleanSync,
			"request_clean_sync":     device.RequestCleanSync,
			"pending_sync_type":      device.PendingSyncType,
			"last_sync":              lastSync,
			"request_cursor_present": request.GetCursor() != "",
			"next_cursor_present":    nextCursor != "",
			"rules_returned":         len(protoRules),
			"removals_returned":      removals,
			"total_rule_count":       totalRuleCount,
			"batch_size":             batchSize,
		})
	}

	response := syncv1.RuleDownloadResponse_builder{
		Rules:  protoRules,
		Cursor: nextCursor,
	}.Build()

	if err := writeSyncResponse(w, format, response); err != nil {
		Errorf("Failed to write ruledownload response for machine_id=%s: %v", machineID, err)
	}
}

// applicableRulesFilter matches every rule that targets the machine: global
// rules and managedapp/machine rules it is assigned to.
func applicableRulesFilter(machineID string) bson.M {
	return bson.M{
		"$or": []bson.M{
			{"scope": "global"},
			{"scope": "managedapp", "assigned": machineID},
			{"scope": "machine", "assigned": machineID},
		},
	}
}

// activeRulesFilter matches the rules the machine should currently have.
func activeRulesFilter(machineID string) bson.M {
	return bson.M{
		"$and": []bson.M{
			applicableRulesFilter(machineID),
			{"deleted": bson.M{"$ne": true}},
		},
	}
}

// ruleDownloadFilter builds the query for one page of rules.
//
// A full sync returns only active rules, since the client has just deleted
// everything. An incremental sync returns everything changed since last_sync,
// including tombstones (deleted rules and rules the machine was unassigned
// from) so the client can be told to REMOVE them.
func ruleDownloadFilter(machineID string, fullSync bool, lastSync time.Time, cursor *CursorMetadata) bson.M {
	andFilters := make([]bson.M, 0, 3)

	if fullSync {
		andFilters = append(andFilters, activeRulesFilter(machineID))
	} else {
		andFilters = append(andFilters,
			bson.M{"$or": []bson.M{
				applicableRulesFilter(machineID),
				{"removed_from": machineID},
			}},
			bson.M{"last_updated": bson.M{"$gt": lastSync}},
		)
	}

	if cursor != nil {
		cursorTime := time.UnixMilli(cursor.LastUpdated).UTC()
		andFilters = append(andFilters, bson.M{
			"$or": []bson.M{
				{"last_updated": bson.M{"$gt": cursorTime}},
				{
					"$and": []bson.M{
						{"last_updated": cursorTime},
						{"identifier": bson.M{"$gt": cursor.Identifier}},
					},
				},
			},
		})
	}

	return bson.M{"$and": andFilters}
}

// isActiveFor reports whether the rule currently applies to the machine.
func (r *Rule) isActiveFor(machineID string) bool {
	if r.Deleted {
		return false
	}
	if r.Scope == "global" {
		return true
	}
	return slices.Contains(r.Assigned, machineID)
}

// toProto converts a stored rule to the protocol rule sent to machineID. A
// rule that no longer applies to the machine becomes a REMOVE. Only protocol
// fields are copied, so internal fields such as the list of assigned machines
// never reach the client.
func (r *Rule) toProto(machineID string) (*syncv1.Rule, error) {
	ruleType, ok := syncv1.RuleType_value[strings.ToUpper(strings.TrimSpace(r.RuleType))]
	if !ok || ruleType == 0 {
		return nil, errors.New("unknown rule_type " + strconv.Quote(r.RuleType))
	}

	if !r.isActiveFor(machineID) {
		return syncv1.Rule_builder{
			Identifier: r.Identifier,
			Policy:     syncv1.Policy_REMOVE,
			RuleType:   syncv1.RuleType(ruleType),
		}.Build(), nil
	}

	policy, ok := syncv1.Policy_value[strings.ToUpper(strings.TrimSpace(r.Policy))]
	if !ok || policy == 0 {
		return nil, errors.New("unknown policy " + strconv.Quote(r.Policy))
	}

	if syncv1.Policy(policy) == syncv1.Policy_CEL && strings.TrimSpace(r.CELExpr) == "" {
		return nil, errors.New("CEL policy without cel_expr")
	}

	rule := syncv1.Rule_builder{
		Identifier:             r.Identifier,
		Policy:                 syncv1.Policy(policy),
		RuleType:               syncv1.RuleType(ruleType),
		CustomMsg:              r.CustomMessage,
		CustomUrl:              r.CustomURL,
		NotificationAppName:    r.NotificationAppName,
		EventDetailButtonLabel: r.EventDetailButtonLabel,
		FileBundleHash:         r.FileBundleHash,
	}.Build()

	if syncv1.Policy(policy) == syncv1.Policy_CEL {
		rule.SetCelExpr(r.CELExpr)
	}

	if count, err := strconv.ParseUint(r.FileBundleBinaryCount, 10, 32); err == nil {
		rule.SetFileBundleBinaryCount(uint32(count))
	}

	return rule, nil
}

// fullSyncReason reports whether the device must receive its complete rule set
// rather than only rules changed since last_sync, and why. A CLEAN, CLEAN_ALL
// or CLEAN_RULES sync makes the client delete its rules, so anything less than
// the full set would leave it missing rules.
func fullSyncReason(device *Device) (bool, string) {
	switch {
	case device.PendingSyncType == "CLEAN" || device.PendingSyncType == "CLEAN_ALL" || device.PendingSyncType == "CLEAN_RULES":
		return true, "pending_sync_type=" + device.PendingSyncType
	case device.NeedsCleanSync:
		return true, "needs_clean_sync"
	case device.RequestCleanSync:
		return true, "request_clean_sync"
	case device.LastSync == 0:
		return true, "first_sync"
	default:
		return false, "incremental"
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	responseJSON, err := json.Marshal(payload)
	if err != nil {
		Errorf("Failed to encode response JSON: %v", err)
		http.Error(w, `{"error":"Failed to encode response JSON"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if _, err := w.Write(responseJSON); err != nil {
		Errorf("Failed to write response: %v", err)
	}
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
