package ruledownload

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func getGlobalRules(ctx context.Context, client *mongo.Client) ([]*Rule, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	filter := bson.M{"scope": "global"}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

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
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	filter := bson.M{
		"scope":    "managedapp",
		"assigned": ID,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

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
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")

	filter := bson.M{
		"scope":    "machine",
		"assigned": ID,
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch rules: %v", err)
	}
	defer cursor.Close(ctx)

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

func encodeCursor(cursor CursorMetadata) string {
	data, _ := json.Marshal(cursor)
	return base64.StdEncoding.EncodeToString(data)
}

func decodeCursor(cursorStr string) (CursorMetadata, error) {
	data, err := base64.StdEncoding.DecodeString(cursorStr)
	if err != nil {
		return CursorMetadata{}, err
	}
	var cursor CursorMetadata
	err = json.Unmarshal(data, &cursor)
	return cursor, err
}

func paginateRules(rules []*Rule, r *http.Request) ([]*Rule, string) {
	cursorStr := r.URL.Query().Get("cursor")
	startIndex := 0

	if cursorStr != "" {
		cursor, err := decodeCursor(cursorStr)
		if err == nil {
			for i, rule := range rules {
				if rule.Identifier == cursor.Identifier {
					startIndex = i + 1
					break
				}
			}
		}
	}

	endIndex := startIndex + batchSize
	if endIndex > len(rules) {
		endIndex = len(rules)
	}

	paginatedRules := rules[startIndex:endIndex]

	nextCursor := ""
	if endIndex < len(rules) {
		lastRule := rules[endIndex-1]
		nextCursor = encodeCursor(CursorMetadata{Identifier: lastRule.Identifier})
	}

	return paginatedRules, nextCursor
}

func ruledownloadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	var reponse Response
	var rules []*Rule

	apiKey := r.Header.Get("X-API-Key")

	if apiKey != validAPIKey {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Unauthorized"}`)
		return
	}

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

	reqBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Cannot parse request body: %v\n", err)
	}
	defer r.Body.Close()

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

	globalrules, err := getGlobalRules(ctx, client)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get global rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, globalrules...)

	managedapprules, err := getMunkiRules(ctx, client, machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get managedapp rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, managedapprules...)

	machinerules, err := getMachineRules(ctx, client, machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine rules: %v", err), http.StatusInternalServerError)
		return
	}
	rules = append(rules, machinerules...)

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
