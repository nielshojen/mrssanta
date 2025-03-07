package ruledownload

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

const batchSize = 100

func init() {
	var err error
	// ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		panic("GCP_PROJECT environment variable is not set")
	}

	// firestoreDatabasePath = os.Getenv("FIRESTORE_DATABASE")
	// if firestoreDatabasePath == "" {
	// 	log.Fatal("FIRESTORE_DATABASE environment variable is not set")
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
	functions.HTTP("ruledownload", ruledownloadHandler)
}
