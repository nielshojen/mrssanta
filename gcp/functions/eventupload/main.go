package eventupload

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	log.SetFlags(0)

	var err error
	ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable is not set")
	}

	dbPrefix := os.Getenv("DB_PREFIX")
	if dbPrefix == "" {
		log.Fatal("DB_PREFIX environment variable is not set")
	}

	client, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Error initializing Cloud Firestore client: %v", err)
	}

	functions.HTTP("eventupload", eventuploadHandler)
}
