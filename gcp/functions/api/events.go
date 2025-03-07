package api

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func getAllEvents(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("events")

	// Find all documents
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		log.Printf("Failed to retrieve events: %v", err)
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}
	defer cursor.Close(ctx)

	// Convert MongoDB documents to a JSON-friendly format
	var events []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			log.Printf("Failed to decode event document: %v", err)
			continue
		}

		// Convert MongoDB timestamps (primitive.DateTime) to RFC3339 formatted strings
		if creationTime, ok := doc["creation_time"].(primitive.DateTime); ok {
			doc["creation_time"] = creationTime.Time().Format(time.RFC3339)
		}
		if lastUpdated, ok := doc["last_updated"].(primitive.DateTime); ok {
			doc["last_updated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		events = append(events, doc)
	}

	// Check for cursor errors
	if err := cursor.Err(); err != nil {
		log.Printf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return events, nil
}

func getEventByID(ctx context.Context, eventID string) (*Event, error) {
	collection := client.Database(os.Getenv("DB_COLLECTION")).Collection("events")

	// Convert eventID to ObjectID if it's stored as an ObjectID
	objID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		log.Printf("Invalid event ID format: %v", err)
		return nil, fmt.Errorf("invalid event ID format: %w", err)
	}

	// Fetch document by ID
	var event Event
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("Event %s not found in MongoDB", eventID)
			return nil, nil // Return nil to indicate no result
		}
		log.Printf("Error fetching event %s: %v", eventID, err)
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	// Convert MongoDB timestamps to time.Time (if needed)
	event.LastUpdated = primitive.NewDateTimeFromTime(event.LastUpdated.Time())

	return &event, nil
}
