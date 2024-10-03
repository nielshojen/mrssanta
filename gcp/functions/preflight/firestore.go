package preflight

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
)

func saveDevice(ctx context.Context, client *firestore.Client, w http.ResponseWriter, device *Device, machineID string) {

	device.LastUpdated = time.Now()

	_, err := client.Collection(os.Getenv("DB_PREFIX")+"_devices").Doc(machineID).Set(ctx, &device)
	if err != nil {
		log.Printf("Failed to update collection: %v", err)
		http.Error(w, "Failed to update collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
