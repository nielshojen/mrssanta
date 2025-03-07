package api

import (
	"context"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// var client *firestore.Client
// var firestoreDatabasePath string

var client *mongo.Client

var validAPIKey = os.Getenv("API_KEY")

func init() {
	var err error
	// ctx := context.Background()

	projectID := os.Getenv("GCP_PROJECT")
	if projectID == "" {
		log.Fatal("GCP_PROJECT environment variable is not set")
	}

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

	functions.HTTP("api", apiHandler)
}
