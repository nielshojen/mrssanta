package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getAllEvents(ctx context.Context) ([]map[string]interface{}, error) {
	// Query Firestore collection
	query := client.Collection(os.Getenv("DB_PREFIX") + "_events")
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		log.Printf("Failed to retrieve events: %v", err)
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}

	// Convert Firestore documents to a generic JSON-friendly format
	var events []map[string]interface{}
	for _, doc := range docs {
		data := doc.Data() // Retrieve document as a map

		// Convert Firestore timestamps to RFC3339 formatted strings
		if creationTime, ok := data["CreationTime"].(time.Time); ok {
			data["CreationTime"] = creationTime.Format(time.RFC3339)
		}
		if lastUpdated, ok := data["LastUpdated"].(time.Time); ok {
			data["LastUpdated"] = lastUpdated.Format(time.RFC3339)
		}

		events = append(events, data)
	}

	return events, nil
}

func getEventByID(ctx context.Context, machineID string) (*Event, error) {
	// Reference Firestore document directly by ID
	docRef := client.Collection(os.Getenv("DB_PREFIX") + "_events").Doc(machineID)

	// Get document snapshot
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		// Handle case where document does not exist
		if status.Code(err) == codes.NotFound {
			log.Printf("Event %s not found in Firestore", machineID)
			return nil, nil // Return nil instead of an empty slice
		}
		log.Printf("Error fetching event %s: %v", machineID, err)
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	// Unmarshal document into Rule struct
	var event Event
	if err := docSnap.DataTo(&event); err != nil {
		log.Printf("Failed to decode rule document: %v", err)
		return nil, fmt.Errorf("failed to decode event: %w", err)
	}

	return &event, nil
}
