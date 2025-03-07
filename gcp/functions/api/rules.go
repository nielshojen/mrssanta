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
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// Find all documents in the collection
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to retrieve rules: %v", err)
		return nil, fmt.Errorf("failed to retrieve rules: %w", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB documents to a generic JSON-friendly format
	var rules []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode document: %v", err)
			continue
		}

		// Convert MongoDB timestamps to RFC3339 formatted strings
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
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// Fetch document by ID into a flexible `bson.M`
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

	// Convert MongoDB timestamps (primitive.DateTime) to RFC3339 formatted strings
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
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// Decode JSON into slice of Rule structs
	var rules []*Rule
	err := json.Unmarshal(jsonData, &rules)
	if err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate rules
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

		// Set timestamps
		now := primitive.NewDateTimeFromTime(time.Now())

		rule.CreationTime = now
		rule.LastUpdated = now

		// Set MongoDB ID if not already set
		if rule.ID == "" {
			rule.ID = primitive.NewObjectID().Hex()
		}
	}

	// Update each rule in MongoDB
	for _, rule := range rules {
		// Set LastUpdated timestamp
		now := primitive.NewDateTimeFromTime(time.Now())

		// Convert rule to bson.M to dynamically update only provided fields
		updateFields := bson.M{}
		jsonData, _ := json.Marshal(rule)       // Convert struct to JSON
		json.Unmarshal(jsonData, &updateFields) // Convert JSON to bson.M

		updateFields["last_updated"] = now // Always update LastUpdated

		// Update MongoDB document without removing missing fields
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
