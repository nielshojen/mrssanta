package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
)

func getAllRules(ctx context.Context) ([]*Rule, error) {

	query := client.Collection(os.Getenv("DB_PREFIX") + "_rules")
	iter := query.Documents(ctx)

	var rules []*Rule
	for {
		var r Rule
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		err = doc.DataTo(&r)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, nil
}

func getRuleByID(ctx context.Context, ruleID string) ([]*Rule, error) {

	query := client.Collection(os.Getenv("DB_PREFIX")+"_rules").Where("name", "==", ruleID)
	iter := query.Documents(ctx)

	var rules []*Rule
	for {
		var r Rule
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		err = doc.DataTo(&r)
		if err != nil {
			return nil, err
		}
		rules = append(rules, &r)
	}
	return rules, nil
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
		err := saveRule(ctx, client, w, rule)
		if err != nil {
			log.Printf("Failed to save rule %s: %v", rule.Identifier, err)
			return nil, fmt.Errorf("failed to save rule %s: %w", rule.Identifier, err)
		}
	}

	// Return the successfully saved rules
	return rules, nil
}
