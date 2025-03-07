package ruledownload

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getGlobalRules(ctx context.Context, client *mongo.Client) ([]*Rule, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// MongoDB query equivalent to Firestore's `.Where("Scope", "==", "global")`
	filter := bson.M{"scope": "global"}

	// Find all matching rules
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB results to `Rule` struct
	var rules []*Rule
	for cursor.Next(ctx) {
		var r Rule
		if err := cursor.Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode rule: %v", err)
		}
		rules = append(rules, &r)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return rules, nil
}

func getMunkiRules(ctx context.Context, client *mongo.Client, ID string) ([]*Rule, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// MongoDB query equivalent to Firestore's `.Where("Scope", "==", "munki").Where("Assigned", "array-contains", ID)`
	filter := bson.M{
		"scope":    "munki",
		"assigned": ID, // MongoDB automatically handles `array-contains`
	}

	// Find all matching rules
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB results to `Rule` struct
	var rules []*Rule
	for cursor.Next(ctx) {
		var r Rule
		if err := cursor.Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode rule: %v", err)
		}
		rules = append(rules, &r)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	return rules, nil
}

func getMachineRules(ctx context.Context, client *mongo.Client, ID string) ([]*Rule, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("rules")

	// MongoDB query equivalent to Firestore's `.Where("Scope", "==", "machine").Where("Assigned", "array-contains", ID)`
	filter := bson.M{
		"scope":    "machine",
		"assigned": ID, // MongoDB automatically handles `array-contains`
	}

	// Find all matching rules
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB results to `Rule` struct
	var rules []*Rule
	for cursor.Next(ctx) {
		var r Rule
		if err := cursor.Decode(&r); err != nil {
			return nil, fmt.Errorf("failed to decode rule: %v", err)
		}
		rules = append(rules, &r)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
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
	var rules []*Rule

	// Extract the API key from the header
	apiKey := r.Header.Get("X-API-Key")

	// Validate the API key
	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

	// Extract machine ID from the query parameters
	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		http.Error(w, "Machine ID is missing in the request URL", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "Content-Type header must be application/json", http.StatusBadRequest)
		return
	}

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot parse request body: %v\n", err)
	}
	defer r.Body.Close()

	// Decompress the data
	decompressedData, err := decompressZlib(reqBody)
	if err != nil {
		log.Printf("Failed to decompress body: %v", err)
		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(decompressedData, &reponse); err != nil {
		log.Printf("Failed to decode JSON from decompressed data: %v", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	// Logic to handle machine_id and cursor
	globalrules, err := getGlobalRules(ctx, client)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get global rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, globalrules...)

	munkirules, err := getMunkiRules(ctx, client, machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get munki rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, munkirules...)

	machinerules, err := getMachineRules(ctx, client, machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, machinerules...)

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

// decompressZlib decompresses zlib-compressed data
func decompressZlib(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib reader: %w", err)
	}
	defer reader.Close()

	var decompressedData bytes.Buffer
	if _, err := io.Copy(&decompressedData, reader); err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}
	return decompressedData.Bytes(), nil
}
