package api

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func getAllEvents(ctx context.Context) ([]map[string]interface{}, error) {
	collection := client.Database(dbName).Collection("events")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		Errorf("Failed to retrieve events: %v", err)
		return nil, fmt.Errorf("failed to retrieve events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			Warnf("Failed to decode event document: %v", err)
			continue
		}

		if creationTime, ok := doc["creation_time"].(bson.DateTime); ok {
			doc["creation_time"] = creationTime.Time().Format(time.RFC3339)
		}

		if lastUpdated, ok := doc["last_updated"].(bson.DateTime); ok {
			doc["last_updated"] = lastUpdated.Time().Format(time.RFC3339)
		}

		events = append(events, doc)
	}

	if err := cursor.Err(); err != nil {
		Errorf("Cursor error: %v", err)
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	return events, nil
}

func getEventByID(ctx context.Context, eventID string) (*Event, error) {
	collection := client.Database(dbName).Collection("events")

	var event Event
	err := collection.FindOne(ctx, bson.M{"_id": eventID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}

	return &event, nil
}
