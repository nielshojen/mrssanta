package eventupload

import (
	"context"
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// var client *firestore.Client
// var firestoreDatabasePath string

var client *mongo.Client

var validAPIKey = os.Getenv("API_KEY")

func init() {
	log.SetFlags(0)

	var err error
	// ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable is not set")
	}

	// firestoreDatabasePath = os.Getenv("FIRESTORE_DATABASE")
	// if firestoreDatabasePath == "" {
	// 	log.Fatal("FIRESTORE_DATABASE environment variable is not set")
	// }

	// dbPrefix := os.Getenv("DB_PREFIX")
	// if dbPrefix == "" {
	// 	log.Fatal("DB_PREFIX environment variable is not set")
	// }

	// client, err = firestore.NewClientWithDatabase(ctx, projectID, firestoreDatabasePath)
	// if err != nil {
	// 	log.Fatalf("Error initializing Cloud Firestore client: %v", err)
	// }

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("Missing MONGO_URI environment variable")
	}

	println("Connecting using URI: ", mongoURI)

	client, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}

	functions.HTTP("eventupload", eventuploadHandler)
}
