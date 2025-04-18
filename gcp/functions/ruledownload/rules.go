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
	cursor, err := collection.Find(ctx, bson.M{"scope": "global"})
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
	return rules, cursor.Err()
}

func getMunkiRules(ctx context.Context, client *mongo.Client, ID string) ([]*Rule, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")
	cursor, err := collection.Find(ctx, bson.M{"scope": "managedapp", "assigned": ID})
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
	return rules, cursor.Err()
}

func getMachineRules(ctx context.Context, client *mongo.Client, ID string) ([]*Rule, error) {
	collection := client.Database(os.Getenv("MONGO_DB")).Collection("rules")
	cursor, err := collection.Find(ctx, bson.M{"scope": "machine", "assigned": ID})
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
	return rules, cursor.Err()
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

func paginateRules(rules []*Rule, cursorStr string) ([]*Rule, string) {
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
	var request Request

	apiKey := r.Header.Get("X-API-Key")
	if apiKey != validAPIKey {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
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
		log.Printf("Cannot read request body: %v\n", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	decompressedData, err := decompressZlib(reqBody)
	if err != nil {
		log.Printf("Failed to decompress body: %v", err)
		http.Error(w, "Failed to decompress request body", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(decompressedData, &request); err != nil {
		log.Printf("Failed to decode JSON: %v", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Incoming cursor: %s", request.Cursor)

	var rules []*Rule

	if globalrules, err := getGlobalRules(ctx, client); err == nil {
		rules = append(rules, globalrules...)
	} else {
		http.Error(w, fmt.Sprintf("Failed to get global rules: %v", err), http.StatusInternalServerError)
		return
	}

	if managedapprules, err := getMunkiRules(ctx, client, machineID); err == nil {
		rules = append(rules, managedapprules...)
	} else {
		http.Error(w, fmt.Sprintf("Failed to get managedapp rules: %v", err), http.StatusInternalServerError)
		return
	}

	if machinerules, err := getMachineRules(ctx, client, machineID); err == nil {
		rules = append(rules, machinerules...)
	} else {
		http.Error(w, fmt.Sprintf("Failed to get machine rules: %v", err), http.StatusInternalServerError)
		return
	}

	paginatedRules, cursor := paginateRules(rules, request.Cursor)

	log.Printf("Outgoing cursor: %s", cursor)

	response := Response{
		Rules:  paginatedRules,
		Cursor: cursor,
	}

	responseJSON, err := json.Marshal(response)
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
