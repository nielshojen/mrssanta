package ruledownload

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var client *firestore.Client
var firestoreDatabasePath string

var validAPIKey = os.Getenv("API_KEY")

const batchSize = 100

func init() {
	var err error
	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		panic("GCP_PROJECT environment variable is not set")
	}

	firestoreDatabasePath = os.Getenv("FIRESTORE_DATABASE")
	if firestoreDatabasePath == "" {
		log.Fatal("FIRESTORE_DATABASE environment variable is not set")
	}

	client, err = firestore.NewClientWithDatabase(ctx, projectID, firestoreDatabasePath)
	if err != nil {
		log.Fatalf("Error initializing Cloud Firestore client: %v", err)
	}

	functions.HTTP("ruledownload", ruledownloadHandler)
}
