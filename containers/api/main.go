package api

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	client      *mongo.Client
	validAPIKey string
	dbName      string
)

func Init(ctx context.Context) error {
	var err error

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("missing MONGO_URI environment variable")
	}

	dbName = os.Getenv("MONGO_DB")
	if dbName == "" {
		return fmt.Errorf("missing MONGO_DB environment variable")
	}

	validAPIKey = os.Getenv("API_KEY")
	if validAPIKey == "" {
		return fmt.Errorf("missing API_KEY environment variable")
	}

	client, err = mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb connection failed: %w", err)
	}

	return nil
}

func Close(ctx context.Context) error {
	if client == nil {
		return nil
	}

	return client.Disconnect(ctx)
}
