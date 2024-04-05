package ruledownload

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

var client *firestore.Client

func getGlobalRules(ctx context.Context) ([]*Rule, error) {

	query := client.Collection(os.Getenv("DB_PREFIX")+"_rules").Where("scope", "==", "global")
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

func paginateRules(rules []*Rule, r *http.Request) ([]*Rule, string) {
	cursor := r.URL.Query().Get("cursor")
	startIndex := 0

	// Find the start index based on the provided cursor
	for i, rule := range rules {
		if rule.Identifier == cursor {
			startIndex = i + 1
			break
		}
	}

	// Paginate rules based on the start index and batchSize
	endIndex := startIndex + batchSize
	if endIndex > len(rules) {
		endIndex = len(rules)
	}
	paginatedRules := rules[startIndex:endIndex]

	// Determine the cursor for the next page
	nextCursor := ""
	if endIndex < len(rules) {
		nextCursor = time.Now().Format(time.RFC3339)
	}

	return paginatedRules, nextCursor
}

func ruledownloadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var reponse Response

	if err := json.NewDecoder(r.Body).Decode(&reponse); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Logic to handle machine_id and cursor

	rules, err := getGlobalRules(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get global rules: %v", err), http.StatusInternalServerError)
		return
	}

	// Paginate rules
	paginatedRules, cursor := paginateRules(rules, r)

	responseData := map[string]interface{}{
		"rules": paginatedRules,
	}
	if cursor != "" {
		responseData["cursor"] = cursor
	}

	responseJSON, err := json.Marshal(responseData)
	if err != nil {
		http.Error(w, "Failed to encode response JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
