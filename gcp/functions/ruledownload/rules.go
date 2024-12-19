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

	"google.golang.org/api/iterator"
)

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
