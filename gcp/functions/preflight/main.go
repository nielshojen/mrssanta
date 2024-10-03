package preflight

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var client *firestore.Client
var firestoreDatabasePath string

func init() {
	var err error
	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable is not set")
	}

	firestoreDatabasePath = os.Getenv("FIRESTORE_DATABASE")
	if firestoreDatabasePath == "" {
		log.Fatal("FIRESTORE_DATABASE environment variable is not set")
	}

	dbPrefix := os.Getenv("DB_PREFIX")
	if dbPrefix == "" {
		log.Fatal("DB_PREFIX environment variable is not set")
	}

	client, err = firestore.NewClientWithDatabase(ctx, projectID, firestoreDatabasePath)
	if err != nil {
		log.Fatalf("error initializing Cloud Firestore client: %v", err)
	}

	functions.HTTP("preflight", preflightHandler)
}
