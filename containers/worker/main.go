package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// The worker runs the background jobs that aren't part of handling a
// request. Each job checks its own settings and does nothing while it is
// turned off.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		Fatalf("Missing MONGO_URI environment variable")
	}
	dbName := os.Getenv("MONGO_DB")
	if dbName == "" {
		Fatalf("Missing MONGO_DB environment variable")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		Fatalf("MongoDB connection failed: %v", err)
	}

	db := client.Database(dbName)
	malwareBazaar := &bazaarJob{
		store:   &bazaarMongoStore{db: db},
		exports: newExportsClient(),
		now:     func() time.Time { return time.Now().UTC() },
	}

	jobs := []job{
		{name: "MalwareBazaar", interval: bazaarCycleInterval, runOnce: malwareBazaar.runOnce},
	}
	for _, j := range jobs {
		Infof("Running %s every %s", j.name, j.interval)
	}
	runJobs(ctx, jobs)
}
