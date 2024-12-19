package postflight

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

func postflight(w http.ResponseWriter, r *http.Request) {
	var request Request

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

	if err := json.Unmarshal(decompressedData, &request); err != nil {
		log.Printf("Failed to decode JSON from decompressed data: %v", err)
		http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
		return
	}

	if request.RulesReceived == 0 && request.RulesProcessed == 0 {
		http.Error(w, "Request body is empty or does not contain required fields", http.StatusBadRequest)
		return
	}

	machineID := r.URL.Query().Get("machine_id")

	response := Response{
		ID:             machineID,
		RulesReceived:  request.RulesReceived,
		RulesProcessed: request.RulesProcessed,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to marshal response to JSON", http.StatusInternalServerError)
		return
	}

	fmt.Println(string(responseJSON))

	// Set response headers and write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
