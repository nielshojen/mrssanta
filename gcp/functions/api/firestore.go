package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func saveRule(ctx context.Context, client *firestore.Client, w http.ResponseWriter, rule *Rule) {
	rulesCollection := client.Collection(os.Getenv("DB_PREFIX") + "_rules")
	docRef := rulesCollection.Doc(rule.Identifier)

	// Retrieve the existing document (if it exists)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		// If the document does NOT exist, set CreationTime
		if status.Code(err) == codes.NotFound {
			rule.CreationTime = time.Now()
		} else {
			// If there's another Firestore error, return
			log.Printf("Failed to check existing rule: %v", err)
			http.Error(w, "Failed to check existing rule", http.StatusInternalServerError)
			return
		}
	} else {
		// If the document exists, preserve its CreationTime
		var existingRule Rule
		if err := docSnap.DataTo(&existingRule); err != nil {
			log.Printf("Failed to decode existing rule: %v", err)
			http.Error(w, "Failed to decode existing rule", http.StatusInternalServerError)
			return
		}
		rule.CreationTime = existingRule.CreationTime // Preserve original timestamp
	}

	// Always update LastUpdated to current time
	rule.LastUpdated = time.Now()

	// Save rule to Firestore
	_, err = docRef.Set(ctx, rule)
	if err != nil {
		log.Printf("Failed to update collection: %v", err)
		http.Error(w, "Failed to update collection", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
