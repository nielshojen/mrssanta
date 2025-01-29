package preflight

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func saveDevice(ctx context.Context, device *Device, machineID string) error {
	// Reference Firestore document
	docRef := client.Collection(os.Getenv("DB_PREFIX") + "_devices").Doc(machineID)

	// Save the device (Firestore will create or update it)
	_, err := docRef.Set(ctx, device)
	if err != nil {
		log.Printf("Failed to save device data: %v", err)
		return fmt.Errorf("failed to save device data: %w", err)
	}
	return nil
}

func getDevice(ctx context.Context, machineID string) (*Device, error) {
	// Reference Firestore document
	docRef := client.Collection(os.Getenv("DB_PREFIX") + "_devices").Doc(machineID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		// Return nil if the document does not exist
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		log.Printf("Failed to retrieve existing device: %v", err)
		return nil, fmt.Errorf("failed to retrieve existing device: %w", err)
	}

	// Unmarshal into Device struct
	var existingDevice Device
	if err := docSnap.DataTo(&existingDevice); err != nil {
		log.Printf("Failed to decode existing device: %v", err)
		return nil, fmt.Errorf("failed to decode existing device: %w", err)
	}

	return &existingDevice, nil
}
