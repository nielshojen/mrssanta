package eventupload

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/firestore"
)

var client *firestore.Client

func saveEvent(ctx context.Context, client *firestore.Client, event Event) (Event, error) {
	// Check if file_sha256 field exists and is a string
	if event.FileSha256 == "" {
		return Event{}, fmt.Errorf("no file_sha256 field in the event")
	}

	// Add Timestamp
	event.LastUpdated = time.Now()

	// Set data in Firestore
	_, err := client.Collection(os.Getenv("DB_PREFIX")+"_binaries").Doc(event.FileSha256).Set(ctx, event)
	if err != nil {
		return Event{}, fmt.Errorf("failed to store data in Firestore: %v", err)
	}

	return event, nil
}
