package ruledownload

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client      *mongo.Client
	validAPIKey string
)

func Init(ctx context.Context) error {
	var err error

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		return fmt.Errorf("missing MONGO_URI environment variable")
	}

	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		return fmt.Errorf("missing MONGO_DB environment variable")
	}

	validAPIKey = os.Getenv("API_KEY")
	if validAPIKey == "" {
		return fmt.Errorf("missing API_KEY environment variable")
	}

	if err := loadXSRFConfig(); err != nil {
		return err
	}

	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb connection failed: %w", err)
	}

	if err := ensureIndexes(ctx, client, dbName); err != nil {
		return fmt.Errorf("failed to ensure indexes: %w", err)
	}

	Infof("ruledownload initialized with batchSize=%d", getBatchSize())
	return nil
}

func Close(ctx context.Context) error {
	if client == nil {
		return nil
	}
	return client.Disconnect(ctx)
}

func ensureIndexes(ctx context.Context, client *mongo.Client, dbName string) error {
	devicesColl := client.Database(dbName).Collection("devices")
	rulesColl := client.Database(dbName).Collection("rules")

	_, err := devicesColl.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "machine_id", Value: 1},
		},
		Options: options.Index().SetName("machine_id_idx").SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create devices index: %w", err)
	}

	_, err = rulesColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "scope", Value: 1},
				{Key: "last_updated", Value: 1},
				{Key: "identifier", Value: 1},
			},
			Options: options.Index().SetName("scope_lastupdated_identifier_idx"),
		},
		{
			Keys: bson.D{
				{Key: "scope", Value: 1},
				{Key: "assigned", Value: 1},
				{Key: "last_updated", Value: 1},
				{Key: "identifier", Value: 1},
			},
			Options: options.Index().SetName("scope_assigned_lastupdated_identifier_idx"),
		},
		{
			Keys: bson.D{
				{Key: "removed_from", Value: 1},
				{Key: "last_updated", Value: 1},
			},
			Options: options.Index().SetName("removedfrom_lastupdated_idx").SetSparse(true),
		},
	})
	if err != nil {
		return fmt.Errorf("create rules indexes: %w", err)
	}

	return nil
}
