package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var errRuleNotFound = errors.New("rule not found")

func getAllRules(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(dbName).Collection("rules")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		Errorf("Failed to retrieve rules: %v", err)
		return nil, fmt.Errorf("failed to retrieve rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			Warnf("Failed to decode document: %v", err)
			continue
		}

		if creationTime, ok := doc["CreationTime"].(bson.DateTime); ok {
			doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
		}
		if lastUpdated, ok := doc["LastUpdated"].(bson.DateTime); ok {
			doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		rules = append(rules, doc)
	}

	if err := cursor.Err(); err != nil {
		Errorf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return rules, nil
}

func getRuleByID(ctx context.Context, ruleID string) ([]map[string]interface{}, error) {
	collection := client.Database(dbName).Collection("rules")

	var rules []map[string]interface{}
	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"_id": ruleID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			Infof("Rule %s not found in MongoDB", ruleID)
			return nil, nil
		}
		Errorf("Error fetching rule %s: %v", ruleID, err)
		return nil, fmt.Errorf("failed to fetch rule: %w", err)
	}

	if creationTime, ok := doc["CreationTime"].(bson.DateTime); ok {
		doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
	}

	if lastUpdated, ok := doc["LastUpdated"].(bson.DateTime); ok {
		doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
	}

	rules = append(rules, doc)
	return rules, nil
}

func createRules(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Rule, error) {
	collection := client.Database(dbName).Collection("rules")

	var rules []*Rule
	err := json.Unmarshal(jsonData, &rules)
	if err != nil {
		Warnf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	for _, rule := range rules {
		if err := normalizeRule(rule); err != nil {
			return nil, err
		}

		now := bson.NewDateTimeFromTime(time.Now())
		rule.CreationTime = now
		rule.LastUpdated = now
		if rule.ID == "" {
			rule.ID = bson.NewObjectID().Hex()
		}
	}

	for _, rule := range rules {
		now := bson.NewDateTimeFromTime(time.Now())

		updateFields := bson.M{}
		marshaled, _ := json.Marshal(rule)
		_ = json.Unmarshal(marshaled, &updateFields)
		updateFields["creation_time"] = rule.CreationTime
		updateFields["last_updated"] = now

		filter := bson.M{"_id": rule.Identifier}
		update := bson.M{
			"$set": updateFields,
			// Re-creating a deleted rule must clear its tombstone.
			"$unset": bson.M{"deleted": "", "deleted_at": ""},
		}
		opts := options.UpdateOne().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			Errorf("Failed to update rule %s: %v", rule.Identifier, err)
			return nil, fmt.Errorf("failed to update rule %s: %w", rule.Identifier, err)
		}
	}

	return rules, nil
}

func assignRule(ctx context.Context, w http.ResponseWriter, jsonData []byte, ruleID string) error {
	collection := client.Database(dbName).Collection("rules")

	var device Device
	if err := json.Unmarshal(jsonData, &device); err != nil {
		Warnf("Failed to decode JSON: %v", err)
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	now := bson.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": ruleID}
	update := bson.M{
		"$addToSet": bson.M{"assigned": device.Identifier},
		"$pull":     bson.M{"removed_from": device.Identifier},
		"$set":      bson.M{"last_updated": now},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		Errorf("Error updating rule %s: %v", ruleID, err)
		return fmt.Errorf("failed to update rule: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("rule %s not found in MongoDB", ruleID)
	}

	Infof("Successfully assigned device %s to rule %s (LastUpdated: %v)",
		device.Identifier, ruleID, now.Time())

	return nil
}

func unassignRule(ctx context.Context, w http.ResponseWriter, jsonData []byte, ruleID string) error {
	collection := client.Database(dbName).Collection("rules")

	var device Device
	if err := json.Unmarshal(jsonData, &device); err != nil {
		Warnf("Failed to decode JSON: %v", err)
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	now := bson.NewDateTimeFromTime(time.Now())

	filter := bson.M{"_id": ruleID}
	update := unassignUpdate(device.Identifier, now)

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		Errorf("Error updating rule %s: %v", ruleID, err)
		return fmt.Errorf("failed to update rule: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("rule %s not found in MongoDB", ruleID)
	}

	Infof("Successfully unassigned device %s from rule %s (LastUpdated: %v)",
		device.Identifier, ruleID, now.Time())

	return nil
}

func unassignManagedAppRules(ctx context.Context, w http.ResponseWriter, jsonData []byte) error {
	collection := client.Database(dbName).Collection("rules")

	var device Device
	if err := json.Unmarshal(jsonData, &device); err != nil {
		Warnf("Failed to decode JSON: %v", err)
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	if device.Identifier == "" {
		Warnf("Device identifier is empty, skipping managedapp unassignment")
		return fmt.Errorf("device identifier is empty")
	}

	now := bson.NewDateTimeFromTime(time.Now())

	filter := bson.M{
		"scope":    "managedapp",
		"assigned": device.Identifier,
	}

	update := unassignUpdate(device.Identifier, now)

	result, err := collection.UpdateMany(ctx, filter, update)
	if err != nil {
		Errorf("Error unassigning device %s from managedapp rules: %v", device.Identifier, err)
		return fmt.Errorf("failed to update managedapp rules: %w", err)
	}

	if result.MatchedCount == 0 {
		Infof("No managedapp rules found for device %s", device.Identifier)
	} else {
		Infof("Unassigned device %s from %d managedapp rule(s) (LastUpdated: %v)",
			device.Identifier, result.ModifiedCount, now.Time())
	}

	return nil
}

// unassignUpdate removes a machine from a rule and records it in
// removed_from, so ruledownload sends that machine a REMOVE on its next
// incremental sync instead of needing a clean sync.
func unassignUpdate(machineID string, now bson.DateTime) bson.M {
	return bson.M{
		"$pull":     bson.M{"assigned": machineID},
		"$addToSet": bson.M{"removed_from": machineID},
		"$set":      bson.M{"last_updated": now},
	}
}

// deleteRuleUpdate turns a rule into a tombstone. The document is kept so
// ruledownload can send REMOVE to machines that still have the rule; full
// syncs skip tombstones.
func deleteRuleUpdate(now bson.DateTime) bson.M {
	return bson.M{
		"$set": bson.M{
			"deleted":      true,
			"deleted_at":   now,
			"last_updated": now,
		},
	}
}

func deleteRule(ctx context.Context, ruleID string) error {
	collection := client.Database(dbName).Collection("rules")

	now := bson.NewDateTimeFromTime(time.Now())
	result, err := collection.UpdateOne(ctx, bson.M{"_id": ruleID}, deleteRuleUpdate(now))
	if err != nil {
		Errorf("Error deleting rule %s: %v", ruleID, err)
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	if result.MatchedCount == 0 {
		return errRuleNotFound
	}

	Infof("Deleted rule %s (tombstoned at %v)", ruleID, now.Time())
	return nil
}
