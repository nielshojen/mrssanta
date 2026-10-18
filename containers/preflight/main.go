package preflight

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// defaultCleanSyncMaxAgeDays is how many days a device may go without a
// completed clean sync before preflight forces a CLEAN_ALL.
const defaultCleanSyncMaxAgeDays = 30

var (
	client               *mongo.Client
	validAPIKey          string
	dbName               string
	syncIntervalLockdown uint32 = 60
	syncIntervalMonitor  uint32 = 1800
	cleanSyncMaxAge             = cleanSyncMaxAgeFromDays(defaultCleanSyncMaxAgeDays)
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

	if err := loadXSRFConfig(); err != nil {
		return err
	}

	overrideUint32FromEnv(&syncIntervalLockdown, "SYNC_INTERVAL_LOCKDOWN")
	overrideUint32FromEnv(&syncIntervalMonitor, "SYNC_INTERVAL_MONITOR")
	cleanSyncMaxAge = parseCleanSyncMaxAge(os.Getenv("CLEAN_SYNC_MAX_AGE_DAYS"))

	client, err = mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb connection failed: %w", err)
	}

	Infof(
		"preflight initialized with lockdown interval=%d, monitor interval=%d and clean sync max age=%s",
		syncIntervalLockdown,
		syncIntervalMonitor,
		cleanSyncMaxAge,
	)

	return nil
}

func Close(ctx context.Context) error {
	if client == nil {
		return nil
	}

	return client.Disconnect(ctx)
}

func overrideUint32FromEnv(target *uint32, envKey string) {
	valueStr := os.Getenv(envKey)
	if valueStr == "" {
		return
	}

	value, err := strconv.ParseUint(valueStr, 10, 32)
	if err != nil {
		Warnf("Invalid value for %s (%s) - keeping default %d", envKey, valueStr, *target)
		return
	}

	*target = uint32(value)
	Infof("%s overridden via env: %d", envKey, *target)
}

// parseCleanSyncMaxAge converts the CLEAN_SYNC_MAX_AGE_DAYS value into a
// duration. An empty, malformed or zero value falls back to the default.
func parseCleanSyncMaxAge(valueStr string) time.Duration {
	if valueStr == "" {
		return cleanSyncMaxAgeFromDays(defaultCleanSyncMaxAgeDays)
	}

	days, err := strconv.ParseUint(valueStr, 10, 16)
	if err != nil || days == 0 {
		Warnf("Invalid value for CLEAN_SYNC_MAX_AGE_DAYS (%s) - keeping default %d days",
			valueStr, defaultCleanSyncMaxAgeDays)
		return cleanSyncMaxAgeFromDays(defaultCleanSyncMaxAgeDays)
	}

	Infof("CLEAN_SYNC_MAX_AGE_DAYS overridden via env: %d", days)
	return cleanSyncMaxAgeFromDays(days)
}

func cleanSyncMaxAgeFromDays(days uint64) time.Duration {
	return time.Duration(days) * 24 * time.Hour
}
