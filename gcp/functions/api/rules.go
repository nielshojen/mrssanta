package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rs/zerolog/log"
)

func getAllRules(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to retrieve rules: %v", err)
		return nil, fmt.Errorf("failed to retrieve rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode document: %v", err)
			continue
		}

		if creationTime, ok := doc["CreationTime"].(primitive.DateTime); ok {
			doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
		}
		if lastUpdated, ok := doc["LastUpdated"].(primitive.DateTime); ok {
			doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		rules = append(rules, doc)
	}

	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return rules, nil
}

func getRuleByID(ctx context.Context, ruleID string) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	var rules []map[string]interface{}
	var doc bson.M
	err := collection.FindOne(ctx, bson.M{"_id": ruleID}).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("Rule %s not found in MongoDB", ruleID)
			return nil, nil // Return nil to indicate no result
		}
		log.Printf("Error fetching rule %s: %v", ruleID, err)
		return nil, fmt.Errorf("failed to fetch rule: %w", err)
	}

	if creationTime, ok := doc["CreationTime"].(primitive.DateTime); ok {
		doc["CreationTime"] = creationTime.Time().Format(time.RFC3339)
	}

	if lastUpdated, ok := doc["LastUpdated"].(primitive.DateTime); ok {
		doc["LastUpdated"] = lastUpdated.Time().Format(time.RFC3339)
	}

	rules = append(rules, doc)

	return rules, nil
}

func createRules(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Rule, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	var rules []*Rule
	err := json.Unmarshal(jsonData, &rules)
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	for _, rule := range rules {
		if rule.Identifier == "" {
			return nil, fmt.Errorf("rule is missing identifier")
		}
		if rule.Policy == "" {
			return nil, fmt.Errorf("rule with identifier %s is missing policy", rule.Identifier)
		}
		if rule.RuleType == "" {
			return nil, fmt.Errorf("rule with identifier %s is missing rule type", rule.Identifier)
		}

		now := primitive.NewDateTimeFromTime(time.Now())

		rule.CreationTime = now
		rule.LastUpdated = now

		if rule.ID == "" {
			rule.ID = primitive.NewObjectID().Hex()
		}
	}

	for _, rule := range rules {
		now := primitive.NewDateTimeFromTime(time.Now())

		updateFields := bson.M{}
		jsonData, _ := json.Marshal(rule)
		json.Unmarshal(jsonData, &updateFields)
		updateFields["creation_time"] = rule.CreationTime
		updateFields["last_updated"] = now

		filter := bson.M{"_id": rule.Identifier}
		update := bson.M{"$set": updateFields}
		opts := options.Update().SetUpsert(true)

		_, err = collection.UpdateOne(ctx, filter, update, opts)
		if err != nil {
			log.Printf("Failed to update rule %s: %v", rule.Identifier, err)
			return nil, fmt.Errorf("failed to update rule %s: %w", rule.Identifier, err)
		}
	}

	return rules, nil
}

func assignRule(ctx context.Context, w http.ResponseWriter, jsonData []byte, ruleID string) error {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	var device Device
	err := json.Unmarshal(jsonData, &device)
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	filter := bson.M{"_id": ruleID}
	update := bson.M{
		"$addToSet": bson.M{"assigned": device.Identifier},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating rule %s: %v", ruleID, err)
		return fmt.Errorf("failed to update rule: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("rule %s not found in MongoDB", ruleID)
	}

	log.Printf("Successfully assigned device %s to rule %s", device.Identifier, ruleID)
	return nil
}
