package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getAllRules(ctx context.Context) ([]map[string]interface{}, error) {
	// Query Firestore collection
	query := client.Collection(os.Getenv("DB_PREFIX") + "_rules")
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Failed to retrieve rules: %v", err)
		return nil, fmt.Errorf("failed to retrieve rules: %w", err)
	}

	// Convert Firestore documents to a generic JSON-friendly format
	var rules []map[string]interface{}
	for _, doc := range docs {
		data := doc.Data() // Retrieve document as a map

		// Convert Firestore timestamps to RFC3339 formatted strings
		if creationTime, ok := data["CreationTime"].(time.Time); ok {
			data["CreationTime"] = creationTime.Format(time.RFC3339)
		}
		if lastUpdated, ok := data["LastUpdated"].(time.Time); ok {
			data["LastUpdated"] = lastUpdated.Format(time.RFC3339)
		}

		rules = append(rules, data)
	}

	return rules, nil
}

func getRuleByID(ctx context.Context, ruleID string) (*Rule, error) {
	// Reference Firestore document directly by ID
	docRef := client.Collection(os.Getenv("DB_PREFIX") + "_rules").Doc(ruleID)

	// Get document snapshot
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		// Handle case where document does not exist
		if status.Code(err) == codes.NotFound {
			log.Printf("Rule %s not found in Firestore", ruleID)
			return nil, nil // Return nil instead of an empty slice
		}
		log.Printf("Error fetching rule %s: %v", ruleID, err)
		return nil, fmt.Errorf("failed to fetch rule: %w", err)
	}

	// Unmarshal document into Rule struct
	var rule Rule
	if err := docSnap.DataTo(&rule); err != nil {
		log.Printf("Failed to decode rule document: %v", err)
		return nil, fmt.Errorf("failed to decode rule: %w", err)
	}

	return &rule, nil
}

func createRules(ctx context.Context, w http.ResponseWriter, jsonData []byte) ([]*Rule, error) {
	// Define a slice to hold the rules
	var rules []*Rule

	// Unmarshal the JSON data into the slice
	err := json.Unmarshal(jsonData, &rules)
	if err != nil {
		// Log and return an error if JSON parsing fails
		log.Printf("Failed to decode JSON: %v", err)
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate each rule
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
	}

	// Save each rule to Firestore
	for _, rule := range rules {
		saveRule(ctx, client, w, rule)
	}

	// Return the successfully saved rules
	return rules, nil
}
